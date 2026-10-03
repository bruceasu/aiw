// FD-021 R6: one real Provider request; no fallback/retry/service mutation.
import { readFile, writeFile, realpath } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { dirname, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { createHash } from 'node:crypto';
import { isDeepStrictEqual } from 'node:util';
const root = resolve(import.meta.dirname, '..');
const service = resolve(root, 'program/aiw-agent');
const install = 'C:/green/aiw/plugins/aiw-gw';
const require = createRequire(resolve(service, 'package.json'));
const ts = require('typescript');
const digests = {}, modules = new Map();
const session = 'fd021-external-acceptance-20261004-c813f0';
const command = 'node tests/fd021_gateway_acceptance.mjs';
const hash = text => createHash('sha256').update(text).digest('hex');
const fdSource = await readFile(resolve(root, 'docs/features/FD-021_AIW_AGENT_OPENAI_SDK.md'), 'utf8');
const testSource = await readFile(import.meta.filename, 'utf8');
const sdkVersion = JSON.parse(await readFile(resolve(service, 'node_modules/openai/package.json'), 'utf8')).version;
const gatewayDigest = hash(await readFile(resolve(install, 'agent-gateway.exe')));
async function load(path) {
  if (modules.has(path)) return modules.get(path);
  const source = await readFile(path, 'utf8');
  digests[path.slice(root.length + 1).replaceAll('\\', '/')] = hash(source);
  let code = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 } }).outputText;
  for (const name of new Set([...code.matchAll(/(?:from\s*|import\s*)["']([^"']+)["']/g)].map(m => m[1]))) {
    const replacement = name.startsWith('.') ? await load(resolve(dirname(path), name.replace(/\.js$/, '.ts'))) : name.startsWith('node:') ? null : pathToFileURL(require.resolve(name)).href;
    if (replacement) code = code.replace(/(from\s*|import\s*)(["'])([^"']+)\2/g, (match, prefix, quote, specifier) => specifier === name ? prefix + JSON.stringify(replacement) : match);
  }
  const url = 'data:text/javascript;base64,' + Buffer.from(code).toString('base64'); modules.set(path, url); return url;
}
const count = v => Number.isSafeInteger(v) && v >= 0 ? v : null;
const identifier = v => typeof v === 'string' && /^[A-Za-z0-9._:/-]{1,128}$/.test(v) ? v : null;
const requestId = v => typeof v === 'string' && /^[A-Za-z0-9_-]{1,128}$/.test(v) ? v : null;
function mapWire(data) {
  return { input_tokens: count(data?.usage?.input_tokens), output_tokens: count(data?.usage?.output_tokens), cost: null,
    openai_usage: { response_id: identifier(data?.id), response_model: identifier(data?.model), service_tier: identifier(data?.service_tier), response_status: identifier(data?.status), total_tokens: count(data?.usage?.total_tokens), cached_input_tokens: count(data?.usage?.input_tokens_details?.cached_tokens), cache_write_input_tokens: count(data?.usage?.input_tokens_details?.cache_write_tokens), reasoning_output_tokens: count(data?.usage?.output_tokens_details?.reasoning_tokens) } };
}
const results = [];
function check(id, behavior, ok) {
  results.push({ id, behavior, status: ok ? 'passed' : 'failed' });
  if (!ok) throw new Error('safe-assertion-failed');
}
const liveInventory = [
  ['R6-LIVE-01', '当前Provider通过已安装官方SDK完成真实Gateway请求'],
  ['R6-LIVE-02', 'Gateway模式真实请求仅input/model且POST/redirect:error'],
  ['R6-LIVE-03', '真实文本结果精确匹配标记'],
  ['R6-LIVE-04', '返回completed并映射响应标识/模型/状态元数据'],
  ['R6-LIVE-05', 'Provider用量等于安全响应映射且合法未知字段为null'],
  ['R6-LIVE-06', '本次成功路径仅一次HTTP请求且无fallback'],
  ['R6-LIVE-07', '关联HTTP审计succeeded且execution_started为true'],
];
const additionalInventory = [
  '缺省openai模式保留8192和reasoning', '显式openai模式保留8192和reasoning',
  'Gateway离线标准output和usage映射', 'Gateway instructions与JSON支持字段',
  'Gateway effort发送前拒绝', 'Gateway summary=true发送前拒绝',
  '未知profile发送前拒绝', '空profile发送前拒绝',
  '非法JSON结果明确拒绝', '官方模式instructions保持', 'published reasoning summary提取与缺失',
  'reasoning summary16Ki边界与超界拒绝', 'headers前60秒超时取消且错误分类保持',
  '成功正文完整读取受deadline保护', '错误正文完整读取受deadline保护',
  'HTTP401/429/500错误脱敏及不重试', '连接错误脱敏及不重试',
  'SDK debug日志不泄露请求和响应正文', '文本结果256Ki边界及超界拒绝',
  'usage完全缺失及显式null nested details容错', '非法usage计数和identifier净化',
  'loopbackIPv4/IPv6/localhost HTTP允许', '远程HTTP拒绝和远程HTTPS允许',
  'URL userinfo/query/fragment/非法协议拒绝', '缺失API key发送前拒绝',
  '默认官方baseURL保持', '重定向拒绝且错误不重试', '空输出拒绝并保usage',
  'legacy顶层output_text兼容', '成功/HTTP错误/JSON解析错误后timer清理',
  'ws及其类型和TypeScript仍保留', '扩展文档记录request/provider/storage/audit与无知识库',
  '服务HTTP/WebSocket/Copilot/Codex既有契约未改变',
];
const savedEnv = { ...process.env }, savedFetch = globalThis.fetch;
const output = { fd_id: 'FD-021', source_event: null, tester_session: session,
  fd_revision: Number(fdSource.match(/\*\*Revision:\*\*\s*(\d+)/)?.[1]), fd_digest: hash(fdSource),
  test_source_digest: hash(testSource), source_digests: digests, sdk_version: sdkVersion,
  gateway_binary_digest: gatewayDigest, profile: 'aiw_gateway', endpoint: 'http://127.0.0.1:43127/v1/responses',
  model: 'gpt-6-luna', requests: [], results, historical_evidence: [], scenario_inventory: [
    ...liveInventory.map(([id, behavior]) => ({ id, behavior, evidence_route: 'current-single-live-request' })),
    ...additionalInventory.map((behavior, i) => ({ id: 'FD021-A-' + String(i + 1).padStart(2, '0'), behavior, evidence_route: i < 8 ? 'R5 history only if all available source/SDK/test digests match' : 'unrun-current-version; static review or stale history required' }))
  ], model_execution_limit: 1, fallback_policy: 'none', raw_path: 'docs/features/reports/FD-021-test-results-r6.json' };
let key, stateDir, overallTimer, wireUsage;
const overall = new AbortController();
try {
  delete process.env.AIW_AGENT_PROXY_HOST; delete process.env.AIW_AGENT_PROXY_PORT; delete process.env.OPENAI_LOG;
  const { complete } = await import(await load(resolve(service, 'src/providers.ts')));
  const authorization = JSON.parse(await readFile(resolve(root, 'docs/features/reports/FD-021-test-authorization-r6.json'), 'utf8'));
  const ad = authorization.data;
  if (ad?.decision !== 'approved' || ad?.command !== command || ad?.tester_session !== session
      || ad?.fd_digest !== output.fd_digest || ad?.test_source_digest !== output.test_source_digest
      || ad?.fd_revision !== output.fd_revision
      || ad?.gateway_binary_digest !== gatewayDigest
      || !isDeepStrictEqual(ad?.source_digests, digests)) throw new Error('safe-authorization-mismatch');
  for (const [path, script] of [
    ['docs/features/reports/FD-021-test-results-r2.json', 'tests/fd021_sdk_blackbox.mjs'],
    ['docs/features/reports/FD-021-test-results-r3-retry.json', 'tests/fd021_sdk_edges.mjs'],
    ['docs/features/reports/FD-021-test-results-r5.json', 'tests/fd021_gateway_profile.mjs'],
  ]) {
    const evidence = JSON.parse(await readFile(resolve(root, path), 'utf8'));
    const matches = evidence.sdk_version === sdkVersion
      && Object.entries(digests).every(([p, v]) => evidence.source_digests?.[p] === v)
      && evidence.test_source_digest === hash(await readFile(resolve(root, script), 'utf8'));
    output.historical_evidence.push({ path, available_digest_scope_matches: matches, historical_passed: evidence.passed, historical_failed: evidence.failed, counted_as_current_execution: false });
  }
  const config = JSON.parse(await readFile(resolve(install, 'gateway.json'), 'utf8'));
  const principal = config.principals?.find(p => p.enabled === true && p.allowed_models?.includes('gpt-6-luna') && typeof p.keys?.[0] === 'string' && p.keys[0]);
  if (config.listen !== '127.0.0.1:43127' || !principal || typeof config.state_dir !== 'string') throw new Error('safe-local-config-mismatch');
  key = principal.keys[0]; stateDir = config.state_dir;
  process.env.OPENAI_API_KEY = key; process.env.OPENAI_BASE_URL = 'http://127.0.0.1:43127/v1'; process.env.OPENAI_API_PROFILE = 'aiw_gateway';
  overallTimer = setTimeout(() => overall.abort(), 65_000);
  globalThis.fetch = async (url, options) => {
    const target = new URL(String(url));
    if (target.origin !== 'http://127.0.0.1:43127' || target.pathname !== '/v1/responses' || target.search || target.hash || target.username || target.password || options?.method !== 'POST' || options.redirect !== 'error' || output.requests.length !== 0) throw new Error('safe-request-budget-or-target');
    const body = JSON.parse(options.body);
    const fields = Object.keys(body).sort();
    if (JSON.stringify(fields) !== JSON.stringify(['input', 'model']) || body.model !== 'gpt-6-luna') throw new Error('safe-request-fields-mismatch');
    const record = { method: 'POST', redirect: 'error', request_fields: fields, http_status: null, request_id: null };
    output.requests.push(record);
    const signal = options.signal ? AbortSignal.any([options.signal, overall.signal]) : overall.signal;
    const response = await savedFetch(url, { ...options, signal, redirect: 'error' });
    record.http_status = response.status; record.request_id = requestId(response.headers.get('x-request-id'));
    if (response.ok) wireUsage = mapWire(await response.clone().json());
    return response;
  };
  const marker = 'FD021_R6_OK';
  const completion = await complete({ client_id: 'fd021-r6-acceptance', provider: 'openai', model: 'gpt-6-luna', output_format: 'markdown', prompt: 'Reply exactly ' + marker + '.' });
  check(...liveInventory[0], output.requests.length === 1 && output.requests[0].http_status === 200);
  check(...liveInventory[1], JSON.stringify(output.requests[0].request_fields) === JSON.stringify(['input', 'model']));
  output.text_present = completion.text.trim().length > 0; output.text_match = completion.text.trim() === marker;
  check(...liveInventory[2], output.text_present && output.text_match);
  output.usage = wireUsage;
  check(...liveInventory[3], wireUsage?.openai_usage.response_status === 'completed'
    && completion.usage.openai_usage?.response_id === wireUsage.openai_usage.response_id
    && completion.usage.openai_usage?.response_model === wireUsage.openai_usage.response_model
    && completion.usage.openai_usage?.response_status === 'completed');
  check(...liveInventory[4], isDeepStrictEqual(completion.usage, wireUsage));
  check(...liveInventory[5], output.requests.length === 1);
  const id = output.requests[0].request_id;
  if (!id) throw new Error('safe-request-id-unavailable');
  const auditBase = await realpath(resolve(stateDir, 'requests'));
  const candidate = resolve(auditBase, id + '.json');
  if (dirname(candidate) !== auditBase) throw new Error('safe-audit-path');
  const deadline = Date.now() + 1500;
  let audit;
  while (Date.now() < deadline) {
    try {
      const canonical = await realpath(candidate);
      if (dirname(canonical) !== auditBase) throw new Error('safe-audit-path');
      const row = JSON.parse(await readFile(canonical, 'utf8'));
      audit = { request_id: id, http_status: count(row.http_status), state: ['succeeded', 'failed', 'timed_out', 'cancelled', 'rejected', 'in_progress'].includes(row.state) ? row.state : null, execution_started: row.execution_started === true };
      if (audit.state !== 'in_progress') break;
    } catch (error) { if (error?.code !== 'ENOENT') throw new Error('safe-audit-read'); }
    await new Promise(resolve => setTimeout(resolve, 25));
  }
  output.audit = audit ?? { request_id: id, state: 'not-visible-before-read-deadline' };
  check(...liveInventory[6], audit?.state === 'succeeded' && audit.http_status === 200 && audit.execution_started === true);
} catch (error) {
  output.failure_class = overall.signal.aborted ? 'overall-timeout' : ['provider_error', 'provider_timeout', 'provider_not_configured', 'unsupported_option'].includes(error?.code) ? error.code : 'safe-assertion-or-harness-failure';
} finally {
  if (overallTimer) clearTimeout(overallTimer);
  globalThis.fetch = savedFetch;
  for (const name of Object.keys(process.env)) if (!(name in savedEnv)) delete process.env[name];
  Object.assign(process.env, savedEnv); key = undefined; stateDir = undefined;
}
output.live_request_count = output.requests.length; output.passed = results.filter(r => r.status === 'passed').length;
output.failed = results.filter(r => r.status === 'failed').length;
output.unrun_live_checks = liveInventory.length - results.length;
output.outcome = output.passed === liveInventory.length && output.failed === 0 ? 'passed' : 'failed';
await writeFile(resolve(root, output.raw_path), JSON.stringify(output, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify(output, null, 2)); process.exitCode = output.outcome === 'passed' ? 0 : 1;
