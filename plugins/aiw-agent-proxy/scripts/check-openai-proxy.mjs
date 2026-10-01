import { randomUUID } from "node:crypto";
import { once } from "node:events";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { createServer } from "node:net";
import { tmpdir } from "node:os";
import { join } from "node:path";

const apiKey = process.env.OPENAI_API_KEY;
if (!apiKey) {
  console.error("OPENAI_API_KEY is not set.");
  process.exitCode = 2;
} else {
  const state = await mkdtemp(join(tmpdir(), "aiw-openai-proxy-check-"));
  process.env.AIW_AGENT_PROXY_STATE_DIR = state;
  let service;
  try {
    const { startService } = await import("../dist/service.js");
    const listener = createServer();
    listener.listen(0, "127.0.0.1");
    await once(listener, "listening");
    const port = listener.address().port;
    await new Promise((resolve) => listener.close(resolve));
    service = await startService(port);

    const requestId = randomUUID();
    const prompt = "Reply with OK only.";
    const model = process.env.OPENAI_MODEL || "gpt-6-luna";
    const response = await fetch(`http://127.0.0.1:${port}/v1/requests`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        client_id: "manual-openai-check", request_id: requestId,
        provider: "openai", model, output_format: "markdown", prompt,
      }),
      signal: AbortSignal.timeout(65_000),
    });
    const body = await response.json();
    console.log(`HTTP ${response.status}; model=${model}; text_received=${Boolean(body.result)}`);
    if (!response.ok) {
      console.log(`error_code=${body.error?.code || "unknown"}`);
      process.exitCode = 1;
    } else {
      console.log(`input_tokens=${body.usage?.input_tokens ?? "unknown"}; output_tokens=${body.usage?.output_tokens ?? "unknown"}`);
      const audit = await readFile(join(state, "audit.jsonl"), "utf8");
      const record = audit.trim().split("\n").map((line) => JSON.parse(line))
        .find((row) => row.request_id === requestId);
      const clean = !audit.includes(apiKey) && !audit.includes(prompt)
        && !audit.includes(body.result);
      const usageMatch = JSON.stringify(record?.openai_usage) === JSON.stringify(body.usage?.openai_usage);
      console.log(`audit_status=${record?.status || "missing"}; audit_redacted=${clean}`);
      console.log(`audit_input_tokens=${record?.input_tokens ?? "unknown"}; audit_output_tokens=${record?.output_tokens ?? "unknown"}`);
      console.log(`openai_usage=${JSON.stringify(body.usage?.openai_usage ?? null)}`);
      console.log(`audit_usage_match=${usageMatch}`);
      if (!body.result || record?.status !== "succeeded" || !clean || !usageMatch) process.exitCode = 1;
    }
  } catch {
    console.error("Agent Proxy check failed. Ensure the plugin has been built.");
    process.exitCode = 1;
  } finally {
    if (service) await service.close();
    await rm(state, { recursive: true, force: true });
  }
}
