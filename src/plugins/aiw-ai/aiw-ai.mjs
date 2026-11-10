#!/usr/bin/env node
import { open } from "node:fs/promises";
import { spawnSync } from "node:child_process";
import { request as httpRequest } from "node:http";
import { request as httpsRequest } from "node:https";
import { join, resolve } from "node:path";
import { createInterface } from "node:readline";
import { editPrompt } from "./editor.mjs";

const MAX_PROMPT_CODE_POINTS = 64 * 1024;
const MAX_STDIN_BYTES = 4 * MAX_PROMPT_CODE_POINTS;
const MAX_PROMPT_FILE_BYTES = 4 * MAX_PROMPT_CODE_POINTS;
const MAX_SYSTEM_FILE_BYTES = 64 * 1024;
const MAX_RESPONSE_BYTES = 4 * 1024 * 1024;
// Keep synchronized with maxBridgeOutputBytes in the Go AIW Provider bridge.
const MAX_BRIDGE_OUTPUT_BYTES = 25 * 1024 * 1024;
const DEFAULT_TIMEOUT_SECONDS = 660;
const DEFAULT_MODEL = "gpt-6-luna";
const HELP = `用法：aiw ai [--backend aiw|gateway] [选项] ["提示词"|-]

发送一次请求。默认使用 AIW [ai] 配置的 Provider；可显式选择 Agent Gateway。需要 Node.js 22.12+。

路由：
  --backend aiw|gateway  后端选择，默认 aiw。仅提供 --url 时兼容旧用法并选择 gateway。
  AIW 路由             使用 aiw.toml [ai] 与 AIW_LLM_* 配置。
  Gateway 路由         需要网关 Key，且模型须由 Gateway 配置并授权。

选项：
  -h, --help            显示帮助，不调用模型。
  --url <地址>          Gateway HTTP(S) origin；未附 --backend 时也会选择 Gateway。
  --system <文本|@文件> 系统提示词；@文件在本机以 UTF-8 读取。
  --file <路径>         将文件内容作为请求输入，可与位置提示词组合。
  --encoding <标签>     文件编码，默认 utf-8；仅用于 --file。
  --timeout-seconds <秒> 请求总期限，1–3660 的十进制整数；默认 660。
  --json                要求并输出单个 JSON 对象。
  --verbose             向 stderr 显示可得的路由、模型和用量信息。
  --                    后续内容按提示词处理。

Gateway 设置：
  AIW_AI_API_KEY        网关 Key（至少 43 字符，无空白）。
  AIW_AI_MODEL          网关逻辑模型，默认 ${DEFAULT_MODEL}。
  AIW_AGENT_PROXY_PORT  本机网关端口，默认 43127；默认地址为 http://127.0.0.1:43127。
  网关须启动，服务端 principals[].keys 与客户端使用相同 Key，模型须已映射并获授权。

输入：非选项参数按空格组成提示词；单独的 - 读取完整 stdin。--file 与 - 不能组合。
文件默认严格按 UTF-8 解码，可用 Node TextDecoder 支持的编码标签覆盖；最多 256 KiB。
文件内容追加在位置提示词之后，以空行分隔；总提示词最多 65536 个 Unicode 码点。
无参数且未指定文件时读取一行终端输入；Ctrl+E 用 EDITOR 编辑当前文字，支持多行。
空白/EOF 不发送请求。文件路径不会发送给 Provider；只发送解码后的请求文本。
stdout 输出结果，stderr 输出错误和诊断，不自动切换路由或重试。

示例：
  aiw ai "How to learn English"
  aiw ai --backend aiw "Use the configured AIW Provider"
  aiw ai --file report.txt "Summarize this report"
  aiw ai --file legacy.txt --encoding gb18030 "Summarize this report"
  aiw ai --backend gateway "Explain this text"
  aiw ai --backend gateway --url https://gateway.example "Explain this text"
  aiw ai --system @instructions.txt --json --verbose "Return a JSON object"
  Get-Content input.txt -Raw | aiw ai -

PowerShell Gateway setup:
  $env:AIW_AI_API_KEY = '<Gateway Key>'
  aiw ai --backend gateway "How to learn English"

Linux Gateway setup:
  export AIW_AI_API_KEY='<Gateway Key>'
  aiw ai --backend gateway "How to learn English"
`;

class CliError extends Error {
  constructor(message, exitCode = 1) {
    super(message);
    this.exitCode = exitCode;
  }
}

function parseArgs(args) {
  let systemValue;
  let urlValue;
  let backendValue;
  let filePath;
  let encodingValue;
  let json = false;
  let verbose = false;
  let timeoutSeconds;
  const inputs = [];
  let optionsEnded = false;
  for (let index = 0; index < args.length; index++) {
    const arg = args[index];
    if (!optionsEnded && arg === "--") { optionsEnded = true; continue; }
    if (!optionsEnded && (arg === "--help" || arg === "-h")) return { help: true };
    if (!optionsEnded && arg === "--json") { json = true; continue; }
    if (!optionsEnded && arg === "--verbose") { verbose = true; continue; }
    if (!optionsEnded && arg === "--backend") {
      const value = args[++index];
      if (value === undefined || !["aiw", "gateway"].includes(value.toLowerCase())) {
        throw new CliError("--backend requires aiw or gateway", 2);
      }
      if (backendValue !== undefined) throw new CliError("--backend may be specified only once", 2);
      backendValue = value.toLowerCase();
      continue;
    }
    if (!optionsEnded && arg === "--file") {
      const value = args[++index];
      if (value === undefined || value === "" || value === "--") throw new CliError("--file requires a filename", 2);
      if (filePath !== undefined) throw new CliError("--file may be specified only once", 2);
      filePath = value;
      continue;
    }
    if (!optionsEnded && arg === "--encoding") {
      const value = args[++index];
      if (value === undefined || value === "" || value.startsWith("--")) throw new CliError("--encoding requires a TextDecoder encoding label", 2);
      if (encodingValue !== undefined) throw new CliError("--encoding may be specified only once", 2);
      encodingValue = value;
      continue;
    }
    if (!optionsEnded && arg === "--timeout-seconds") {
      const value = args[++index];
      if (value === undefined || !/^[0-9]+$/.test(value) || !Number.isSafeInteger(Number(value)) ||
          Number(value) < 1 || Number(value) > 3660) {
        throw new CliError("--timeout-seconds requires an integer from 1 to 3660", 2);
      }
      if (timeoutSeconds !== undefined) throw new CliError("--timeout-seconds may be specified only once", 2);
      timeoutSeconds = Number(value);
      continue;
    }
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
    inputs.push(arg);
  }
  if (inputs.length > 1 && inputs.includes("-")) throw new CliError("- must be the only prompt argument", 2);
  if (inputs.includes("-") && filePath !== undefined) throw new CliError("--file cannot be combined with stdin input '-'", 2);
  if (encodingValue !== undefined && filePath === undefined) throw new CliError("--encoding requires --file", 2);
  const backend = backendValue ?? (urlValue !== undefined ? "gateway" : "aiw");
  if (backend === "aiw" && urlValue !== undefined) throw new CliError("--url is available only with --backend gateway", 2);
  const input = inputs.length ? inputs.join(" ") : undefined;
  return {
    help: false, backend, systemValue, urlValue, filePath, encodingValue,
    json, verbose, timeoutSeconds: timeoutSeconds ?? DEFAULT_TIMEOUT_SECONDS, input,
  };
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
  return new URL("/v1/responses", url).href;
}

function decodeUtf8(bytes, label) {
  try { return new TextDecoder("utf-8", { fatal: true }).decode(bytes); }
  catch { throw new CliError(`${label} is not valid UTF-8`, 2); }
}

function unicodeCodePointLength(value) {
  let count = 0;
  for (const _ of value) count++;
  return count;
}

async function readSystem(value) {
  if (value === undefined) return undefined;
  if (!value.startsWith("@")) {
    if (!value.trim() || unicodeCodePointLength(value) > MAX_PROMPT_CODE_POINTS) throw new CliError("system prompt must contain 1-65536 Unicode code points", 2);
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
  if (!content.trim() || unicodeCodePointLength(content) > MAX_PROMPT_CODE_POINTS) throw new CliError("system prompt file is empty or too long", 2);
  return content;
}

async function readPromptFile(filename, encodingValue) {
  let handle;
  let bytes;
  try {
    handle = await open(resolve(process.cwd(), filename), "r");
    const buffer = Buffer.alloc(MAX_PROMPT_FILE_BYTES + 1);
    let size = 0;
    while (size < buffer.length) {
      const result = await handle.read(buffer, size, buffer.length - size, size);
      if (!result.bytesRead) break;
      size += result.bytesRead;
    }
    bytes = buffer.subarray(0, size);
  } catch {
    throw new CliError("cannot read prompt input file", 2);
  } finally {
    await handle?.close();
  }
  if (bytes.length > MAX_PROMPT_FILE_BYTES) throw new CliError("prompt input file exceeds 256 KiB", 2);
  const label = encodingValue ?? "utf-8";
  let decoder;
  try { decoder = new TextDecoder(label, { fatal: true }); }
  catch { throw new CliError(`unsupported file encoding: ${label}`, 2); }
  try { return decoder.decode(bytes); }
  catch { throw new CliError(`prompt input file is not valid ${label}`, 2); }
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
  return await new Promise((done, reject) => {
    const rl = createInterface({ input: process.stdin, output: process.stderr });
    let editing = false;
    function onKeypress(_text, key) {
      if (!key?.ctrl || key.name !== "e" || editing) return;
      editing = true;
      const initial = rl.line;
      rl.close();
      process.stdin.pause();
      process.stderr.write("\n");
      editPrompt(initial, MAX_STDIN_BYTES).then(done, (error) => reject(new CliError(error.message, 2)));
    }
    process.stdin.on("keypress", onKeypress);
    rl.once("close", () => {
      process.stdin.off("keypress", onKeypress);
      if (!editing) done("");
    });
    rl.once("SIGINT", () => rl.close());
    rl.question("ai> （Enter 提交，Ctrl+E 编辑，Ctrl+C 取消） ", (answer) => { done(answer); rl.close(); });
  });
}

function config(urlValue) {
  const model = process.env.AIW_AI_MODEL || DEFAULT_MODEL;
  const apiKey = process.env.AIW_AI_API_KEY;
  if (!apiKey || apiKey.length < 43 || /\s/.test(apiKey)) throw new CliError(
    `请设置 AIW_AI_API_KEY 为网关 Key（至少 43 字符，无空白）。默认模型为 ${DEFAULT_MODEL}；网关需配置并授权该逻辑模型。运行 aiw ai --help 查看最少配置。`, 2);
  if (process.env.AIW_AI_PROVIDER) throw new CliError("AIW_AI_PROVIDER was removed; use a gateway logical model", 2);
  if (urlValue !== undefined) return { model, apiKey, endpoint: proxyEndpoint(urlValue) };
  const port = Number(process.env.AIW_AGENT_PROXY_PORT || "43127");
  if (!Number.isInteger(port) || port < 1 || port > 65535) throw new CliError("AIW_AGENT_PROXY_PORT must be 1-65535", 2);
  return { model, apiKey, endpoint: `http://127.0.0.1:${port}/v1/responses` };
}

async function readResponse(response) {
  const chunks = [];
  let size = 0;
  for await (const value of response) {
    size += value.byteLength;
    if (size > MAX_RESPONSE_BYTES) {
      response.destroy();
      throw new CliError("Agent Proxy response is too large");
    }
    chunks.push(Buffer.from(value));
  }
  try { return JSON.parse(Buffer.concat(chunks).toString("utf8")); }
  catch { throw new CliError("Agent Proxy returned invalid JSON"); }
}

function sendRequest(endpoint, apiKey, requestBody, signal) {
  return new Promise((resolve, reject) => {
    const url = new URL(endpoint);
    const send = url.protocol === "https:" ? httpsRequest : httpRequest;
    const request = send(url, {
      method: "POST",
      headers: { "content-type": "application/json", authorization: `Bearer ${apiKey}` },
      signal,
    }, async (response) => {
      try {
        const body = await readResponse(response);
        resolve({ ok: response.statusCode >= 200 && response.statusCode < 300, body });
      } catch (error) { reject(error); }
    });
    request.on("error", reject);
    request.end(JSON.stringify(requestBody));
  });
}

function shown(value) { return value === null || value === undefined ? "unknown" : String(value); }

function printVerbose(response, request) {
  const usage = response.usage && typeof response.usage === "object" ? response.usage : {};
  const inputDetails = usage.input_tokens_details || {};
  const outputDetails = usage.output_tokens_details || {};
  const lines = [
    "--- Request details ---",
    `Model: ${shown(response.model ?? request.model)}`,
    `Request ID: ${shown(response.id)}`,
    `Input tokens: ${shown(usage.input_tokens)}`,
    `Output tokens: ${shown(usage.output_tokens)}`,
    `Total tokens: ${shown(usage.total_tokens)}`,
    `Cached input tokens: ${shown(inputDetails.cached_tokens)}`,
    `Reasoning output tokens: ${shown(outputDetails.reasoning_tokens)}`,
    "Cost: unknown",
    "Reasoning summary:",
    typeof response.reasoning_summary === "string" && response.reasoning_summary
      ? response.reasoning_summary : "unavailable",
  ];
  process.stderr.write(`${lines.join("\n")}\n`);
}

function invokeAiwProvider(options, prompt, system) {
  const executableName = process.platform === "win32" ? "aiw.exe" : "aiw";
  const executable = process.env.AIW_ROOT
    ? resolve(process.env.AIW_ROOT, executableName)
    : executableName;
  const request = JSON.stringify({
    prompt,
    system_prompt: system ?? "",
    json: options.json,
    timeout_seconds: options.timeoutSeconds,
  });
  const result = spawnSync(executable, ["__aiw_ai_provider"], {
    input: request,
    encoding: "utf8",
    timeout: options.timeoutSeconds * 1000 + 5000,
    maxBuffer: MAX_BRIDGE_OUTPUT_BYTES,
    windowsHide: true,
  });
  if (result.error) {
    if (result.error.code === "ETIMEDOUT") throw new CliError("AIW Provider request timed out");
    if (result.error.code === "ENOENT") throw new CliError("AIW CLI is unavailable for the configured Provider route");
    if (result.error.code === "ENOBUFS") throw new CliError("AIW Provider response is too large");
    throw new CliError("could not invoke the AIW configured Provider");
  }
  if (result.status !== 0) {
    const diagnostic = String(result.stderr ?? "").trim().slice(0, 240);
    throw new CliError(diagnostic ? `AIW Provider failed: ${diagnostic}` : "AIW Provider request failed");
  }
  let response;
  try { response = JSON.parse(result.stdout); }
  catch { throw new CliError("AIW Provider bridge returned invalid JSON"); }
  if (!response || typeof response.text !== "string" ||
      typeof response.provider !== "string" ||
      (response.model !== undefined && typeof response.model !== "string")) {
    throw new CliError("AIW Provider bridge returned an invalid response");
  }
  return response;
}

function printProviderVerbose(response) {
  const lines = [
    "--- Request details ---",
    "Route: AIW configured Provider",
    `Configured provider: ${shown(response.provider)}`,
    `Configured model: ${shown(response.model)}`,
    "Request ID: unavailable",
    "Input tokens: unknown",
    "Output tokens: unknown",
    "Total tokens: unknown",
    "Cost: unknown",
    "Reasoning summary: unavailable",
  ];
  process.stderr.write(`${lines.join("\n")}\n`);
}

function writeResult(result, json) {
  if (json) {
    let value;
    try { value = JSON.parse(result); }
    catch { throw new CliError("AI provider returned invalid result JSON"); }
    if (value === null || Array.isArray(value) || typeof value !== "object") {
      throw new CliError("AI provider JSON result must be an object");
    }
    process.stdout.write(`${JSON.stringify(value, null, 2)}\n`);
  } else {
    process.stdout.write(result.endsWith("\n") ? result : `${result}\n`);
  }
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  if (options.help) {
    process.stdout.write(HELP);
    return;
  }
  if (process.env.AIW_AI_PROVIDER) throw new CliError("AIW_AI_PROVIDER was removed; use --backend and AIW_LLM_PROVIDER instead", 2);
  let prompt = options.input === "-" ? await readStdin()
    : options.input === undefined && options.filePath === undefined ? await readInteractive()
      : options.input ?? "";
  if (options.filePath !== undefined) {
    const content = await readPromptFile(options.filePath, options.encodingValue);
    prompt = prompt ? `${prompt}\n\n${content}` : content;
  }
  if (!prompt.trim()) return;
  if (unicodeCodePointLength(prompt) > MAX_PROMPT_CODE_POINTS) throw new CliError("input exceeds 65536 Unicode code points", 2);
  const system = await readSystem(options.systemValue);
  if (options.backend === "aiw") {
    const providerResponse = invokeAiwProvider(options, prompt, system);
    writeResult(providerResponse.text, options.json);
    if (options.verbose) printProviderVerbose(providerResponse);
    return;
  }

  const { model, apiKey, endpoint } = config(options.urlValue);
  const request = {
    model, input: prompt, store: false,
    text: { format: { type: options.json ? "json_object" : "text" } },
    ...(system === undefined ? {} : { instructions: system }),
  };
  const signal = AbortSignal.timeout(options.timeoutSeconds * 1000);
  let response;
  let body;
  try {
    ({ ok: response, body } = await sendRequest(endpoint, apiKey, request, signal));
  } catch (error) {
    if (error instanceof CliError) throw error;
    throw new CliError(signal.aborted || error?.name === "TimeoutError" || error?.name === "AbortError"
      ? "Agent Proxy request timed out" : "Agent Proxy is unavailable");
  }
  if (!response) {
    const code = typeof body?.error?.code === "string" ? body.error.code : "proxy_error";
    const message = typeof body?.error?.message === "string" ? body.error.message.slice(0, 240) : "request failed";
    throw new CliError(`Agent Proxy ${code}: ${message}`);
  }
  if (body?.object !== "response" || body.status !== "completed" || !Array.isArray(body.output)) throw new CliError("Agent Proxy returned an invalid response");
  const texts = body.output.filter((item) => item?.type === "message" && item.role === "assistant")
    .flatMap((item) => Array.isArray(item.content) ? item.content : [])
    .filter((part) => part?.type === "output_text" && typeof part.text === "string")
    .map((part) => part.text);
  if (!texts.length) throw new CliError("Agent Proxy returned no result text");
  const result = texts.join("\n");
  writeResult(result, options.json);
  if (options.verbose) printVerbose(body, request);
}

main().catch((error) => {
  const message = error instanceof CliError ? error.message : "unexpected client error";
  process.stderr.write(`ai: ${message}\n`);
  process.exitCode = error instanceof CliError ? error.exitCode : 1;
});
