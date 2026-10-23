# FD-104：Python 经 agent-gateway 调用函数

**Status:** In Progress
**Revision:** 9
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

- [x] 1.1 仅接受 `title` 字符串注解，并保持 gateway 对函数参数的 strict 校验。
- [x] 1.2 在 `workers/process-python/openai_api_runtime.py` 增加 `AIW_OPENAI_PARALLEL_TOOL_CALLS`（缺省 `true`，只接受 `true`/`false`，无效值 fail-fast）；更新 `workers/process-python/README.md`，说明连接此 gateway 时必须设为 `false`。
- [x] 1.3 在 worker 配置测试中覆盖默认值、显式 `false` 和非法值；新增 `workers/demo-python-with-openai-agents/fd104_test_tool_call.py` 顺序往返验收脚本，校验带 `title` 的 schema、工具只执行一次和最终输出。原 `test_tool_call.py` 位于仓库本地忽略目录，未覆盖或修改。
- [x] 1.4 更新 agent-gateway 稳定规格，记录允许的字符串 `title` 注解、拒绝非法/未知关键字，以及顺序调用边界。

工作项编号在实现开始后保持不变。

## Acceptance

- Agents SDK 生成的 `add_numbers(a: int, b: int)` strict schema 在根节点及属性节点含字符串 `title` 时，gateway 接受工具定义；非字符串 `title`、未知关键字、非整数参数、缺少必填参数和额外参数仍拒绝。
- 顺序函数调用返回有效 `function_call`；SDK 恰好执行本地工具一次，提交结果后得到 `TOOL_CALL_OK:42`。
- `process-python` 使用 `AIW_OPENAI_PARALLEL_TOOL_CALLS=false` 连接 gateway 时不要求并行工具调用；未设置时默认 `true`，保持其他后端的当前行为；无效配置 fail-fast。
- 普通文本 Responses 请求行为不变。

## Verification

- Static review covered the Python worker setting, gateway schema parser, stable spec, and the acceptance script. The focused Go compile-only check and Python syntax compilation passed earlier in this FD cycle.
- The installed gateway binary used for the first configured run was revision `4ddd993`, older than this FD. It returned HTTP 400 `invalid_request` before execution; that result does not validate the FD source.
- The gateway was then started from this FD worktree using `C:\green\aiw\gateway.json`, and stopped gracefully after the attempt. The existing demo virtualenv ran `fd104_test_tool_call.py` against `gpt-6-luna`.
- Agents SDK raised `MaxTurnsExceeded` (exit code 1). No `TOOL_CALL_OK:42` result was observed. Gateway audit recorded three `/v1/responses` requests, all HTTP 200 with `execution_started=true` (request IDs: `resp_59a031771c0154bb55e67e8c5c3649e8`, `resp_8645f0db9614a1f57ff927167d4f4316`, and `resp_965122a7e2b50b5d4012fc13a06c6ad6`).
- Cause remains undetermined because audit records omit SDK turn output and tool-call payload. Do not mark this acceptance as passed. At that point, the runtime attempt budget was exhausted pending new user direction.
- After a new explicit user request, one more run used the same source gateway and config. It again ended with `MaxTurnsExceeded`; audit recorded three HTTP 200 backend executions (`resp_91af732deb4cf13637b3c0fc971f0b22`, `resp_f322da52ce89824edffaaf31b19d62fd`, `resp_a673ffdd5ca5b3b0c99762d89e2177e0`). Gateway shutdown completed. The repeated result still does not establish whether the SDK repeated tool calls or the round trip failed to terminate.
- Any further runtime attempt needs new explicit direction and a diagnostic change that captures the missing turn/tool evidence.
- No Go tests, Python unit tests, or other test suites were run. No dependency was installed or downloaded.
- Reviewer report `reviews/FD-104-review-r1.md` remains changes-requested because the required acceptance result is absent.

## Sources

- Approved Issue: `ISSUE-00012`.
- Stable capability spec: `openspec/specs/agent-gateway/spec.md`.
- Gateway implementation: `src/programs/agent-gateway/schema.go` and `protocol.go` in the FD-104 worktree.
- Python worker implementation: `workers/process-python/openai_api_runtime.py` in the aiw-de FD-104 worktree, commit `405744d`.
- Acceptance script: `workers/demo-python-with-openai-agents/fd104_test_tool_call.py` in the aiw-de FD-104 worktree.
- Runtime evidence: blocker report `reports/FD-104-blocker-20261022T161320Z.md` and its JSON sidecar.
