# FD-038 测试豁免与 Reviewer 路由决策

<!-- aiw-data: FD-038-test-waiver-r1.json -->

## 决策记录

- FD：`FD-038`，决策时 Revision 5，摘要：`3baf0eafa1afb625b6e3f8a4e59dc76f265c7ae2dc7294e3026cf27169521baf`
- 来源事件：`FD-038-000005-implementation-ready`
- 旧 Tester 会话：`fd038-tester-20261009-7d32c91e`
- 决策者／会话：PM；`fd038-pm-20261008-7b2f6a`
- 决策时间：2026-10-08T15:24:13Z

## 决策

用户明确指示“Skip test and fixed the status”，并选择一次性手工迁移已派发的 Tester 收据。对此 FD 豁免 Tester 阶段，不再运行测试；将 `Test policy` 改为 `Waived`，结束旧 Tester 交接，并把 FD 状态修正为 `Pending Verification` 后请求独立 Reviewer。不得生成 `test-accepted` 事件，也不得把既有命令结果描述为测试通过。

此前旧流程授权并执行过一次命令：`python -m unittest discover -s tests -p test_fd038_cli_blackbox.py -v`。据当时记录，该命令共执行 8 个用例，1 个通过、7 个失败，退出码为 1。失败显示 PATH 将 `aiw` 解析为 `C:\green\aiw\aiw.exe`，该入口不包含本 worktree 新增的命令；因此这次运行没有验证 FD-038 worktree 中的实现。没有保留完整 stdout/stderr，也没有 Tester 报告或 `test-report-ready` 事件。此结果既不是通过证据，也不足以判断目标实现运行时行为。

## 收据迁移与复核要求

一次性手工迁移仅针对 `FD-038-000005-implementation-ready`：保留事件 ID、来源、原 Tester session 与领取时间，将 dispatch 状态标为取消，并附上决策引用和原因；随后由受支持的 `request-review` 命令创建 Reviewer 事件。此人工迁移由用户在本会话明确授权，不构成通用流程或 CLI 能力。

迁移已完成：旧收据由 `FD-038-000006-review-requested` 替代；FD 当前为 Revision 6、`Pending Verification`，Reviewer 交接处于 `pending`。

Reviewer 应按 FD 验收标准独立检查实现和静态证据，并明确列出因豁免而未获得的运行时证据。任何新增测试都需另行授权；本决策不授权测试。

## 剩余风险

FD-038 的运行时行为未由有效测试验证。Reviewer 的通过结论只能基于其实际完成的独立审查，不应声称测试通过或覆盖率已满足。
