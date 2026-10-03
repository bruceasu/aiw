// FD-023: offline black-box checks of the documented ai client contract.
import { spawn } from 'node:child_process';
import { createServer } from 'node:http';
import { once } from 'node:events';
import { writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';

const root = resolve(import.meta.dirname, '..');
const client = resolve(root, 'plugins/aiw-ai/aiw-ai.mjs');
const rawPath = resolve(root, 'docs/features/reports/FD-023-test-results-r1-retry.json');
const key = 'a'.repeat(43);
const requests = [];
let behavior = 'success';
let disconnected = false;
const server = createServer((request, response) => {
  const chunks = [];
  request.on('data', chunk => chunks.push(chunk));
  request.on('end', () => {
    let body;
    try { body = JSON.parse(Buffer.concat(chunks).toString('utf8')); }
    catch { body = null; }
    requests.push({ method: request.method, path: request.url, authorization: request.headers.authorization === `Bearer ${key}`, body });
    if (behavior === 'wait-for-headers') {
      response.on('close', () => { disconnected = true; });
      return;
    }
    response.writeHead(200, { 'content-type': 'application/json' });
    if (behavior === 'wait-for-body') {
      response.write('{');
      response.on('close', () => { disconnected = true; });
      return;
    }
    response.end(JSON.stringify({ object: 'response', id: 'resp_fd023', model: 'gpt-6-luna', status: 'completed',
      output: [{ type: 'message', role: 'assistant', content: [{ type: 'output_text', text: 'FD023_OK' }] }], usage: null }));
  });
});
server.listen(0, '127.0.0.1');
await once(server, 'listening');
const port = server.address().port;
const url = `http://127.0.0.1:${port}`;
const scenarios = [];

function run(args, input = '') {
  return new Promise((resolveRun, reject) => {
    const env = { ...process.env, AIW_AI_API_KEY: key, AIW_AI_MODEL: 'gpt-6-luna', AIW_AGENT_PROXY_PORT: String(port) };
    delete env.AIW_AI_PROVIDER;
    const child = spawn(process.execPath, [client, ...args], {
      cwd: root, windowsHide: true,
      env,
      stdio: ['pipe', 'pipe', 'pipe'],
    });
    let stdout = '', stderr = '';
    const watchdog = setTimeout(() => child.kill(), 4500);
    child.stdout.on('data', chunk => { stdout += chunk.toString(); });
    child.stderr.on('data', chunk => { stderr += chunk.toString(); });
    child.on('error', reject);
    child.on('close', code => { clearTimeout(watchdog); resolveRun({ code, stdout, stderr }); });
    child.stdin.end(input);
  });
}

async function check(id, description, fn) {
  try { const detail = await fn(); scenarios.push({ id, description, status: 'passed', detail }); }
  catch (error) { scenarios.push({ id, description, status: 'failed', detail: String(error?.message ?? error) }); }
}
function assert(value, message) { if (!value) throw new Error(message); }
function promptRequest() { return ['--url', url, 'Explain this text']; }

try {
  await check('T01', '帮助显示独立的客户端 660 秒期限且不发请求', async () => {
    const before = requests.length;
    const result = await run(['--help']);
    assert(result.code === 0 && /660/.test(result.stdout + result.stderr) && /timeout-seconds/.test(result.stdout + result.stderr), 'help contract');
    assert(requests.length === before, 'help sent request');
    return 'exit 0; help contains 660 and option; no request';
  });
  const invalid = [
    ['T02', '缺值', ['--timeout-seconds']],
    ['T03', '重复', ['--timeout-seconds', '1', '--timeout-seconds', '2']],
    ['T04', '零值', ['--timeout-seconds', '0']],
    ['T05', '超过上限', ['--timeout-seconds', '3661']],
    ['T06', '非十进制整数', ['--timeout-seconds', '1.5']],
  ];
  for (const [id, description, args] of invalid) {
    await check(id, `${description}在发请求前退出 2`, async () => {
      const before = requests.length;
      const result = await run([...args, ...promptRequest()]);
      assert(result.code === 2, `exit ${result.code}`);
      assert(requests.length === before, 'unexpected request');
      return 'exit 2; no request';
    });
  }
  await check('T07', '300 秒参数不进入 Prompt 或 Responses JSON，保留认证与输出', async () => {
    behavior = 'success';
    const before = requests.length;
    const result = await run(['--timeout-seconds', '300', ...promptRequest()]);
    const received = requests.slice(before);
    assert(result.code === 0 && result.stdout.trim() === 'FD023_OK', `exit ${result.code}; output mismatch`);
    assert(received.length === 1 && received[0].method === 'POST' && received[0].path === '/v1/responses' && received[0].authorization, 'request contract');
    assert(received[0].body?.input === 'Explain this text' && !('timeout_seconds' in received[0].body) && !JSON.stringify(received[0].body).includes('300'), 'timeout leaked into body');
    return 'one authenticated POST; expected prompt and output; no timeout field';
  });
  await check('T08', '双连字符后 timeout 字样保留为 Prompt', async () => {
    const before = requests.length;
    const result = await run(['--url', url, '--', '--timeout-seconds', '300']);
    const received = requests.slice(before);
    assert(result.code === 0 && received.length === 1 && received[0].body?.input === '--timeout-seconds 300', 'terminator behavior');
    return 'one request; literal prompt retained';
  });
  for (const [id, mode, description] of [
    ['T09', 'wait-for-headers', '请求头前超时并取消'],
    ['T10', 'wait-for-body', '响应体未完成时超时并取消'],
  ]) {
    await check(id, description, async () => {
      behavior = mode; disconnected = false;
      const before = requests.length;
      const started = Date.now();
      const result = await run(['--timeout-seconds', '1', ...promptRequest()]);
      await new Promise(resolveWait => setTimeout(resolveWait, 100));
      const elapsed = Date.now() - started;
      assert(result.code !== 0 && elapsed >= 700 && elapsed < 3500, `exit ${result.code}; elapsed ${elapsed}ms`);
      assert(requests.length === before + 1 && disconnected, 'request not cancelled');
      return `nonzero exit; elapsed ${elapsed}ms; local socket closed`;
    });
  }
} finally {
  server.closeAllConnections();
  server.close();
}

const evidence = { fd_id: 'FD-023', source_event: 'FD-023-000005-implementation-ready',
  tester_session: 'fd023-tester-20261004-a9e4bd', command: 'node tests/fd023_client_blackbox.mjs',
  environment: 'Node local child process and loopback HTTP mock only', scenarios,
  passed: scenarios.filter(item => item.status === 'passed').length,
  failed: scenarios.filter(item => item.status === 'failed').length };
await writeFile(rawPath, JSON.stringify(evidence, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify(evidence));
if (evidence.failed) process.exitCode = 1;
