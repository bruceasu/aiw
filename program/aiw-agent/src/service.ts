import { randomUUID } from "node:crypto";
import { createServer, type IncomingMessage, type Server, type ServerResponse } from "node:http";
import { isIP } from "node:net";
import { URL } from "node:url";
// @ts-ignore -- runtime and types are declared in package.json; install is a separate gated step.
import { WebSocketServer } from "ws";
import { HOST, MAX_BODY_BYTES, PORT } from "./config.js";
import { ProxyError, safeError } from "./errors.js";
import { complete } from "./providers.js";
import { appendAudit, ResultStore, type PendingResult } from "./store.js";
import { validateRequest } from "./validate.js";
import type { AuditRecord, Completion, RequestInput, Usage } from "./types.js";

const MAX_ACTIVE = 8;
const MAX_CONNECTIONS = 128;
const CLIENT_ID = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
let active = 0;
const pendingRequests = new Map<string, Promise<PendingResult>>();

type Service = { server: Server; close: () => Promise<void> };

export async function startService(port = PORT): Promise<Service> {
  const store = new ResultStore();
  await store.initialize();
  const expiryTimer = setInterval(() => void store.expire().catch(() => undefined), 60_000);
  expiryTimer.unref();
  const server = createServer((request, response) => {
    void handleHttp(request, response, store, port);
  });
  // @ts-ignore -- WebSocketServer is supplied by the package's ws dependency.
  const sockets = new WebSocketServer({ noServer: true, maxPayload: MAX_BODY_BYTES });

  server.on("upgrade", (request, socket, head) => {
    const clientId = upgradeClientId(request, port);
    if (!clientId || !isLoopback(request.socket.remoteAddress) || sockets.clients.size >= MAX_CONNECTIONS) {
      socket.write("HTTP/1.1 403 Forbidden\r\nConnection: close\r\n\r\n");
      socket.destroy();
      return;
    }
    sockets.handleUpgrade(request, socket, head, (websocket: any) => {
      sockets.emit("connection", websocket, request);
    });
  });

  sockets.on("connection", (socket: any, request: IncomingMessage) => {
    const clientId = new URL(request.url || "/", `http://${HOST}:${port}`).searchParams.get("client_id") || "";
    let processing = false;
    const deliverPending = async (): Promise<void> => {
      for (const item of await store.pending(clientId)) {
        if (!socketOpen(socket)) break;
        if (!await sendJson(socket, resultEvent(item))) break;
      }
    };
    void deliverPending();

    socket.on("message", (data: unknown) => {
      void (async () => {
        let message: unknown;
        try {
          const text = typeof data === "string" ? data : Buffer.isBuffer(data) ? data.toString("utf8") : Buffer.from(data as ArrayBuffer).toString("utf8");
          message = JSON.parse(text);
        }
        catch { sendJson(socket, { type: "error", error: { code: "invalid_json", message: "Message must be JSON" } }); return; }
        if (!message || typeof message !== "object" || Array.isArray(message)) {
          sendJson(socket, { type: "error", error: { code: "invalid_event", message: "Event must be an object" } }); return;
        }
        const event = message as Record<string, unknown>;
        if (event.type === "ack") {
          const eventId = typeof event.event_id === "string" ? event.event_id : "";
          const accepted = await store.acknowledge(clientId, eventId);
          await sendJson(socket, { type: "acknowledged", event_id: eventId, deleted: accepted });
          return;
        }
        if (event.type !== "request" || processing) {
          await sendJson(socket, { type: "error", error: { code: processing ? "request_in_progress" : "invalid_event", message: processing ? "Only one request may run per connection" : "Expected request or ack event" } });
          return;
        }
        let input: RequestInput;
        try { input = validateRequest(event.request); }
        catch (error) { await sendJson(socket, { type: "error", error: safeError(error) }); return; }
        if (input.client_id !== clientId) {
          await sendJson(socket, { type: "error", error: { code: "client_id_mismatch", message: "Request client_id must match the connection" } }); return;
        }
        input.request_id ||= randomUUID();
        const previous = await store.findPending(clientId, input.request_id);
        if (previous) { await sendJson(socket, resultEvent(previous)); return; }
        processing = true;
        try {
          const key = JSON.stringify([clientId, input.request_id]);
          let work = pendingRequests.get(key);
          if (!work) {
            work = executeAndStore(input, store);
            pendingRequests.set(key, work);
          }
          let pending: PendingResult;
          try { pending = await work; }
          finally { if (pendingRequests.get(key) === work) pendingRequests.delete(key); }
          if (!await sendJson(socket, resultEvent(pending))) {
            // Keep the result file for the next connection; send failure is not ACK.
          }
        } catch (error) {
          const safe = safeError(error);
          await sendJson(socket, { type: "error", request_id: input.request_id, error: { code: safe.code, message: safe.message } });
        } finally {
          processing = false;
        }
      })();
    });
  });

  server.headersTimeout = 10_000;
  server.requestTimeout = 30_000;
  server.keepAliveTimeout = 5_000;

  await new Promise<void>((resolve, reject) => {
    server.once("error", reject);
    server.listen(port, HOST, () => resolve());
  });

  return {
    server,
    close: async () => {
      clearInterval(expiryTimer);
      for (const socket of sockets.clients) socket.close(1001, "service stopping");
      sockets.close();
      await new Promise<void>((resolve) => server.close(() => resolve()));
    },
  };
}

async function handleHttp(request: IncomingMessage, response: ServerResponse, store: ResultStore, port: number): Promise<void> {
  const localOnly = HOST === "127.0.0.1";
  if ((localOnly && !isLoopback(request.socket.remoteAddress)) ||
      !(localOnly ? localHostHeader(request.headers.host, port) : remoteHostHeader(request.headers.host, port))) {
    json(response, 403, { error: localOnly
      ? { code: "local_only", message: "Only local loopback requests are accepted" }
      : { code: "invalid_host", message: "Use the configured IPv4 address and port" } });
    return;
  }
  if (request.method === "GET" && request.url === "/health") {
    json(response, 200, { status: "ok", host: HOST, port });
    return;
  }
  if (request.method !== "POST" || request.url !== "/v1/requests") {
    json(response, 404, { error: { code: "not_found", message: "Endpoint not found" } });
    return;
  }
  if (!/^application\/json(?:\s*;|$)/i.test(request.headers["content-type"] || "")) {
    json(response, 415, { error: { code: "content_type_required", message: "Content-Type must be application/json" } });
    return;
  }
  try {
    const input = validateRequest(await readJson(request));
    input.request_id ||= randomUUID();
    if (active >= MAX_ACTIVE) throw new ProxyError("busy", "Too many active requests", 429);
    active++;
    try {
      const result = await execute(input);
      json(response, 200, { request_id: input.request_id, provider: input.provider, model: input.model, output_format: input.output_format, result: result.text, usage: result.usage, ...(result.reasoning_summary !== undefined ? { reasoning_summary: result.reasoning_summary } : {}) });
    } finally { active--; }
  } catch (error) {
    const safe = safeError(error);
    json(response, safe.statusCode, { error: { code: safe.code, message: safe.message } });
  }
}

async function execute(input: RequestInput): Promise<Completion> {
  const requestId = input.request_id || randomUUID();
  const start = Date.now();
  try {
    const result = await complete(input);
    await writeAudit({
      timestamp: new Date().toISOString(), request_id: requestId, client_id: input.client_id,
      provider: input.provider, model: input.model, status: "succeeded", duration_ms: Date.now() - start,
      ...auditUsage(result.usage),
    });
    return result;
  } catch (error) {
    const safe = safeError(error);
    const usage = error instanceof ProxyError ? error.usage : undefined;
    await writeAudit({
      timestamp: new Date().toISOString(), request_id: requestId, client_id: input.client_id,
      provider: input.provider, model: input.model, status: "failed", duration_ms: Date.now() - start,
      ...auditUsage(usage), error_code: safe.code,
    });
    throw error;
  }
}

async function executeAndStore(input: RequestInput, store: ResultStore): Promise<PendingResult> {
  const requestId = input.request_id || randomUUID();
  const start = Date.now();
  let completion: Completion | undefined;
  try {
    if (active >= MAX_ACTIVE) throw new ProxyError("busy", "Too many active requests", 429);
    active++;
    try {
      completion = await complete(input);
      const pending = await store.save(requestId, input.client_id, completion);
      await writeAudit({
        timestamp: new Date().toISOString(), request_id: requestId, event_id: pending.event_id,
        client_id: input.client_id, provider: input.provider, model: input.model, status: "succeeded",
        duration_ms: Date.now() - start, ...auditUsage(completion.usage),
      });
      return pending;
    } finally { active--; }
  } catch (error) {
    const safe = safeError(error);
    await writeAudit({
      timestamp: new Date().toISOString(), request_id: requestId, client_id: input.client_id,
      provider: input.provider, model: input.model, status: "failed", duration_ms: Date.now() - start,
      ...auditUsage(completion?.usage ?? (error instanceof ProxyError ? error.usage : undefined)),
      error_code: safe.code,
    });
    throw error;
  }
}

function auditUsage(usage: Usage | undefined): Pick<AuditRecord, "input_tokens" | "output_tokens" | "cost" | "openai_usage"> {
  return {
    input_tokens: usage?.input_tokens ?? null,
    output_tokens: usage?.output_tokens ?? null,
    cost: usage?.cost ?? null,
    ...(usage?.openai_usage ? { openai_usage: usage.openai_usage } : {}),
  };
}

async function writeAudit(record: AuditRecord): Promise<void> {
  try { await appendAudit(record); }
  catch { process.stderr.write("aiw-agent-proxy: audit_write_failed\n"); }
}

async function readJson(request: IncomingMessage): Promise<unknown> {
  const chunks: Buffer[] = [];
  let size = 0;
  for await (const chunk of request) {
    const buffer = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
    size += buffer.length;
    if (size > MAX_BODY_BYTES) throw new ProxyError("body_too_large", "Request body exceeds 1 MiB", 413);
    chunks.push(buffer);
  }
  try { return JSON.parse(Buffer.concat(chunks).toString("utf8")); }
  catch { throw new ProxyError("invalid_json", "Request body must be valid JSON"); }
}

function upgradeClientId(request: IncomingMessage, port: number): string | undefined {
  if (!localHostHeader(request.headers.host, port)) return undefined;
  const origin = request.headers.origin;
  if (origin) {
    try {
      const parsed = new URL(origin);
      if (parsed.protocol !== "http:" || !["localhost", "127.0.0.1"].includes(parsed.hostname) || Number(parsed.port) !== port) return undefined;
    } catch { return undefined; }
  }
  try {
    const url = new URL(request.url || "/", `http://${HOST}:${port}`);
    const clientId = url.searchParams.get("client_id") || "";
    return url.pathname === "/v1/events" && CLIENT_ID.test(clientId) ? clientId : undefined;
  } catch { return undefined; }
}

function localHostHeader(host: string | undefined, port: number): boolean {
  if (!host) return false;
  try {
    const parsed = new URL(`http://${host}`);
    return ["127.0.0.1", "localhost"].includes(parsed.hostname) && (!parsed.port || Number(parsed.port) === port);
  } catch { return false; }
}

function remoteHostHeader(host: string | undefined, port: number): boolean {
  if (!host) return false;
  const authority = /^((?:[0-9]{1,3}\.){3}[0-9]{1,3})(?::([0-9]{1,5}))?$/.exec(host);
  if (!authority || isIP(authority[1]) !== 4) return false;
  return Number(authority[2] ?? 80) === port &&
    (HOST === "0.0.0.0" || authority[1] === HOST);
}

function isLoopback(address: string | undefined): boolean {
  if (!address || !isIP(address)) return false;
  return address === "127.0.0.1" || address === "::1" || address.startsWith("::ffff:127.");
}

function resultEvent(item: PendingResult): object {
  return { type: "result", ...item };
}

function socketOpen(socket: { readyState?: number }): boolean {
  return socket.readyState === 1;
}

function sendJson(socket: { send(data: string, callback: (error?: Error) => void): void; readyState?: number }, value: unknown): Promise<boolean> {
  if (!socketOpen(socket)) return Promise.resolve(false);
  return new Promise((resolve) => socket.send(JSON.stringify(value), (error) => resolve(!error)));
}

function json(response: ServerResponse, statusCode: number, value: unknown): void {
  const body = JSON.stringify(value);
  response.writeHead(statusCode, { "content-type": "application/json; charset=utf-8", "content-length": Buffer.byteLength(body), "cache-control": "no-store", "x-content-type-options": "nosniff" });
  response.end(body);
}
