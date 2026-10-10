import { EFFORTS, PROVIDERS, type Effort, type OutputFormat, type Provider, type RequestInput } from "./types.js";
import { MAX_PROMPT_CHARS } from "./config.js";
import { ProxyError } from "./errors.js";

const ID = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;

export function validateRequest(value: unknown): RequestInput {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new ProxyError("invalid_request", "Request must be an object");
  const row = value as Record<string, unknown>;
  if (typeof row.client_id !== "string" || !ID.test(row.client_id)) throw new ProxyError("invalid_client_id", "client_id must be 1-128 safe characters");
  if (row.request_id !== undefined && (typeof row.request_id !== "string" || !ID.test(row.request_id))) throw new ProxyError("invalid_request_id", "request_id must be 1-128 safe characters");
  if (row.provider !== undefined && (typeof row.provider !== "string" || !PROVIDERS.includes(row.provider as Provider))) throw new ProxyError("invalid_provider", "provider must be openai when specified");
  if (typeof row.model !== "string" || !/^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$/.test(row.model)) throw new ProxyError("invalid_model", "model is required and must be 1-128 safe characters");
  if (row.effort !== undefined && (typeof row.effort !== "string" || !EFFORTS.includes(row.effort as Effort))) throw new ProxyError("invalid_effort", "effort must be minimal, low, medium, high, or xhigh");
  if (row.output_format !== "json" && row.output_format !== "markdown") throw new ProxyError("invalid_output_format", "output_format must be json or markdown");
  if (typeof row.prompt !== "string" || !row.prompt.trim() || row.prompt.length > MAX_PROMPT_CHARS) throw new ProxyError("invalid_prompt", `prompt must contain 1-${MAX_PROMPT_CHARS} characters`);
  if (row.system_prompt !== undefined && (typeof row.system_prompt !== "string" || !row.system_prompt.trim() || row.system_prompt.length > MAX_PROMPT_CHARS)) throw new ProxyError("invalid_system_prompt", `system_prompt must contain 1-${MAX_PROMPT_CHARS} characters`);
  if (row.include_reasoning_summary !== undefined && typeof row.include_reasoning_summary !== "boolean") throw new ProxyError("invalid_reasoning_summary", "include_reasoning_summary must be a boolean");
  return {
    client_id: row.client_id,
    ...(row.request_id === undefined ? {} : { request_id: row.request_id as string }),
    provider: "openai",
    model: row.model,
    ...(row.effort === undefined ? {} : { effort: row.effort as Effort }),
    output_format: row.output_format as OutputFormat,
    prompt: row.prompt,
    ...(row.system_prompt === undefined ? {} : { system_prompt: row.system_prompt as string }),
    ...(row.include_reasoning_summary === undefined ? {} : { include_reasoning_summary: row.include_reasoning_summary as boolean }),
  };
}
