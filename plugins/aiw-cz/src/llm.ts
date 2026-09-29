import { execFileSync } from "node:child_process";
import { existsSync } from "node:fs";
import { delimiter, dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import type { Config, Provider } from "./config.js";

export type Draft = { type: string; scope: string; subject: string; body: string; breaking: string; footer: string };
const order: Provider[] = ["copilot", "codex", "openai"];
const schema = {
  type: "object", additionalProperties: false,
  properties: { candidates: { type: "array", items: {
    type: "object", additionalProperties: false,
    properties: Object.fromEntries(["type", "scope", "subject", "body", "breaking", "footer"].map((key) => [key, { type: "string" }])),
    required: ["type", "scope", "subject", "body", "breaking", "footer"],
  } } }, required: ["candidates"],
};

function git(...args: string[]): string {
  return execFileSync("git", args, { encoding: "utf8", maxBuffer: 8 * 1024 * 1024 }).trim();
}

export function buildPrompt(config: Config): { prompt: string; context: string } {
  const files = git("diff", "--cached", "--name-only");
  const diff = git("diff", "--cached", "--");
  let history = "";
  try { history = git("log", "--oneline", "-n", "5"); } catch { /* A new repository has no history. */ }
  const context = `${files}\n${diff}\n${history}`;
  const preview = Array.from(diff).slice(0, 1000).join("");
  return { context, prompt: `Generate ${config.candidates} Conventional Commit candidates. Return only JSON: {"candidates":[{"type":"feat","scope":"","subject":"","body":"","breaking":"","footer":""}]}. All six fields are strings. Allowed types: ${config.types.map((type) => type.value).join(", ")}. Keep subjects concise and imperative. Use the language of the changed code and comments. Do not invent issue references. Do not execute tools or edit files.\n\nChanged files:\n${files}\nRecent commits:\n${history}\nStaged diff:\n${preview}${Array.from(diff).length > 1000 ? "\n... (truncated)" : ""}` };
}

function parseCandidates(raw: string, config: Config, context: string): Draft[] {
  const match = raw.match(/\{[\s\S]*\}/);
  const parsed = JSON.parse(match?.[0] || raw) as { candidates?: unknown };
  if (!Array.isArray(parsed.candidates)) throw new Error("missing candidates array");
  const allowed = new Set(config.types.map((type) => type.value));
  const issueRefs = new Set(context.match(/#\d+/g) || []);
  const results = parsed.candidates.flatMap((candidate): Draft[] => {
    if (!candidate || typeof candidate !== "object") return [];
    const row = candidate as Record<string, unknown>;
    if (typeof row.type !== "string" || !allowed.has(row.type.trim()) || typeof row.subject !== "string" || !row.subject.trim()) return [];
    const field = (key: string): string => typeof row[key] === "string" ? (row[key] as string).replaceAll("\0", "").trim() : "";
    const footer = field("footer");
    if ([...footer.matchAll(/#\d+/g)].some((match) => !issueRefs.has(match[0]))) return [];
    const subject = field("subject").replace(/[\r\n]+/g, " ");
    if ([...subject].length > config.maxSubjectLength) return [];
    return [{ type: field("type"), scope: field("scope").replace(/[\r\n()]/g, ""), subject, body: field("body"), breaking: field("breaking"), footer }];
  });
  if (!results.length) throw new Error("no valid commit candidates");
  return results.slice(0, config.candidates);
}

async function generate(provider: Provider, prompt: string, config: Config): Promise<string> {
  const settings = config.providers[provider];
  const pluginDir = dirname(dirname(fileURLToPath(import.meta.url)));
  const packaged = existsSync(join(pluginDir, "release-manifest.json"));
  if (provider === "copilot") {
    if (packaged && !process.env.COPILOT_CLI_PATH) {
      const glibcVersion = (process.report?.getReport() as
        { header?: { glibcVersionRuntime?: string } } | undefined)?.header?.glibcVersionRuntime;
      const platform = process.platform === "linux" && !glibcVersion
        ? `linuxmusl-${process.arch}` : `${process.platform}-${process.arch}`;
      const binary = join(pluginDir, "runtime", "copilot", "prebuilds", platform,
        process.platform === "win32" ? "copilot-runtime.exe" : "copilot-runtime");
      if (!existsSync(binary)) throw new Error(`Copilot runtime missing: ${binary}`);
      process.env.COPILOT_CLI_PATH = binary;
    }
    const { CopilotClient } = await import("@github/copilot-sdk");
    const client = new CopilotClient(settings.apiKey ? { gitHubToken: settings.apiKey } : undefined);
    await client.start();
    try {
      const session = await client.createSession({
        ...(settings.model ? { model: settings.model } : {}),
        onPermissionRequest: async () => ({ kind: "reject" as const }),
      });
      try {
        const response = await session.sendAndWait({ prompt }, 60_000);
        return response?.data.content || "";
      } finally { await session.disconnect(); }
    } finally { await client.stop(); }
  }
  if (provider === "codex") {
    let binary: string | undefined;
    let codexEnv: Record<string, string> | undefined;
    if (packaged) {
      const triples: Record<string, string> = {
        "win32-x64": "x86_64-pc-windows-msvc",
        "win32-arm64": "aarch64-pc-windows-msvc",
        "linux-x64": "x86_64-unknown-linux-musl",
        "linux-arm64": "aarch64-unknown-linux-musl",
      };
      const triple = triples[`${process.platform}-${process.arch}`];
      if (!triple) throw new Error(`Codex runtime unsupported: ${process.platform}-${process.arch}`);
      binary = join(pluginDir, "runtime", "codex", "vendor", triple, "bin",
        process.platform === "win32" ? "codex.exe" : "codex");
      if (!existsSync(binary)) throw new Error(`Codex runtime missing: ${binary}`);
      const pathDir = join(pluginDir, "runtime", "codex", "vendor", triple, "codex-path");
      if (existsSync(pathDir)) {
        codexEnv = Object.fromEntries(Object.entries(process.env)
          .filter((entry): entry is [string, string] => entry[1] !== undefined));
        const pathKey = Object.keys(codexEnv).find((key) => key.toLowerCase() === "path") || "PATH";
        codexEnv[pathKey] = `${pathDir}${delimiter}${codexEnv[pathKey] || ""}`;
      }
    }
    const { Codex } = await import("@openai/codex-sdk");
    const codex = new Codex({ ...(settings.apiKey ? { apiKey: settings.apiKey } : {}),
      ...(binary ? { codexPathOverride: binary } : {}), ...(codexEnv ? { env: codexEnv } : {}) });
    const thread = codex.startThread({ workingDirectory: process.cwd(), sandboxMode: "read-only", approvalPolicy: "never", ...(settings.model ? { model: settings.model } : {}) });
    const turn = await thread.run(prompt, { outputSchema: schema, signal: AbortSignal.timeout(60_000) });
    return turn.finalResponse || "";
  }
  const { default: OpenAI } = await import("openai");
  const apiKey = settings.apiKey || process.env.OPENAI_API_KEY;
  if (!apiKey) throw new Error("OpenAI API key is not configured");
  if (!settings.model) throw new Error("OpenAI model is not configured");
  const client = new OpenAI({ apiKey, ...(settings.baseURL ? { baseURL: settings.baseURL } : {}), timeout: 60_000, maxRetries: 0 });
  const response = await client.responses.create({ model: settings.model, input: prompt, text: { format: { type: "json_schema", name: "commit_candidates", strict: true, schema } } });
  return response.output_text;
}

export async function candidates(config: Config, selected?: Provider): Promise<Draft[] | undefined> {
  const { prompt, context } = buildPrompt(config);
  const providers = selected ? [selected] : order;
  for (const provider of providers) {
    try {
      return parseCandidates(await generate(provider, prompt, config), config, context);
    } catch (error) {
      const reason = error instanceof Error ? error.message : String(error);
      process.stderr.write(`cz: ${provider} unavailable: ${reason.slice(0, 180)}\n`);
    }
  }
  process.stderr.write("cz: LLM unavailable; continuing with the manual wizard.\n");
  return undefined;
}
