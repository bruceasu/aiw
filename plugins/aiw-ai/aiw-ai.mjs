#!/usr/bin/env node
import { randomUUID } from "node:crypto";
import { open } from "node:fs/promises";
import { resolve } from "node:path";
import { createInterface } from "node:readline";

const MAX_PROMPT_CHARS = 64 * 1024;
const MAX_STDIN_BYTES = 4 * MAX_PROMPT_CHARS;
const MAX_SYSTEM_FILE_BYTES = 64 * 1024;
const MAX_RESPONSE_BYTES = 4 * 1024 * 1024;
const TIMEOUT_MS = 65_000;
const PROVIDERS = new Set(["openai", "codex", "copilot"]);

class CliError extends Error {
  constructor(message, exitCode = 1) {
    super(message);
    this.exitCode = exitCode;
  }
}

function parseArgs(args) {
  let systemValue;
  let urlValue;
  let json = false;
  let verbose = false;
  let input;
  let optionsEnded = false;
  for (let index = 0; index < args.length; index++) {
    const arg = args[index];
    if (!optionsEnded && arg === "--") { optionsEnded = true; continue; }
    if (!optionsEnded && arg === "--help") return { help: true };
    if (!optionsEnded && arg === "--json") { json = true; continue; }
    if (!optionsEnded && arg === "--verbose") { verbose = true; continue; }
    if (!optionsEnded && arg === "--url") {
      const value = args[++index];
      if (value === undefined || value === "" || value.startsWith("--")) throw new CliError("--url requires an HTTP(S) service URL", 2);
      if (urlValue !== undefined) throw new CliError("--url may be specified only once", 2);
      urlValue = value;
      continue;
    }
    if (!optionsEnded && arg === "--system") {
      const value = args[++index];
      if (value === undefined || value === "" || value.startsWith("--")) throw new CliError("--system requires content or @<file>", 2);
      if (systemValue !== undefined) throw new CliError("--system may be specified only once", 2);
      systemValue = value;
      continue;
    }
    if (!optionsEnded && arg.startsWith("-") && arg !== "-") throw new CliError(`unknown option: ${arg}`, 2);
    if (input !== undefined) throw new CliError("only one input argument is allowed", 2);
    input = arg;
  }
  return { help: false, systemValue, urlValue, json, verbose, input };
}

function proxyEndpoint(value) {
  if (!/^https?:\/\/[^/?#]+\/?$/i.test(value)) {
    throw new CliError("--url must be an HTTP(S) service origin without credentials, path, query, or fragment", 2);
  }
  let url;
  try { url = new URL(value); }
  catch { throw new CliError("--url must be an HTTP(S) service origin", 2); }
  if ((url.protocol !== "http:" && url.protocol !== "https:") || !url.hostname ||
      url.username || url.password || url.pathname !== "/" || url.search || url.hash) {
    throw new CliError("--url must be an HTTP(S) service origin without credentials, path, query, or fragment", 2);
  }
  return new URL("/v1/requests", url).href;
}

function decodeUtf8(bytes, label) {
  try { return new TextDecoder("utf-8", { fatal: true }).decode(bytes); }
  catch { throw new CliError(`${label} is not valid UTF-8`, 2); }
}

async function readSystem(value) {
  if (value === undefined) return undefined;
  if (!value.startsWith("@")) {
    if (!value.trim() || value.length > MAX_PROMPT_CHARS) throw new CliError("system prompt must contain 1-65536 characters", 2);
    return value;
  }
  if (value.length === 1) throw new CliError("--system @ requires a filename", 2);
  let handle;
  let bytes;
  try {
    handle = await open(resolve(process.cwd(), value.slice(1)), "r");
    const buffer = Buffer.alloc(MAX_SYSTEM_FILE_BYTES + 1);
    let size = 0;
    while (size < buffer.length) {
      const result = await handle.read(buffer, size, buffer.length - size, size);
      if (!result.bytesRead) break;
      size += result.bytesRead;
    }
    bytes = buffer.subarray(0, size);
  } catch {
    throw new CliError("cannot read system prompt file", 2);
  } finally {
    await handle?.close();
  }
  if (bytes.length > MAX_SYSTEM_FILE_BYTES) throw new CliError("system prompt file exceeds 64 KiB", 2);
  const content = decodeUtf8(bytes, "system prompt file");
  if (!content.trim() || content.length > MAX_PROMPT_CHARS) throw new CliError("system prompt file is empty or too long", 2);
  return content;
}

async function readStdin() {
  const chunks = [];
  let size = 0;
  for await (const chunk of process.stdin) {
    const bytes = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
    size += bytes.length;
    if (size > MAX_STDIN_BYTES) throw new CliError("standard input is too large", 2);
    chunks.push(bytes);
  }
  return decodeUtf8(Buffer.concat(chunks), "standard input");
}

async function readInteractive() {
  if (!process.stdin.isTTY) throw new CliError("interactive input requires a terminal; use - to read stdin", 2);
  return await new Promise((done) => {
    const rl = createInterface({ input: process.stdin, output: process.stderr });
    rl.once("close", () => done(""));
    rl.question("ai> ", (answer) => { done(answer); rl.close(); });
  });
}

function config(urlValue) {
  const provider = process.env.AIW_AI_PROVIDER || "openai";
  if (!PROVIDERS.has(provider)) throw new CliError("AIW_AI_PROVIDER must be openai, codex, or copilot", 2);
  const model = process.env.AIW_AI_MODEL || (provider === "openai" ? "gpt-6-luna" : "");
  if (!model) throw new CliError("set AIW_AI_MODEL for codex or copilot", 2);
  if (urlValue !== undefined) return { provider, model, endpoint: proxyEndpoint(urlValue) };
  const port = Number(process.env.AIW_AGENT_PROXY_PORT || "43127");
  if (!Number.isInteger(port) || port < 1 || port > 65535) throw new CliError("AIW_AGENT_PROXY_PORT must be 1-65535", 2);
  return { provider, model, endpoint: `http://127.0.0.1:${port}/v1/requests` };
}

async function readResponse(response) {
  if (!response.body) throw new CliError("Agent Proxy returned no response body");
  const reader = response.body.getReader();
  const chunks = [];
  let size = 0;
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    size += value.byteLength;
    if (size > MAX_RESPONSE_BYTES) {
      await reader.cancel();
      throw new CliError("Agent Proxy response is too large");
    }
    chunks.push(Buffer.from(value));
  }
  try { return JSON.parse(Buffer.concat(chunks).toString("utf8")); }
  catch { throw new CliError("Agent Proxy returned invalid JSON"); }
}

function shown(value) { return value === null || value === undefined ? "unknown" : String(value); }

function printVerbose(response, request) {
  const usage = response.usage && typeof response.usage === "object" ? response.usage : {};
  const details = usage.openai_usage && typeof usage.openai_usage === "object" ? usage.openai_usage : {};
  const lines = [
    "--- Request details ---",
    `Provider: ${shown(response.provider ?? request.provider)}`,
    `Model: ${shown(details.response_model ?? response.model ?? request.model)}`,
    `Request ID: ${shown(response.request_id ?? request.request_id)}`,
    `Input tokens: ${shown(usage.input_tokens)}`,
    `Output tokens: ${shown(usage.output_tokens)}`,
    `Total tokens: ${shown(details.total_tokens)}`,
    `Cached input tokens: ${shown(details.cached_input_tokens)}`,
    `Cache-write input tokens: ${shown(details.cache_write_input_tokens)}`,
    `Reasoning output tokens: ${shown(details.reasoning_output_tokens)}`,
    `Cost: ${shown(usage.cost)}`,
    "Reasoning summary:",
    typeof response.reasoning_summary === "string" && response.reasoning_summary
      ? response.reasoning_summary : "unavailable",
  ];
  process.stderr.write(`${lines.join("\n")}\n`);
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  if (options.help) {
    process.stdout.write("Usage: ai [--url <http(s)://host:port>] [--system <content|@file>] [--json] [--verbose] [input|-]\n");
    return;
  }
  const prompt = options.input === "-" ? await readStdin()
    : options.input === undefined ? await readInteractive() : options.input;
  if (!prompt.trim()) return;
  if (prompt.length > MAX_PROMPT_CHARS) throw new CliError("input exceeds 65536 characters", 2);
  const { provider, model, endpoint } = config(options.urlValue);
  if (options.systemValue !== undefined && provider !== "openai") throw new CliError("--system is supported only with OpenAI", 2);
  const system = await readSystem(options.systemValue);
  const request = {
    client_id: "ai-cli", request_id: randomUUID(), provider, model,
    output_format: options.json ? "json" : "markdown", prompt,
    ...(system === undefined ? {} : { system_prompt: system }),
    ...(options.verbose ? { include_reasoning_summary: true } : {}),
  };
  let response;
  let body;
  try {
    response = await fetch(endpoint, {
      method: "POST", headers: { "content-type": "application/json" },
      body: JSON.stringify(request), redirect: "error",
      signal: AbortSignal.timeout(TIMEOUT_MS),
    });
    body = await readResponse(response);
  } catch (error) {
    if (error instanceof CliError) throw error;
    throw new CliError(error?.name === "TimeoutError" || error?.name === "AbortError"
      ? "Agent Proxy request timed out" : "Agent Proxy is unavailable");
  }
  if (!response.ok) {
    const code = typeof body?.error?.code === "string" ? body.error.code : "proxy_error";
    const message = typeof body?.error?.message === "string" ? body.error.message.slice(0, 240) : "request failed";
    throw new CliError(`Agent Proxy ${code}: ${message}`);
  }
  if (typeof body?.result !== "string") throw new CliError("Agent Proxy returned no result text");
  if (options.json) {
    let value;
    try { value = JSON.parse(body.result); }
    catch { throw new CliError("Agent Proxy returned invalid result JSON"); }
    process.stdout.write(`${JSON.stringify(value, null, 2)}\n`);
  } else {
    process.stdout.write(body.result.endsWith("\n") ? body.result : `${body.result}\n`);
  }
  if (options.verbose) printVerbose(body, request);
}

main().catch((error) => {
  const message = error instanceof CliError ? error.message : "unexpected client error";
  process.stderr.write(`ai: ${message}\n`);
  process.exitCode = error instanceof CliError ? error.exitCode : 1;
});
