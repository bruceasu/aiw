# FD-006: 本机 TypeScript Agent Proxy

**Status:** Open
**Revision:** 6
**Priority:** Medium

## Problem

Agent 客户端需要使用 Codex CLI、Copilot CLI 或 OpenAI API 发起模型请求。目前 `.ai/tmp/ts-agent/` 只有 provider/config 参考代码，没有可复用的通用 HTTP/WebSocket 服务、断线结果暂存和最小审计记录。客户端因而要自行适配多个 Provider，也无法在连接中断后可靠取回结果。

目标是提供一个简单的、仅监听本机的 TypeScript 服务，支持同步单发与 WebSocket 异步单发，并在不把完整 prompt/response 写入审计日志的前提下尽可能记录用量。

## Options and decision

1. **独立 TypeScript Node 服务（选定）。** 在 `plugins/aiw-agent-proxy/` 建立独立服务入口与 Provider 适配层；HTTP 使用 Node 标准库，WebSocket 采用小型专用依赖。复用 `.ai/tmp/ts-agent/` 的 Provider 参考逻辑，但不复用其中面向 CZ 的配置、Git 上下文和候选校验。此方案与已批准 Issue 和现有参考代码一致，边界小，可独立启动和验证。
2. 扩展现有 Go AIW Server：复用服务框架，但需把 TypeScript Provider 逻辑跨进程或改写为 Go，并将代理能力绑定到现有服务部署。当前 Issue 明确以 TypeScript 参考代码为基础，且要求快速形成独立本机代理，新增跨语言服务边界没有已知收益。
3. 仅提供 Provider 库：不能满足统一 HTTP/WebSocket 接入、ACK、重连重发和结果暂存要求。

## Solution

- 提供独立命令 `aiw-agent-proxy start`，默认绑定 `127.0.0.1`；监听地址不可配置为非 loopback。端口可由启动参数配置，避免依赖固定端口。该服务不纳入 `aiw` Go 常驻服务或 Supervisor。
- 用统一请求类型表达 `client_id`、`request_id`、`provider`、`model`、`effort`、`output_format`（`json`/`markdown`）和 `prompt`。Provider 适配器负责校验并映射各自支持的模型、effort 与输出格式；不支持的参数明确报错，不静默忽略。初版不承诺 JSON Schema 约束，只承诺 JSON 输出格式。
- `POST /v1/requests` 同步执行一次请求，返回对应结果或结构化错误。`GET /v1/events` 建立 WebSocket 长连接；客户端发送一次 request event，服务执行一次请求并发送带 `request_id`/`event_id` 的 result event。客户端收到后发送 ACK event。相同 `event_id` 重复 ACK 幂等处理。
- 结果送达 ACK 前写入本机临时结果队列。队列以每个结果一个文件保存完整待交付结果，采用原子写入；记录 `client_id`、`request_id`、创建时间和过期时间。客户端重连并声明相同 `client_id` 后重发未确认结果；ACK 成功后立即删除文件；一小时后清理过期文件。启动和定期清理都执行过期清理。临时结果队列不是审计日志。
- `client_id` 是客户端自报的关联标签，不是身份凭据；本机绑定限制远程主机访问，但不隔离本机上的进程。任何本机进程均可能自报另一个 ID，因此不得把 ID 用作权限边界或保密保证。
- 审计日志使用本机 JSONL，仅记录时间、请求/事件 ID、客户端 ID、Provider、模型、状态、耗时、可获得的输入/输出 Token、费用和错误类别。不得记录完整 prompt/response、API key 或完整环境变量；错误信息需过滤凭据和正文。Provider 未提供用量/费用时写 `null`，不估算为实际花费。
- OpenAI 通过 API 使用配置的 API key、model 和 base URL。Codex/Copilot 仅在其实际 SDK/CLI 可可靠禁止 Shell 执行、文件修改及敏感文件读取时启用；无法建立并验证这些限制时 fail closed，拒绝调用并给出原因。只读 sandbox 本身不作为“禁止 Shell”的充分证据。

## Scope

- 新建独立 TypeScript 服务包、HTTP/WebSocket 接口、Provider 适配器、请求参数校验、结果文件队列、过期清理和审计日志。
- 支持 Codex CLI、Copilot CLI、OpenAI API；各 Provider 能力差异应显式报告。
- 提供最小配置与本机启动/停止说明；不要求守护进程管理器、UI 或远程访问。
- 为已实现的稳定 HTTP/WebSocket 行为补充 `openspec/specs/` 能力规格，并更新使用文档。

不包含：远程监听或网络身份认证、多个请求的批处理/流式 token 输出、完整 prompt/response 审计、通用队列服务、Supervisor/AIW Work Core 集成、自动估算实际费用、修改 `.ai/tmp/ts-agent/` 作为库或添加 OpenSpec change。

## Work items

- [ ] 1.1 验证 Codex 与 Copilot Provider 的安全隔离能力。静态检查发现当前 Codex SDK 没有工具白名单，故实现为 fail-closed；Copilot 使用 `mode: "empty"` 和 `availableTools: []`，仍需隔离负向运行验证后才能启用。验收：有可复核的能力证据和负向验证；任一 Provider 无法满足时，该 Provider 在初版中保持禁用，不阻塞其他 Provider。
- [ ] 1.2 建立独立 TypeScript 服务与统一 Provider 接口。已实现 HTTP 路由、请求校验和 OpenAI Responses 调用；Codex 保持禁用，Copilot 代码受未通过验证的安全门禁保护。尚未安装声明的运行依赖或执行 Provider 验证。
- [ ] 1.3 实现 WebSocket 单发请求、结果事件、ACK 和重连重发。已实现事件关联、待 ACK 文件、重连重发、ACK 删除、幂等 ACK 和 TTL 清理逻辑；尚无运行证据。
- [ ] 1.4 实现最小审计日志和用量采集。已实现不含 prompt/response/key 的 JSONL 元数据记录；OpenAI Token 计数可读，费用保持未知；Copilot 用量映射待安全门通过后验证。尚未进行运行时日志检查。
- [x] 1.5 补齐运行与稳定规格文档。已新增插件启动/停止和 API 说明及 `openspec/specs/agent-proxy/spec.md`；没有创建 OpenSpec change。

保持 Work Item 编号稳定。每项完成前须具有其验收描述要求的真实证据；没有运行的验证不得标为通过。

## Acceptance

1. 服务仅绑定 `127.0.0.1`，可配置端口，不接受远端绑定配置；HTTP 与 WebSocket 均只能经本机回环接口访问。
2. HTTP 单次请求支持三个 Provider 目标、客户端指定模型/effort 和 `json`/`markdown` 输出；配置/Provider 不支持的组合清晰拒绝。
3. WebSocket 长连接可接收异步单次请求，以带请求和事件关联 ID 的 result event 返回；结果由客户端 ACK 确认，同一 ID 的重连可重新收到未 ACK 结果。
4. 未 ACK 结果在本机临时文件中最多保留一小时，ACK 后立即删除，过期自动清除；重启后仍能读取有效结果。
5. Codex/Copilot 不能执行 Shell、写文件或读取敏感文件；能力限制不可证实时相应 Provider 调用失败关闭，不降级为无限制调用。
6. 审计记录含可获取的请求元数据、状态、Token 与费用；缺失用量明确标记未知；完整 prompt/response 和凭据不写入审计日志。
7. 错误、断线、重复 ACK、重启恢复和过期清理行为可观察且有解释；自报 `client_id` 只用于关联，不被表述为身份认证。

## Verification

- 设计依据：已阅读已批准的 `REQ00005-ai-agent-proxy`、其决策记录、`.ai/tmp/ts-agent/config.ts` 与 `llm.ts`、FD 操作约定。
- 已静态实现 `plugins/aiw-agent-proxy/` 中的 HTTP/WebSocket 服务、请求校验、结果队列、审计和 OpenAI adapter；新增稳定规格及启动文档。
- Codex SDK 类型仅提供只读 sandbox，没有线程工具白名单；Codex 请求始终返回 `provider_disabled`。Copilot SDK 类型支持空模式与空工具 allowlist，但 `isolationVerified` 保持 `false`，直到负向运行验证通过。
- Compile-only 检查通过：`node ..\\aiw-cz\\node_modules\\typescript\\bin\\tsc --noEmit --typeRoots ..\\aiw-cz\\node_modules\\@types -p tsconfig.json`（在插件目录运行）。此检查使用现有 Node 类型，尚未安装/解析 `ws` 与 Copilot 包的声明。
- 未运行 Provider、服务或测试；未安装 npm 依赖，未执行网络调用。
- 实施报告：`docs/features/reports/FD-006-implementation.md`。
- CLI 安全门尚未通过；Codex 与 Copilot 保持 fail-closed，不请求放宽禁止 Shell/文件访问的规则。

- 用户已确认可在 `plugins/aiw-agent-proxy` 执行一次 `npm install --ignore-scripts`。依赖已安装并生成 `package-lock.json`；随后 `node node_modules/typescript/bin/tsc --noEmit -p tsconfig.json` 通过。首次检查发现 Copilot SDK 类型不匹配，修正后同一命令通过。
- npm 安装报告 1 项高危依赖告警；尚未定位或修复。未运行服务、Provider、安全负向验证或测试。

- `%% NEEDS_INPUT: 是否授权在 plugins/aiw-agent-proxy 运行一次 npm run build（生成被忽略的 dist/），随后运行 node scripts/smoke-local.mjs？脚本只监听并访问本机回环地址，用模拟 OpenAI 响应验证 HTTP、WebSocket 重连/ACK、过期清理与审计；不访问真实 Provider。`

## Sources

- Issue: `REQ00005-ai-agent-proxy` — `docs/requirements/REQ00005-ai-agent-proxy/requirement-plan.md`
- Approved Issue decision: `docs/requirements/REQ00005-ai-agent-proxy/decision-log.md`
- Provider/config reference: `.ai/tmp/ts-agent/config.ts`
- Provider invocation reference: `.ai/tmp/ts-agent/llm.ts`
- FD lifecycle contract: `skills/work-management.md`

## Deferred engineering decisions

- SDK/CLI 版本及安全能力验证方式；Provider 不满足要求时，初版按 1.1 禁用，不要求人放宽安全规则。
- 各 Provider 对 effort、JSON/Markdown 的精确字段映射与参数范围，由 1.2 依据实际 SDK/API 能力落实并记录。
- Windows/macOS/Linux 的临时目录权限细节与审计文件轮转；必须遵守 1 小时结果 TTL 和最小日志规则。
- WebSocket 消息大小、请求超时和每个客户端并发上限；实现需设有限制，避免本机请求无限占用资源。
