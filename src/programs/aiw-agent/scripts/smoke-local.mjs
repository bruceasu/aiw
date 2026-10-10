import assert from "node:assert/strict";
import { createServer } from "node:net";
import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { once } from "node:events";
import WebSocket from "ws";

const state = await mkdtemp(join(tmpdir(), "aiw-agent-proxy-smoke-"));
process.env.AIW_AGENT_PROXY_STATE_DIR = state;
process.env.OPENAI_API_KEY = "smoke-only";
const openaiUsage = {
  input_tokens: 3, input_tokens_details: { cached_tokens: 1, cache_write_tokens: 0 },
  output_tokens: 2, output_tokens_details: { reasoning_tokens: 1 }, total_tokens: 5,
};
const expectedDetails = {
  response_id: "resp_smoke", response_model: "smoke-model", service_tier: "default",
  response_status: "completed", total_tokens: 5, cached_input_tokens: 1,
  cache_write_input_tokens: 0, reasoning_output_tokens: 1,
};
const realFetch = globalThis.fetch;
globalThis.fetch = async (url, options) => {
  if (String(url) !== "https://api.openai.com/v1/responses") return realFetch(url, options);
  assert.equal(options.headers.authorization, "Bearer smoke-only");
  const empty = JSON.parse(options.body).input === "empty output";
  return new Response(JSON.stringify({
    id: "resp_smoke", model: "smoke-model", service_tier: "default", status: "completed",
    output_text: empty ? "" : "synthetic result", usage: openaiUsage,
  }), {
    status: 200, headers: { "content-type": "application/json" },
  });
};

const { startService } = await import("../dist/service.js");
const { ResultStore } = await import("../dist/store.js");
const listener = createServer();
listener.listen(0, "127.0.0.1");
await once(listener, "listening");
const port = listener.address().port;
await new Promise((resolve) => listener.close(resolve));
let service = await startService(port);

function connect() {
  const socket = new WebSocket(`ws://127.0.0.1:${port}/v1/events?client_id=smoke`);
  const queue = [];
  const waiters = [];
  socket.on("message", (raw) => {
    const event = JSON.parse(raw.toString());
    if (waiters.length) waiters.shift()(event);
    else queue.push(event);
  });
  return {
    socket,
    ready: once(socket, "open"),
    next: () => queue.length ? Promise.resolve(queue.shift()) : Promise.race([
      new Promise((resolve) => waiters.push(resolve)),
      new Promise((_, reject) => setTimeout(() => reject(new Error("WebSocket event timeout")), 5000)),
    ]),
  };
}

try {
  const request = { client_id: "smoke", request_id: "smoke-1", provider: "openai", model: "smoke-model", output_format: "markdown", prompt: "synthetic prompt" };
  const http = await realFetch(`http://127.0.0.1:${port}/v1/requests`, {
    method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify(request),
  });
  assert.equal(http.status, 200);
  const httpResult = await http.json();
  assert.equal(httpResult.result, "synthetic result");
  assert.deepEqual(httpResult.usage.openai_usage, expectedDetails);

  const failed = await realFetch(`http://127.0.0.1:${port}/v1/requests`, {
    method: "POST", headers: { "content-type": "application/json" },
    body: JSON.stringify({ ...request, request_id: "smoke-no-text", prompt: "empty output" }),
  });
  assert.equal(failed.status, 502);
  assert.equal((await failed.json()).error.code, "provider_error");

  const first = connect();
  await first.ready;
  first.socket.send(JSON.stringify({ type: "request", request: { ...request, request_id: "smoke-ws" } }));
  const result = await first.next();
  assert.equal(result.type, "result");
  assert.equal(result.result.text, "synthetic result");
  assert.deepEqual(result.result.usage.openai_usage, expectedDetails);
  first.socket.close();
  await once(first.socket, "close");
  await service.close();
  service = await startService(port);

  const second = connect();
  await second.ready;
  const replay = await second.next();
  assert.equal(replay.event_id, result.event_id);
  second.socket.send(JSON.stringify({ type: "ack", event_id: result.event_id }));
  const ack = await second.next();
  assert.equal(ack.type, "acknowledged");
  assert.equal(ack.deleted, true);
  second.socket.close();
  await once(second.socket, "close");

  const store = new ResultStore();
  assert.equal((await store.pending("smoke")).length, 0);
  const expiring = await store.save("expiring", "smoke", { text: "synthetic", usage: { input_tokens: null, output_tokens: null, cost: null } });
  const resultPath = join(state, "results", `${expiring.event_id}.json`);
  await writeFile(resultPath, JSON.stringify({ ...expiring, expires_at: new Date(0).toISOString() }));
  await store.expire();
  assert.equal((await store.pending("smoke")).length, 0);

  const audit = await readFile(join(state, "audit.jsonl"), "utf8");
  assert.equal(audit.includes("synthetic prompt"), false);
  assert.equal(audit.includes("synthetic result"), false);
  assert.equal(audit.includes("smoke-only"), false);
  assert.equal(audit.includes("empty output"), false);
  const records = audit.trim().split("\n").map((line) => JSON.parse(line));
  assert.equal(records.length, 3);
  for (const record of records) {
    assert.ok(Date.parse(record.timestamp));
    assert.equal(record.client_id, "smoke");
    assert.equal(record.provider, "openai");
    assert.equal(record.model, "smoke-model");
    assert.ok(record.duration_ms >= 0);
    assert.equal(record.input_tokens, 3);
    assert.equal(record.output_tokens, 2);
    assert.equal(record.cost, null);
    assert.deepEqual(record.openai_usage, expectedDetails);
  }
  assert.equal(records[0].request_id, "smoke-1");
  assert.equal(records[0].status, "succeeded");
  assert.equal(records[1].request_id, "smoke-no-text");
  assert.equal(records[1].status, "failed");
  assert.equal(records[1].error_code, "provider_error");
  assert.equal(records[2].request_id, "smoke-ws");
  assert.equal(records[2].status, "succeeded");
  assert.equal(records[2].event_id, result.event_id);
  process.stdout.write("local smoke passed\n");
} finally {
  await service.close();
  globalThis.fetch = realFetch;
  await rm(state, { recursive: true, force: true });
}
