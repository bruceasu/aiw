# FD-104：Python 经 agent-gateway 调用函数

**Status:** Open
**Revision:** 6
**Priority:** Medium  
**Evidence policy:** Dual

## Problem

OpenAI Agents SDK 为函数参数生成的 strict JSON Schema 含有 `title` 注解；当前 agent-gateway 的 schema 子集校验会把未识别的关键字拒绝。另一个已确认的调用方问题是 `process-python` 固定发送 `parallel_tool_calls=true`，而 agent-gateway 只支持顺序函数调用。用户提供的网关记录显示普通 Responses 请求成功，工具请求在执行前被拒绝。

受影响的调用方是当前 Python Agents SDK 示例，以及预期连接该网关的 `process-python` worker。目标是在不执行调用方本地函数、不放宽参数校验且不改变其他后端行为的情况下，完成顺序函数调用往返。

## Options and decision

1. 放宽或移除 strict schema 校验：拒绝。这样会削弱当前函数参数的类型、必填字段和边界验证。
2. 只接受 `title` 这类不改变参数语义的 SDK 注解，并继续拒绝不支持的 schema 关键字；对连接该网关的 Python 调用使用顺序工具调用。采用此方案。`title` 仅作为字符串注解忽略，不进入参数校验语义。

`process-python` 使用显式配置 `AIW_OPENAI_PARALLEL_TOOL_CALLS` 控制并行请求，缺省为 `true` 以保留现有行为。连接只支持顺序调用的 agent-gateway 时，部署配置必须设为 `false`。这是 Issue 用户确认的配置决策。

## Solution

- 在 agent-gateway 的 schema parser 中，仅在当前支持的各层 schema 节点接受字符串 `title` 注解；不将其纳入参数语义。继续执行现有类型、必填项、`additionalProperties`、枚举和值约束校验；非字符串 `title` 和其他不支持的关键字仍拒绝。
- 在 `process-python` 中读取 `AIW_OPENAI_PARALLEL_TOOL_CALLS`；未设置时缺省为 `true`，大小写不敏感的 `true`/`false` 分别启用/关闭并行；无效值在启动时以稳定配置错误失败。gateway 部署必须显式设为 `false`，其他后端无需改变配置。
- 保持 Responses 函数调用协议：网关返回 `function_call`，Agents SDK 执行本地函数，再提交匹配的调用结果；网关不执行 Python 函数。
- 更新 agent-gateway 稳定规格，记录 `title` 注解的有限兼容范围和不变的顺序调用边界。

## Scope

- Go `agent-gateway` strict function schema 兼容性。
- 外部 worker 仓库 `aiw-de` 中的 `process-python` 开关、单元覆盖及 `workers/process-python/README.md` 配置说明。实现前必须将 `D:\03_projects\AI-tools\aiw-de` 纳入可写工作区，或在该仓库建立经授权的专用 worktree。
- SDK schema 注解及顺序调用设置的聚焦兼容覆盖。
- 不包含并行或流式工具执行、任意 JSON Schema 支持、移除 strict 校验、网关执行 Python 函数、认证/配额/持久化策略变化。

## Work items

- [ ] 1.1 仅接受 `title` 字符串注解，并保持 gateway 对函数参数的 strict 校验。
- [ ] 1.2 在 `workers/process-python/openai_api_runtime.py` 增加 `AIW_OPENAI_PARALLEL_TOOL_CALLS`（缺省 `true`，只接受 `true`/`false`，无效值 fail-fast）；更新 `workers/process-python/README.md`，说明连接此 gateway 时必须设为 `false`。
- [ ] 1.3 在 worker 配置测试中覆盖默认值、显式 `false` 和非法值；将 `workers/demo-python-with-openai-agents/test_tool_call.py` 的 Agents SDK 顺序往返作为集成覆盖，校验带 `title` 的 schema、工具只执行一次和最终输出。
- [ ] 1.4 更新 agent-gateway 稳定规格，记录允许的字符串 `title` 注解、拒绝非法/未知关键字，以及顺序调用边界。

工作项编号在实现开始后保持不变。

## Acceptance

- Agents SDK 生成的 `add_numbers(a: int, b: int)` strict schema 在根节点及属性节点含字符串 `title` 时，gateway 接受工具定义；非字符串 `title`、未知关键字、非整数参数、缺少必填参数和额外参数仍拒绝。
- 顺序函数调用返回有效 `function_call`；SDK 恰好执行本地工具一次，提交结果后得到 `TOOL_CALL_OK:42`。
- `process-python` 使用 `AIW_OPENAI_PARALLEL_TOOL_CALLS=false` 连接 gateway 时不要求并行工具调用；未设置时默认 `true`，保持其他后端的当前行为；无效配置 fail-fast。
- 普通文本 Responses 请求行为不变。

## Verification

- Issue 请求聚焦兼容覆盖：在 worker 配置测试中覆盖默认/显式/非法值；Agents SDK demo 覆盖带 `title` schema 与一次顺序工具往返。Go 子树规则禁止新增或运行 Go 测试，因此不创建 Go 测试。实现后执行窄范围 compile-only 检查和静态调用链审查。
- 独立验收范围为一次 Agents SDK 到 gateway 的顺序工具往返。端到端命令需要访问本地 gateway 并触发 Codex backend；Tester 命令及其 revision 必须经代码/副作用检查后另行授权，当前不执行。配置单元测试如需执行，也先确认 worker repo 的局部规则和具体命令。有限预算：每个获授权命令先运行一次；只有相关代码/环境修正后才允许一次重跑。任何未运行的检查都不得报告为通过。
- 用户日志只提供通用 `error_code=invalid_request`，没有 HTTP 响应体。静态校验顺序先检查 `parallel_tool_calls=true`，再校验 strict schema。两项确定的不兼容都纳入方案，但无法断言该请求首先触发哪一项；该根因不确定性不改变修复范围，仍保留“还可能有其他不支持字段”的风险。

## Sources

- 已批准 Issue：`ISSUE-00012`。
- 稳定规格：[`agent-gateway`](../../../openspec/specs/agent-gateway/spec.md)。
- Gateway 源码基线：`4ddd9931935084f69d561b889c6207dff2624322`；`protocol.go` 先拒绝并行调用，再校验 strict schema；`schema.go` 拒绝未识别关键字。
- Worker 源码基线：`D:\03_projects\AI-tools\aiw-de` revision `31abb1b8b560eab5cc35e1c0675ec114461b7946`；`process-python/openai_api_runtime.py` 通过 `OPENAI_BASE_URL` 选择 endpoint，并固定设置 `parallel_tool_calls=True`。
- 用户提供的日志：gateway 在 `127.0.0.1:43127` 以 `mode=development backend_mode=codex` 运行；普通请求 `resp_f8350ab048cf0457de9ce782dabb2e6b` 返回 200；工具请求 `resp_40dc8c16d1a1cbfeefbffec10474370c` 返回 400、`error_code=invalid_request`、`execution_started=false`。
- Issue Plan 是批准前的捕获快照；批准与提升由 ISSUE 元数据及 FD-104 handoff 记录，新增日志证据和根因限制记录于本 FD。
