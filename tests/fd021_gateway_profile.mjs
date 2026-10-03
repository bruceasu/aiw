// FD-021 R5: explicit API profiles, offline black-box contracts only.
import assert from 'node:assert/strict';
import { readFile, writeFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { dirname, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { createHash } from 'node:crypto';
const root = resolve(import.meta.dirname, '..');
const service = resolve(root, 'program/aiw-agent');
const require = createRequire(resolve(service, 'package.json'));
const ts = require('typescript');
const hashes = {}, modules = new Map();
function hash(value) { return createHash('sha256').update(value).digest('hex'); }
async function load(path) {
  if (modules.has(path)) return modules.get(path);
  const source = await readFile(path, 'utf8');
  hashes[path.slice(root.length + 1).replaceAll('\\', '/')] = hash(source);
  let code = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 } }).outputText;
  for (const name of new Set([...code.matchAll(/(?:from\s*|import\s*)["']([^"']+)["']/g)].map(m => m[1]))) {
    const replacement = name.startsWith('.') ? await load(resolve(dirname(path), name.replace(/\.js$/, '.ts'))) : name.startsWith('node:') ? null : pathToFileURL(require.resolve(name)).href;
    if (replacement) code = code.replace(/(from\s*|import\s*)(["'])([^"']+)\2/g, (match, prefix, quote, specifier) => specifier === name ? prefix + JSON.stringify(replacement) : match);
  }
  const url = 'data:text/javascript;base64,' + Buffer.from(code).toString('base64');
  modules.set(path, url); return url;
}
const fdSource = await readFile(resolve(root, 'docs/features/FD-021_AIW_AGENT_OPENAI_SDK.md'), 'utf8');
const testSource = await readFile(import.meta.filename, 'utf8');
const savedEnv = { ...process.env }, savedFetch = globalThis.fetch;
delete process.env.AIW_AGENT_PROXY_HOST; delete process.env.AIW_AGENT_PROXY_PORT; delete process.env.OPENAI_LOG;
let calls = [], responseText = 'profile answer';
globalThis.fetch = async (url, options) => {
  calls.push({ url: String(url), method: options.method, redirect: options.redirect, body: JSON.parse(options.body) });
  return new Response(JSON.stringify({ id: 'resp_profile', object: 'response', model: 'gpt-6-luna', status: 'completed', service_tier: 'default', output: [{ type: 'message', role: 'assistant', content: [{ type: 'output_text', text: responseText, annotations: [] }] }], usage: { input_tokens: 9, output_tokens: 3, total_tokens: 12 } }), { headers: { 'content-type': 'application/json' } });
};
const request = { client_id: 'fd021-profile', provider: 'openai', model: 'gpt-6-luna', output_format: 'markdown', prompt: 'profile test prompt' };
const results = [];
let complete;
async function test(name, profile, fn) {
  calls = []; responseText = 'profile answer';
  process.env.OPENAI_API_KEY = 'dummy-fd021-profile-key'; process.env.OPENAI_BASE_URL = 'http://127.0.0.1:43127/v1';
  if (profile === undefined) delete process.env.OPENAI_API_PROFILE; else process.env.OPENAI_API_PROFILE = profile;
  try { await fn(); results.push({ name, status: 'passed', request_count: calls.length }); }
  catch (error) { results.push({ name, status: 'failed', request_count: calls.length, message: String(error.message).slice(0, 1000) }); }
}
async function rejection(input, code, status) {
  try { await complete(input); assert.fail('Expected rejection'); }
  catch (error) { if (error.code === 'ERR_ASSERTION') throw error; assert.equal(error.code, code); assert.equal(error.statusCode, status); assert.equal(calls.length, 0); }
}
function transport() { assert.equal(calls.length, 1); assert.equal(calls[0].method, 'POST'); assert.equal(calls[0].url, 'http://127.0.0.1:43127/v1/responses'); assert.equal(calls[0].redirect, 'error'); }
try {
  ({ complete } = await import(await load(resolve(service, 'src/providers.ts'))));
  for (const profile of [undefined, 'openai']) await test(profile === undefined ? '缺省official模式保留token上限和reasoning' : '显式official模式保留token上限和reasoning', profile, async () => {
    const result = await complete({ ...request, effort: 'high', include_reasoning_summary: true });
    transport(); assert.equal(calls[0].body.max_output_tokens, 8192); assert.deepEqual(calls[0].body.reasoning, { effort: 'high', summary: 'auto' });
    assert.equal(calls[0].body.input, request.prompt); assert.equal(result.text, 'profile answer'); assert.equal(result.reasoning_summary, null);
  });
  await test('Gateway最小文本参数与标准output/usage', 'aiw_gateway', async () => {
    const result = await complete({ ...request, include_reasoning_summary: false });
    transport(); assert.deepEqual(Object.keys(calls[0].body).sort(), ['input', 'model']);
    assert.equal(calls[0].body.model, 'gpt-6-luna'); assert.equal(calls[0].body.input, request.prompt);
    assert.equal(result.text, 'profile answer'); assert.equal(result.usage.input_tokens, 9); assert.equal(result.usage.output_tokens, 3); assert.equal(result.usage.cost, null);
    assert.equal(result.usage.openai_usage.total_tokens, 12); assert.equal(result.usage.openai_usage.response_id, 'resp_profile'); assert.equal(result.usage.openai_usage.response_status, 'completed');
    assert.equal(result.usage.openai_usage.cached_input_tokens, null); assert.equal(Object.hasOwn(result, 'reasoning_summary'), false);
  });
  await test('Gateway instructions与JSON仅发送支持字段', 'aiw_gateway', async () => {
    responseText = '{"ok":true}';
    const result = await complete({ ...request, output_format: 'json', system_prompt: 'profile system instruction' });
    transport(); assert.deepEqual(Object.keys(calls[0].body).sort(), ['input', 'instructions', 'model', 'text']);
    assert.equal(calls[0].body.instructions, 'profile system instruction'); assert.deepEqual(calls[0].body.text, { format: { type: 'json_object' } }); assert.match(calls[0].body.input, /Return JSON only/);
    assert.deepEqual(JSON.parse(result.text), { ok: true });
  });
  await test('Gateway effort调用前拒绝', 'aiw_gateway', () => rejection({ ...request, effort: 'medium' }, 'unsupported_option', 400));
  await test('Gateway reasoning summary调用前拒绝', 'aiw_gateway', () => rejection({ ...request, include_reasoning_summary: true }, 'unsupported_option', 400));
  await test('未知profile配置调用前拒绝', 'unknown_profile', () => rejection(request, 'provider_not_configured', 503));
  await test('空profile配置调用前拒绝', '', () => rejection(request, 'provider_not_configured', 503));
} catch (error) { results.push({ name: 'profile测试装载', status: 'failed', message: String(error.message).slice(0, 1000) }); }
finally {
  globalThis.fetch = savedFetch;
  for (const name of Object.keys(process.env)) if (!(name in savedEnv)) delete process.env[name];
  Object.assign(process.env, savedEnv);
}
const output = { fd_id: 'FD-021', fd_revision: Number(fdSource.match(/\*\*Revision:\*\*\s*(\d+)/)?.[1]), fd_digest: hash(fdSource), test_source_digest: hash(testSource), source_digests: hashes, sdk_version: JSON.parse(await readFile(resolve(service, 'node_modules/openai/package.json'), 'utf8')).version, transport: 'real installed SDK; in-process fetch mock; zero network/socket/model execution', results, passed: results.filter(r => r.status === 'passed').length, failed: results.filter(r => r.status === 'failed').length, raw_path: 'docs/features/reports/FD-021-test-results-r5.json' };
await writeFile(resolve(root, output.raw_path), JSON.stringify(output, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify(output, null, 2)); process.exitCode = output.failed ? 1 : 0;
