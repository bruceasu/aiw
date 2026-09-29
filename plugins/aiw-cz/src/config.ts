import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { parse } from "smol-toml";

export type Provider = "copilot" | "codex" | "openai";
export type Choice = { value: string; name: string };
export type ProviderConfig = { model?: string; apiKey?: string; baseURL?: string };
export type Config = {
  llm: boolean;
  provider?: Provider;
  candidates: number;
  editor?: string;
  maxSubjectLength: number;
  language: string;
  messages: Record<string, string>;
  types: Choice[];
  scopes: Choice[];
  providers: Record<Provider, ProviderConfig>;
};

type Table = Record<string, unknown>;
const pluginDir = dirname(dirname(fileURLToPath(import.meta.url)));
const asTable = (value: unknown): Table => value && typeof value === "object" && !Array.isArray(value) ? value as Table : {};
const asString = (value: unknown): string | undefined => typeof value === "string" ? value.trim() : undefined;
const firstExisting = (...paths: string[]): string | undefined => paths.find(existsSync);

export function projectRoot(): string {
  return execFileSync("git", ["rev-parse", "--show-toplevel"], { encoding: "utf8" }).trim();
}

function readToml(path: string | undefined): Table {
  if (!path) return {};
  return asTable(parse(readFileSync(path, "utf8")));
}

function choices(value: unknown): Choice[] | undefined {
  if (!Array.isArray(value)) return undefined;
  return value.flatMap((item) => {
    const row = asTable(item);
    const name = asString(row.name);
    const code = asString(row.value);
    return code ? [{ value: code, name: name || code }] : [];
  });
}

function localeName(raw: string): string {
  return raw.toLowerCase().split(/[.@_-]/, 1)[0] || "en";
}

function applyFile(config: Config, data: Table, localeOverrides: Table): void {
  const cz = asTable(data.cz);
  const i18n = asTable(data.i18n);
  const language = asString(i18n.default_language);
  if (language) config.language = localeName(language);
  if (typeof cz.llm === "boolean") config.llm = cz.llm;
  if (typeof cz.use_llm === "boolean") config.llm = cz.use_llm;
  const provider = asString(cz.provider || cz.llm_provider);
  if (provider) {
    if (!["auto", "copilot", "codex", "openai"].includes(provider)) throw new Error(`unsupported cz provider: ${provider}`);
    config.provider = provider === "auto" ? undefined : provider as Provider;
  }
  if (Number.isInteger(cz.candidates) && Number(cz.candidates) > 0) config.candidates = Number(cz.candidates);
  const editor = asString(cz.editor);
  if (editor) config.editor = editor;
  if (Number.isInteger(cz.max_subject_length) && Number(cz.max_subject_length) > 0) config.maxSubjectLength = Number(cz.max_subject_length);
  for (const providerName of ["copilot", "codex", "openai"] as const) {
    const section = asTable(cz[providerName]);
    const target = config.providers[providerName];
    const model = asString(section.model);
    if (model) target.model = model;
    const key = asString(section.api_key);
    if (key) target.apiKey = key;
    const url = asString(section.base_url);
    if (url) target.baseURL = url;
  }
  const types = choices(cz.types);
  if (types) config.types = types;
  const scopes = choices(cz.scopes);
  if (scopes) config.scopes = scopes;
  for (const [key, value] of Object.entries(asTable(cz.messages))) {
    if (typeof value === "string") config.messages[key.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`)] = value;
  }
  Object.assign(localeOverrides, asTable(cz.locales));
}

export function loadConfig(requestedLanguage?: string): Config {
  const root = projectRoot();
  const config: Config = {
    llm: false, candidates: 3, maxSubjectLength: 72, language: "",
    messages: {}, types: [], scopes: [],
    providers: { copilot: {}, codex: {}, openai: {} },
  };
  const overrides: Table = {};
  const files = [
    firstExisting(join(pluginDir, "cz.toml"), join(pluginDir, ".cz.toml")),
    process.env.AIW_ROOT ? firstExisting(join(process.env.AIW_ROOT, "aiw.toml"), join(process.env.AIW_ROOT, ".aiw.toml")) : undefined,
    firstExisting(join(root, "aiw.toml"), join(root, ".aiw.toml")),
  ];
  for (const path of files) applyFile(config, readToml(path), overrides);
  config.language = localeName(requestedLanguage || config.language || process.env.LC_ALL || process.env.LANG || "en");
  const builtIn = firstExisting(join(pluginDir, "locales", `${config.language}.toml`));
  if (!builtIn && !asTable(overrides[config.language]).messages) {
    if (requestedLanguage) throw new Error(`unsupported language: ${requestedLanguage}`);
    config.language = "en";
  }
  const locale = readToml(firstExisting(join(pluginDir, "locales", `${config.language}.toml`), join(pluginDir, "locales", "en.toml")));
  const custom = asTable(overrides[config.language]);
  config.messages = { ...asTable(locale.messages), ...config.messages, ...asTable(custom.messages) } as Record<string, string>;
  config.types = choices(custom.types) ?? (config.types.length ? config.types : choices(locale.types) || []);
  config.scopes = choices(custom.scopes) ?? (config.scopes.length ? config.scopes : choices(locale.scopes) || []);
  for (const name of ["copilot", "codex", "openai"] as const) {
    const prefix = `CZ_${name.toUpperCase()}_`;
    config.providers[name].model = process.env[`${prefix}MODEL`] || config.providers[name].model;
    config.providers[name].apiKey = process.env[`${prefix}API_KEY`] || config.providers[name].apiKey;
    config.providers[name].baseURL = process.env[`${prefix}BASE_URL`] || config.providers[name].baseURL;
  }
  const selected = process.env.CZ_LLM_PROVIDER;
  if (selected) {
    if (!["auto", "copilot", "codex", "openai"].includes(selected)) throw new Error(`unsupported CZ_LLM_PROVIDER: ${selected}`);
    config.provider = selected === "auto" ? undefined : selected as Provider;
  }
  if (!config.types.length) throw new Error("cz requires at least one commit type");
  return config;
}
