# FD-038 自动流程阻塞反馈

<!-- aiw-data: FD-038-blocker-20261008T151153Z-tester-removal.json -->

## 定位

- FD：`FD-038`
- 阶段与角色：Pending Test 路由恢复；PM
- 关联交接事件：`FD-038-000005-implementation-ready`
- 记录时间（含时区）：2026-10-08T15:11:53+00:00
- 记录者／会话：PM；`fd038-pm-20261008-7b2f6a`

## 阻塞事实

- 观察到的表现及停止位置：用户指出新流程已移除 Tester。PM 已停止并中断 Tester 子代理，不再运行测试或继续 Tester 报告/交接。用户指示前，已按旧流程授权并执行一次测试命令；命令报告 8 个用例中 7 个失败、1 个通过。失败摘要显示 PATH 中的 `C:\green\aiw\aiw.exe` 对 `show-report`/`show-review` 报 `invalid choice`，且 `fd show` 未显示 feature branch。完整 stdout/stderr 未留存；未生成 Tester 报告或 `test-report-ready` 事件。
- 已确认的根因：执行入口解析为 `C:\green\aiw\aiw.exe`，该入口没有暴露 FD-038 worktree 中新增的查询命令；本次命令未能验证目标 worktree 的 CLI 实现。失败不是实现通过证据，也不能作为交付测试结论。
- 已尝试的恢复及结果：1) Planner 审查并授权的精确命令已运行一次；2) 收到用户更新后中断 Tester，未重跑；3) 删除了未提交的临时 Tester 测试代码，未发现遗留 fixture；4) 检查当前路由：`aiw fd request-review` 仅接受 Pending Verification，`refresh-tester` 仅接受尚未领取的 pending Tester handoff。没有发现可将已 dispatched Tester handoff 安全迁移到 Reviewer 的 CLI 操作；未手工编辑 receipt。
- 当前状态：已解决。FD 为 Revision 6、Pending Verification；旧事件 `FD-038-000005-implementation-ready` 已标记取消，保留 Tester session `fd038-tester-20261009-7d32c91e`，并由 Reviewer 事件 `FD-038-000006-review-requested` 替代。
- 是否需要人工决策：否。用户已明确指示跳过测试、修正状态，并一次性授权手工迁移已派发的 Tester 收据。PM 决策见 `FD-038-test-waiver-r1.md`。

## 结果与改进

- 实际解决方案：新增 PM 测试豁免决策，将 FD 的 Test policy 设为 Waived 并转为 Pending Verification；按用户一次性授权把旧 Tester 收据标记 cancelled，保留原事件与 session 信息，再由 `request-review` 创建 `FD-038-000006-review-requested`。旧测试记录仍保留为历史事实；未创建 `test-accepted`，未再次运行测试。
- 解决时间（含时区）：以 `FD-038-000006-review-requested` 收据的 `created_at` 为准
- 剩余风险或下一步：Reviewer 交接处于 pending，等待独立复核。运行时行为仍未由有效测试验证；Reviewer 必须明确这一证据缺口。只有 Reviewer 通过后才能继续 Auto 的合并与归档阶段。
- 可复用的流程改进建议：更新 `fd-workflow` Auto 与 `aiw fd` 状态机文档/CLI，明确移除 Tester 后如何迁移已领取的 Pending Test handoff，并校验测试命令使用 FD worktree 的入口。该流程/API 变更超出 FD-038 当前查询功能范围，未实施。
- 建议处理状态：待评估；该建议不授权改写事件或跳过 Gate。
