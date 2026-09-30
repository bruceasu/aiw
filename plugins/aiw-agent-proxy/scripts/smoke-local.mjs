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
const realFetch = globalThis.fetch;
globalThis.fetch = async (url, options) => {
  if (String(url) !== "https://api.openai.com/v1/responses") return realFetch(url, options);
  assert.equal(options.headers.authorization, "Bearer smoke-only");
  return new Response(JSON.stringify({ output_text: "synthetic result", usage: { input_tokens: 3, output_tokens: 2 } }), {
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
const service = await startService(port);

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
  assert.equal((await http.json()).result, "synthetic result");

  for (const provider of ["codex", "copilot"]) {
    const response = await realFetch(`http://127.0.0.1:${port}/v1/requests`, {
      method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ ...request, provider }),
    });
    assert.equal(response.status, 503);
    assert.equal((await response.json()).error.code, "provider_disabled");
  }

  const first = connect();
  await first.ready;
  first.socket.send(JSON.stringify({ type: "request", request: { ...request, request_id: "smoke-ws" } }));
  const result = await first.next();
  assert.equal(result.type, "result");
  assert.equal(result.result.text, "synthetic result");
  first.socket.close();
  await once(first.socket, "close");

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
  assert.equal(audit.trim().split("\n").length, 4);
  process.stdout.write("local smoke passed\n");
} finally {
  await service.close();
  globalThis.fetch = realFetch;
  await rm(state, { recursive: true, force: true });
}
