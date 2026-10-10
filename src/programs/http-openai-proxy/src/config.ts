import { tmpdir } from "node:os";
import { join } from "node:path";
import { isIP } from "node:net";

export const HOST = parseHost(process.env.HTTP_OPENAI_PROXY_HOST || process.env.AIW_AGENT_PROXY_HOST);
export const PORT = parsePort(process.env.HTTP_OPENAI_PROXY_PORT || process.env.AIW_AGENT_PROXY_PORT);
export const RESULT_TTL_MS = 60 * 60 * 1000;
export const REQUEST_TIMEOUT_MS = 60_000;
export const MAX_BODY_BYTES = 1024 * 1024;
export const MAX_PROMPT_CHARS = 64 * 1024;
export const MAX_RESULT_CHARS = 256 * 1024;
export const MAX_PENDING_RESULTS = 256;
export const STATE_DIR = process.env.HTTP_OPENAI_PROXY_STATE_DIR || process.env.AIW_AGENT_PROXY_STATE_DIR || join(tmpdir(), "http-openai-proxy");
export const RESULTS_DIR = join(STATE_DIR, "results");
export const AUDIT_FILE = join(STATE_DIR, "audit.jsonl");

function parseHost(raw: string | undefined): string {
  if (!raw) return "127.0.0.1";
  if (isIP(raw) !== 4) throw new Error("HTTP_OPENAI_PROXY_HOST must be an IPv4 address");
  return raw;
}

function parsePort(raw: string | undefined): number {
  if (!raw) return 43127;
  const port = Number(raw);
  if (!Number.isInteger(port) || port < 1 || port > 65535) throw new Error("HTTP_OPENAI_PROXY_PORT must be 1-65535");
  return port;
}
