import { tmpdir } from "node:os";
import { join } from "node:path";

export const HOST = "127.0.0.1" as const;
export const PORT = parsePort(process.env.AIW_AGENT_PROXY_PORT);
export const RESULT_TTL_MS = 60 * 60 * 1000;
export const REQUEST_TIMEOUT_MS = 60_000;
export const MAX_BODY_BYTES = 1024 * 1024;
export const MAX_PROMPT_CHARS = 64 * 1024;
export const MAX_RESULT_CHARS = 256 * 1024;
export const MAX_PENDING_RESULTS = 256;
export const STATE_DIR = process.env.AIW_AGENT_PROXY_STATE_DIR || join(tmpdir(), "aiw-agent-proxy");
export const RESULTS_DIR = join(STATE_DIR, "results");
export const AUDIT_FILE = join(STATE_DIR, "audit.jsonl");

function parsePort(raw: string | undefined): number {
  if (!raw) return 43127;
  const port = Number(raw);
  if (!Number.isInteger(port) || port < 1 || port > 65535) throw new Error("AIW_AGENT_PROXY_PORT must be 1-65535");
  return port;
}
