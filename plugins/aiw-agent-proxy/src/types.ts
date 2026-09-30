export type Provider = "codex" | "copilot" | "openai";
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
};

export type Usage = { input_tokens: number | null; output_tokens: number | null; cost: number | null };
export type Completion = { text: string; usage: Usage };

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
  error_code?: string;
};

export const PROVIDERS: readonly Provider[] = ["codex", "copilot", "openai"];
export const EFFORTS: readonly Effort[] = ["minimal", "low", "medium", "high", "xhigh"];
