# FD-104：Python 经 agent-gateway 调用函数

**Status:** Complete
**Revision:** 25
**Priority:** Medium
**Evidence policy:** Dual

## Problem

OpenAI Agents SDK 为函数参数生成的 strict JSON Schema 含有 `title` 注解；当前 agent-gateway 的 schema 子集校验会把未识别的关键字拒绝。另一个已确认的调用方问题是 `process-python` 固定发送 `parallel_tool_calls=true`，而 agent-gateway 只支持顺序函数调用。用户提供的网关记录显示普通 Responses 请求成功，工具请求在执行前被拒绝。

受影响的调用方是当前 Python Agents SDK 示例，以及预期连接该网关的 `process-python` worker。目标是在不执行调用方本地函数、不放宽参数校验且不改变其他后端行为的情况下，完成顺序函数调用往返。

## Options and decision

1. 放宽或移除 strict schema 校验：拒绝。这样会削弱当前函数参数的类型、必填字段和边界验证。
2. 只接受 `title` 这类不改变参数语义的 SDK 注解，并继续拒绝不支持的 schema 关键字；对连接该网关的 Python 调用使用顺序工具调用。采用此方案。`title` 仅作为字符串注解忽略，不进入参数校验语义。

`process-python` 使用显式配置 `AIW_OPENAI_PARALLEL_TOOL_CALLS` 控制并行请求，缺省为 `true` 以保留现有行为。连接只支持顺序调用的 agent-gateway 时，部署配置必须设为 `false`。这是 Issue 用户确认的配置决策。

2026-11-08 用户调整验收方式：端到端验收使用 `.env` 中的 `DEEPSEEK_MODEL`，通过 agent-gateway 的 DeepSeek Responses 代理执行。运行程序可加载 `.env`，不得直接展示、记录或提交其中的值。DeepSeek 官方文档说明 `parallel_tool_calls` 会被忽略，因此本轮只用它验证 schema 接受、一次函数调用/结果往返和最终文本；不据此声称 DeepSeek 服务端遵守了该并行开关。

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
- [x] 1.2 将 `AIW_OPENAI_PARALLEL_TOOL_CALLS` 配置实现、README 说明和配置测试交付到 aiw-de `develop`；保留原 `true` 默认值，并保持旧位置参数构造兼容。
- [x] 1.3 将顺序往返验收脚本交付到 aiw-de `develop`，加入不记录凭据和原始请求体的逐轮请求/响应摘要。原 `test_tool_call.py` 位于仓库本地忽略目录，未覆盖或修改。
- [x] 1.4 更新 agent-gateway 稳定规格，记录允许的字符串 `title` 注解、拒绝非法/未知关键字，以及顺序调用边界。
- [x] 1.5 恢复异常归档的 FD-104 状态，保留旧归档快照，并通过 `recover-worker` 记录已停止的旧 Worker 会话。
- [x] 1.6 使用用户指定的 DeepSeek 模型完成顺序函数调用往返验收；根据脱敏轨迹确认单次工具调用和最终文本。

工作项编号在实现开始后保持不变。

## Acceptance

- Agents SDK 生成的 `add_numbers(a: int, b: int)` strict schema 在根节点及属性节点含字符串 `title` 时，gateway 接受工具定义；非字符串 `title`、未知关键字、非整数参数、缺少必填参数和额外参数仍拒绝。
- 顺序函数调用返回有效 `function_call`；SDK 恰好执行本地工具一次，提交结果后得到 `TOOL_CALL_OK:42`。
- 本轮在线验收从 `DEEPSEEK_MODEL` 读取模型配置，经本地代理转发至 DeepSeek Responses API；只记录来源标签和脱敏调用轨迹，不打印模型名或密钥。`parallel_tool_calls=false` 仅作为请求字段观察，不作为 DeepSeek 服务端顺序保证的验收依据。
- `process-python` 使用 `AIW_OPENAI_PARALLEL_TOOL_CALLS=false` 连接 gateway 时不要求并行工具调用；未设置时默认 `true`，保持其他后端的当前行为；无效配置 fail-fast。
- 普通文本 Responses 请求行为不变。

## Verification

- 新测试周期（用户明确授权）：精确命令为 `& (Join-Path $env:TEMP 'aiw-fd104-20261108\\Scripts\\python.exe') (Join-Path $env:TEMP 'aiw-fd104-20261108\\fd104_deepseek_runner.py')`，cwd 为 AIW 仓库根目录。临时 runner 用 `python-dotenv==1.2.4` 程序化加载 `C:\green\aiw\.env`，启动当前源码 gateway 的 DeepSeek proxy 并运行 `fd104_test_tool_call.py` 一次；最长 7 分钟、最多三轮外部 API 请求，结果不输出模型名或 `.env` 值。依赖版本：`openai-agents==0.24.0`、`openai==3.46.0`、`httpx==0.28.1`。测试基线：AIW `8e73956d3fe1ed12e97906d4959aad11be8b1aee`，aiw-de `82c2988`（验收脚本已提交，无 dirty diff）；临时 runner SHA-256 `38f88a1941726af79256f35819f7c38ab9d3e0c70132f0e464130e84bf56fadf`。若失败，需先有相关修正，最多允许一次重跑。
- Static review covered the Python worker setting, gateway schema parser, stable spec, and the acceptance script. The focused Go compile-only check and Python syntax compilation passed earlier in this FD cycle.
- The installed gateway binary used for the first configured run was revision `4ddd993`, older than this FD. It returned HTTP 400 `invalid_request` before execution; that result does not validate the FD source.
- The gateway was then started from this FD worktree using `C:\green\aiw\gateway.json`, and stopped gracefully after the attempt. The existing demo virtualenv ran `fd104_test_tool_call.py` against `gpt-6-luna`.
- Agents SDK raised `MaxTurnsExceeded` (exit code 1). No `TOOL_CALL_OK:42` result was observed. Gateway audit recorded three `/v1/responses` requests, all HTTP 200 with `execution_started=true` (request IDs: `resp_59a031771c0154bb55e67e8c5c3649e8`, `resp_8645f0db9614a1f57ff927167d4f4316`, and `resp_965122a7e2b50b5d4012fc13a06c6ad6`).
- Cause remains undetermined because audit records omit SDK turn output and tool-call payload. Do not mark this acceptance as passed. At that point, the runtime attempt budget was exhausted pending new user direction.
- After a new explicit user request, one more run used the same source gateway and config. It again ended with `MaxTurnsExceeded`; audit recorded three HTTP 200 backend executions (`resp_91af732deb4cf13637b3c0fc971f0b22`, `resp_f322da52ce89824edffaaf31b19d62fd`, `resp_a673ffdd5ca5b3b0c99762d89e2177e0`). Gateway shutdown completed. The repeated result still does not establish whether the SDK repeated tool calls or the round trip failed to terminate.
- Any further runtime attempt needs new explicit direction and a diagnostic change that captures the missing turn/tool evidence.
- No Go tests, Python unit tests, or other test suites were run. No dependency was installed or downloaded.
- Reviewer report `docs/features/archive/FD-104-pre-recovery-20261108/reviews/FD-104-review-r1.md` remains changes-requested because the required acceptance result is absent.
- On 2026-11-08, the user confirmed the old Worker stopped. The archived design and evidence were preserved under `docs/features/archive/FD-104-pre-recovery-20261108/`; the design was restored to the active path, and `aiw fd recover-worker` created handoff `FD-104-000011-work-requested`, claimed as `fd104-worker-20261108-recovery`.
- Before recovery, aiw-de `develop` lacked worker commit `405744d` and its acceptance script. The changes were recovered from the local object database and delivered as `cd5d632`.
- Ported the missing worker changes to aiw-de `develop` and committed them as `cd5d632`. The boolean field is appended with a default so `model_runtime.py`'s existing positional `ApiRuntimeConfig` construction remains compatible.
- The Python `compile()` check passed for `openai_api_runtime.py`, `test_parallel_tool_call_setting.py`, and `fd104_test_tool_call.py`. `git diff --check` passed in both repositories before commit.
- At the time of the r2 report, the focused unit test and diagnostic endpoint script had not run because the available interpreter lacked dependencies. This was superseded by the 2026-11-08 follow-up below.
- On 2026-11-08, created a temporary Python 3.12 virtual environment and installed `workers/process-python/requirements.txt` plus `httpx==0.28.1`; no project dependency manifest changed. Installed versions included `openai-agents==0.24.0` and `openai==3.46.0`.
- `python -m unittest test_parallel_tool_call_setting` first exposed an incomplete test fixture (`service_tier` missing). The fixture was corrected, and the same command passed all four tests; the scoped correction was committed in aiw-de as `857c280`.
- The diagnostic script first failed at import because the requirements file does not provide the directly imported `httpx` module. After installing `httpx` in the temporary environment, one attempt against the installed gateway returned HTTP 400 `invalid_request` before execution. Audit recorded `execution_started=false`; the installed executable is older than the FD source.
- Retried once against the current gateway source using `GOTOOLCHAIN=local GOPROXY=off go run . start --config C:\green\aiw\gateway.json`. The SDK sent `parallel_tool_calls=false`; the source gateway returned HTTP 200 and a valid `add_numbers(19, 23)` function call. After receiving the submitted output `42`, the backend returned another `function_call` on each of turns 2 and 3. The SDK executed the function three times and ended with `MaxTurnsExceeded` (exit code 1); no `TOOL_CALL_OK:42` was produced.
- Static diagnosis: `decodeRequest` validates matching `function_call`/`function_call_output` IDs and flattens both into the next prompt; `appInstructions` includes the caller's instructions and the tool-result policy; `executeApp` returns the backend's next decision as a new Responses `function_call`. The trace confirms this path returned 200, the SDK submitted output `42`, and the backend then emitted another call. This supports a model/backend response or model-dependent acceptance-oracle hypothesis; it does not confirm a gateway schema/sequential-setting defect or establish why the backend ignored the instruction. Do not mark endpoint acceptance as passed.
- Both gateway processes were stopped gracefully. No Go tests, broad Python suite, or additional endpoint retries were run during the prior test cycle. That cycle's budget was exhausted; the user later authorized a separate DeepSeek cycle, whose result appears below.
- Historical Reviewer report `docs/features/archive/FD-104-pre-recovery-20261108/reviews/FD-104-review-r1.md` remains changes-requested. Current review `docs/features/FD-104/reviews/FD-104-review-r2.md` has no blocking findings and emitted `FD-104-000022-verification-passed`.
- 新周期验收通过：临时 launcher 用 `python-dotenv==1.2.4` 加载 `C:\green\aiw\.env`，没有展示或保存其中的变量值。aiw-de 验收脚本提交为 `82c2988`；临时 runner SHA-256 为 `38f88a1941726af79256f35819f7c38ab9d3e0c70132f0e464130e84bf56fadf`。测试命令与 cwd 使用上方记录值，依赖版本为 `openai-agents==0.24.0`、`openai==3.46.0`、`httpx==0.28.1`。
- 当前 gateway 源码以 `GOTOOLCHAIN=local GOPROXY=off go run . start --config <temporary-config>` 在 `openai_proxy` 模式转发至 `https://api.deepseek.com`。两轮 `/v1/responses` 请求均 HTTP 200；第 1 轮返回唯一 `add_numbers(19, 23)` 调用；SDK 提交 `42` 后，第 2 轮返回 `TOOL_CALL_OK:42`。脚本结果：`model_source=DEEPSEEK_MODEL`、`title_annotations=true`、`tool_calls=[[19, 23]]`、`status=ok`、exit code 0。
- DeepSeek 文档说明 `parallel_tool_calls` 会被忽略；本结果验证 schema 和一次工具往返，不证明 DeepSeek 服务端执行了该开关。旧 gpt-6-luna 重复调用失败记录继续保留。gateway 优雅停止，临时代理配置随 launcher 退出清理。
- Reviewer r2 reviewed the adjusted acceptance and current code, found no blocking issues, and preserved the Codex-backed residual risk in the report.
- 本轮后续验收报告：`docs/features/FD-104/reports/FD-104-implementation-r3.md`，其 Dual sidecar 绑定本轮来源事件 `FD-104-000017-needs-decision`。

用户于 2026-11-08 决定改用 `.env` 中的 `DEEPSEEK_MODEL` 验收；新的 DeepSeek 测试周期已获明确授权。旧周期的运行预算不延续到本次新授权周期。

%% RISK: The prior Codex-backed `gpt-6-luna` run accepted the first function call and matching output but repeated calls until `MaxTurnsExceeded`. The user-directed DeepSeek proxy acceptance passed, but `openai_proxy` bypasses the Codex-backed schema/parser/app_runner path; this risk remains unresolved and is not hidden by the adjusted acceptance.

## TODO

- No required implementation or verification items remain. Preserve the residual risk above for any future Codex-backed follow-up.

## Sources

- Approved Issue: `ISSUE-00012`.
- Stable capability spec: `openspec/specs/agent-gateway/spec.md`.
- Gateway implementation: `src/programs/agent-gateway/schema.go` and `protocol.go` in AIW `develop` (FD-104 squash `271df81`).
- Python worker implementation and configuration coverage: aiw-de `develop`, commits `cd5d632` and `857c280`.
- Acceptance script: `workers/demo-python-with-openai-agents/fd104_test_tool_call.py` in aiw-de `develop`.
- DeepSeek Responses API and compatibility docs: [Responses API guide](https://api-docs.deepseek.com/guides/responses_api/) and [create response](https://api-docs.deepseek.com/api/create-response/).
- Prior independent review and blocker evidence: `docs/features/archive/FD-104-pre-recovery-20261108/`.

**Review policy:** Independent

**Completed:** 2026-11-08
