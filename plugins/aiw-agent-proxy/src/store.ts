import { randomUUID } from "node:crypto";
import { chmod, mkdir, readdir, readFile, rename, rm, writeFile, appendFile } from "node:fs/promises";
import { join } from "node:path";
import { AUDIT_FILE, MAX_PENDING_RESULTS, MAX_RESULT_CHARS, RESULTS_DIR, RESULT_TTL_MS, STATE_DIR } from "./config.js";
import { ProxyError } from "./errors.js";
import type { AuditRecord, Completion } from "./types.js";

export type PendingResult = {
  event_id: string;
  request_id: string;
  client_id: string;
  created_at: string;
  expires_at: string;
  result: Completion;
};

export class ResultStore {
  private readonly expiryTimers = new Map<string, NodeJS.Timeout>();

  async initialize(): Promise<void> {
    await mkdir(RESULTS_DIR, { recursive: true, mode: 0o700 });
    await chmod(RESULTS_DIR, 0o700).catch(() => undefined);
    await this.expire();
    for (const item of await this.allPending()) this.scheduleExpiry(item.event_id, Date.parse(item.expires_at));
  }

  async save(requestId: string, clientId: string, result: Completion): Promise<PendingResult> {
    if (result.text.length > MAX_RESULT_CHARS) throw new ProxyError("result_too_large", "Provider response exceeds the result limit", 502);
    const files = await readdir(RESULTS_DIR, { withFileTypes: true });
    if (files.filter((entry) => entry.isFile() && entry.name.endsWith(".json")).length >= MAX_PENDING_RESULTS) {
      throw new ProxyError("result_queue_full", "Pending result queue is full", 503);
    }
    const now = Date.now();
    const pending: PendingResult = {
      event_id: randomUUID(),
      request_id: requestId,
      client_id: clientId,
      created_at: new Date(now).toISOString(),
      expires_at: new Date(now + RESULT_TTL_MS).toISOString(),
      result,
    };
    const finalPath = join(RESULTS_DIR, `${pending.event_id}.json`);
    const tempPath = join(RESULTS_DIR, `.${pending.event_id}.${randomUUID()}.tmp`);
    try {
      await writeFile(tempPath, JSON.stringify(pending), { encoding: "utf8", mode: 0o600, flag: "wx" });
      await rename(tempPath, finalPath);
    } catch {
      await rm(tempPath, { force: true });
      throw new ProxyError("result_store_failed", "Could not save the pending result", 507);
    }
    this.scheduleExpiry(pending.event_id, now + RESULT_TTL_MS);
    return pending;
  }

  async pending(clientId: string): Promise<PendingResult[]> {
    const result: PendingResult[] = [];
    for (const entry of await readdir(RESULTS_DIR, { withFileTypes: true })) {
      if (!entry.isFile() || !/^[0-9a-f-]{36}\.json$/i.test(entry.name)) continue;
      try {
        const item = JSON.parse(await readFile(join(RESULTS_DIR, entry.name), "utf8")) as PendingResult;
        if (Date.parse(item.expires_at) <= Date.now()) {
          this.clearExpiry(item.event_id);
          await rm(join(RESULTS_DIR, entry.name), { force: true });
        } else if (item.client_id === clientId && item.event_id === entry.name.slice(0, -5)) {
          result.push(item);
        }
      } catch {
        await rm(join(RESULTS_DIR, entry.name), { force: true }).catch(() => undefined);
      }
    }
    return result.sort((a, b) => a.created_at.localeCompare(b.created_at));
  }

  async findPending(clientId: string, requestId: string): Promise<PendingResult | undefined> {
    return (await this.pending(clientId)).find((item) => item.request_id === requestId);
  }

  async acknowledge(clientId: string, eventId: string): Promise<boolean> {
    if (!/^[0-9a-f-]{36}$/i.test(eventId)) return false;
    const path = join(RESULTS_DIR, `${eventId}.json`);
    try {
      const pending = JSON.parse(await readFile(path, "utf8")) as PendingResult;
      if (pending.client_id !== clientId) return false;
      await rm(path, { force: true });
      this.clearExpiry(eventId);
      return true;
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code === "ENOENT") return true;
      return false;
    }
  }

  async expire(): Promise<void> {
    await mkdir(RESULTS_DIR, { recursive: true, mode: 0o700 });
    const now = Date.now();
    for (const entry of await readdir(RESULTS_DIR, { withFileTypes: true })) {
      if (!entry.isFile()) continue;
      const path = join(RESULTS_DIR, entry.name);
      try {
        if (entry.name.endsWith(".tmp")) {
          await rm(path, { force: true });
          continue;
        }
        const item = JSON.parse(await readFile(path, "utf8")) as PendingResult;
        if (!Number.isFinite(Date.parse(item.expires_at)) || Date.parse(item.expires_at) <= now) {
          this.clearExpiry(item.event_id);
          await rm(path, { force: true });
        }
      } catch {
        await rm(path, { force: true }).catch(() => undefined);
      }
    }
  }

  private async allPending(): Promise<PendingResult[]> {
    const result: PendingResult[] = [];
    for (const entry of await readdir(RESULTS_DIR, { withFileTypes: true })) {
      if (!entry.isFile() || !/^[0-9a-f-]{36}\.json$/i.test(entry.name)) continue;
      try { result.push(JSON.parse(await readFile(join(RESULTS_DIR, entry.name), "utf8")) as PendingResult); }
      catch { /* Expiry cleanup will remove invalid files. */ }
    }
    return result;
  }

  private scheduleExpiry(eventId: string, expiresAt: number): void {
    this.clearExpiry(eventId);
    const timer = setTimeout(() => {
      void rm(join(RESULTS_DIR, `${eventId}.json`), { force: true }).catch(() => undefined);
      this.expiryTimers.delete(eventId);
    }, Math.max(0, expiresAt - Date.now()));
    timer.unref();
    this.expiryTimers.set(eventId, timer);
  }

  private clearExpiry(eventId: string): void {
    const timer = this.expiryTimers.get(eventId);
    if (timer) clearTimeout(timer);
    this.expiryTimers.delete(eventId);
  }
}

export async function appendAudit(record: AuditRecord): Promise<void> {
  await mkdir(STATE_DIR, { recursive: true, mode: 0o700 });
  await appendFile(AUDIT_FILE, `${JSON.stringify(record)}\n`, { encoding: "utf8", mode: 0o600 });
  await chmod(AUDIT_FILE, 0o600).catch(() => undefined);
}
