import { MAX_RESULT_CHARS, REQUEST_TIMEOUT_MS } from "./config.js";
import { ProxyError } from "./errors.js";
import type { Completion, RequestInput, Usage } from "./types.js";
import { isIP } from "node:net";
import OpenAI, { APIConnectionTimeoutError, APIError } from "openai";

const safeInteger = (value: unknown): number | null => Number.isSafeInteger(value) && Number(value) >= 0 ? Number(value) : null;
const safeIdentifier = (value: unknown): string | null =>
  typeof value === "string" && /^[A-Za-z0-9._:/-]{1,128}$/.test(value) ? value : null;
const MAX_REASONING_SUMMARY_CHARS = 16 * 1024;

export async function complete(input: RequestInput): Promise<Completion> {
  const result = await completeOpenAI(input);
  if (result.text.length > MAX_RESULT_CHARS) throw new ProxyError("result_too_large", "Provider response exceeds the 256 KiB limit", 502, result.usage);
  if (input.output_format === "json") {
    try { JSON.parse(result.text); }
    catch { throw new ProxyError("invalid_provider_output", "Provider did not return valid JSON", 502, result.usage); }
  }
  if (input.include_reasoning_summary && result.reasoning_summary === undefined) result.reasoning_summary = null;
  return result;
}

async function completeOpenAI(input: RequestInput): Promise<Completion> {
  const apiKey = process.env.HTTP_OPENAI_PROXY_API_KEY || process.env.OPENAI_API_KEY;
  if (!apiKey) throw new ProxyError("provider_not_configured", "HTTP_OPENAI_PROXY_API_KEY is not configured", 503);
  const profile = process.env.HTTP_OPENAI_PROXY_API_PROFILE || process.env.OPENAI_API_PROFILE || "openai";
  if (profile !== "openai" && profile !== "aiw_gateway") {
    throw new ProxyError("provider_not_configured", "HTTP_OPENAI_PROXY_API_PROFILE must be openai or aiw_gateway", 503);
  }
  if (profile === "aiw_gateway" && (input.effort || input.include_reasoning_summary)) {
    throw new ProxyError("unsupported_option", "The AIW Gateway profile does not support reasoning options", 400);
  }

  let baseURL: string;
  try {
    const endpoint = new URL(process.env.HTTP_OPENAI_PROXY_BASE_URL || process.env.OPENAI_BASE_URL || "https://api.openai.com/v1");
    const hostname = endpoint.hostname.replace(/^\[|\]$/g, "").toLowerCase();
    const loopback = hostname === "localhost" || hostname.endsWith(".localhost")
      || (isIP(hostname) === 4 && hostname.startsWith("127."))
      || (isIP(hostname) === 6 && hostname === "::1");
    if (endpoint.username || endpoint.password || endpoint.search || endpoint.hash) throw new Error("invalid URL components");
    if (endpoint.protocol !== "https:" && !(endpoint.protocol === "http:" && loopback)) throw new Error("unsupported protocol");
    baseURL = endpoint.toString().replace(/\/+$/, "");
  } catch {
    throw new ProxyError("provider_not_configured", "HTTP_OPENAI_PROXY_BASE_URL must be a valid HTTPS URL or a loopback HTTP URL", 503);
  }

  const client = new OpenAI({
    apiKey,
    baseURL,
    timeout: REQUEST_TIMEOUT_MS,
    maxRetries: 0,
    logLevel: "off",
    fetchOptions: { redirect: "error" },
  });
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  try {
    // Use the typed SDK transport so compatible endpoints' output_text is preserved.
    const response = await client.post<OpenAI.Responses.Response>("/responses", { body: {
      model: input.model,
      input: input.output_format === "json" ? `Return JSON only.\n\n${input.prompt}` : input.prompt,
      ...(input.system_prompt ? { instructions: input.system_prompt } : {}),
      ...(profile === "openai" ? { max_output_tokens: 8192 } : {}),
      ...(input.effort || input.include_reasoning_summary ? { reasoning: {
        ...(input.effort ? { effort: input.effort } : {}),
        ...(input.include_reasoning_summary ? { summary: "auto" } : {}),
      } } : {}),
      ...(input.output_format === "json" ? { text: { format: { type: "json_object" } } } : {}),
    } satisfies OpenAI.Responses.ResponseCreateParamsNonStreaming, signal: controller.signal });

    const usageRow = response.usage;
    const usage: Usage = {
      input_tokens: safeInteger(usageRow?.input_tokens),
      output_tokens: safeInteger(usageRow?.output_tokens),
      cost: null,
      openai_usage: {
        response_id: safeIdentifier(response.id),
        response_model: safeIdentifier(response.model),
        service_tier: safeIdentifier(response.service_tier),
        response_status: safeIdentifier(response.status),
        total_tokens: safeInteger(usageRow?.total_tokens),
        cached_input_tokens: safeInteger(usageRow?.input_tokens_details?.cached_tokens),
        cache_write_input_tokens: safeInteger(usageRow?.input_tokens_details?.cache_write_tokens),
        reasoning_output_tokens: safeInteger(usageRow?.output_tokens_details?.reasoning_tokens),
      },
    };
    const text = typeof response.output_text === "string" ? response.output_text : extractOutput(response.output);
    if (!text) throw new ProxyError("provider_error", "Provider returned no text", 502, usage);
    const summary = input.include_reasoning_summary ? extractReasoningSummary(response.output, usage) : undefined;
    return { text, usage, ...(input.include_reasoning_summary ? { reasoning_summary: summary } : {}) };
  } catch (error) {
    if (controller.signal.aborted) throw new ProxyError("provider_timeout", "Provider request timed out", 502);
    if (error instanceof ProxyError) throw error;
    if (error instanceof APIConnectionTimeoutError) throw new ProxyError("provider_timeout", "Provider request timed out", 502);
    if (error instanceof APIError) throw new ProxyError("provider_error", `Provider returned HTTP ${error.status ?? "unknown"}`, 502);
    throw new ProxyError("provider_error", "Provider request failed", 502);
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
          && typeof (part as Record<string, unknown>).text === "string") chunks.push((part as Record<string, string>).text);
    }
  }
  const text = chunks.join("\n\n");
  if (text.length > MAX_REASONING_SUMMARY_CHARS) throw new ProxyError("result_too_large", "Provider reasoning summary exceeds the limit", 502, usage);
  return text || null;
}
