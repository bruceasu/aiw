# FD-036 自动流程阻塞反馈

<!-- aiw-data: FD-036-blocker-20261008T080700Z-emit-reconcile.json -->

## 定位

- FD：`FD-036`
- 阶段与角色：Worker handoff／PM
- 关联交接事件：`FD-036-000005-work-requested`
- 记录时间（含时区）：2026-10-08 17:07:50+09:00（恢复结果补记）
- 记录者／会话：Codex PM／fd036-pm-20261008-7e91bc

## 阻塞事实

- 观察到的表现及停止位置：`aiw fd emit FD-036 implementation-ready --producer worker --artifact docs/features/reports/FD-036-implementation-r1.md --source-event FD-036-000005-work-requested` 未成功。CLI 报告全部 Work Items 必须完成或取消，并报告 FD 当前 Revision 6 与 Worker 收据绑定的 Revision 5 不一致。读取收据确认该 Worker 事件已由 `fd036-worker-20261008-8f3c2a` 领取。
- 已确认的根因：Worker 领取后更新 FD 进度和版本，导致当前内容摘要与已领取事件不一致；当时 Tester 场景项仍未完成。
- 已尝试的恢复及结果：读取 `aiw fd --help` 和 `aiw fd resume FD-036`；可用命令没有为已派发 Worker 收据提供直接摘要刷新操作，resume 明确报告版本不一致。未编辑收据。将 Work Item 1.8 的测试代码与 Tester 场景报告拆分后，决定取消重复描述后续 Pending Test 阶段的 1.8.2，保留并继续要求独立 Tester 阶段报告。
- 当前状态：已解决（2026-10-08 17:07:50+09:00）
- 是否需要人工决策：否；PM 确认本 Worker 实施结果已提交并结束，选择用受支持的 `recover-worker` 创建绑定当前 FD 的新 Worker handoff；由新 Worker 仅核对既有报告与 FD 状态后重新交接。

## 结果与改进

- 实际解决方案：PM 执行 `aiw fd recover-worker FD-036 --expected-event FD-036-000005-work-requested --expected-session fd036-worker-20261008-8f3c2a --reason "Worker implementation report is committed; PM confirms the session ended, and the FD revision was updated after its claim. Re-issue a digest-bound handoff for the current FD and have a new Worker reconcile the report before implementation-ready."`，生成 `FD-036-000008-work-requested`。新 Worker 会话 `fd036-worker-20261008-d72c01` 已领取；收据绑定 FD Revision 8 和当前摘要。没有手工编辑收据。
- 解决时间（含时区）：2026-10-08 17:07:50+09:00
- 剩余风险或下一步：当前 Worker 重新提交实现交接；Tester 场景清单和覆盖证据仍待独立 Tester。
- 可复用的流程改进建议：建议更新 `fd-workflow` Skill 的 Worker 说明，明确 FD progress revision/digest 更新与 claimed Worker 收据的协调步骤，避免完成实施后才发现 implementation-ready 无法匹配原事件。建议处理状态：待评估，须走正常 FD 与独立审查流程。
