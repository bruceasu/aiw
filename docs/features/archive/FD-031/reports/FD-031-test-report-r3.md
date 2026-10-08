# FD-031 独立测试报告，第 3 轮

<!-- aiw-data: FD-031-test-report-r3.json -->

## 结论

针对 FD 修订 13、事件 `FD-031-000013-implementation-ready`，独立 Tester 执行了 Planner 新授权的一条聚焦命令。**13 个黑箱测试全部通过**，用时 23.666 秒。需求可观察场景覆盖为 **19/21（90.5%）**；业务代码分支覆盖率未测量。

Reviewer r1 指出的 S21 索引读取异常已通过隔离夹具验证：在另一个编号 FD 文件写入无效 UTF-8 后，`recover-worker` 返回失败，目标 FD、旧回执、索引原始字节与事件文件集合均与调用前相同。建议 PM 接纳本轮行为证据，并明确记录 S13、S20 和未测量分支覆盖率的风险。

## 场景与证据

| ID | 可观察场景 | 状态 | 测试或缺口 |
| --- | --- | --- | --- |
| S01 | `issue --help` 列出 promote | 通过 | `test_both_help_entries_name_promote(issue)` |
| S02 | `req --help` 列出 promote | 通过 | `test_both_help_entries_name_promote(req)` |
| S03 | Issue 不存在时拒绝且不创建 FD | 通过 | `test_missing_issue_is_rejected` |
| S04 | Issue 状态未批准时拒绝 | 通过 | `test_unapproved_issue_is_rejected` |
| S05 | 批准记录未批准时拒绝 | 通过 | `test_issue_and_approval_must_both_be_approved` |
| S06 | `issue promote` 保留含引号标题及 Issue ID | 通过 | `test_issue_alias_creates_fd_and_planner_handoff` |
| S07 | `req promote` 兼容入口创建 FD | 通过 | `test_req_alias_creates_fd` |
| S08 | 创建 FD 后留有 Planner 待处理回执 | 通过 | `test_issue_alias_creates_fd_and_planner_handoff` |
| S09 | 旧 `[promotion]` 元数据字节不变 | 通过 | `test_issue_alias_creates_fd_and_planner_handoff`、`test_req_alias_creates_fd` |
| S10 | promote 不创建 Task | 通过 | promote 成功与拒绝用例 |
| S11 | 已关联 Issue 拒绝第二次创建 | 通过 | `test_duplicate_link_is_rejected` |
| S12 | FD 创建失败时返回错误且不留 FD 或 handoff | 通过 | `test_fd_creation_failure_is_reported` |
| S13 | FD CLI 不可用时返回错误 | 未覆盖 | 未在隔离夹具中注入 CLI 缺失 |
| S14 | recover-worker 审计取消旧回执并记录原因、后继 | 通过 | `test_recovery_creates_auditable_claimable_worker_event` |
| S15 | 新 Worker 回执绑定新修订与摘要且可由另一 session 认领 | 通过 | 同一用例 |
| S16 | 不匹配的旧 event 被拒绝且状态不变 | 通过 | `test_wrong_event_or_session_preserves_state` |
| S17 | 不匹配的旧 session 被拒绝且状态不变 | 通过 | 同一用例 |
| S18 | 空、换行、超过 500 字符的原因被拒绝 | 通过 | `test_invalid_reason_preserves_state` |
| S19 | 非 Worker 或非 dispatched 交接不能恢复 | 通过 | `test_non_dispatched_or_non_worker_handoff_cannot_be_recovered` |
| S20 | 归档 FD 不能恢复 | 未覆盖 | 未构造归档 FD 夹具 |
| S21 | 索引读取失败时恢复 FD、旧回执和索引且不留孤儿事件 | 通过 | `test_index_decode_failure_rolls_back_all_recovery_writes` |

S21 夹具使用公开 FD CLI 建立并认领 Worker 交接，先快照目标 FD、全部旧事件回执及索引字节，再在另一编号 FD 文件放入无效 UTF-8。恢复命令返回非零后，逐字节比较所有快照和事件文件集合；未出现新事件。该用例验证 Reviewer 指出的具体失败路径，不代表所有磁盘写入错误均已被模拟。文档与稳定规格的一致性、旧 session 确已停止的人工确认不计入运行时覆盖。

## 命令与风险

- 授权记录：`docs/features/reports/FD-031-test-authorization-r3.md`，绑定来源事件、修订 13、摘要 `7a5783b35df0209e193dbfc330bcb6af6204b00c7f5235465695ef8fdd16464a` 和 Tester session `fd031-tester-r2-20261008-4433f97b`。
- 实际命令：`python -B -m unittest tests.test_fd031_promote_blackbox tests.test_fd031_recover_worker_blackbox -v`；工作目录 `C:\Users\svictor\workspace\tools\ai-tools\aiw\.wt\FD-031`；退出码 0。原始摘要：`Ran 13 tests in 23.666s`、`OK`；13 个用例均显示 `ok`。
- 测试文件：`tests/test_fd031_promote_blackbox.py` 和 `tests/test_fd031_recover_worker_blackbox.py`。前者离线构建临时 CLI；两者在系统临时目录创建并清理隔离夹具，没有执行网络命令。
- 未执行其他测试、覆盖率、最终构建、格式化、lint、vet 或网络命令。分支覆盖率无法由 unittest 输出推断。
- 剩余风险：S13、S20 没有运行时证据；S21 仅覆盖索引解码失败注入，未覆盖每一种文件系统故障。PM 应明确决定是否接受这些缺口及未测量的分支覆盖率。

本报告来源事件：`FD-031-000013-implementation-ready`；独立 Tester session：`fd031-tester-r2-20261008-4433f97b`。
