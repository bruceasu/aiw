# FD-104 独立实现审查（r2）

<!-- aiw-data: FD-104-review-r2.json -->

## 结论

**通过当前用户调整后的 DeepSeek 验收，发出 `verification-passed`。** 本结论仅覆盖 FD 当前记录的 DeepSeek Responses 代理验收。它不表示 Codex-backed gateway 已完成 `TOOL_CALL_OK:42` 的单次往返；旧 gpt-6-luna 的 `MaxTurnsExceeded` 仍是未解决的历史结果。

Reviewer 会话：`fd104-review-deepseek-20261108`。来源事件：`FD-104-000021-review-requested`。

## 审查范围与修订

- FD `FD-104` 修订 21；主仓库当前 HEAD `56a0d21d161ec72cb99bf0e4cf0215089ed53765`。FD 文件在审查时已有未提交修改；本 Reviewer 未改动它。
- Gateway 实现：主仓库 `271df81d6395b329e4a9175f40c0e4664877cb8c`（相对首父提交检查 `schema.go` 改动）；检查当前 `server.go`、`protocol.go` 的实际调用分支。
- Worker 仓库当前 HEAD `82c29888bdb6d8195e492982bf33eb4e0e26f3d9`；检查 `cd5d632`（配置实现、单测、README）、`857c280`（测试夹具修正）和 `82c2988`（验收脚本使用 `DEEPSEEK_MODEL`）提交差异。
- 阅读当前 FD、批准的 `ISSUE-00012`、`openspec/specs/agent-gateway/spec.md`、Worker r2/r3 报告及 JSON sidecar，以及归档的 r1 changes-requested 报告及 JSON sidecar。

## 结果

- `schema.go` 仅在原有受限 schema parser 中接受字符串 `title`；非字符串标题拒绝，未知关键字仍按既有规则拒绝。原有字段、类型、必填项及值约束校验路径保持存在。主仓库 `271df81` 的差异与稳定规格一致。
- `process-python` 把 `AIW_OPENAI_PARALLEL_TOOL_CALLS` 解析为布尔值，缺省 `true`，仅接受大小写不敏感的 `true`/`false`，其它输入启动失败；配置传入 OpenAI 和 DeepSeek 的 Responses/Chat Completions model preferences。旧构造方式通过 dataclass 默认值保留兼容性。Worker 单测覆盖缺省、显式值、非法值及 Responses preferences 传递；Worker r2 报告记录 4 项测试通过，本 Reviewer 未重跑。
- 验收脚本源码检查确认它只输出 `model_source` 标签，不输出所选模型名；HTTP trace 摘要不包含请求体的 `model` 或认证头。错误处理仅输出异常类型。脚本会记录固定工具参数及最终文本摘要，不会读取或打印 `.env` 文件；本 Reviewer 没有读取 `.env`。
- Worker r3 报告记录的 DeepSeek 两轮 HTTP 200、一次 `add_numbers(19,23)`、提交 `42` 后得到 `TOOL_CALL_OK:42`、`status=ok` 和 exit code 0，与当前 FD 明确调整后的验收一致。按报告记录，DeepSeek 忽略 `parallel_tool_calls`；因此该结果只证明本次代理往返，不证明 DeepSeek 提供方遵守该参数。此项限制已在 FD 和报告中披露，没有作为提供方保证。
- **代理路径边界：** `server.go` 在 `backend_mode=openai_proxy` 分支中调用 `ReverseProxy` 后直接返回。该分支不会进入 Codex-backed `responses()`、`decodeRequest`、`schema.go` 或 `app_runner` 执行路径。因此 DeepSeek 结果不作为 `schema.go` 或 Codex runner 的运行时证据。另据 r2 报告，Codex-backed 运行的首轮 HTTP 200 返回有效 function call，后续匹配的 function-call output 被接受，但 backend 重复返回新 function call，最终 `MaxTurnsExceeded`，未得到 `TOOL_CALL_OK:42`。该失败被准确保留，没有被 DeepSeek 结果覆盖或归因成已解决。
- 归档 r1 的缺失验收证据 finding 原本要求 SDK→Codex-backed gateway 的完整往返。用户后来明确要求调整验收并测试 DeepSeek，当前 FD 已将验收改为 `DEEPSEEK_MODEL` 经本地代理的两轮往返。基于此明确调整，本次按当前 FD 验收通过；Codex-backed 单次完成仍作为未验证的边界风险记录，不把代理证据归给 parser/runner。

## Findings

无阻塞当前已调整验收的实现 finding。

## 命令与未运行检查

本 Reviewer 运行的命令均为本地只读检查：`aiw fd show FD-104`、`aiw fd --help`、`aiw fd emit --help`、`Get-Content`、`rg`、`git status`、`git log`、`git show`、`git rev-parse`。有一次 PowerShell 对 brace-expanded 路径的解析失败；修正为逐个文件参数后完成静态检查。没有读取、输出、复制或提交任何 `.env`/`.env.*` 内容或值。

未运行测试、编译、构建、格式化、静态分析器、网络请求或其他运行时命令；未读取临时 launcher。报告中的运行结果均引用 Worker r2/r3 记录，本 Reviewer 未复现。

## 剩余风险

- Codex-backed gateway 曾能解析并返回首轮 function call，也曾接收匹配的工具结果；其后续模型重复调用导致完整 `TOOL_CALL_OK:42` 验收失败。根因仍不确定，DeepSeek 代理结果不解决此问题。
- DeepSeek 对 `parallel_tool_calls` 的忽略意味着本次不能证明提供方的顺序调用保证。
- 本次静态审查与既有报告可支持当前 FD 调整后的验收结论；没有独立运行 Worker 单测或重新执行任何外部 API 验收。
