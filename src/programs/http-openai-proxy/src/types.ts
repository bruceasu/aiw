export type Provider = "openai";
export type OutputFormat = "json" | "markdown";
export type Effort = "minimal" | "low" | "medium" | "high" | "xhigh";

export type RequestInput = {
  client_id: string;
  request_id?: string;
  provider: Provider;
  model: string;
  effort?: Effort;
  output_format: OutputFormat;
  prompt: string;
  system_prompt?: string;
  include_reasoning_summary?: boolean;
};

export type OpenAIUsage = {
  response_id: string | null;
  response_model: string | null;
  service_tier: string | null;
  response_status: string | null;
  total_tokens: number | null;
  cached_input_tokens: number | null;
  cache_write_input_tokens: number | null;
  reasoning_output_tokens: number | null;
};

export type Usage = {
  input_tokens: number | null;
  output_tokens: number | null;
  cost: number | null;
  openai_usage?: OpenAIUsage;
};
export type Completion = { text: string; usage: Usage; reasoning_summary?: string | null };

export type AuditRecord = {
  timestamp: string;
  request_id: string;
  event_id?: string;
  client_id: string;
  provider: Provider;
  model: string;
  status: "succeeded" | "failed";
  duration_ms: number;
  input_tokens: number | null;
  output_tokens: number | null;
  cost: number | null;
  openai_usage?: OpenAIUsage;
  error_code?: string;
};

export const PROVIDERS: readonly Provider[] = ["openai"];
export const EFFORTS: readonly Effort[] = ["minimal", "low", "medium", "high", "xhigh"];
