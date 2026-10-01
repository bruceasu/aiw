# FD-006: 本机 TypeScript Agent Proxy

**Status:** Complete
**Revision:** 19
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
- 对 OpenAI Responses 保留额外的计费核算元数据：响应 ID、实际模型、服务层级、响应状态，以及返回的总 Token、缓存命中、缓存写入和推理 Token。仅复制已知字段并校验类型；不记录整个响应或动态价格，不把推理 Token 再加到已包含它的输出 Token 中。
- OpenAI 通过 API 使用配置的 API key、model 和 base URL。Codex CLI 调用显式设置 `features.shell_tool=false`、只读 sandbox、`approval_policy=never`，关闭 hooks/apps，并通过高优先级指令禁止工具调用、Shell 权限申请与本地文件读取。用户确认这些措施已是当前可做到的程度，接受无法证明绝不读取敏感文件的残余风险；提示语本身不被表述为硬隔离。Copilot 直接调用 CLI，使用空 `--available-tools=` 列表、显式拒绝工具权限、关闭内置 MCP 与交互提问；依据 CLI 参数声明启用，不增加隔离负向运行验证门槛，也不依赖 Copilot SDK。

## Scope

- 新建独立 TypeScript 服务包、HTTP/WebSocket 接口、Provider 适配器、请求参数校验、结果文件队列、过期清理和审计日志。
- 支持 Codex CLI、Copilot CLI、OpenAI API；各 Provider 能力差异应显式报告。
- 提供最小配置与本机启动/停止说明；不要求守护进程管理器、UI 或远程访问。
- 为已实现的稳定 HTTP/WebSocket 行为补充 `openspec/specs/` 能力规格，并更新使用文档。

不包含：远程监听或网络身份认证、多个请求的批处理/流式 token 输出、完整 prompt/response 审计、通用队列服务、Supervisor/AIW Work Core 集成、自动估算实际费用、修改 `.ai/tmp/ts-agent/` 作为库或添加 OpenSpec change。

## Work items

- [x] 1.1 核对 Codex 与 Copilot Provider 的安全隔离能力。Codex CLI 路径关闭默认 Shell 工具、使用只读 sandbox 和禁止审批申请，并以高优先级指令禁止工具及本地文件读取。用户接受敏感文件读取无法绝对证明的残余风险；没有把提示语记为硬隔离。Copilot CLI 依据其参数声明使用空工具列表和显式工具拒绝，关闭内置 MCP 与交互提问；按用户决策免除负向运行验证门槛。
- [x] 1.2 建立独立 TypeScript 服务与统一 Provider 接口。HTTP 路由、请求校验和三个 Provider 适配器已实现；Codex、Copilot CLI 均有真实完成请求证据。用户独立运行 `check-openai-proxy.mjs`，经 Agent Proxy 的 OpenAI 请求返回 HTTP 200 与文本。
- [x] 1.3 实现 WebSocket 单发请求、结果事件、ACK 和重连重发。本机 smoke 已覆盖单发结果、断线后服务重启、同一客户端重连重发、ACK 删除及 TTL 过期清理。
- [x] 1.4 实现最小审计日志和用量采集。本机 smoke 验证 JSONL 元数据字段及正文排除；Codex 真实请求返回 `12964/5` token，Copilot CLI 未给出可解析 Token，保持 `null`。用户独立运行真实 OpenAI 代理请求，响应和审计均记录 `11/5` token，审计状态为 `succeeded` 且未包含 Key、提示词或结果正文。
- [x] 1.5 补齐运行与稳定规格文档。已新增插件启动/停止和 API 说明及 `openspec/specs/agent-proxy/spec.md`；没有创建 OpenSpec change。
- [x] 1.6 扩展 OpenAI 用量审计。向同步/异步结果和 JSONL 审计追加可选的 `openai_usage` 明细；保留原有 `input_tokens`、`output_tokens`、`cost`，缺失值为 `null`，不推算金额。若响应已产生用量但没有可用正文，失败审计仍保留该用量。类型、映射和两条审计路径静态一致，本机模拟响应已验证新增字段不泄露正文或 Key。

保持 Work Item 编号稳定。每项完成前须具有其验收描述要求的真实证据；没有运行的验证不得标为通过。

## Acceptance

1. 服务仅绑定 `127.0.0.1`，可配置端口，不接受远端绑定配置；HTTP 与 WebSocket 均只能经本机回环接口访问。
2. HTTP 单次请求支持三个 Provider 目标、客户端指定模型/effort 和 `json`/`markdown` 输出；配置/Provider 不支持的组合清晰拒绝。
3. WebSocket 长连接可接收异步单次请求，以带请求和事件关联 ID 的 result event 返回；结果由客户端 ACK 确认，同一 ID 的重连可重新收到未 ACK 结果。
4. 未 ACK 结果在本机临时文件中最多保留一小时，ACK 后立即删除，过期自动清除；重启后仍能读取有效结果。
5. Codex 禁用默认 Shell 工具、采用只读 sandbox 和禁止审批申请，并被明确指示不调用工具或读取敏感文件；Copilot CLI 采用空工具列表和显式工具拒绝。敏感文件读取无法绝对证明的残余风险已由用户接受，不得将 prompt 声称为强制隔离。
6. 审计记录含可获取的请求元数据、状态、Token 与费用；缺失用量明确标记未知；完整 prompt/response 和凭据不写入审计日志。
7. 错误、断线、重复 ACK、重启恢复和过期清理行为可观察且有解释；自报 `client_id` 只用于关联，不被表述为身份认证。

## Verification

- Independent review: `docs/features/archive/FD-006/reviews/FD-006-review.md`; outcome `VERIFICATION_PASSED`. The review assessed the latest recorded human decisions, including the accepted Codex sensitive-file-read risk and the waived Copilot negative isolation test. No FD-006 review-requested event was present, so this direct review did not claim or emit a lifecycle event. The human subsequently requested closure; the standalone FD was archived using the portable close operation without fabricating a lifecycle event.

- 设计依据：已阅读已批准的 `REQ00005-ai-agent-proxy`、其决策记录、`.ai/tmp/ts-agent/config.ts` 与 `llm.ts`、FD 操作约定。
- 已静态实现 `plugins/aiw-agent-proxy/` 中的 HTTP/WebSocket 服务、请求校验、结果队列、审计和 OpenAI adapter；新增稳定规格及启动文档。
- 更正 1.1 判断：[OpenAI 配置文档](https://learn.chatgpt.com/docs/config-file/config-reference)说明 `features.shell_tool = false` 会关闭默认 Shell 工具，先前“没有白名单就无法禁用 Shell”的判断不成立。Codex 适配器现通过 CLI 配置关闭 Shell、采用只读 sandbox 和 `approval_policy=never`，并用高优先级指令禁止工具调用及本地文件读取；用户接受敏感文件读取无法绝对证明的残余风险。Copilot SDK 1.0.14 的本地类型声明支持空模式及空工具列表；本轮未运行真实 Codex 或 Copilot Provider。
- Compile-only 检查通过：`node ..\\aiw-cz\\node_modules\\typescript\\bin\\tsc --noEmit --typeRoots ..\\aiw-cz\\node_modules\\@types -p tsconfig.json`（在插件目录运行）。此检查使用现有 Node 类型，尚未安装/解析 `ws` 与 Copilot 包的声明。
- 未运行 Provider、服务或测试；未安装 npm 依赖，未执行网络调用。
- 实施报告：`docs/features/archive/FD-006/reports/FD-006-implementation.md`。
- Codex 已按用户确认的限制接入 CLI；敏感文件读取的强制隔离仍无法证明。Copilot 按用户决策采用 SDK 声明的空工具配置。
- 1.2 静态补齐：Copilot 在 SDK 会话创建前查询 `listModels()`，拒绝未知或禁用模型及模型不支持的 effort；预备 smoke 脚本只检查模拟 OpenAI 路径，不会调用真实 CLI Provider。尚未调用真实 Provider 或运行脚本。
- 本轮对 `src/providers.ts` 使用本地 TypeScript `transpileModule` 做无产物语法编译，通过；插件目录缺少本地 `node_modules`，未做完整项目类型检查或运行验证。
- 用户授权本地依赖安装、构建和 smoke 检查；`npm` PowerShell 入口因执行策略受阻，改用 `npm.cmd install --ignore-scripts` 后下载 `ws@8.18.3` 遭系统 `EACCES` 拒绝，清理时另见 `EPERM`。依赖安装未完成，故未运行 `npm run build` 或 `node scripts/smoke-local.mjs`。1.2、1.3、1.4 仍缺运行证据。
- Codex 适配器改用 Node `spawn` 直接启动现有 CLI，配置 `features.shell_tool=false`、`--sandbox read-only`、`--ask-for-approval never` 和禁止工具/文件读取的高优先级指令；`--json` 输出只提取最终文本与已报告的 token。`src/providers.ts` 无产物语法编译通过；未运行 Codex CLI，也未作完整类型检查。提示语和只读 sandbox 无法证明敏感文件绝不被读取，用户已接受该残余风险。

- 用户已确认可在 `plugins/aiw-agent-proxy` 执行一次 `npm install --ignore-scripts`。依赖已安装并生成 `package-lock.json`；随后 `node node_modules/typescript/bin/tsc --noEmit -p tsconfig.json` 通过。首次检查发现 Copilot SDK 类型不匹配，修正后同一命令通过。
- npm 安装报告 1 项高危依赖告警；尚未定位或修复。未运行服务、Provider、安全负向验证或测试；用户明确免除 Copilot 安全负向验证门槛。

- 当前权限环境下 `npm.cmd install --ignore-scripts` 成功，`npm.cmd run build` 成功，`node scripts/smoke-local.mjs` 输出 `local smoke passed`。脚本使用模拟 OpenAI 响应与本机回环接口，未调用真实 Provider；覆盖 HTTP、WebSocket ACK/重连、过期清理与审计正文排除。
- 本轮 `node --check scripts/smoke-local.mjs` 通过；扩展后的 `node scripts/smoke-local.mjs` 通过，覆盖服务重启恢复和审计字段/模拟用量。首次运行仅因断言误写状态值 `success` 而失败；实际契约为 `succeeded`，修正断言后通过。
- 经用户确认做最小真实请求：Codex `gpt-5` 请求返回 `provider_error`。初次默认命中 VS Code 内置 CLI `0.155.0-alpha.16.3`；改用已登录的命令行安装所带原生 CLI `0.159.3` 后重试一次，仍返回 `provider_error`。Copilot SDK 在 `listModels()` 返回 `-32603`，未能进入完成请求；`OPENAI_API_KEY` 未设置，OpenAI 未调用。未输出回复正文或凭据。
- 后续诊断：`codex doctor --json` 显示 ChatGPT 认证和连接检查通过、当前模型为 `gpt-6-luna`；本地模型目录不含先前手工选用的 `gpt-5`。改用当前 CLI 原生程序及 `gpt-6-luna` 后真实请求成功，返回文本与 `12964/5` token。Windows 默认入口现优先使用 PATH 中 npm Codex 启动脚本；`node node_modules/typescript/bin/tsc --noEmit -p tsconfig.json` 和 `npm.cmd run build` 通过，不设置可执行文件覆盖项的真实请求也成功。
- 用户决定不依赖 Copilot SDK。核对本机 Copilot CLI 参数及 [GitHub CLI 命令文档](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference) 后，使用 stdin 提示词、空工具列表、显式工具拒绝和 JSONL 输出完成最小真实请求；新适配器的真实 `auto` 模型请求返回正文。`npm.cmd uninstall @github/copilot-sdk --package-lock-only --offline --ignore-scripts` 移除了包依赖，报告 0 项漏洞；无产物 TypeScript 编译与构建通过。
- 已准备独立的 `plugins/aiw-agent-proxy/scripts/check-openai.mjs` 供用户自行验证 OpenAI API：仅从环境读取 Key，固定官方 HTTPS Responses 地址，不输出 Key、响应正文或错误正文；默认 `gpt-5.4-mini`，可用 `OPENAI_MODEL` 覆盖。用户明确要求自行运行，故本轮没有执行脚本或发送 API 请求；1.2 和 1.4 的真实 OpenAI 证据仍待用户反馈。
- 用户反馈独立脚本结果：`HTTP 200; model=gpt-6-luna; text_received=true; input_tokens=11; output_tokens=5`。这证明该 Key 的 OpenAI Responses API 调用成功，不等同于经 Agent Proxy 的真实 HTTP 与审计路径。用户当前脚本的默认模型已是 `gpt-6-luna`；文档随之更正。另备 `scripts/check-openai-proxy.mjs`，由用户自行运行一次真实代理请求并检查临时审计，脚本本轮未执行。
- 用户反馈 `node check-openai-proxy.mjs` 结果：`HTTP 200; model=gpt-6-luna; text_received=true; input_tokens=11; output_tokens=5; audit_status=succeeded; audit_redacted=true; audit_input_tokens=11; audit_output_tokens=5`。该脚本经本机 Agent Proxy HTTP 路径调用真实 OpenAI API，并检查了临时审计记录；Worker 未接触 Key，也未再次运行网络请求。
- 1.6：新增 OpenAI 响应 ID、实际模型、服务层级、响应状态、总 Token、缓存读取/写入输入 Token、推理输出 Token 明细。`node node_modules/typescript/bin/tsc --noEmit -p tsconfig.json`、`npm.cmd run build` 和 `node scripts/smoke-local.mjs` 通过。smoke 仅模拟 OpenAI API，验证 HTTP/WebSocket 结果、成功及无正文失败审计中用量一致，审计不含 Key、提示词或正文。
- 用户独立运行更新后的 `node check-openai-proxy.mjs`：HTTP 200，`gpt-6-luna` 返回输入/输出 `11/5`、总计 `16` Token；实际响应字段含响应 ID、模型 `gpt-6-luna`、服务层级 `default`、状态 `completed`、缓存读取 `0`、缓存写入 `0`、推理输出 `0`，`audit_status=succeeded`、`audit_redacted=true`、`audit_usage_match=true`。这验证了本次真实 OpenAI HTTP 结果与审计明细一致；Worker 未接触 Key 或重复请求。
- `%% RISK: 其他模型或请求类型可能不返回相同的 OpenAI 明细；缺失字段保持 null，cost 不估算。`
- `%% RISK: Copilot CLI 本次未给出可解析的 token 数，审计中保持 null；Codex 对无效模型目前返回通用 provider_error，错误分类可继续改善。`

## Sources

- Issue: `REQ00005-ai-agent-proxy` — `docs/requirements/REQ00005-ai-agent-proxy/requirement-plan.md`
- Approved Issue decision: `docs/requirements/REQ00005-ai-agent-proxy/decision-log.md`
- Provider/config reference: `.ai/tmp/ts-agent/config.ts`
- Provider invocation reference: `.ai/tmp/ts-agent/llm.ts`
- FD lifecycle contract: `skills/work-management.md`

## Deferred engineering decisions

- CLI 版本及安全能力验证方式；Provider 不满足要求时，初版按 1.1 禁用，不要求人放宽安全规则。
- 各 Provider 对 effort、JSON/Markdown 的精确字段映射与参数范围，由 1.2 依据实际 CLI/API 能力落实并记录。
- Windows/macOS/Linux 的临时目录权限细节与审计文件轮转；必须遵守 1 小时结果 TTL 和最小日志规则。
- WebSocket 消息大小、请求超时和每个客户端并发上限；实现需设有限制，避免本机请求无限占用资源。

**Completed:** 2026-10-01
