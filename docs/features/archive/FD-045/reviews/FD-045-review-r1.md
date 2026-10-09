# FD-045 独立审查 r1

<!-- aiw-data: FD-045-review-r1.json -->

**结论：** Verification Passed  
**审查人会话：** `fd045-reviewer-20261010-a64b`  
**Reviewer claim：** `FD-045-000010-implementation-ready`  
**审查差异：** `develop` (`891156b`) 到 `feature/FD-045` (`da1e73a`)，并包含工作树中 AIW 更新的 FD 状态/revision 与索引；FD revision 10 的 SHA-256 与当前 receipt 一致。  
**Worker 报告：** `docs/features/reports/FD-045-implementation-r1.md` 及同名 JSON sidecar

## 发现

无阻断验收的问题。

## 验收审查

1. **Responses 兼容子集：通过静态检查。** `protocol.go` 保留请求输入和功能边界；`app_runner.go` 将文本、JSON object 与顺序 function call 映射到 Responses 输出。未声称覆盖完整 OpenAI API 或 Agents SDK。
2. **函数调用及严格 schema：通过静态检查。** `protocol.go` 重放匹配的调用/结果历史；`schema.go` 对声明的 schema 子集做复杂度限制和参数验证；`app_runner.go` 在工具名 allowlist 和参数检查通过后才构造 `function_call`。
3. **结构化输出与拒绝：通过静态检查。** 非支持选项在请求解码或 turn 前拒绝；`json_object` 在完成后验证 JSON object。未运行模型验证。
4. **Responses item、SSE、状态及 usage：通过静态检查。** `server.go` 只发送模型增量并在收尾时结束文本事件；`app_runner.go` 以当前 turn 的事件/usage 生成最终状态，不从 thread 累加用量。Worker 报告记录两次 compile-only 通过；Reviewer 未重跑。
5. **线程/进程生命周期：通过静态检查。** `app_runner.go` 每请求创建临时 cwd 和 ephemeral thread；`apppool.go` 限制并发、对异常进程淘汰且不重试未知执行；关闭路径会等待有限清理并传播错误。
6. **权限及 MCP 边界：通过静态检查。** `appserver.go` 显式传入 feature overrides、执行 initialize/config/MCP preflight，遇到启用或未知 MCP 状态时拒绝；turn 使用固定模型、`approvalPolicy=never` 和禁网只读 sandbox。有效运行时配置和安装版本行为未实际验证。
7. **隐私和 rollout 风险：证据与限制一致。** `childEnv` 白名单复用 `CODEX_HOME`，没有复制凭据或修改用户配置；报告明确说明 ephemeral thread 不能证明进程内存或 rollout 数据必然清除。

Worker Dual 报告的 JSON sidecar 通过 `aiw fd show-report FD-045 --last` 读取；Markdown 含唯一同名 `aiw-data` 引用，报告内的 Work Item/命令/未运行检查与提交和本次授权边界一致。FD Problem 保留原始问题描述；Verification 有 Worker 静态证据以及本审查结论。

## 命令与未运行检查

Reviewer 实际运行的只读/静态命令：

- `aiw fd show FD-045`
- `aiw fd resume FD-045`
- `aiw fd claim FD-045 FD-045-000010-implementation-ready --session fd045-reviewer-20261010-a64b`
- `git diff --stat develop...HEAD`
- `git diff --name-status develop...HEAD`
- `git diff --check develop`
- `aiw fd show-report FD-045 --last`
- `aiw fd emit --help`
- `aiw fd show-review --help`
- 针对实现、FD、spec 和报告字段的 `rg`/文件静态读取

`git diff --check develop` 报告 Worker Markdown 的两处行尾空格和 `appserver.go` 文件尾的额外空行；这些不影响运行行为或 FD 验收。

Reviewer 未运行测试、compile、最终构建、formatter/linter、Codex SDK、App Server 或真实账户请求。原因是仓库预算与本次授权明确禁止 Reviewer 运行这些命令。

## 剩余风险

Worker 报告中的 compile-only 结果由 Worker 报告，Reviewer 未独立重跑。已安装 Codex/App Server 版本、RPC 实际交互、MCP/feature override 生效情况、登录状态，以及进程/rollout 的实际数据清理仍需单独授权的运行时验证。Worker 已在 Dual 报告中列明这些风险。
