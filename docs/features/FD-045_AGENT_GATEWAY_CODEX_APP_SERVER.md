# FD-045：Agent Gateway 改用 Codex App Server 并兼容 Agents SDK

**Status:** Design
**Revision:** 2
**Priority:** Medium
**Evidence policy:** Dual

## Problem

Agent Gateway 面向调用方提供 OpenAI Responses API。当前实现用 `codex exec --json` 启动一次性进程，并只在 `item.completed` 时收取完整消息。FD-022 已定义 Agents SDK 对 Responses API 的兼容目标，包括函数工具回合、流式文本和结构化输出；但它把 `codex exec --output-schema` 的行为作为关键实现前提，且该路径尚未得到运行验证。

本 FD 合并 FD-022 的 API 兼容范围与 FD-045 的后端迁移范围。Gateway 对外继续提供受限的 OpenAI Responses API；内部改用 Codex App Server。更换后端可能移除对 `codex exec --output-schema` 输出形状的依赖，但不能预先断定它会消除所有协议、安全和生命周期 Gate。

## Options and decision

1. 保留 `codex exec`：进程生命周期简单，但每次请求重复启动，输出仅在完整消息完成后可用；工具调用依赖尚未验证的 `--output-schema` 路径。
2. 每请求启动 App Server：使用双向 RPC 和增量事件，但仍重复初始化进程。
3. 建立有界 App Server 进程池：每个池槽复用一个进程，每次请求创建独立 thread；需要明确 RPC、取消、超时、状态恢复和 rollout 生命周期。

决定采用方案 3。每个进程同一时刻只处理一个请求，池大小沿用 `global_concurrency`；HTTP 容量拒绝行为不变，不引入排队。App Server 仅作为模型后端。Gateway 继续拥有认证、模型授权、RPM、日额度、并发、审计和 OpenAI 响应格式。

OpenAI Responses 是外部兼容边界。目标是满足固定版本 `@openai/agents` 默认 `OpenAIResponsesModel` 所需的、在本 FD 明确列出的 API 子集，而不是实现全部 OpenAI API 或 Agents SDK 的所有 provider、工具和运行模式。实现前先以官方 SDK 与 App Server 契约证据确认具体请求和事件映射；Codex 原生工具仍关闭，调用方提供的 function tools 只作为模型输出选择，不授予 App Server 本地工具权限。

## Solution

### 合并范围与 API 契约

- 保持 `/v1/responses`、`/v1/models`、`/v1/usage`、现有认证和错误 envelope。SDK 通过标准 OpenAI Responses HTTP 客户端接入；不增加 `/v1/agents` 或 App Server HTTP 端点。
- 保留 FD-022 已确定的目标版本 `@openai/agents` v0.18.0。以默认 Responses provider 为兼容对象，不承诺 Chat Completions provider、handoffs、hosted tools、sessions、`previous_response_id` 或完整 OpenAI API。
- 第一阶段支持非流式顺序 function tool 回合、无工具的流式文本回合及有界结构化输出。函数调用由调用方执行；follow-up 请求重放 `function_call` 与匹配的 `function_call_output`。
- 流式工具回合、并行工具调用及工具与结构化输出的组合，只有在 SDK 与 API 契约证据完整且实现验收后才可加入；未选或未支持的组合必须在启动 Runner 前明确拒绝。
- `strict:true` 必须兑现 schema 约束：Gateway 对模型生成的参数按声明 schema 校验；无法支持的 schema 子集在 Runner 前拒绝。不得把“是 JSON 对象”当作严格 schema 校验。
- App Server 的 `outputSchema`、原生 item 类型、事件顺序和 usage 含义需先核对目标 Codex 版本。只有实际契约支持且映射无损时，才能替代 `codex exec --output-schema` 路径；不得根据命令帮助或字段名称推断等价性。

### App Server 后端边界

- 在 `program/agent-gateway` 内集中定义 JSONL RPC、请求/响应 ID、thread/turn/item ID、事件和终态；不得把 JSON-RPC 细节扩散到 HTTP、配额或存储模块。
- 通过 stdin/stdout JSONL 完成 initialize/initialized 握手；保留 `codex_path` 和 Windows Node 启动配置，不新增第三方依赖。
- 每个请求创建独立 cwd 和新 thread，不接受外部 thread ID，不调用 thread/resume。完成后释放 thread 资源并清理临时目录。
- 服务端固定命令、环境白名单、read-only、Codex 工具禁用范围继续生效；不因 App Server 增加 shell、MCP、apps、hooks、web 或子 agent 权限。
- 每个池槽最多一个活动请求。超时或客户端断开时发送 turn/interrupt，并在清理预算内确认终态；协议失同步或终态未知时淘汰该池槽，不把不确定执行交给下一请求，不自动重试。
- Linux process group 与 Windows Job Object 继续负责进程树清理。正常 turn 完成不杀健康的常驻进程；关闭服务时停止接收、取消活动 turn、关闭进程池并等待有界清理。
- thread/rollout 可能持久化的正文生命周期需按目标版本实证。保持受控目录、清理策略、凭据隔离和最小留存；在证明清理前，不得把 App Server 本身描述为不持久化。

### 用量、隐私与兼容映射

- 使用本次请求的 turn 用量，不将 thread 累计值重复计费；缺失用量保持 unknown/null，不当成零。
- 只有确认 turn/start 已启动后，才把额度 reservation 标为 started；未知执行状态按保守占用规则处理，不自动退款或重放。
- OpenAI Responses 文本、function_call、结构化输出和 SSE 终态都由 Gateway 组装。SSE 增量只来自实际 App Server 增量事件；不重复发送最终汇总文本，也不把失败伪装为 completed。
- Prompts、工具参数、工具结果、API Key 和模型正文不得进入日志或持久化审计。记录 request ID、主体、模型、终态和经确认语义的用量。

## Scope

包含 `program/agent-gateway` 的 App Server RPC 客户端、进程池与 thread/turn 生命周期、Responses API 子集映射、严格工具参数校验、SSE/结构化输出、usage/审计边界，以及 agent-proxy 稳定 spec 和 Gateway 文档。

不改变调用方执行函数工具的职责、认证主体与 Key、模型授权、额度策略、存储格式或其他服务 API。不实现通用 OpenAI API、Agents SDK 所有 provider、Codex 本地工具执行或远程 App Server。

## Work items

- [ ] 1.1 固定 Codex App Server 与 Agents SDK 版本，核对 JSONL 握手、turn/start、output schema、item/event、usage、interrupt、错误及正文留存契约。验收：记录一手来源、精确字段和版本；未知行为单独列明。大小：小；难度：中。
- [ ] 1.2 定义 Responses 到 App Server 的映射表和支持矩阵。验收：非流式 function tool、文本流、结构化输出的请求/响应映射明确；不支持组合在 Runner 前拒绝；保留认证、额度和审计边界。依赖：1.1。大小：小；难度：中。
- [ ] 1.3 实现有界 stdio JSONL RPC 客户端和 initialize 握手。验收：ID 路由、并发请求隔离、缓冲上限、错误/EOF 和进程失效处理有界。依赖：1.1。大小：中；难度：中。
- [ ] 1.4 实现独立 thread/turn 执行与资源清理。验收：每请求新 thread/cwd；请求完成、失败和取消都释放资源；未知终态不重试。依赖：1.3。大小：中；难度：中。
- [ ] 1.5 实现有界 App Server 进程池和启动/关闭生命周期。验收：池大小等于现有并发限制；单槽单请求；启动失败替换一次后停止；shutdown 有界等待。依赖：1.4。大小：中；难度：中。
- [ ] 1.6 实现非流式 Responses 文本与 function_call 映射、工具结果重放和 `strict:true` schema 校验。验收：标准 item/ID 及调用方执行语义正确；超出支持 schema 的请求在 Runner 前拒绝。依赖：1.2、1.4。大小：中；难度：中。
- [ ] 1.7 实现文本 SSE 与结构化输出映射。验收：事件序号、增量、completed/failed/cancelled 终态和格式校验符合 1.1/1.2 的证据；未支持组合前置拒绝。依赖：1.2、1.4。大小：中；难度：中。
- [ ] 1.8 对齐 turn 用量、reservation、审计和隐私处理。验收：不重复累计 thread 用量；未知状态保守记账；正文和凭据不进入持久化记录。依赖：1.4、1.6。大小：中；难度：中。
- [ ] 1.9 实现中断、进程树清理和服务停止路径。验收：超时/断连中断活动 turn；失同步进程槽被淘汰；Windows/Linux 清理及失败可观测。依赖：1.5、1.7。大小：中；难度：中。
- [ ] 1.10 更新 agent-proxy 稳定 spec、Gateway 文档及 Dual 实施报告。验收：只描述实际支持的 SDK/API 组合、App Server 版本和留存限制；报告说明静态证据及未执行的验证。依赖：1.6–1.9。大小：小；难度：低。

工作项按依赖顺序执行；每项独立提交。实施后保持编号稳定。仅在明确取消时使用 `- [-]` 并写明原因。

## Acceptance

1. `@openai/agents` v0.18.0 默认 Responses provider 可通过标准 Gateway HTTP 配置使用本 FD 明确列出的兼容子集；不宣称完整 OpenAI API 或全部 Agents SDK 功能。
2. 非流式顺序 function tool 回合按标准 `function_call` / `function_call_output` 形状往返，工具始终由调用方执行；严格模式按工具 schema 验参。
3. 文本流与结构化输出只支持经证据确认的格式和组合；不支持的组合在 App Server 执行前失败。
4. Responses item、SSE 序号和终态映射不重复、不伪造成功；用量以当前 turn 计，不重复计算 thread 累计值。
5. 每个请求使用独立 thread/cwd；取消、崩溃、协议失同步、重启和 shutdown 都有界，未知执行不自动重试。
6. App Server 的本地工具、MCP、apps、hooks、web、子 agent 和非预期审批能力保持关闭；服务端身份、模型权限、RPM、日额度和并发限制保持有效。
7. Prompt、工具参数/结果、API Key 和生成正文不进入日志或持久化审计；App Server rollout 留存和清理行为有版本化证据。

## TODO

- [ ] 用 App Server 作为唯一后端计划，整合 FD-022 的 Agents SDK / Responses API 验收范围。
- [ ] 在 1.1–1.2 中以 pinned source 证据替代所有 `codex exec` 假设；若 App Server 不支持所需输出契约，先更新决策与验收，再进入实现。
- [ ] 实施与 compile-only 验证尚未开始；运行时 SDK/backend 验证仍须单独授权精确命令。
- [ ] FD-022 现存活动副本尚未归档；本 FD 作为合并后主计划，后续需通过受支持的 FD 流程记录其 superseded disposition。

## Verification

- 当前静态证据：FD-022 记录了 `@openai/agents` v0.18.0 默认使用 Responses API 的源代码定位；Gateway 当前仍以 `codex exec --json` 加 `--output-schema` 产生工具调用。代码已实现部分非流式 function tool，但不验证参数 schema；流式工具和工具加结构化输出仍拒绝。
- FD-045 原始设计记录了 App Server 池、thread/turn 隔离、进程树清理、响应映射和隐私生命周期要求；其中 `outputSchema`、事件及 rollout 行为仍待目标版本的一手证据核实。
- 本次没有实现代码、安装 SDK、调用网络、执行测试或编译。实施阶段按仓库规则只做静态审查和一次 compile-only；可执行 SDK/App Server 验证须先提出确切命令、范围、时长及风险并取得授权。
- 旧 Planner handoff `FD-045-000002-design-requested` 因其引用 revision 2 而活动文件为 revision 1，已由用户授权的 PM `cancel-event` 取消。该操作只取消收据，没有证明或停止原 Planner 会话；审计记录在 `.ai/fd/FD-045/operations/9b3aec7514a14b75b5f2a0f80b6a253f.json`。

## Sources

- `docs/features/FD-022_AGENT_GATEWAY_OPENAI_AGENTS_SDK.md` 及其归档决策：Agents SDK Responses 兼容范围、既有证据和未完成问题。
- `docs/features/FD-045_AGENT_GATEWAY_CODEX_APP_SERVER.md`：App Server 池、生命周期、进程清理与 Gateway 边界。
- `program/agent-gateway/protocol.go`
- `program/agent-gateway/runner.go`
- `program/agent-gateway/server.go`
- `program/agent-gateway/process_windows.go`
- `program/agent-gateway/process_linux.go`
- `openspec/specs/agent-proxy/spec.md`

**Consolidation decision (2026-10-09):** FD-045 is the canonical plan for both the App Server backend and the bounded OpenAI Responses compatibility scope previously tracked by FD-022. FD-022 remains historical evidence until its disposition can be recorded through the supported archive workflow.