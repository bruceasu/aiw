# FD-032 PM 测试决策：第 1 轮

<!-- aiw-data: FD-032-test-decision-r1.json -->

## 决策

三位独立评估者均投票 `repair`，因此本轮拒绝交付并交回 Worker。决定依据是关键行为仍缺少运行证据，而非测试通过率或场景覆盖率的固定门槛。Tester 的两次获授权运行分别停在测试夹具的证据格式与 PM 错误 claim 上；未证明生产实现失败，也未证明生产实现正确。

## 证据与风险

- Tester 报告：`docs/features/reports/FD-032-test-report-r1.md`。13 个目标场景均为 blocked；需求覆盖率 0%；分支覆盖率未测；已触达的目标行为测试数为 0，失败行为测试数为 0。
- 评估 A：`docs/features/reports/FD-032-test-risk-assessment-r1-a.md`，`repair`。核心多数路由和身份边界缺少运行证据，预计修正夹具并聚焦重测需数小时，结果仍不确定。
- 评估 B：`docs/features/reports/FD-032-test-risk-assessment-r1-b.md`，`repair`。影响本 FD 的 PM 和 Reviewer 交接可信度，预计一个短工作时段，重测结果未知。
- 评估 C：`docs/features/reports/FD-032-test-risk-assessment-r1-c.md`，`repair`。建议先完成已静态修正的夹具复核，估计约 1 至 2 小时；再排错与授权耗时未知。
- 多数：`accept-with-risk` 0 票，`repair` 3 票。

## Worker 后续

核对并完成黑箱夹具的 PM handoff 修正，提交新的实现报告和可审查的测试文件差异，再交给独立 Tester。新 Tester 的运行仍须按当前修订单独获 Planner 授权。保留本轮两次失败与 0% 覆盖事实，不将其改写为实现缺陷。

剩余风险：多数票路由、无效评估拒绝以及 Reviewer 身份隔离尚无运行证据；任何修正后的测试结果尚未知。
