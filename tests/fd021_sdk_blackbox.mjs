// FD-021 external black-box tests: real installed SDK, in-process fetch only.
import assert from 'node:assert/strict';
import { readFile, writeFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { dirname, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { createHash } from 'node:crypto';

const root = resolve(import.meta.dirname, '..');
const service = resolve(root, 'src/programs/aiw-agent');
const require = createRequire(resolve(service, 'package.json'));
const ts = require('typescript');
const cache = new Map();
const revisions = {};
// Source text is used only to transpile/load modules, never to derive assertions.
async function load(path) {
  if (cache.has(path)) return cache.get(path);
  const source = await readFile(path, 'utf8');
  revisions[path.slice(root.length + 1).replaceAll('\\', '/')] = createHash('sha256').update(source).digest('hex');
  let code = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 } }).outputText;
  const imports = [...code.matchAll(/(?:from\s*|import\s*)["']([^"']+)["']/g)].map(m => m[1]);
  for (const name of new Set(imports)) {
    let replacement;
    if (name.startsWith('.')) replacement = await load(resolve(dirname(path), name.replace(/\.js$/, '.ts')));
    else if (!name.startsWith('node:')) replacement = pathToFileURL(require.resolve(name)).href;
    if (replacement) code = code.replace(/(from\s*|import\s*)(["'])([^"']+)\2/g, (match, prefix, quote, specifier) => specifier === name ? prefix + JSON.stringify(replacement) : match);
  }
  const url = 'data:text/javascript;base64,' + Buffer.from(code).toString('base64');
  cache.set(path, url);
  return url;
}

const oldEnv = { ...process.env };
process.env.OPENAI_API_KEY = 'dummy-fd021-key';
process.env.OPENAI_BASE_URL = 'http://127.0.0.1:43127/v1';
delete process.env.AIW_AGENT_PROXY_HOST;
delete process.env.AIW_AGENT_PROXY_PORT;
delete process.env.OPENAI_LOG;
let handler;
let calls = [];
const originalFetch = globalThis.fetch;
globalThis.fetch = async (url, options) => {
  calls.push({ url: String(url), method: options?.method, body: JSON.parse(options?.body || '{}'), signal: options?.signal });
  return handler(url, options);
};
const originalSetTimeout = globalThis.setTimeout;
let timeoutObserved = null;
globalThis.setTimeout = (fn, delay, ...args) => {
  if (delay === 60_000) timeoutObserved = delay;
  return originalSetTimeout(fn, delay === 60_000 && accelerateTimeout ? 5 : delay, ...args);
};
let accelerateTimeout = false;
const results = [];
function response(text = 'answer', extra = {}) {
  return { id: 'resp_blackbox', object: 'response', model: 'gpt-5-actual', status: 'completed', service_tier: 'default', output: [{ type: 'message', role: 'assistant', content: [{ type: 'output_text', text, annotations: [] }] }], usage: { input_tokens: 12, output_tokens: 8, total_tokens: 20, input_tokens_details: { cached_tokens: 3, cache_write_tokens: 2 }, output_tokens_details: { reasoning_tokens: 4 } }, ...extra };
}
function reply(value, status = 200) { return new Response(JSON.stringify(value), { status, headers: { 'content-type': 'application/json' } }); }
const request = { client_id: 'fd021-test', provider: 'openai', model: 'gpt-5', output_format: 'markdown', prompt: 'test prompt' };
let complete;
async function test(name, fn) {
  calls = []; timeoutObserved = null; accelerateTimeout = false;
  process.env.OPENAI_API_KEY = 'dummy-fd021-key';
  process.env.OPENAI_BASE_URL = 'http://127.0.0.1:43127/v1';
  handler = () => reply(response());
  try { await fn(); results.push({ name, status: 'passed' }); }
  catch (e) { results.push({ name, status: 'failed', message: String(e.message).slice(0, 1200) }); }
}
async function rejection(input = request) {
  try { await complete(input); assert.fail('Expected rejection'); }
  catch (error) { if (error.code === 'ERR_ASSERTION') throw error; assert.equal(typeof error.code, 'string'); assert.equal(typeof error.statusCode, 'number'); return error; }
}
function expectError(error, code, statusCode) { assert.equal(error.code, code); assert.equal(error.statusCode, statusCode); }
try {
  ({ complete } = await import(await load(resolve(service, 'src/providers.ts'))));
  assert.equal(typeof complete, 'function');
  await test('文本请求字段、SDK端点、文本与usage映射', async () => {
    const result = await complete({ ...request, system_prompt: 'system instruction', effort: 'medium', include_reasoning_summary: true });
    assert.equal(calls.length, 1); assert.equal(calls[0].url, 'http://127.0.0.1:43127/v1/responses');
    assert.equal(calls[0].method, 'POST');
    assert.equal(calls[0].body.model, 'gpt-5'); assert.equal(calls[0].body.input, 'test prompt');
    assert.equal(calls[0].body.instructions, 'system instruction');
    assert.equal(calls[0].body.reasoning.effort, 'medium'); assert.equal(calls[0].body.reasoning.summary, 'auto');
    assert.equal(result.text, 'answer'); assert.equal(result.usage.input_tokens, 12); assert.equal(result.usage.output_tokens, 8); assert.equal(result.usage.cost, null);
    assert.equal(result.usage.openai_usage.response_id, 'resp_blackbox'); assert.equal(result.usage.openai_usage.response_model, 'gpt-5-actual');
    assert.equal(result.usage.openai_usage.response_status, 'completed'); assert.equal(result.usage.openai_usage.service_tier, 'default');
    assert.equal(result.usage.openai_usage.total_tokens, 20); assert.equal(result.usage.openai_usage.cached_input_tokens, 3);
    assert.equal(result.usage.openai_usage.cache_write_input_tokens, 2); assert.equal(result.usage.openai_usage.reasoning_output_tokens, 4);
  });
  await test('JSON请求与有效JSON结果', async () => {
    handler = () => reply(response('{"ok":true}'));
    const result = await complete({ ...request, output_format: 'json' });
    assert.equal(calls[0].body.text.format.type, 'json_object'); assert.match(calls[0].body.input, /Return JSON only/); assert.deepEqual(JSON.parse(result.text), { ok: true });
  });
  await test('无效JSON结果拒绝', async () => { handler = () => reply(response('invalid json')); expectError(await rejection({ ...request, output_format: 'json' }), 'invalid_provider_output', 502); });
  await test('usage nested details缺失保持null', async () => {
    handler = () => reply(response('answer', { usage: { input_tokens: 12, output_tokens: 8, total_tokens: 20 } }));
    const result = await complete(request);
    assert.equal(result.usage.input_tokens, 12); assert.equal(result.usage.output_tokens, 8); assert.equal(result.usage.openai_usage.total_tokens, 20);
    assert.equal(result.usage.openai_usage.cached_input_tokens, null); assert.equal(result.usage.openai_usage.cache_write_input_tokens, null); assert.equal(result.usage.openai_usage.reasoning_output_tokens, null);
  });
  await test('usage nested details显式null保持已知tokens', async () => {
    handler = () => reply(response('answer', { usage: { input_tokens: 12, output_tokens: 8, total_tokens: 20, input_tokens_details: null, output_tokens_details: null } }));
    const result = await complete(request);
    assert.equal(result.usage.input_tokens, 12); assert.equal(result.usage.output_tokens, 8); assert.equal(result.usage.openai_usage.total_tokens, 20);
    assert.equal(result.usage.openai_usage.cached_input_tokens, null); assert.equal(result.usage.openai_usage.cache_write_input_tokens, null); assert.equal(result.usage.openai_usage.reasoning_output_tokens, null);
  });
  await test('usage完全缺失保持null', async () => {
    handler = () => reply(response('answer', { usage: null })); const result = await complete(request);
    assert.equal(result.usage.input_tokens, null); assert.equal(result.usage.output_tokens, null); assert.equal(result.usage.openai_usage.total_tokens, null);
  });
  await test('reasoning summary返回与缺失', async () => {
    handler = () => reply(response('answer', { output: [{ type: 'reasoning', summary: [{ type: 'summary_text', text: 'published summary' }] }, ...response().output] }));
    const result = await complete({ ...request, include_reasoning_summary: true }); assert.equal(result.reasoning_summary, 'published summary');
    handler = () => reply(response()); assert.equal((await complete({ ...request, include_reasoning_summary: true })).reasoning_summary, null);
  });
  for (const status of [401, 429, 500]) await test(`HTTP ${status}错误脱敏与不重试`, async () => {
    handler = () => reply({ error: { message: 'dummy-fd021-key raw-upstream-body', type: 'server_error' } }, status);
    const error = await rejection(); assert.equal(calls.length, 1); assert.doesNotMatch(error.message, /dummy-fd021-key|raw-upstream-body/);
    expectError(error, 'provider_error', 502); assert.equal(error.message, `OpenAI returned HTTP ${status}`);
  });
  await test('连接错误脱敏与不重试', async () => {
    handler = () => { throw new Error('dummy-fd021-key raw-upstream-body'); };
    const error = await rejection(); assert.equal(calls.length, 1); assert.doesNotMatch(error.message, /dummy-fd021-key|raw-upstream-body/);
    expectError(error, 'provider_error', 502);
  });
  await test('60秒SDK超时配置、取消与不重试（缩短定时器）', async () => {
    accelerateTimeout = true;
    handler = (_, options) => new Promise((_, reject) => { options.signal.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')), { once: true }); });
    const error = await rejection(); expectError(error, 'provider_timeout', 502); assert.equal(error.message, 'OpenAI request timed out'); assert.equal(timeoutObserved, 60_000); assert.equal(calls.length, 1); assert.equal(calls[0].signal.aborted, true);
  });
  for (const status of [200, 500]) await test(`HTTP ${status} headers后body仍受完整响应deadline约束（缩短定时器）`, async () => {
    accelerateTimeout = true;
    handler = (_, options) => new Response(new ReadableStream({ start(controller) {
      const timer = originalSetTimeout(() => { controller.enqueue(new TextEncoder().encode(JSON.stringify(status === 200 ? response() : { error: { message: 'late-body-canary', type: 'server_error' } }))); controller.close(); }, 30);
      options.signal.addEventListener('abort', () => { clearTimeout(timer); controller.error(new DOMException('Aborted', 'AbortError')); }, { once: true });
    } }), { status, headers: { 'content-type': 'application/json' } });
    expectError(await rejection(), 'provider_timeout', 502); assert.equal(timeoutObserved, 60_000); assert.equal(calls.length, 1); assert.equal(calls[0].signal.aborted, true);
  });
  await test('OPENAI_LOG=debug不泄露请求或上游正文canary', async () => {
    process.env.OPENAI_LOG = 'debug';
    const logs = [];
    const saved = new Map(['log', 'debug', 'info', 'warn', 'error'].map(name => [name, console[name]]));
    for (const name of saved.keys()) console[name] = (...args) => logs.push(args.map(x => typeof x === 'string' ? x : JSON.stringify(x)).join(' '));
    try {
      handler = () => reply(response('fd021-response-canary'));
      await complete({ ...request, prompt: 'fd021-prompt-canary', system_prompt: 'fd021-instructions-canary' });
      handler = () => reply({ error: { message: 'fd021-error-body-canary', type: 'server_error' } }, 500);
      const error = await rejection({ ...request, prompt: 'fd021-prompt-canary', system_prompt: 'fd021-instructions-canary' });
      expectError(error, 'provider_error', 502); assert.equal(calls.length, 2);
      assert.doesNotMatch(error.message, /fd021-(?:prompt|instructions|response|error-body)-canary/);
      assert.doesNotMatch(logs.join('\n'), /fd021-(?:prompt|instructions|response|error-body)-canary/);
    } finally { for (const [name, fn] of saved) console[name] = fn; delete process.env.OPENAI_LOG; }
  });
  for (const url of ['http://localhost:43127/v1', 'http://[::1]:43127/v1', 'https://example.invalid/v1']) await test(`允许URL ${url}`, async () => {
    process.env.OPENAI_BASE_URL = url; await complete(request); assert.equal(calls.length, 1);
  });
  for (const url of ['http://example.invalid/v1', 'ftp://127.0.0.1/v1', 'not-a-url']) await test(`拒绝URL ${url}`, async () => {
    process.env.OPENAI_BASE_URL = url; expectError(await rejection(), 'provider_not_configured', 503); assert.equal(calls.length, 0);
  });
  await test('缺少API key在发送前拒绝', async () => { delete process.env.OPENAI_API_KEY; expectError(await rejection(), 'provider_not_configured', 503); assert.equal(calls.length, 0); });
  await test('结果256Ki字符边界允许', async () => { handler = () => reply(response('x'.repeat(256 * 1024))); assert.equal((await complete(request)).text.length, 256 * 1024); });
  await test('结果超过256Ki字符拒绝', async () => { handler = () => reply(response('x'.repeat(256 * 1024 + 1))); expectError(await rejection(), 'result_too_large', 502); });
} catch (e) { results.push({ name: '测试装载', status: 'failed', message: String(e.message).slice(0, 1200) }); }
finally {
  globalThis.fetch = originalFetch; globalThis.setTimeout = originalSetTimeout;
  for (const name of Object.keys(process.env)) if (!(name in oldEnv)) delete process.env[name];
  Object.assign(process.env, oldEnv);
}
const fdSource = await readFile(resolve(root, 'docs/features/FD-021_AIW_AGENT_OPENAI_SDK.md'), 'utf8');
const sdkVersion = JSON.parse(await readFile(resolve(service, 'node_modules/openai/package.json'), 'utf8')).version;
const testSource = await readFile(import.meta.filename, 'utf8');
const output = { fd_id: 'FD-021', fd_revision: Number(fdSource.match(/\*\*Revision:\*\*\s*(\d+)/)?.[1]), fd_digest: createHash('sha256').update(fdSource).digest('hex'), test_source_digest: createHash('sha256').update(testSource).digest('hex'), source_digests: revisions, sdk_version: sdkVersion, transport: 'in-process fetch mock; real installed SDK; no network/socket', timer_policy: 'Only configured 60000ms timeout accelerated to 5ms for cancellation case', results, passed: results.filter(r => r.status === 'passed').length, failed: results.filter(r => r.status === 'failed').length };
await writeFile(resolve(root, 'docs/features/reports/FD-021-test-results-r2.json'), JSON.stringify(output, null, 2) + '\n');
console.log(JSON.stringify(output, null, 2));
process.exitCode = output.failed ? 1 : 0;
