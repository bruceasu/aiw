# AIW Agent Proxy

The TypeScript service binds to `127.0.0.1` by default. Start it from this directory
after installing its package dependencies and building:

```powershell
npm install
npm run build
node ./aiw-agent-proxy.js start --port 43127
```

`AIW_AGENT_PROXY_PORT` sets the default port. `AIW_AGENT_PROXY_STATE_DIR` may
set a private local temporary state directory; it stores ACK-pending results
and `audit.jsonl`. Results expire one hour after creation. The service removes
each result file after ACK. Do not put the state directory in a synced or
shared folder. Stop it with Ctrl+C or SIGTERM.

For direct access from another machine, set `AIW_AGENT_PROXY_HOST` to a
specific local IPv4 address or `0.0.0.0` before starting the service. Remote
HTTP clients must use an IPv4 address in the URL; the WebSocket endpoint stays
loopback-only. For example, on the service machine:

```powershell
$env:AIW_AGENT_PROXY_HOST = "0.0.0.0"
node ./aiw-agent-proxy.js start --port 43127
```

Then use `ai --url http://<service-ip>:43127 "Question"` on the client
machine. No authentication is applied to remote HTTP requests. Anyone able to
reach that port can submit Provider requests and consume the service's quota;
restrict access with a host firewall or a trusted private network. The
service itself serves plain HTTP, so use an HTTPS terminating proxy when
transport encryption is needed. Leave the host variable unset for local-only
operation.

## HTTP

`POST /v1/requests` accepts JSON:

```json
{
  "client_id": "editor-1",
  "request_id": "optional-id",
  "provider": "openai",
  "model": "gpt-5",
  "effort": "medium",
  "output_format": "json",
  "prompt": "Summarize the requested change."
}
```

`provider` is `codex`, `copilot`, or `openai`; `output_format` is `json` or
`markdown`. Provider/model combinations and unsupported effort values are
rejected by the provider. OpenAI reads `OPENAI_API_KEY` and optionally
`OPENAI_BASE_URL` from the service environment. Cost is reported as `null`
unless a Provider supplies a documented monetary amount; token totals are
reported when supplied by the Provider.

OpenAI Responses requests use the official OpenAI Node API SDK. The SDK uses
the same `OPENAI_API_KEY` and optional `OPENAI_BASE_URL` configuration.
Requests use the SDK's public typed POST method for `/responses` so compatible
endpoints' existing `output_text` is preserved. Standard `output` text is still
extracted when `output_text` is absent; SDK authentication, transport, JSON
decoding, and error handling remain in use.
`OPENAI_BASE_URL` defaults to `https://api.openai.com/v1`; HTTP is accepted only
for loopback hosts such as `http://127.0.0.1:43127/v1` when using the local
Agent Gateway simulator. Other hosts must use HTTPS. SDK retries are disabled
and the request timeout is 60 seconds.

`OPENAI_API_PROFILE` defaults to `openai`, preserving `max_output_tokens: 8192`
and reasoning options. To use the current Go Agent Gateway, explicitly set
`OPENAI_API_PROFILE=aiw_gateway`, `OPENAI_BASE_URL` to its `/v1` URL, and
`OPENAI_API_KEY` to a Gateway key. This profile omits `max_output_tokens` and
rejects `effort` or `include_reasoning_summary: true` before sending a request;
the Gateway does not support these options. Model, instructions and JSON
requests remain available. The existing Gateway and client result limits
still apply; character/byte limits are not equivalent to a token limit.
Unknown profile values are rejected. Profiles are explicitly configured and
are never selected automatically based on the URL. Offline SDK tests use
simulated responses; separate live evidence records actual Gateway calls.

Optional `system_prompt` sends a separate OpenAI Responses `instructions`
value; Codex and Copilot requests with this field fail explicitly. Optional
`include_reasoning_summary: true` asks OpenAI for a published reasoning
summary and adds `reasoning_summary` to the result (`null` when unavailable).
The summary is never written to the audit JSONL. Existing requests need
neither field. The separate [Agent Proxy client](../aiw-agent-proxy-client/README.md)
offers the one-shot `ai` command for this endpoint.

For OpenAI Responses, `usage.openai_usage` also returns the response ID, actual
model, service tier, response status, total tokens, cached input tokens,
cache-write input tokens, and reasoning output tokens. The same object is
written to the audit record. Missing values are `null`; `cost` stays `null`
because Responses does not supply a monetary amount. To calculate a charge,
use the model and tier's prices for uncached input, cached input, cache writes,
and output at the time of the request. Reasoning tokens are already included
in `output_tokens`; do not add them a second time. Pricing can also depend on
context length, processing mode, and tools, so reconcile estimates with the
provider's billing records.

## Standalone OpenAI API check

From this directory, run `node ./scripts/check-openai.mjs` after setting
`OPENAI_API_KEY` in the process environment. The script needs only Node.js;
the Agent Proxy service does not need to be started. It makes one short request
to the official Responses endpoint and prints only the HTTP status, whether
text was returned, and available token counts. The default model is
`gpt-6-luna`; set `OPENAI_MODEL` to use another model available to your key.
It does not print the key, response body, or API error body.

In PowerShell, enter the key without putting it in command history:

```powershell
$secret = Read-Host "OpenAI API key" -AsSecureString
try {
  $env:OPENAI_API_KEY = [System.Net.NetworkCredential]::new("", $secret).Password
  node ./scripts/check-openai.mjs
} finally {
  Remove-Item Env:OPENAI_API_KEY -ErrorAction SilentlyContinue
}
```

To check the OpenAI path through the built Agent Proxy, run
`node ./scripts/check-openai-proxy.mjs` with the same environment variable.
This script starts a temporary loopback service, makes one real request, checks
the audit record, and removes its temporary state. It prints no key, prompt,
response body, or audit record content. It prints the allowed OpenAI usage
metadata and whether it matches the audit. Run it yourself when ready; it is
not part of the automated build.

## WebSocket

Connect to `ws://127.0.0.1:43127/v1/events?client_id=<self-asserted-id>`.
Send one request event at a time:

```json
{"type":"request","request":{"client_id":"editor-1","provider":"openai","model":"gpt-5","output_format":"markdown","prompt":"Write a short summary."}}
```

The service sends a `result` event containing `event_id`, `request_id`, result
text and usage. Send `{"type":"ack","event_id":"<event_id>"}` after
receiving it. If the connection drops before ACK, reconnect using the same
`client_id`; pending result events are sent again. Duplicate ACKs are safe.

The client ID is only a self-asserted routing label. It is not authentication
or an authorization boundary; another local process can claim the same value.
The service does not expose CORS access and rejects non-loopback network peers.

## Provider safety

OpenAI uses HTTPS for the official API and remote endpoints; loopback HTTP is
available for a local compatible Gateway. Copilot invokes the CLI with an empty
`--available-tools=` list, explicit tool denials, disabled built-in MCP servers,
and no interactive questions. On Windows it prefers the npm Copilot entrypoint
on PATH; `AIW_AGENT_PROXY_COPILOT_EXECUTABLE` can select an existing executable.
Its CLI-declared tool restrictions are used without a separate negative runtime
check. Codex is invoked with `features.shell_tool=false`,
read-only sandboxing, no approval requests, disabled hooks/apps, and instructions
to use no tools or local files. `AIW_AGENT_PROXY_CODEX_EXECUTABLE` may name an
existing Codex executable; otherwise on Windows the service prefers the npm
Codex entrypoint found on PATH and falls back to `codex.exe`. Other platforms
use `codex`. If it is unavailable, the request fails with
`provider_not_configured`. The Codex restrictions reduce access but cannot prove
that no sensitive file is read; this residual risk is accepted in FD-006.

Audit JSONL records request/event IDs, self-asserted client ID, Provider/model,
status, duration and available usage metadata. It does not record the prompt,
result body, API key or complete environment. The result spool is separate
from the audit log and temporarily contains the result until ACK or expiry.

## 扩展功能与功能开发流程

`aiw-agent` 是一个请求转发服务：HTTP 和 WebSocket 请求都会先经过输入校验，再按 Provider 调用模型，最后保存待确认结果并写入脱敏审计记录。它目前不包含知识库存储、文档导入或检索功能。

### 从哪里开始改

1. 在 `docs/features/` 新建或更新编号 FD，写明用户场景、范围、兼容性、数据与权限边界、验收条件和待确认问题；阅读相关稳定规范后再实现。
2. 请求字段和返回类型在 [`src/types.ts`](src/types.ts)，HTTP 与 WebSocket 共用的请求校验在 [`src/validate.ts`](src/validate.ts)。新增输入要同步定义类型、校验、错误行为和文档。
3. 两种入口都在 [`src/service.ts`](src/service.ts) 汇入 Provider 调用。模型调用和 Provider 差异放在 [`src/providers.ts`](src/providers.ts)；通用配置在 [`src/config.ts`](src/config.ts)。不要只改一个入口而漏掉另一个。
4. 结果暂存和 ACK/过期清理在 [`src/store.ts`](src/store.ts)，审计字段在 [`src/types.ts`](src/types.ts) 与 service 写入逻辑中。持久化新数据前，先定义访问权限、保留期限、清理方式和审计脱敏规则。
5. 同步更新本 README 和受影响的稳定规范。完成后更新 FD 的 Work Items、TODO 和 Verification；静态检查为默认，执行测试或真实 API 请求前按仓库规则取得授权。

### 示例：设计知识库检索

先把知识库当成一项独立能力设计，而不是直接把所有文档拼进 `prompt`。一个可评审的 FD 至少应明确：

- **语料生命周期：** 谁能导入、支持哪些文件、大小与数量上限、更新/删除如何生效，索引存放在哪里以及何时清理。
- **隔离与授权：** 每个知识库由谁拥有、请求者如何获得访问权。当前 `client_id` 是调用方自报的路由标签，不是身份认证；不能单独用它保护远程知识库数据。
- **检索契约：** 请求如何选择知识库，如何限制检索条数和上下文大小，是否返回来源引用，以及空结果、索引不可用和超时如何处理。
- **模型与安全：** 检索片段属于不可信内容；明确如何避免片段覆盖系统指令，并确认 OpenAI、Codex、Copilot 三种 Provider 是否都要支持同一行为。
- **隐私与费用：** 明确原文、查询、检索片段、生成结果分别是否落盘；审计不得意外记录文档正文或凭据，并估算存储、索引和额外模型用量。

确定范围后，再选择最小改动面：若知识库由外部服务管理，增加受授权的检索适配层；若由本服务管理，还需单独设计导入接口、索引存储、更新/删除与数据迁移。两种方案的持久化和安全边界不同，先通过 FD 做决定，不要在 Provider 方法里临时读取任意文件或把整份语料写入提示词。以上只是设计清单，当前版本没有实现这些接口。
