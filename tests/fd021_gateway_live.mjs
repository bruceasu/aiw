// FD-021 R4: one real model execution maximum; never log key, prompt or response body.
import { readFile, writeFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { dirname, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { createHash } from 'node:crypto';
const root = resolve(import.meta.dirname, '..');
const service = resolve(root, 'program/aiw-agent');
const require = createRequire(resolve(service, 'package.json'));
const ts = require('typescript');
const hashes = {};
const loaded = new Map();
function hash(value) { return createHash('sha256').update(value).digest('hex'); }
async function load(path) {
  if (loaded.has(path)) return loaded.get(path);
  const source = await readFile(path, 'utf8');
  hashes[path.slice(root.length + 1).replaceAll('\\', '/')] = hash(source);
  let code = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 } }).outputText;
  for (const name of new Set([...code.matchAll(/(?:from\s*|import\s*)["']([^"']+)["']/g)].map(m => m[1]))) {
    const replacement = name.startsWith('.') ? await load(resolve(dirname(path), name.replace(/\.js$/, '.ts'))) : name.startsWith('node:') ? null : pathToFileURL(require.resolve(name)).href;
    if (replacement) code = code.replace(/(from\s*|import\s*)(["'])([^"']+)\2/g, (match, prefix, quote, specifier) => specifier === name ? prefix + JSON.stringify(replacement) : match);
  }
  const url = 'data:text/javascript;base64,' + Buffer.from(code).toString('base64'); loaded.set(path, url); return url;
}
function count(value) { return Number.isSafeInteger(value) && value >= 0 ? value : null; }
function safeIdentifier(value) { return typeof value === 'string' && /^[A-Za-z0-9._:/-]{1,128}$/.test(value) ? value : null; }
function safeUsage(usage) {
  return { input_tokens: count(usage?.input_tokens), output_tokens: count(usage?.output_tokens), total_tokens: count(usage?.total_tokens ?? usage?.openai_usage?.total_tokens), cached_input_tokens: count(usage?.input_tokens_details?.cached_tokens ?? usage?.openai_usage?.cached_input_tokens), reasoning_output_tokens: count(usage?.output_tokens_details?.reasoning_tokens ?? usage?.openai_usage?.reasoning_output_tokens), cost: null };
}
const fd = await readFile(resolve(root, 'docs/features/FD-021_AIW_AGENT_OPENAI_SDK.md'), 'utf8');
const script = await readFile(import.meta.filename, 'utf8');
const sdkVersion = JSON.parse(await readFile(resolve(service, 'node_modules/openai/package.json'), 'utf8')).version;
const savedEnv = { ...process.env };
const savedFetch = globalThis.fetch;
const output = { fd_id: 'FD-021', fd_revision: Number(fd.match(/\*\*Revision:\*\*\s*(\d+)/)?.[1]), fd_digest: hash(fd), test_source_digest: hash(script), source_digests: hashes, sdk_version: sdkVersion, endpoint: 'http://127.0.0.1:43127/v1/responses', model: 'gpt-6-luna', requests: [], provider: { outcome: 'not-run' }, direct_sdk: { outcome: 'not-run' }, model_execution_limit: 1, model_completion_observed: false, authorization: 'Explicit user: allow one minimal real model request; no retry', raw_path: 'docs/features/reports/FD-021-test-results-r4.json' };
let overallTimer;
let phase = 'provider';
let allowFallback = false;
let key;
let liveController;
try {
  // Load modules before reading any credential. No provider implementation analysis.
  delete process.env.AIW_AGENT_PROXY_HOST; delete process.env.AIW_AGENT_PROXY_PORT; delete process.env.OPENAI_LOG;
  const { complete } = await import(await load(resolve(service, 'src/providers.ts')));
  const { default: OpenAI } = await import(pathToFileURL(require.resolve('openai')).href);
  const config = JSON.parse(await readFile('D:/green/aiw/plugins/aiw-gw/gateway.json', 'utf8'));
  const principal = config.principals?.find(p => p.id === 'team-a' && p.enabled === true);
  if (config.listen !== '127.0.0.1:43127' || !principal?.allowed_models?.includes('gpt-6-luna') || typeof principal.keys?.[0] !== 'string' || !principal.keys[0]) throw new Error('unsafe-local-config');
  key = principal.keys[0];
  process.env.OPENAI_API_KEY = key;
  process.env.OPENAI_BASE_URL = 'http://127.0.0.1:43127/v1';
  liveController = new AbortController();
  overallTimer = setTimeout(() => liveController.abort(), 65_000);
  globalThis.fetch = async (url, options) => {
    const endpoint = new URL(String(url));
    if (endpoint.origin !== 'http://127.0.0.1:43127' || endpoint.pathname !== '/v1/responses' || endpoint.search || endpoint.hash || endpoint.username || endpoint.password || options?.method !== 'POST') throw new Error('restricted-live-target');
    if (output.requests.length >= 2 || (output.requests.length === 1 && (!allowFallback || phase !== 'direct_sdk'))) throw new Error('restricted-request-budget');
    const body = JSON.parse(options.body);
    const record = { phase, method: 'POST', request_fields: Object.keys(body).sort(), http_status: null, request_id: null };
    output.requests.push(record);
    const signal = options.signal ? AbortSignal.any([options.signal, liveController.signal]) : liveController.signal;
    const response = await savedFetch(url, { ...options, signal, redirect: 'error' });
    record.http_status = response.status;
    record.request_id = safeIdentifier(response.headers.get('x-request-id'));
    return response;
  };
  try {
    const result = await complete({ client_id: 'fd021-live', provider: 'openai', model: 'gpt-6-luna', output_format: 'markdown', prompt: 'Reply exactly FD021_OK.' });
    const textMatch = result.text.trim() === 'FD021_OK';
    output.provider = { outcome: textMatch ? 'succeeded' : 'unexpected_output', text_present: result.text.trim().length > 0, text_match: textMatch, response_status: safeIdentifier(result.usage?.openai_usage?.response_status), usage: safeUsage(result.usage) };
    output.model_completion_observed = true;
  } catch (error) {
    const code = ['provider_error', 'provider_timeout', 'provider_not_configured', 'result_too_large', 'invalid_provider_output'].includes(error?.code) ? error.code : 'unclassified_failure';
    const first = output.requests[0];
    output.provider = { outcome: 'failed', error_code: code, service_status: Number.isInteger(error?.statusCode) ? error.statusCode : null, upstream_http_status: first?.http_status ?? null };
    // Gateway HTTP 400 is statically established as decodeRequest rejection before Runner.
    allowFallback = output.requests.length === 1 && first.http_status === 400 && code === 'provider_error' && error.message === 'OpenAI returned HTTP 400' && !liveController.signal.aborted;
  }
  if (allowFallback) {
    phase = 'direct_sdk';
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 60_000);
    try {
      const client = new OpenAI({ apiKey: key, baseURL: 'http://127.0.0.1:43127/v1', maxRetries: 0, timeout: 60_000, logLevel: 'off', fetchOptions: { redirect: 'error' } });
      const result = await client.post('/responses', { body: { model: 'gpt-6-luna', input: 'Reply exactly FD021_OK.' }, signal: AbortSignal.any([controller.signal, liveController.signal]) });
      const text = typeof result.output_text === 'string' ? result.output_text : (Array.isArray(result.output) ? result.output.flatMap(item => Array.isArray(item.content) ? item.content.filter(part => part.type === 'output_text' && typeof part.text === 'string').map(part => part.text) : []).join('') : '');
      const textMatch = text.trim() === 'FD021_OK';
      output.direct_sdk = { outcome: textMatch ? 'succeeded' : 'unexpected_output', text_present: text.trim().length > 0, text_match: textMatch, response_status: safeIdentifier(result.status), usage: safeUsage(result.usage) };
      output.model_completion_observed = true;
    } catch (error) {
      output.direct_sdk = { outcome: 'failed', error_code: controller.signal.aborted || liveController.signal.aborted ? 'request_timeout' : 'sdk_request_failed', http_status: Number.isInteger(error?.status) ? error.status : null };
    } finally { clearTimeout(timer); }
  }
} catch {
  output.harness_outcome = 'stopped-safe-error';
} finally {
  if (overallTimer) clearTimeout(overallTimer);
  globalThis.fetch = savedFetch;
  for (const name of Object.keys(process.env)) if (!(name in savedEnv)) delete process.env[name];
  Object.assign(process.env, savedEnv);
  key = undefined;
}
output.live_request_count = output.requests.length;
output.fallback_policy = 'Only exact Provider HTTP 400 pre-Runner rejection permits one minimal direct SDK request; all other failures stop.';
await writeFile(resolve(root, output.raw_path), JSON.stringify(output, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify(output, null, 2));
process.exitCode = output.provider.outcome === 'succeeded' ? 0 : output.direct_sdk.outcome === 'succeeded' ? 2 : 1;
