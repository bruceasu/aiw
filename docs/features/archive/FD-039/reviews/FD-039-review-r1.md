# FD-039 独立评审 R1

<!-- aiw-data: FD-039-review-r1.json -->

## 结论

`changes-requested`。默认 FD 路由、独立 `fd-test` 边界和旧 Tester CLI 兼容实现均有静态证据；FD-039 的 Verification 文本有一处控制字符损坏，需要修复后再完成评审。

## 发现

### F-1：Verification 中的 Skill 名称含控制字符

- 严重度：低；本轮文档验收阻塞项。
- 位置：[docs/features/FD-039_SEPARATE_FD_TEST_SKILL_FROM_FD_WORKFLOW.md]，第 58 行。
- 现状：`可选 d-test 隔离边界` 中包含 U+000C（form feed），Skill 名称无法按字面读取。
- 修复：将 U+000C 替换为字面字符 `$fd-test`，保留其余 Verification 内容。

## 验收核对

- 默认路由：新 FD 模板与 `aiw fd new` 模板不再包含 `Test policy: Independent`；CLI 的默认 `implementation-ready` 目标为 Reviewer。路由代码仍按显式旧 policy 进入 legacy Tester 分支。
- fd-test 边界：独立 Skill 要求显式调用；不创建 FD handoff、不改状态、不改变验收项；报告只记录测试事实，不能交 Reviewer、PM 或风险评估 Agent 评价。
- 默认评审链：Workflow、Implement、Reviewer 和 PM 角色说明已移除默认 Tester/测试报告决策链。Reviewer 只核对 FD、实现差异和实现证据。
- CLI 兼容：Tester 事件、状态路由、报告校验及 `refresh-tester` 命令仍保留；新 FD 默认不启用该 policy。
- Worker 报告记录了 Python 内存式 compile-only 检查通过。Reviewer 未重跑该命令，也未运行测试、覆盖率、最终构建、格式化或 lint。

## 审查范围与证据

- FD：`FD-039`，Revision 4。
- Source event：`FD-039-000004-implementation-ready`。
- Reviewer session：`fd039-reviewer-20261008-a41d`，与 Worker session 独立。
- Diff：`676e52d` 之后的提交 `20518be`、`b6e1f4e`、`883582f`、`78cba67` 及当前工作区差异。
- 静态核对了 FD、Worker 实施报告、FD Workflow/角色 Skill、`fd-test` Skill、`aiw fd` 模板和路由代码、使用说明及 `openspec/specs/fd-workflow/spec.md`。
- 未读取、引用或评价任何可选 fd-test 测试报告；本 FD 没有 Tester 报告。

## 实际执行的命令

- `aiw fd --help`
- `aiw fd show FD-039`
- `aiw fd claim FD-039 FD-039-000004-implementation-ready --session fd039-reviewer-20261008-a41d`
- `git status --short; git log -5 --oneline`
- `git diff --stat 676e52d; git diff --name-status 676e52d`
- `git diff 676e52d --`（按审查范围读取 Skill、模板、CLI、使用说明和 OpenSpec 文件）
- `rg -n -i -C 4 'Test policy|implementation-ready|test-report-ready|refresh-tester|Pending Test|test-accepted|test-rejected'`（限定于相关实现与文档）
- PowerShell 静态扫描 FD 文件中的控制字符，确认第 58 行含 U+000C。

## 未验证项与剩余风险

- 未运行测试、coverage、最终构建、lint 或格式化；未在运行时模拟 CLI 路由。当前结论依据源码和文档静态核对。
- `$fd-test` 场景生成与测试命令执行效果未在本次工作中运行验证。
