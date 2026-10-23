# FD-104 独立实现审查

<!-- aiw-data: FD-104-review-r1.json -->

## 结论

**要求补充证据，发出 `changes-requested`。** 本 Reviewer 会话：`fd104-reviewer-20261023-c`。来源 handoff：`FD-104-000007-implementation-ready`。

## 审查范围

- AIW worktree `feature/FD-104`：实现提交 `395283341729b3a565059f7dca3ceb037f0094c6`，handoff 元数据提交 `f27c537afbece16ee8ef2b2d95049d4d26dfcded`；按实际提交差异审查，代码基线为 `3952833^`（FD 来源基线 `4ddd9931935084f69d561b889c6207dff2624322`）。
- Worker worktree `D:\03_projects\AI-tools\aiw-de\.wt\FD-104`：`405744ddabf0e3422900d42ac77cd76b6e1f05f3`，父提交 `405744d^`。
- 检查了 FD-104、APPROVED 的 ISSUE-00012、`openspec/specs/agent-gateway/spec.md`、Worker report 与两个仓库根级规则。

## 发现

1. **验收证据缺失：Agents SDK 到 gateway 的顺序函数调用尚未实际验证。** FD-104 Acceptance 要求一次有效 `function_call` 往返、工具恰好执行一次并最终得到 `TOOL_CALL_OK:42`。`workers/demo-python-with-openai-agents/fd104_test_tool_call.py` 对这些结果设置了明确断言，但 Worker report 明确记载该脚本、配置用例、Go 测试及端到端调用均未运行。因此目前只能确认验收脚本逻辑和实现路径，不能确认该 Acceptance 行为成立。最小后续动作：在获得该精确命令的单独测试授权后运行此脚本，记录所用 gateway/worker revision 与结果；本 Reviewer 未运行测试或网络调用。

## 已支持的静态审查结果

- `schema.go` 的 `parseSchema` 在根节点、`properties` 子节点和 `items` 节点递归解析时接受字符串 `title`；非字符串 `title` 返回错误。其它未知关键字仍进入错误分支，既有 type、required、additionalProperties、enum、const、数值/长度/数组边界校验保持在原逻辑中。协议层仍在 Runner 前调用 `validateFunctionSchema`。
- Worker `from_env()` 将 `AIW_OPENAI_PARALLEL_TOOL_CALLS` 解析进 `ApiRuntimeConfig`，缺省为 `true`；大小写不敏感地接受 true/false，空串及其它值抛出 RuntimeError。`_run()` 将该配置值传给 SDK `ModelSettings(parallel_tool_calls=...)`。Codex runtime 路径未被修改。README 记录接入 gateway 时需设为 false。
- 配置用例覆盖缺省、显式 true/false、无效输入；顺序验收脚本检查 SDK 参数 schema 的 title、单次 `(19, 23)` 工具调用及最终文本。原本地忽略的 `test_tool_call.py` 未覆盖，新增文件名为 `fd104_test_tool_call.py`。
- OpenSpec 对 string title 注解、非字符串和其它不支持关键字的拒绝，以及 gateway 不执行函数的边界描述与实现相符。
- 普通文本 Responses 行为未见被所审变更触及；运行时等价性未实测。

## 命令与未运行检查

只运行了静态读取命令：`aiw fd show`、`aiw fd claim`、`aiw fd --help`、`Get-Content`、`rg`、`git show`、`git log`、`git status`、`git rev-parse`、`git diff --name-status`、`git worktree list`。具体读取范围包括上述 FD/Issue/spec/report、Go schema/protocol、Python runtime/配置用例/验收脚本/README。

未运行 Python 配置用例、Go 测试、编译、格式化、静态检查器、Agents SDK 调用、gateway 端到端/网络检查或最终构建。Worker report 所列 compile-only 和 Python `compile()` 结果仅作为其报告记录，本 Reviewer 未重跑。

## 剩余风险

用户日志只包含 `invalid_request`，没有响应体，无法归因原始 400 是并行标志还是 schema title 导致；其它未支持请求字段也仍可能存在。以上属于已记录诊断限制，不改变本次实现审查范围。
