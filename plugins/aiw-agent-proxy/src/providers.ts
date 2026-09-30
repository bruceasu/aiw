import { MAX_RESULT_CHARS, REQUEST_TIMEOUT_MS, STATE_DIR } from "./config.js";
import { ProxyError } from "./errors.js";
import type { Completion, RequestInput, Usage } from "./types.js";
import { join } from "node:path";

const emptyUsage = (): Usage => ({ input_tokens: null, output_tokens: null, cost: null });
const safeInteger = (value: unknown): number | null => Number.isSafeInteger(value) && Number(value) >= 0 ? Number(value) : null;

export async function complete(input: RequestInput): Promise<Completion> {
  let result: Completion;
  if (input.provider === "codex") {
    // Codex SDK exposes a read-only sandbox but no thread tool allowlist. Shell
    // execution is forbidden by the approved Issue, so this provider stays closed.
    throw new ProxyError("provider_disabled", "Codex is disabled: its current SDK cannot prove that shell tools are unavailable", 503);
  }
  if (input.provider === "copilot") result = await completeCopilot(input);
  else result = await completeOpenAI(input);
  if (result.text.length > MAX_RESULT_CHARS) throw new ProxyError("result_too_large", "Provider response exceeds the 256 KiB limit", 502, result.usage);
  if (input.output_format === "json") {
    try { JSON.parse(result.text); }
    catch { throw new ProxyError("invalid_provider_output", "Provider did not return valid JSON", 502, result.usage); }
  }
  return result;
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
        max_output_tokens: 8192,
        ...(input.effort ? { reasoning: { effort: input.effort } } : {}),
        ...(input.output_format === "json" ? { text: { format: { type: "json_object" } } } : {}),
      }),
    });
    if (!response.ok) throw new ProxyError("provider_error", `OpenAI returned HTTP ${response.status}`, 502);
    const body: unknown = await response.json();
    if (!body || typeof body !== "object") throw new ProxyError("provider_error", "OpenAI returned an invalid response", 502);
    const row = body as Record<string, unknown>;
    const usageRow = row.usage && typeof row.usage === "object" ? row.usage as Record<string, unknown> : {};
    const text = typeof row.output_text === "string" ? row.output_text : extractOutput(row.output);
    if (!text) throw new ProxyError("provider_error", "OpenAI returned no text", 502);
    return {
      text,
      usage: {
        input_tokens: safeInteger(usageRow.input_tokens),
        output_tokens: safeInteger(usageRow.output_tokens),
        cost: null,
      },
    };
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

async function completeCopilot(input: RequestInput): Promise<Completion> {
  // The SDK types describe deny-all configuration, but the bundled runtime still
  // needs a negative execution check before this gate may be opened.
  const isolationVerified = false;
  if (!isolationVerified) throw new ProxyError("provider_disabled", "Copilot is disabled until deny-all tool isolation passes runtime verification", 503);
  if (input.effort === "minimal") throw new ProxyError("invalid_effort", "Copilot does not support minimal effort", 400);
  const { CopilotClient } = await import("@github/copilot-sdk");
  const client = new CopilotClient({ mode: "empty", baseDirectory: join(STATE_DIR, "copilot"), enableRemoteSessions: false });
  try {
    await client.start();
    const session = await client.createSession({
      model: input.model,
      ...(input.effort ? { reasoningEffort: input.effort } : {}),
      availableTools: [],
      onPermissionRequest: async () => ({ kind: "reject" as const }),
    });
    let usage = emptyUsage();
    session.on((event) => {
      if (event.type === "assistant.usage") usage = {
        input_tokens: safeInteger(event.data?.inputTokens),
        output_tokens: safeInteger(event.data?.outputTokens),
        cost: null,
      };
    });
    try {
      const prompt = input.output_format === "json" ? `Return valid JSON only.\n\n${input.prompt}` : input.prompt;
      const response = await session.sendAndWait({ prompt }, REQUEST_TIMEOUT_MS);
      if (!response?.data.content) throw new ProxyError("provider_error", "Copilot returned no text", 502);
      return { text: response.data.content, usage };
    } finally {
      await session.disconnect();
    }
  } finally {
    await client.stop();
  }
}
