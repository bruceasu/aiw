import type { Usage } from "./types.js";

export class ProxyError extends Error {
  constructor(readonly code: string, message: string, readonly statusCode = 400, readonly usage?: Usage) {
    super(message);
  }
}

export function errorCode(error: unknown): string {
  return error instanceof ProxyError ? error.code : "provider_error";
}

export function safeError(error: unknown): { code: string; message: string; statusCode: number } {
  if (error instanceof ProxyError) return { code: error.code, message: error.message, statusCode: error.statusCode };
  return { code: "provider_error", message: "Provider request failed", statusCode: 502 };
}
