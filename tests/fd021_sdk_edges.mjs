// FD-021 r3 edge contracts: independent cases; installed SDK; zero network.
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
const digests = {};
const urls = new Map();
async function load(path) {
  if (urls.has(path)) return urls.get(path);
  const source = await readFile(path, 'utf8');
  digests[path.slice(root.length + 1).replaceAll('\\', '/')] = hash(source);
  let code = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 } }).outputText;
  for (const name of new Set([...code.matchAll(/(?:from\s*|import\s*)["']([^"']+)["']/g)].map(m => m[1]))) {
    const replacement = name.startsWith('.') ? await load(resolve(dirname(path), name.replace(/\.js$/, '.ts'))) : name.startsWith('node:') ? null : pathToFileURL(require.resolve(name)).href;
    if (replacement) code = code.replace(/(from\s*|import\s*)(["'])([^"']+)\2/g, (match, prefix, quote, specifier) => specifier === name ? prefix + JSON.stringify(replacement) : match);
  }
  const url = 'data:text/javascript;base64,' + Buffer.from(code).toString('base64');
  urls.set(path, url); return url;
}
function hash(text) { return createHash('sha256').update(text).digest('hex'); }
const fdSource = await readFile(resolve(root, 'docs/features/FD-021_AIW_AGENT_OPENAI_SDK.md'), 'utf8');
const testSource = await readFile(import.meta.filename, 'utf8');
const savedEnv = { ...process.env };
const savedFetch = globalThis.fetch;
const savedSetTimeout = globalThis.setTimeout;
const savedClearTimeout = globalThis.clearTimeout;
let fast = false;
const pending = new Set();
globalThis.setTimeout = (fn, delay, ...args) => {
  let timer;
  timer = savedSetTimeout(() => { pending.delete(timer); fn(...args); }, fast && delay === 60_000 ? 5 : delay);
  if (delay === 60_000) pending.add(timer);
  return timer;
};
globalThis.clearTimeout = timer => { pending.delete(timer); return savedClearTimeout(timer); };
let calls = [], handler;
globalThis.fetch = async (url, options) => { calls.push({ url: String(url), options }); return handler(url, options); };
delete process.env.AIW_AGENT_PROXY_HOST; delete process.env.AIW_AGENT_PROXY_PORT; delete process.env.OPENAI_LOG;
const request = { client_id: 'fd021-edges', provider: 'openai', model: 'gpt-5', output_format: 'markdown', prompt: 'edge canary prompt' };
const baseUsage = { input_tokens: 12, output_tokens: 8, total_tokens: 20, input_tokens_details: { cached_tokens: 3, cache_write_tokens: 2 }, output_tokens_details: { reasoning_tokens: 4 } };
function body(extra = {}) { return { id: 'resp_edges', object: 'response', model: 'gpt-5', status: 'completed', service_tier: 'default', usage: baseUsage, output: [{ type: 'message', role: 'assistant', content: [{ type: 'output_text', text: 'answer', annotations: [] }] }], ...extra }; }
function reply(value, status = 200) { return new Response(JSON.stringify(value), { status, headers: { 'content-type': 'application/json' } }); }
let complete;
const results = [];
async function test(name, fn, contract = 'existing public contract') {
  calls = []; fast = false;
  process.env.OPENAI_API_KEY = 'dummy-fd021-key'; process.env.OPENAI_BASE_URL = 'http://127.0.0.1:43127/v1';
  handler = () => reply(body());
  try { await fn(); results.push({ name, contract, status: 'passed' }); }
  catch (e) { results.push({ name, contract, status: 'failed', message: String(e.message).slice(0, 1200) }); }
  finally { for (const timer of pending) savedClearTimeout(timer); pending.clear(); }
}
async function rejection(input = request, code = 'provider_error', status = 502) {
  try { await complete(input); assert.fail('Expected rejection'); }
  catch (e) { if (e.code === 'ERR_ASSERTION') throw e; assert.equal(e.code, code); assert.equal(e.statusCode, status); return e; }
}
function assertUsage(usage) {
  assert.equal(usage.input_tokens, 12); assert.equal(usage.output_tokens, 8); assert.equal(usage.cost, null);
  assert.equal(usage.openai_usage.total_tokens, 20); assert.equal(usage.openai_usage.cached_input_tokens, 3);
}
try {
  ({ complete } = await import(await load(resolve(service, 'src/providers.ts'))));
  for (const output of [[], [{ type: 'reasoning', summary: [] }]]) await test('空输出或无文本输出拒绝且保usage ' + JSON.stringify(output), async () => {
    handler = () => reply(body({ output })); const e = await rejection(); assertUsage(e.usage); assert.equal(calls.length, 1);
  });
  for (const size of [16 * 1024, 16 * 1024 + 1]) await test('reasoning summary字符边界 ' + size, async () => {
    const summary = 's'.repeat(size);
    handler = () => reply(body({ output: [{ type: 'reasoning', summary: [{ type: 'summary_text', text: summary }] }, ...body().output] }));
    if (size === 16 * 1024) { const result = await complete({ ...request, include_reasoning_summary: true }); assert.equal(result.reasoning_summary, summary); }
    else { const e = await rejection({ ...request, include_reasoning_summary: true }, 'result_too_large'); assertUsage(e.usage); }
  });
  await test('未请求summary时忽略超长摘要且不返回字段', async () => {
    handler = () => reply(body({ output: [{ type: 'reasoning', summary: [{ type: 'summary_text', text: 's'.repeat(16 * 1024 + 1) }] }, ...body().output] }));
    const result = await complete(request); assert.equal(result.text, 'answer'); assert.equal(Object.hasOwn(result, 'reasoning_summary'), false);
  });
  for (const count of [-1, 1.5, '12', Number.MAX_SAFE_INTEGER + 1]) await test('非法usage计数归null ' + JSON.stringify(count), async () => {
    handler = () => reply(body({ usage: { input_tokens: count, output_tokens: count, total_tokens: count, input_tokens_details: { cached_tokens: count, cache_write_tokens: count }, output_tokens_details: { reasoning_tokens: count } } }));
    const usage = (await complete(request)).usage;
    for (const key of ['input_tokens', 'output_tokens', 'cost']) assert.equal(usage[key], null);
    for (const key of ['total_tokens', 'cached_input_tokens', 'cache_write_input_tokens', 'reasoning_output_tokens']) assert.equal(usage.openai_usage[key], null);
  });
  for (const id of ['invalid identifier canary', 'x'.repeat(129), 42]) await test('非法usage identifier归null ' + JSON.stringify(id), async () => {
    handler = () => reply(body({ id, model: id, status: id, service_tier: id }));
    const usage = (await complete(request)).usage;
    for (const key of ['response_id', 'response_model', 'response_status', 'service_tier']) assert.equal(usage.openai_usage[key], null);
  });
  await test('合法identifier128字符边界', async () => {
    const id = 'x'.repeat(128); handler = () => reply(body({ id, model: 'model/a-b.c_d:1' }));
    const usage = (await complete(request)).usage; assert.equal(usage.openai_usage.response_id, id); assert.equal(usage.openai_usage.response_model, 'model/a-b.c_d:1');
  });
  for (const status of [200, 500]) await test('非JSON HTTP正文错误脱敏 ' + status, async () => {
    handler = () => new Response('raw-body-canary dummy-fd021-key', { status, headers: { 'content-type': 'application/json' } });
    const e = await rejection(); assert.doesNotMatch(e.message, /raw-body-canary|dummy-fd021-key/); assert.equal(calls.length, 1);
  });
  for (const url of ['https://user:password@host.invalid/v1', 'https://host.invalid/v1?canary=query', 'https://host.invalid/v1#canary']) await test('拒绝URL组件 ' + url, async () => {
    process.env.OPENAI_BASE_URL = url; await rejection(request, 'provider_not_configured', 503); assert.equal(calls.length, 0);
  });
  await test('fetch redirect:error且拒绝无重试', async () => {
    handler = (_, options) => { assert.equal(options.redirect, 'error'); throw new TypeError('redirect-canary'); };
    const e = await rejection(); assert.equal(calls.length, 1); assert.doesNotMatch(e.message, /redirect-canary/);
    // The handler assertion is also checked here because SDK wraps fetch failures.
    assert.equal(calls[0].options.redirect, 'error');
  });
  await test('缺省baseURL使用官方Responses端点仅mock', async () => {
    delete process.env.OPENAI_BASE_URL; await complete(request); assert.equal(calls[0].url, 'https://api.openai.com/v1/responses');
  });
  await test('成功后60秒timer清理且signal不再abort（缩短定时器）', async () => {
    fast = true; const result = await complete(request); assert.equal(result.text, 'answer'); assert.equal(pending.size, 0);
    await new Promise(resolve => savedSetTimeout(resolve, 15)); assert.equal(calls[0].options.signal.aborted, false);
  });
  await test('HTTP错误后60秒timer清理且signal不再abort（缩短定时器）', async () => {
    fast = true; handler = () => reply({ error: { message: 'error-canary' } }, 500); await rejection(); assert.equal(pending.size, 0);
    await new Promise(resolve => savedSetTimeout(resolve, 15)); assert.equal(calls[0].options.signal.aborted, false);
  });
  await test('JSON解析错误后60秒timer清理且signal不再abort（缩短定时器）', async () => {
    fast = true; handler = () => new Response('invalid-json-canary', { headers: { 'content-type': 'application/json' } }); await rejection(); assert.equal(pending.size, 0);
    await new Promise(resolve => savedSetTimeout(resolve, 15)); assert.equal(calls[0].options.signal.aborted, false);
  });
  await test('legacy顶层output_text兼容且output为空数组', async () => {
    handler = () => reply(body({ output_text: 'legacy answer', output: [] })); const result = await complete(request); assert.equal(result.text, 'legacy answer'); assertUsage(result.usage);
    assert.equal(calls.length, 1); assert.equal(calls[0].options.method, 'POST');
    assert.equal(new Headers(calls[0].options.headers).get('authorization'), 'Bearer dummy-fd021-key');
  }, 'Legacy pre-SDK accepted shape; not a promise of standard OpenAI wire shape or Gateway support');
  for (const usage of [false, null]) await test('usage非对象false/null保持未知 ' + String(usage), async () => {
    handler = () => reply(body({ usage })); const result = await complete(request); assert.equal(result.text, 'answer'); assert.equal(result.usage.input_tokens, null); assert.equal(result.usage.output_tokens, null); assert.equal(result.usage.openai_usage.total_tokens, null);
  });
} catch (e) { results.push({ name: 'edge测试装载', status: 'failed', message: String(e.message).slice(0, 1200) }); }
finally {
  for (const timer of pending) savedClearTimeout(timer);
  globalThis.fetch = savedFetch; globalThis.setTimeout = savedSetTimeout; globalThis.clearTimeout = savedClearTimeout;
  for (const key of Object.keys(process.env)) if (!(key in savedEnv)) delete process.env[key];
  Object.assign(process.env, savedEnv);
}
const rawPath = 'docs/features/reports/FD-021-test-results-r3-initial.json';
const output = { fd_id: 'FD-021', fd_revision: Number(fdSource.match(/\*\*Revision:\*\*\s*(\d+)/)?.[1]), fd_digest: hash(fdSource), test_source_digest: hash(testSource), source_digests: digests, sdk_version: JSON.parse(await readFile(resolve(service, 'node_modules/openai/package.json'), 'utf8')).version, raw_path: rawPath, transport: 'real installed SDK with in-process fetch mock; no sockets/network', timer_policy: '60000ms timers accelerated to 5ms only in cleanup cases; original 15ms timer observes cleanup', results, passed: results.filter(r => r.status === 'passed').length, failed: results.filter(r => r.status === 'failed').length };
try { await writeFile(resolve(root, rawPath), JSON.stringify(output, null, 2) + '\n', { flag: 'wx' }); }
catch (error) {
  if (error.code !== 'EEXIST') throw error;
  output.raw_path = 'docs/features/reports/FD-021-test-results-r3-retry.json';
  await writeFile(resolve(root, output.raw_path), JSON.stringify(output, null, 2) + '\n', { flag: 'wx' });
}
console.log(JSON.stringify(output, null, 2)); process.exitCode = output.failed ? 1 : 0;
