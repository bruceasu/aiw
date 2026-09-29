import { execFileSync, spawnSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, writeFileSync, mkdirSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createInterface } from "node:readline/promises";
import { stdin, stdout } from "node:process";
import { candidates, type Draft } from "./llm.js";
import { loadConfig, projectRoot, type Choice, type Config, type Provider } from "./config.js";

const help = `aiw cz [options]
  --llm / --no-llm        Generate candidates or use the manual wizard
  -N, --candidates N      Number of candidates (default: 3)
  -r, --retry             Reuse the previous commit message as a draft
  --provider NAME         Use only copilot, codex, or openai
  --model MODEL           Override the selected provider model
  --lang LANGUAGE         Use en, zh, ja, or a configured locale

Automatic --llm order: Copilot SDK -> Codex SDK -> OpenAI SDK -> manual wizard.
Configure each provider under [cz.copilot], [cz.codex], or [cz.openai].
Requires Node.js >=22.12.0.`;

type Options = { llm?: boolean; candidates?: number; retry: boolean; provider?: Provider; model?: string; language?: string; help: boolean };
function options(args: string[]): Options {
  const result: Options = { retry: false, help: false };
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (["-h", "--help", "help"].includes(arg)) result.help = true;
    else if (arg === "--llm") result.llm = true;
    else if (arg === "--no-llm") result.llm = false;
    else if (arg === "-r" || arg === "--retry") result.retry = true;
    else if (["-N", "--candidates", "--provider", "--model", "--lang"].includes(arg)) {
      const value = args[++i];
      if (!value || value.startsWith("--")) throw new Error(`missing value for ${arg}`);
      if (arg === "-N" || arg === "--candidates") {
        const count = Number(value);
        if (!Number.isInteger(count) || count < 1) throw new Error(`invalid candidates value: ${value}`);
        result.candidates = count;
      } else if (arg === "--provider") {
        if (!["copilot", "codex", "openai"].includes(value)) throw new Error(`unknown cz provider: ${value}`);
        result.provider = value as Provider;
      } else if (arg === "--model") result.model = value;
      else result.language = value;
    } else throw new Error(`unknown cz option: ${arg}`);
  }
  return result;
}

function output(...args: string[]): string {
  return execFileSync(args[0], args.slice(1), { encoding: "utf8" }).trim();
}
function label(config: Config, key: string, fallback: string): string { return config.messages[key] || fallback; }
function header(draft: Draft): string { return `${draft.type}${draft.scope ? `(${draft.scope})` : ""}: ${draft.subject}`; }
function message(draft: Draft): string {
  const blocks = [header(draft)];
  if (draft.body) blocks.push(draft.body);
  if (draft.breaking) blocks.push(`BREAKING CHANGE: ${draft.breaking}`);
  if (draft.footer) blocks.push(draft.footer);
  return blocks.join("\n\n").replaceAll("\0", "") + "\n";
}
function normalizeText(value: string): string { return value.trim().replaceAll("|", "\n").replaceAll("\0", ""); }

async function ask(rl: ReturnType<typeof createInterface>, prompt: string, initial = ""): Promise<string> {
  const answer = await rl.question(`${prompt}${initial ? ` [${initial}]` : ""}: `);
  return answer.trim() || initial;
}
async function choose(rl: ReturnType<typeof createInterface>, title: string, list: Choice[]): Promise<string> {
  for (;;) {
    stdout.write(`${title}\n`);
    list.forEach((item, index) => stdout.write(`  ${index + 1}) ${item.value || "(empty)"} ${item.name}\n`));
    const answer = await ask(rl, "Enter number, value, or /search text");
    const number = Number(answer);
    if (Number.isInteger(number) && number >= 1 && number <= list.length) return list[number - 1].value;
    if (list.some((item) => item.value === answer)) return answer;
    if (answer.startsWith("/")) {
      const found = list.filter((item) => `${item.value} ${item.name}`.toLowerCase().includes(answer.slice(1).toLowerCase()));
      if (found.length === 1) return found[0].value;
      stdout.write(`Matches: ${found.map((item) => item.value).join(", ") || "none"}\n`);
    } else process.stderr.write("Invalid selection.\n");
  }
}

function editorCommand(config: Config): string {
  try { return config.editor || process.env.GIT_EDITOR || process.env.VISUAL || process.env.EDITOR || output("git", "config", "--get", "core.editor"); }
  catch { return process.platform === "win32" ? "notepad" : "vi"; }
}
async function multiline(rl: ReturnType<typeof createInterface>, config: Config, title: string, initial = ""): Promise<string> {
  const answer = await ask(rl, `${title} (/edit for editor, | for newline)`, initial);
  if (answer === "-") return "";
  if (answer !== "/edit" && answer !== "^e" && answer.toLowerCase() !== "ctrl+e") return normalizeText(answer);
  const dir = mkdtempSync(join(tmpdir(), "aiw-cz-"));
  const path = join(dir, "message.txt");
  try {
    writeFileSync(path, initial || "# Write text below. Lines starting with # are ignored.\n", "utf8");
    const command = editorCommand(config).match(/(?:[^\s"]+|"[^"]*")+/g)?.map((part) => part.replace(/^"|"$/g, "")) || [];
    if (!command.length) throw new Error("no editor available");
    const args = command.slice(1).map((part) => part.replaceAll("{file}", path));
    if (!command.some((part) => part.includes("{file}"))) args.push(path);
    const run = spawnSync(command[0], args, { stdio: "inherit" });
    if (run.error || run.status !== 0) throw run.error || new Error("editor failed");
    return readFileSync(path, "utf8").split("\n").filter((line) => !line.startsWith("#")).join("\n").trim();
  } finally { rmSync(dir, { recursive: true, force: true }); }
}

async function wizard(rl: ReturnType<typeof createInterface>, config: Config, existing?: Draft): Promise<Draft> {
  const type = await choose(rl, label(config, "type", "Select commit type"), config.types);
  const scope = await choose(rl, label(config, "scope", "Scope"), [
    { value: "-", name: "no scope" }, { value: ".", name: "custom scope" }, ...config.scopes,
  ]);
  const resolvedScope = scope === "-" ? "" : scope === "." ? await ask(rl, label(config, "custom_scope", "Custom scope")) : scope;
  let subject = existing?.subject || "";
  if (existing) subject = await ask(rl, label(config, "subject", "Subject"), subject);
  while (!subject || [...subject].length > config.maxSubjectLength) {
    subject = await ask(rl, `${label(config, "subject", "Subject")} (max ${config.maxSubjectLength})`, subject);
    if (!subject) process.stderr.write(`${label(config, "subject_required", "Subject is required")}\n`);
    else if ([...subject].length > config.maxSubjectLength) process.stderr.write(`${label(config, "subject_too_long", "Subject is too long")}\n`);
  }
  const body = await multiline(rl, config, label(config, "body", "Body"), existing?.body);
  const breaking = await multiline(rl, config, label(config, "breaking", "Breaking changes"), existing?.breaking);
  const prefix = await choose(rl, label(config, "footer_prefixes", "Issue prefix"), [
    { value: "-", name: "skip" }, { value: "#", name: "#" }, { value: "refs", name: "refs" },
    { value: "closes", name: "closes" }, { value: ".", name: "custom" },
  ]);
  const resolvedPrefix = prefix === "." ? await ask(rl, label(config, "custom_footer_prefix", "Custom issue prefix")) : prefix === "-" ? "" : prefix;
  let footer = await multiline(rl, config, label(config, "footer", "Related issue"), existing?.footer);
  if (footer && resolvedPrefix && !footer.toLowerCase().startsWith(resolvedPrefix.toLowerCase())) footer = `${resolvedPrefix} ${footer}`;
  return { type, scope: resolvedScope, subject: subject.replace(/[\r\n]/g, " "), body, breaking, footer };
}

function previousCommit(): Draft {
  const text = output("git", "log", "-1", "--pretty=%B");
  const [first, ...rest] = text.split("\n");
  const match = first.match(/^([a-z][a-z0-9-]*)(?:\(([^)]+)\))?!?:\s*(.+)$/);
  if (!match) return { type: "chore", scope: "", subject: first, body: rest.join("\n").trim(), breaking: "", footer: "" };
  return { type: match[1], scope: match[2] || "", subject: match[3], body: rest.join("\n").trim(), breaking: "", footer: "" };
}

async function review(rl: ReturnType<typeof createInterface>, config: Config, draft: Draft): Promise<void> {
  for (;;) {
    stdout.write(`\n--- ${label(config, "preview", "Commit message preview")} ---\n${message(draft)}--------------------------------\n`);
    const action = (await ask(rl, `${label(config, "confirm_commit", "Commit this message")} [y(commit), e(edit), n(cancel)]`)).toLowerCase();
    if (action === "n" || action === "no") return;
    if (action === "e" || action === "edit") { draft = await wizard(rl, config, draft); continue; }
    if (action === "y" || action === "yes") {
      const root = projectRoot();
      const tempDir = join(root, ".ai", "tmp");
      mkdirSync(tempDir, { recursive: true });
      const dir = mkdtempSync(join(tempDir, "cz-"));
      const path = join(dir, "message.txt");
      try {
        writeFileSync(path, message(draft), "utf8");
        const result = spawnSync("git", ["commit", "-F", path], { stdio: "inherit" });
        if (result.error || result.status !== 0) throw result.error || new Error(`git commit failed (${result.status})`);
        return;
      } finally { rmSync(dir, { recursive: true, force: true }); }
    }
    process.stderr.write("Invalid selection.\n");
  }
}

async function main(): Promise<void> {
  const opts = options(process.argv.slice(2));
  if (opts.help) { stdout.write(help + "\n"); return; }
  const [major, minor] = process.versions.node.split(".").map(Number);
  if (major < 22 || major === 22 && minor < 12) throw new Error("cz requires Node.js >=22.12.0");
  if (!process.stdin.isTTY) throw new Error("cz requires an interactive terminal");
  const config = loadConfig(opts.language);
  if (opts.llm !== undefined) config.llm = opts.llm;
  if (opts.candidates) config.candidates = opts.candidates;
  const provider = opts.provider || config.provider;
  if (opts.model) config.providers[provider || "copilot"].model = opts.model;
  if (!output("git", "diff", "--cached", "--name-only")) throw new Error("no staged changes; run git add first");
  const rl = createInterface({ input: stdin, output: stdout });
  try {
    let draft: Draft | undefined;
    if (opts.retry) draft = previousCommit();
    else if (config.llm) {
      let generated = await candidates(config, provider);
      while (generated) {
        if (generated.length === 1) { draft = generated[0]; break; }
        const selected = await choose(rl, label(config, "candidate", "Select AI candidate"), [
          ...generated.map((item, index) => ({ value: String(index + 1), name: header(item) })),
          { value: "r", name: label(config, "regenerate", "Regenerate candidates") },
        ]);
        if (selected === "r") generated = await candidates(config, provider);
        else { draft = generated[Number(selected) - 1]; break; }
      }
    }
    await review(rl, config, draft || await wizard(rl, config));
  } finally { rl.close(); }
}

main().catch((error: unknown) => {
  process.stderr.write(`cz: ${error instanceof Error ? error.message : String(error)}\n`);
  process.exitCode = 1;
});
