import { MAX_RESULT_CHARS, REQUEST_TIMEOUT_MS, STATE_DIR } from "./config.js";
import { ProxyError } from "./errors.js";
import type { Completion, RequestInput, Usage } from "./types.js";
import { delimiter, join } from "node:path";
import { mkdir } from "node:fs/promises";
import { existsSync } from "node:fs";
import { spawn } from "node:child_process";

const emptyUsage = (): Usage => ({ input_tokens: null, output_tokens: null, cost: null });
const safeInteger = (value: unknown): number | null => Number.isSafeInteger(value) && Number(value) >= 0 ? Number(value) : null;
const safeIdentifier = (value: unknown): string | null =>
  typeof value === "string" && /^[A-Za-z0-9._:/-]{1,128}$/.test(value) ? value : null;
const MAX_REASONING_SUMMARY_CHARS = 16 * 1024;

export async function complete(input: RequestInput): Promise<Completion> {
  let result: Completion;
  if (input.system_prompt && input.provider !== "openai") throw new ProxyError("unsupported_option", "system_prompt is supported only by OpenAI", 400);
  if (input.provider === "codex") result = await completeCodex(input);
  else if (input.provider === "copilot") result = await completeCopilot(input);
  else result = await completeOpenAI(input);
  if (result.text.length > MAX_RESULT_CHARS) throw new ProxyError("result_too_large", "Provider response exceeds the 256 KiB limit", 502, result.usage);
  if (input.output_format === "json") {
    try { JSON.parse(result.text); }
    catch { throw new ProxyError("invalid_provider_output", "Provider did not return valid JSON", 502, result.usage); }
  }
  if (input.include_reasoning_summary && result.reasoning_summary === undefined) result.reasoning_summary = null;
  return result;
}

const CODEX_INSTRUCTIONS = "Answer using only the supplied prompt. Do not call tools, execute Shell commands, or request permission to run them. Do not read or modify local files, especially files containing secrets. Return only the requested answer.";

async function completeCodex(input: RequestInput): Promise<Completion> {
  const workspace = join(STATE_DIR, "codex-workspace");
  await mkdir(workspace, { recursive: true, mode: 0o700 });
  const { executable, prefixArgs } = codexCommand();
  const args = [
    ...prefixArgs,
    "-c", "features.shell_tool=false",
    "-c", "features.hooks=false",
    "-c", "features.apps=false",
    "-c", "features.multi_agent=false",
    "-c", "web_search=disabled",
    "-c", `developer_instructions=${JSON.stringify(CODEX_INSTRUCTIONS)}`,
    ...(input.effort ? ["-c", `model_reasoning_effort=${input.effort}`] : []),
    "--ask-for-approval", "never", "exec", "--sandbox", "read-only",
    "--skip-git-repo-check", "--json", "--model", input.model, "-",
  ];
  const prompt = input.output_format === "json" ? `Return a valid JSON value only.\n\n${input.prompt}` : input.prompt;
  return await new Promise<Completion>((resolve, reject) => {
    const child = spawn(executable, args, {
      cwd: workspace, shell: false, windowsHide: true,
      env: { ...process.env, CI: "1", GIT_TERMINAL_PROMPT: "0" },
    });
    let output = "";
    let timedOut = false;
    let tooLarge = false;
    const timer = setTimeout(() => { timedOut = true; child.kill(); }, REQUEST_TIMEOUT_MS);
    child.stdout.on("data", (chunk: Buffer) => {
      output += chunk.toString("utf8");
      if (output.length > MAX_RESULT_CHARS * 8) { tooLarge = true; child.kill(); }
    });
    child.stderr.on("data", () => undefined);
    child.stdin.on("error", () => undefined);
    child.once("error", (error: NodeJS.ErrnoException) => {
      clearTimeout(timer);
      reject(new ProxyError(error.code === "ENOENT" ? "provider_not_configured" : "provider_error", "Codex executable is unavailable", 503));
    });
    child.once("close", (code) => {
      clearTimeout(timer);
      if (timedOut) { reject(new ProxyError("provider_timeout", "Codex request timed out", 502)); return; }
      if (tooLarge) { reject(new ProxyError("result_too_large", "Codex output exceeds the limit", 502)); return; }
      if (code !== 0) { reject(new ProxyError("provider_error", "Codex request failed", 502)); return; }
      try { resolve(parseCodexOutput(output)); }
      catch { reject(new ProxyError("provider_error", "Codex returned no usable text", 502)); }
    });
    child.stdin.end(prompt);
  });
}

function codexCommand(): { executable: string; prefixArgs: string[] } {
  const override = process.env.AIW_AGENT_PROXY_CODEX_EXECUTABLE;
  if (override) return { executable: override, prefixArgs: [] };
  if (process.platform !== "win32") return { executable: "codex", prefixArgs: [] };
  for (const directory of (process.env.PATH || "").split(delimiter)) {
    if (!directory || !existsSync(join(directory, "codex.cmd"))) continue;
    const script = join(directory, "node_modules", "@openai", "codex", "bin", "codex.js");
    if (existsSync(script)) return { executable: process.execPath, prefixArgs: [script] };
  }
  return { executable: "codex.exe", prefixArgs: [] };
}

function parseCodexOutput(output: string): Completion {
  let text = "";
  let usage = emptyUsage();
  for (const line of output.split(/\r?\n/)) {
    if (!line.trim()) continue;
    const event = JSON.parse(line) as Record<string, unknown>;
    const item = event.item as Record<string, unknown> | undefined;
    if (event.type === "item.completed" && item?.type === "agent_message" && typeof item.text === "string") text = item.text;
    if (event.type === "turn.completed" && event.usage && typeof event.usage === "object") {
      const row = event.usage as Record<string, unknown>;
      usage = { input_tokens: safeInteger(row.input_tokens), output_tokens: safeInteger(row.output_tokens), cost: null };
    }
  }
  if (!text) throw new Error("missing Codex text");
  return { text, usage };
}

async function completeOpenAI(input: RequestInput): Promise<Completion> {
  const apiKey = process.env.OPENAI_API_KEY;
  if (!apiKey) throw new ProxyError("provider_not_configured", "OPENAI_API_KEY is not configured", 503);
  let endpoint: URL;
  try {
    endpoint = new URL(process.env.OPENAI_BASE_URL || "https://api.openai.com/v1");
  } catch {
    throw new ProxyError("provider_not_configured", "OPENAI_BASE_URL is invalid", 503);
  }
  if (endpoint.protocol !== "https:") throw new ProxyError("provider_not_configured", "OPENAI_BASE_URL must use HTTPS", 503);
  endpoint.pathname = `${endpoint.pathname.replace(/\/$/, "")}/responses`;
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  try {
    const response = await fetch(endpoint, {
      method: "POST",
      headers: { authorization: `Bearer ${apiKey}`, "content-type": "application/json" },
      signal: controller.signal,
      redirect: "error",
      body: JSON.stringify({
        model: input.model,
        input: input.output_format === "json" ? `Return JSON only.\n\n${input.prompt}` : input.prompt,
        ...(input.system_prompt ? { instructions: input.system_prompt } : {}),
        max_output_tokens: 8192,
        ...(input.effort || input.include_reasoning_summary ? { reasoning: {
          ...(input.effort ? { effort: input.effort } : {}),
          ...(input.include_reasoning_summary ? { summary: "auto" } : {}),
        } } : {}),
        ...(input.output_format === "json" ? { text: { format: { type: "json_object" } } } : {}),
      }),
    });
    if (!response.ok) throw new ProxyError("provider_error", `OpenAI returned HTTP ${response.status}`, 502);
    const body: unknown = await response.json();
    if (!body || typeof body !== "object") throw new ProxyError("provider_error", "OpenAI returned an invalid response", 502);
    const row = body as Record<string, unknown>;
    const usageRow = row.usage && typeof row.usage === "object" ? row.usage as Record<string, unknown> : {};
    const inputDetails = usageRow.input_tokens_details && typeof usageRow.input_tokens_details === "object"
      ? usageRow.input_tokens_details as Record<string, unknown> : {};
    const outputDetails = usageRow.output_tokens_details && typeof usageRow.output_tokens_details === "object"
      ? usageRow.output_tokens_details as Record<string, unknown> : {};
    const usage: Usage = {
      input_tokens: safeInteger(usageRow.input_tokens),
      output_tokens: safeInteger(usageRow.output_tokens),
      cost: null,
      openai_usage: {
        response_id: safeIdentifier(row.id),
        response_model: safeIdentifier(row.model),
        service_tier: safeIdentifier(row.service_tier),
        response_status: safeIdentifier(row.status),
        total_tokens: safeInteger(usageRow.total_tokens),
        cached_input_tokens: safeInteger(inputDetails.cached_tokens),
        cache_write_input_tokens: safeInteger(inputDetails.cache_write_tokens),
        reasoning_output_tokens: safeInteger(outputDetails.reasoning_tokens),
      },
    };
    const text = typeof row.output_text === "string" ? row.output_text : extractOutput(row.output);
    if (!text) throw new ProxyError("provider_error", "OpenAI returned no text", 502, usage);
    const summary = input.include_reasoning_summary ? extractReasoningSummary(row.output, usage) : undefined;
    return { text, usage, ...(input.include_reasoning_summary ? { reasoning_summary: summary } : {}) };
  } catch (error) {
    if (error instanceof ProxyError) throw error;
    throw new ProxyError(controller.signal.aborted ? "provider_timeout" : "provider_error", controller.signal.aborted ? "OpenAI request timed out" : "OpenAI request failed", 502);
  } finally {
    clearTimeout(timer);
  }
}

function extractOutput(value: unknown): string {
  if (!Array.isArray(value)) return "";
  const chunks: string[] = [];
  for (const item of value) {
    if (!item || typeof item !== "object") continue;
    const content = (item as Record<string, unknown>).content;
    if (!Array.isArray(content)) continue;
    for (const part of content) {
      if (part && typeof part === "object" && typeof (part as Record<string, unknown>).text === "string") chunks.push((part as Record<string, string>).text);
    }
  }
  return chunks.join("");
}

function extractReasoningSummary(value: unknown, usage: Usage): string | null {
  if (!Array.isArray(value)) return null;
  const chunks: string[] = [];
  for (const item of value) {
    if (!item || typeof item !== "object" || (item as Record<string, unknown>).type !== "reasoning") continue;
    const summary = (item as Record<string, unknown>).summary;
    if (!Array.isArray(summary)) continue;
    for (const part of summary) {
      if (part && typeof part === "object" && (part as Record<string, unknown>).type === "summary_text"
          && typeof (part as Record<string, unknown>).text === "string") {
        chunks.push((part as Record<string, string>).text);
      }
    }
  }
  const text = chunks.join("\n\n");
  if (text.length > MAX_REASONING_SUMMARY_CHARS) throw new ProxyError("result_too_large", "OpenAI reasoning summary exceeds the limit", 502, usage);
  return text || null;
}

async function completeCopilot(input: RequestInput): Promise<Completion> {
  const workspace = join(STATE_DIR, "copilot-workspace");
  await mkdir(workspace, { recursive: true, mode: 0o700 });
  const { executable, prefixArgs } = copilotCommand();
  const args = [
    ...prefixArgs,
    "--available-tools=",
    "--deny-tool=shell,read,write,url,memory",
    "--disable-builtin-mcps",
    "--no-custom-instructions",
    "--no-ask-user",
    "--no-auto-update",
    "--no-remote",
    "--output-format=json",
    "--model", input.model,
    ...(input.effort ? ["--reasoning-effort", input.effort] : []),
  ];
  const prompt = input.output_format === "json" ? `Return valid JSON only.\n\n${input.prompt}` : input.prompt;
  return await new Promise<Completion>((resolve, reject) => {
    const child = spawn(executable, args, {
      cwd: workspace, shell: false, windowsHide: true,
      env: { ...process.env, COPILOT_AUTO_UPDATE: "false", COPILOT_ALLOW_ALL: "false" },
    });
    let output = "";
    let timedOut = false;
    let tooLarge = false;
    const timer = setTimeout(() => { timedOut = true; child.kill(); }, REQUEST_TIMEOUT_MS);
    child.stdout.on("data", (chunk: Buffer) => {
      output += chunk.toString("utf8");
      if (output.length > MAX_RESULT_CHARS * 8) { tooLarge = true; child.kill(); }
    });
    child.stderr.on("data", () => undefined);
    child.stdin.on("error", () => undefined);
    child.once("error", (error: NodeJS.ErrnoException) => {
      clearTimeout(timer);
      reject(new ProxyError(error.code === "ENOENT" ? "provider_not_configured" : "provider_error", "Copilot executable is unavailable", 503));
    });
    child.once("close", (code) => {
      clearTimeout(timer);
      if (timedOut) { reject(new ProxyError("provider_timeout", "Copilot request timed out", 502)); return; }
      if (tooLarge) { reject(new ProxyError("result_too_large", "Copilot output exceeds the limit", 502)); return; }
      if (code !== 0) { reject(new ProxyError("provider_error", "Copilot request failed", 502)); return; }
      try { resolve(parseCopilotOutput(output)); }
      catch { reject(new ProxyError("provider_error", "Copilot returned no usable text", 502)); }
    });
    child.stdin.end(prompt);
  });
}

function copilotCommand(): { executable: string; prefixArgs: string[] } {
  const override = process.env.AIW_AGENT_PROXY_COPILOT_EXECUTABLE;
  if (override) return { executable: override, prefixArgs: [] };
  if (process.platform !== "win32") return { executable: "copilot", prefixArgs: [] };
  for (const directory of (process.env.PATH || "").split(delimiter)) {
    if (!directory || !existsSync(join(directory, "copilot.cmd"))) continue;
    const script = join(directory, "node_modules", "@github", "copilot", "npm-loader.js");
    if (existsSync(script)) return { executable: process.execPath, prefixArgs: [script] };
  }
  return { executable: "copilot.exe", prefixArgs: [] };
}

function parseCopilotOutput(output: string): Completion {
  let text = "";
  let usage = emptyUsage();
  let completed = false;
  for (const line of output.split(/\r?\n/)) {
    if (!line.trim()) continue;
    const event = JSON.parse(line) as Record<string, unknown>;
    const data = event.data as Record<string, unknown> | undefined;
    if (event.type === "assistant.message" && typeof data?.content === "string") text = data.content;
    if (event.type === "assistant.usage") usage = copilotUsage(data);
    if (event.type === "result") {
      completed = event.exitCode === 0;
      if (event.usage && typeof event.usage === "object") usage = copilotUsage(event.usage as Record<string, unknown>);
    }
  }
  if (!completed || !text) throw new Error("missing Copilot result");
  return { text, usage };
}

function copilotUsage(row: Record<string, unknown> | undefined): Usage {
  return {
    input_tokens: safeInteger(row?.inputTokens ?? row?.input_tokens),
    output_tokens: safeInteger(row?.outputTokens ?? row?.output_tokens),
    cost: null,
  };
}
