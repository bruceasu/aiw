# FD-031 独立测试报告，第 2 轮

<!-- aiw-data: FD-031-test-report-r2.json -->

## 结论

针对 FD 修订 9、事件 `FD-031-000009-implementation-ready`，独立 Tester 执行了 Planner 精确授权的一条命令。**12 个黑箱测试全部通过**，用时 22.988 秒。需求可观察场景覆盖为 **18/21（85.7%）**。业务代码分支覆盖率未测量；没有执行覆盖率命令。

历史第 1 轮失败的标题保真和 FD 创建失败注入，本轮用例均通过。建议 PM 接纳本轮行为测试证据，并就未覆盖场景及未测量的分支覆盖率作明确风险决定。本报告只证明运行过的场景，不把静态复核或未运行场景记为通过。

## 场景与证据

| ID | 可观察场景 | 状态 | 测试或缺口 |
| --- | --- | --- | --- |
| S01 | `issue --help` 列出 promote | 通过 | `test_both_help_entries_name_promote(issue)` |
| S02 | `req --help` 列出 promote | 通过 | `test_both_help_entries_name_promote(req)` |
| S03 | Issue 不存在时拒绝且不创建 FD | 通过 | `test_missing_issue_is_rejected` |
| S04 | Issue 状态未批准时拒绝 | 通过 | `test_unapproved_issue_is_rejected` |
| S05 | 批准记录未批准时拒绝 | 通过 | `test_issue_and_approval_must_both_be_approved` |
| S06 | `issue promote` 保留含引号的原始标题及 Issue ID | 通过 | `test_issue_alias_creates_fd_and_planner_handoff` |
| S07 | `req promote` 兼容入口创建 FD | 通过 | `test_req_alias_creates_fd` |
| S08 | 创建 FD 后留有 Planner 待处理回执 | 通过 | `test_issue_alias_creates_fd_and_planner_handoff` 检查回执 JSON |
| S09 | 旧 `[promotion]` 元数据字节不变 | 通过 | `test_issue_alias_creates_fd_and_planner_handoff`、`test_req_alias_creates_fd` |
| S10 | promote 不创建 Task | 通过 | promote 成功与拒绝用例检查临时项目 |
| S11 | 已关联 Issue 拒绝第二次创建 | 通过 | `test_duplicate_link_is_rejected` |
| S12 | FD 创建失败时返回错误且不留 FD 或 handoff | 通过 | `test_fd_creation_failure_is_reported`，用模板路径目录阻止写入 |
| S13 | FD CLI 不可用时返回错误 | 未覆盖 | 未在隔离夹具中注入 CLI 缺失 |
| S14 | recover-worker 审计取消旧回执并记录原因、后继 | 通过 | `test_recovery_creates_auditable_claimable_worker_event` |
| S15 | 新 Worker 回执绑定新修订与摘要并可被另一 session 认领 | 通过 | 同一用例核对 digest、状态、Work Item、session |
| S16 | 不匹配的旧 event 被拒绝且状态不变 | 通过 | `test_wrong_event_or_session_preserves_state` |
| S17 | 不匹配的旧 session 被拒绝且状态不变 | 通过 | `test_wrong_event_or_session_preserves_state` |
| S18 | 空、换行、超过 500 字符的原因被拒绝 | 通过 | `test_invalid_reason_preserves_state` |
| S19 | 非 Worker 或非 dispatched 交接不能恢复 | 通过 | `test_non_dispatched_or_non_worker_handoff_cannot_be_recovered` |
| S20 | 归档 FD 不能恢复 | 未覆盖 | 未构造归档 FD 夹具 |
| S21 | 写入失败时回滚旧 FD/回执且无可认领孤儿事件 | 未覆盖 | 未执行文件系统失败注入；仅覆盖普通参数拒绝的无副作用行为 |

文档与稳定规格的一致性、旧 session 确已停止的人工确认属于静态或流程证据，不计入运行时覆盖。S14 至 S19 使用公开 FD CLI、隔离 Git 项目与回执观察，没有读取实现源码设计用例。

## 命令与风险

- 授权记录：`docs/features/reports/FD-031-test-authorization-r2.md`，绑定本事件、修订 9、摘要 `fbcbe2d498663627baaac326e08e59e425184ea1e7adb74f823d7253f4cc5a63` 和 Tester session `fd031-tester-r2-20261008-4433f97b`。
- 实际命令：`python -B -m unittest tests.test_fd031_promote_blackbox tests.test_fd031_recover_worker_blackbox -v`；工作目录 `C:\Users\svictor\workspace\tools\ai-tools\aiw\.wt\FD-031`；退出码 0。原始摘要：`Ran 12 tests in 22.988s`、`OK`；12 个用例均显示 `ok`。
- 测试文件：`tests/test_fd031_promote_blackbox.py` 和 `tests/test_fd031_recover_worker_blackbox.py`。前者离线构建临时 CLI；两者仅在系统临时目录生成和清理夹具，后者直接调用隔离复制的公开 FD 插件命令。未观察到仓库业务文件被测试命令修改。
- 未执行其他测试、覆盖率、最终构建、格式化、lint、vet 或网络命令。分支覆盖率无法从本次 unittest 输出推断。
- 剩余风险：S13、S20、S21 没有运行时证据；尤其恢复过程中文件写入失败后的原子回滚仍需单独故障注入方可确认。PM 应记录是否接受这些缺口及未测量分支覆盖率。

本报告来源事件：`FD-031-000009-implementation-ready`；独立 Tester session：`fd031-tester-r2-20261008-4433f97b`。
