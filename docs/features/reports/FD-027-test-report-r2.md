# FD-027 独立测试报告，第 2 轮

<!-- aiw-data: FD-027-test-report-r2.json -->

## 结论

- 实现事件：`FD-027-000008-implementation-ready`；FD 修订 8；摘要 `e34b0e28ad0133604cd887aef1955facf4f21a798f9b74a7651bd20f28387618`。
- Tester 会话：`fd027-tester-20261006-r2-5f8c1d`。依据 FD 验收、R1 Reviewer 发现和 R2 Worker 报告设计黑盒场景，未阅读实现源码。
- 本轮获准执行一次 `python -B -m unittest tests.test_fd027_wt_blackbox -v`，**12 个测试通过、0 个失败**，耗时 7.412 秒。22 个适用行为场景中 17 个有本轮通过证据，需求场景覆盖率 **17/22（77.27%）**。业务代码分支覆盖率未测量。
- R1 首次 7/9 的夹具失败和修订后 9/9 的结果保留在 `FD-027-test-report-r1.md`。它们是历史证据，不计入本轮的执行与覆盖计数。

## 场景与证据

| ID | 可观察行为 | R2 结果 |
| --- | --- | --- |
| S01 | `wt add` 建立 FD 分支、worktree 及四项坐标 | 通过：`test_add_records_exact_worktree_coordinates` |
| S02 | `wt status` 接受已登记 FD | 通过：`test_status_and_commit_use_recorded_fd_tree` |
| S03 | `wt commit` 仅提交 FD 树的变更 | 通过：同上 |
| S04 | 拒绝 Task ID | 通过：`test_rejects_task_and_unknown_fd_without_creating_tree` |
| S05 | 拒绝未知 FD，且不建树 | 通过：同上 |
| S06 | parent 不干净时拒绝 add，且不建树 | 通过：`test_add_rejects_dirty_parent` |
| S07 | metadata 分支不符时拒绝 commit | 通过：`test_rejects_malformed_record_before_commit` |
| S08 | FD 树在错误分支时拒绝 merge | 通过：`test_rejects_wrong_branch_and_dirty_worktree_before_merge` |
| S09 | FD 树不干净时拒绝 merge | 通过：同上 |
| S10 | 无冲突时交付到记录的 parent | 通过：`test_successful_local_merge_targets_recorded_parent` |
| S11 | parent 冲突后 abort，并恢复原 HEAD 和干净工作树 | 通过：`test_conflict_moves_resolution_to_fd_tree_and_needs_explicit_retry` |
| S12 | 冲突转移至 FD 树的未合并索引 | 通过：同上 |
| S13 | 在 FD 树解决并提交后，仍需显式重试才交付 | 通过：同上 |
| S14 | FD 插件不再分派 `fd worktree` | 通过：`test_fd_plugin_does_not_dispatch_worktree` |
| S15 | 顶层帮助不展示 `fd worktree` | 未覆盖：未调用顶层帮助 |
| S16 | Shell 补全不展示 `fd worktree` | 未覆盖：未调用补全程序 |
| S17 | 非内容冲突 Git 错误不启动反向合并 | 未覆盖：未构造故障 |
| S18 | abort 或恢复检查失败时停止且保留现场 | 未覆盖：未注入故障 |
| S19 | 各路径不自动删除 worktree 或分支 | 未覆盖：未逐一路径断言保留 |
| S20 | 两条管理路径都未忽略时，add 提示 `.wt/` 与 `.ai/`，且不改变 Git 状态 | 通过：`test_add_rejects_when_both_management_paths_are_unignored` |
| S21 | 仅 `.wt/` 未忽略时，add 提示规则且不改变 Git 状态 | 通过：`test_add_rejects_when_worktree_path_is_unignored` |
| S22 | 仅 `.ai/` 未忽略时，add 提示规则且不改变 Git 状态 | 通过：`test_add_rejects_when_metadata_path_is_unignored` |

S20–S22 均核对 parent HEAD、干净状态、FD 分支、worktree 路径与 metadata 文件，确保拒绝发生在 Git 状态变更前。S15–S19 仍须由独立 Reviewer 结合静态证据评估；本报告不将它们写为通过。

## 命令与风险

Planner 授权：`.wt/FD-027/docs/features/reports/FD-027-test-authorization-r3.md`，绑定本轮实现事件、修订、摘要、Tester 会话、测试文件 SHA-256 `9867222CF909D97DB7026D541EA0C0671749FAE0FBA75DC521EBAF1BCE243D69` 和精确命令。

工作目录：`D:\03_projects\AI-tools\aiw\.wt\FD-027`。实际命令：`python -B -m unittest tests.test_fd027_wt_blackbox -v`。退出码 0；原始输出：

```text
test_add_records_exact_worktree_coordinates ... ok
test_add_rejects_dirty_parent ... ok
test_add_rejects_when_both_management_paths_are_unignored ... ok
test_add_rejects_when_metadata_path_is_unignored ... ok
test_add_rejects_when_worktree_path_is_unignored ... ok
test_conflict_moves_resolution_to_fd_tree_and_needs_explicit_retry ... ok
test_fd_plugin_does_not_dispatch_worktree ... ok
test_rejects_malformed_record_before_commit ... ok
test_rejects_task_and_unknown_fd_without_creating_tree ... ok
test_rejects_wrong_branch_and_dirty_worktree_before_merge ... ok
test_status_and_commit_use_recorded_fd_tree ... ok
test_successful_local_merge_targets_recorded_parent ... ok
Ran 12 tests in 7.412s
OK
```

测试仅在系统临时目录创建、提交并清理独立 Git 仓库和 worktree；隔离 Git 全局/系统配置、模板和 hooks；没有网络、下载、提权或发布副作用。未运行覆盖率、其他测试、构建、lint 或格式化。建议 PM 接受本轮有范围说明的证据，并将 5 个未覆盖场景及未测量的分支覆盖率交给 Reviewer 判断。
