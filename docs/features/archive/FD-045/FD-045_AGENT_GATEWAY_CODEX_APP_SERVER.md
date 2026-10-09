# FD-045：Agent Gateway 改用 Codex App Server 并兼容 Agents SDK

**Status:** Complete
**Revision:** 11
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
- 第一阶段支持非流式顺序 function tool 回合、无工具的流式文本回合及 `json_object` 结构化输出。函数调用由调用方执行；follow-up 请求完整重放 `function_call` 与匹配的 `function_call_output`，Gateway 将其作为不可信历史内容编码到新的 turn 输入。
- `@openai/agents` v0.18.0 默认 Responses provider 的工具回合以 `function_call`/`function_call_output` items 往返。Codex App Server 的 `turn/start.input` 只接受其协议列出的 user input，而不是 OpenAI 完整 conversation items；Gateway 继续负责 Responses item 与文本上下文映射。
- 为维持每 HTTP 请求独立、可清理的 thread，函数工具不注册为 App Server `dynamicTools`。该字段在 Codex `rust-v0.160.1` 中标记为 experimental；本方案使用 `turn/start.outputSchema` 约束内部判定对象，再由 Gateway 生成标准 Responses `function_call` item。这样模型只选择工具，Gateway 不执行调用方工具。
- 第一阶段拒绝流式工具、并行工具、工具与结构化输出组合、`previous_response_id`、store/background、非文本输入和不支持的 schema。所有不支持项在 Runner 启动前拒绝。
- `strict:true` 必须兑现 schema 约束：仅接受递归 JSON Schema 子集 `type`、`properties`、`required`、`additionalProperties:false`、`items`、`enum`、`const`、`minimum`/`maximum`、`minLength`/`maxLength`、`minItems`/`maxItems`；schema 深度、节点数和输入字节数有界。拒绝 `$ref`、组合 schema、`pattern`、`format` 和其他未知关键字；超出子集的请求在 Runner 前拒绝。不得把“是 JSON 对象”当作严格 schema 校验。`strict:false` 只要求 function arguments 是 JSON object，不承诺 schema 校验。
- 锁定 Codex CLI/App Server `rust-v0.160.1` 与 Agents SDK `v0.18.0`。该 Codex 版本的 `turn/start` 定义 `input`、`outputSchema` 和 `toolOutput`；turn completed 通知携带 turn 级 usage；`turn/interrupt` 使用 thread/turn ID。SDK 的 Responses model 默认路径映射到 Responses API。

### 版本化来源与决策

- [Agents SDK v0.18.0 Responses model](https://github.com/openai/openai-agents-js/blob/v0.18.0/packages/agents-openai/src/openaiResponsesModel.ts)：固定默认 Responses provider 兼容目标。
- [Codex v0.160.1 thread protocol](https://github.com/openai/codex/blob/rust-v0.160.1/codex-rs/app-server-protocol/src/protocol/v2/thread.rs)：`thread/start` 定义 ephemeral thread；`dynamicTools` 为 experimental 字段，本 FD 不使用。
- [Codex v0.160.1 turn protocol](https://github.com/openai/codex/blob/rust-v0.160.1/codex-rs/app-server-protocol/src/protocol/v2/turn.rs)：turn 输入、工具输出、输出 schema、usage 和 interrupt wire contract。
- [Codex v0.160.1 App Server RPC methods](https://github.com/openai/codex/blob/rust-v0.160.1/codex-rs/app-server-protocol/src/protocol/common.rs)：JSON-RPC client requests 与 server `item/tool/call` 请求分类。
- [Codex v0.160.1 feature flags](https://github.com/openai/codex/blob/rust-v0.160.1/codex-rs/features/src/lib.rs)：本地工具 feature flags 和默认值；部署配置必须明确关闭不需要的功能。
- 来源只证明静态协议形状和字段，不证明绑定版本的二进制运行行为；运行时 SDK/App Server 验证仍需独立授权。

### App Server 后端边界

- 在 `program/agent-gateway` 内集中定义 JSONL RPC、请求/响应 ID、thread/turn/item ID、事件和终态；不得把 JSON-RPC 细节扩散到 HTTP、配额或存储模块。
- 通过 stdin/stdout JSONL 完成 initialize/initialized 握手；保留 `codex_path` 和 Windows Node 启动配置，不新增第三方依赖。
- 每个 HTTP 请求创建独立 cwd 和 `ephemeral:true` 新 thread，不接受外部 thread ID，不调用 thread/resume。请求结束后关闭 thread/process 资源并清理临时目录；ephemeral 不等同于进程内存清零。
- JSONL RPC 只在 Gateway 的 backend 层处理，必须有有限行长、输出总量和 stderr 上限；请求 ID 路由到响应，通知与 server request 由明确分支处理。未预期的 server request（包括 `item/tool/call`）视为协议错误并淘汰进程。
- 使用 `initialize`/`initialized` 一次握手。只有收到 initialize 成功且 server 能力满足要求后进程槽才可接收 turn。
- 服务端固定命令、环境白名单、read-only 和 Codex 工具禁用范围继续生效；不因 App Server 增加 shell、MCP、apps、hooks、web、plugins、browser、computer 或子 agent 权限。`thread/start`/`turn/start` 显式提供 cwd、approval policy、sandbox 和禁用工具配置，不继承外部 thread 设置。
- **认证与配置决策（2026-10-10）：**复用现有 `CODEX_HOME`、Codex 登录态和当前白名单环境，不复制认证材料，也不新增认证机制。App Server 使用显式 config overrides 关闭本地工具 feature flags、apps、plugins、hooks、web 和子 agent。初始化后先读取有效配置和 `mcpServerStatus/list`；若发现任何启用或状态不明的 MCP server，关闭该 App Server 进程槽并拒绝 turn，不允许模型获得 MCP 工具。未启用的配置项只有在 App Server 明确报告 disabled 时才可接受。该 Gate 失败关闭，不修改用户的 Codex 配置文件。
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

- [x] 1.1 固定 Codex `rust-v0.160.1` 与 Agents SDK `v0.18.0` 的版本化协议来源。验收：记录 initialize、ephemeral thread、turn input/outputSchema/usage/interrupt、RPC method 分类及实验性 dynamicTools 的字段来源；说明运行时未验证。大小：小；难度：低。
- [x] 1.2 定义 Responses 映射与支持矩阵。验收：本 FD 已列明文本、非流式顺序 function call、无工具文本流、json_object 和不支持组合；明确函数调用历史由文本重放，不启用 dynamicTools。大小：小；难度：低。依赖：1.1。
- [x] 1.3 实现有界 JSONL framing/writer 与 RPC ID dispatcher。验收：有限行长/缓冲；按 JSON-RPC ID 路由响应；通知和 server request 分类；EOF、非法 JSON、重复/未知 ID 使进程失效。大小：小；难度：中。
- [x] 1.4 实现 App Server 子进程启动、标准流和环境策略。验收：复用现有可执行路径/Windows Node 启动约定及 `CODEX_HOME`/登录态；不复制凭据；stderr 与 stdout 均有界；不把正文写入 Gateway 日志。依赖：1.3。大小：小；难度：中。
- [x] 1.5 实现 initialize 握手与安全配置 preflight。验收：initialize/initialized 顺序正确；显式关闭本地工具 feature flags、apps、plugins、hooks、web 和子 agent；通过 `config/read` 与 `mcpServerStatus/list` 验证有效配置；任何启用/状态不明的 MCP 使进程槽不可用；失败、超时或能力/配置不匹配时不接收 turn。依赖：1.3、1.4。大小：中；难度：中。
- [x] 1.6 实现单请求 ephemeral thread 的 turn/start 与取消。验收：新 thread/cwd；使用固定模型、instructions、read-only sandbox、approval policy 和输出 schema；取消走 turn/interrupt；外部 thread ID 不接受。依赖：1.5。大小：中；难度：中。
- [x] 1.7 实现 App Server event/item 到 Runner 结果的终态映射。验收：只从已完成 turn 得到成功；区分 failed/interrupted/unknown；不重复拼接 delta 与完成 item；未知终态不重试。依赖：1.6。大小：小；难度：中。
- [x] 1.8 实现 Responses 文本、顺序 function_call 和 function_call_output 历史重放。验收：稳定 Responses IDs/item/status；工具由调用方执行；模型输出经 `outputSchema` 和 Gateway allowlist 校验后才能返回 function_call。依赖：1.2、1.7。大小：中；难度：中。
- [x] 1.9 实现 `strict:true` 支持的 JSON Schema 子集检查。验收：支持 `type`、`properties`、`required`、`additionalProperties:false`、`items`、`enum`、`const`、数值/字符串/数组界限；schema 深度、节点和字节有上限；拒绝 `$ref`、组合 schema、`pattern`、`format` 和未知关键字；参数与子 schema 全部匹配。依赖：1.8。大小：中；难度：中。
- [x] 1.10 实现无工具文本 SSE。验收：仅转发真实文本增量；sequence_number 单调；不重复发送最终文本；完成/失败/取消终态与非流式映射一致；带工具 stream 预先拒绝。依赖：1.7。大小：小；难度：中。
- [x] 1.11 实现 turn usage、reservation 与审计状态对接。验收：按当前 turn 计费；缺失 usage 保持 unknown；只在 turn 确实启动后标记 started；不持久化 prompt、工具参数/结果、响应正文或 API Key。依赖：1.7、1.8。大小：小；难度：中。
- [x] 1.12 实现有界 App Server 进程池及故障槽淘汰。验收：槽数等于 `global_concurrency`；每槽一个活动请求；启动失败最多补建一次；失同步/未知执行淘汰槽且不自动重试。依赖：1.5–1.7。大小：中；难度：中。
- [x] 1.13 对接客户端断开、超时、进程树清理和服务 shutdown。验收：中断 turn 后在清理预算内确认终态；Linux process group/Windows Job Object 都有界；失败关闭 Gateway 接收新请求。依赖：1.6、1.12。大小：中；难度：中。
- [x] 1.14 更新 agent-proxy 稳定 spec、Gateway 文档、TODO/Verification 和 Dual 实施报告。验收：版本、支持矩阵、配置/认证边界、隐私限制、静态检查及未运行验证一致。依赖：1.3–1.13。大小：小；难度：低。

旧项映射：原 1.1–1.2 拆为新 1.1–1.2；原 1.3–1.5 拆为新 1.3–1.7 与 1.12–1.13；原 1.6 拆为新 1.8–1.9；原 1.7 拆为新 1.10；原 1.8 拆为新 1.11；原 1.9 拆为新 1.6、1.12–1.13；原 1.10 对应新 1.14。工作项按依赖顺序执行；每项独立提交。实施开始后保持编号稳定，仅新增子项，不复用 ID。

## Acceptance

1. `@openai/agents` v0.18.0 默认 Responses provider 可通过标准 Gateway HTTP 配置使用本 FD 明确列出的兼容子集；不宣称完整 OpenAI API 或全部 Agents SDK 功能。
2. 非流式顺序 function tool 回合按标准 `function_call` / `function_call_output` 形状往返，工具始终由调用方执行；严格模式按工具 schema 验参。
3. 文本流与结构化输出只支持经证据确认的格式和组合；不支持的组合在 App Server 执行前失败。
4. Responses item、SSE 序号和终态映射不重复、不伪造成功；用量以当前 turn 计，不重复计算 thread 累计值。
5. 每个请求使用独立 thread/cwd；取消、崩溃、协议失同步、重启和 shutdown 都有界，未知执行不自动重试。
6. App Server 的本地工具、MCP、apps、plugins、hooks、web、子 agent 和非预期审批能力保持关闭；有效配置或 MCP 状态不能证明安全时，不启动 turn；服务端身份、模型权限、RPM、日额度和并发限制保持有效。
7. Prompt、工具参数/结果、API Key 和生成正文不进入日志或持久化审计；App Server rollout 留存和清理行为有版本化证据。

## TODO

- [x] 以 App Server 为唯一后端，整合 FD-022 的 Responses SDK 兼容范围；版本协议和映射/拒绝矩阵已录入。
- [x] 认证与 Codex home/MCP 工具隔离决策已记录：复用现有 home/login，关闭工具配置并在 turn 前拒绝活动或状态不明的 MCP。
- [x] 实施与 compile-only 验证已完成；运行时 SDK/backend 验证仍须单独授权精确命令。
- [ ] FD-022 已归档为 Deferred；后续在本 FD 交付时，通过受支持流程记录其被本 FD supersede 的处置。

## Verification
- Worker 静态追踪覆盖 JSONL 限额/ID 路由、initialize/config/MCP preflight、turn start/interrupt、终态和 usage、function call 历史映射、严格 schema 校验、SSE delta、配额审计、池回收及 shutdown。
- `python -B scripts/compile.py` 在最终源码后通过；未运行 Go tests、SDK/App Server 或其他运行时验证。App Server 双进程运行行为和本机认证仍需单独授权验证。
- App Server 使用现有 CODEX_HOME；每次借用 slot 均检查 effective config 与 MCP 状态，启用或未知状态 fail closed。turn/start 响应不明时保留 reservation 且不重试。
- 实现工作树：`.wt/FD-045`，分支 `feature/FD-045`；当前等待独立 Reviewer。

- Worker 静态证据：已查阅锁定版本的 Agents SDK、Codex App Server thread/turn/common 协议和 feature flags；当前实现已切换为 App Server JSON-RPC、严格 schema 和真实事件映射。
- Reviewer 独立审查：`docs/features/reviews/FD-045-review-r1.md`（Dual sidecar）；结论 `verification-passed`。Reviewer 未运行测试、compile、SDK/App Server 或其他运行时验证；对应风险仍见 Worker 报告。
- 版本化源码不证明 App Server 二进制行为。Worker 阶段已运行 `python -B scripts/compile.py` compile-only；未执行 Go tests、SDK 或 App Server。若要运行 SDK/App Server 黑盒验证，须先给出确切命令、范围、时长和风险并另行获批。
- 用户已在 `FD-045-000005-needs-decision` 后选择复用现有 Codex home，不复制认证材料；PM 以 `FD-045-000007-decision-recorded` 记录该决定，Planner 已 claim。实现必须通过 App Server 有效配置/MCP 状态 preflight 失败关闭。当前尚未发 `design-ready`，亦未创建 implementation worktree。
- 原 Planner handoff `FD-045-000002-design-requested` 被用户确认停止后，PM 以 `force-emit` 创建 `FD-045-000003-design-requested`；此命令跳过 status、needs-input、previous-handoff 等门槛，审计文件在 `.ai/fd/FD-045/operations/8785419cdb124f668c12bccd367cb52d.json`。新 Planner 已 claim；旧操作的 `agent_stopped:false` 表示命令本身未停止 Agent，重派依据为用户明确确认。
- 旧 Planner handoff `FD-045-000002-design-requested` 因其引用 revision 2 而活动文件为 revision 1，已由用户授权的 PM `cancel-event` 取消。该操作只取消收据，没有证明或停止原 Planner 会话；审计记录在 `.ai/fd/FD-045/operations/9b3aec7514a14b75b5f2a0f80b6a253f.json`。

## Sources

- `docs/features/archive/FD-022/FD-022_AGENT_GATEWAY_OPENAI_AGENTS_SDK.md` 及其归档报告：Agents SDK Responses 兼容范围、既有证据和未完成问题。
- `docs/features/FD-045_AGENT_GATEWAY_CODEX_APP_SERVER.md`：App Server 池、生命周期、进程清理与 Gateway 边界。
- `program/agent-gateway/protocol.go`
- `program/agent-gateway/runner.go`
- `program/agent-gateway/server.go`
- `program/agent-gateway/process_windows.go`
- `program/agent-gateway/process_linux.go`
- `openspec/specs/agent-proxy/spec.md`

**Consolidation decision (2026-10-09):** FD-045 is the canonical plan for both the App Server backend and the bounded OpenAI Responses compatibility scope previously tracked by FD-022. FD-022 remains historical evidence until its disposition can be recorded through the supported archive workflow.

**Completed:** 2026-10-09
