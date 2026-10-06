# FD-027 独立测试报告，第 1 轮

<!-- aiw-data: FD-027-test-report-r1.json -->

## 结论

- 实现事件：`FD-027-000004-implementation-ready`；FD 修订 4；摘要 `5ac7cadd8e2906f8706d214fe54034ea7114d158df14ed5fec064add2e27f5b7`。
- Tester 会话：`fd027-tester-20261006-9c3e7a`。仅按 FD 验收和公开 CLI 合同设计黑盒用例，未阅读实现源码。
- 最终有效执行：9 个测试通过、0 个失败，耗时 5.774 秒。适用行为场景 19 个，其中 14 个有通过证据，需求场景覆盖率 **14/19（73.68%）**。业务代码分支覆盖率未测量。
- 首次执行 9 个测试中 7 个通过、2 个失败。失败由临时仓库夹具缺少 `.ai/`、`.wt/` 忽略规则引起；修正夹具并取得第 2 次版本绑定授权后，唯一一次重跑全部通过。首次失败保留为历史执行记录，不能作为产品通过证据。

## 场景与证据

| ID | 可观察行为 | 最终证据 |
| --- | --- | --- |
| S01 | `wt add` 创建 `feature/FD-027` 和 `.wt/FD-027`，写四项坐标 | `test_add_records_exact_worktree_coordinates` 通过 |
| S02 | `wt status` 接受已登记的 FD | `test_status_and_commit_use_recorded_fd_tree` 通过 |
| S03 | `wt commit` 只提交 FD worktree 的变更 | 同上，通过 |
| S04 | Task ID 被拒绝 | `test_rejects_task_and_unknown_fd_without_creating_tree` 通过 |
| S05 | 未知 FD 被拒绝且不建树 | 同上，通过 |
| S06 | parent 不干净时 `wt add` 拒绝且不建树 | `test_add_rejects_dirty_parent` 通过 |
| S07 | metadata 中分支不符时拒绝 commit，未提交变更 | `test_rejects_malformed_record_before_commit` 通过 |
| S08 | FD worktree 在错误分支时拒绝 merge | `test_rejects_wrong_branch_and_dirty_worktree_before_merge` 通过 |
| S09 | FD worktree 不干净时拒绝 merge | 同上，通过 |
| S10 | 无冲突时 FD 分支合入记录的 parent | `test_successful_local_merge_targets_recorded_parent` 通过 |
| S11 | parent 内容冲突后 abort，parent HEAD 与工作树恢复 | `test_conflict_moves_resolution_to_fd_tree_and_needs_explicit_retry` 通过 |
| S12 | 冲突转移到 FD worktree，未合并索引留在该树 | 同上，通过 |
| S13 | FD 树解决并提交后，parent 仍未交付；显式重试才交付 | 同上，通过 |
| S14 | `aiw fd worktree` 不再由 FD 插件分派 | `test_fd_plugin_does_not_dispatch_worktree` 通过 |
| S15 | 顶层帮助不展示 `fd worktree` | 未覆盖：当前命令仅测试 FD 插件分派 |
| S16 | Shell 补全不展示 `fd worktree` | 未覆盖：当前命令没有调用补全程序 |
| S17 | 非内容冲突的 Git merge 错误不启动反向合并 | 未覆盖：未构造该故障 |
| S18 | abort 或恢复检查失败时保持现场并停止 | 未覆盖：未注入该故障 |
| S19 | 命令不自动删除 worktree 或分支 | 未覆盖：测试未逐一断言所有路径的保留状态 |

`S15`、`S16` 属于 CLI 集成要求，不能由插件分派测试代替。`S17`、`S18`、`S19` 留给静态审查与后续聚焦验证；本报告不把它们写成通过。

## 命令与风险

工作目录均为 `D:\03_projects\AI-tools\aiw\.wt\FD-027`，精确命令均为 `python -B -m unittest tests.test_fd027_wt_blackbox -v`。测试在系统临时目录创建独立 Git 仓库、worktree 和本地提交，结束时清理；隔离 Git 全局/系统配置、模板与 hooks；不调用网络、下载、提权或发布。授权记录：`FD-027-test-authorization-r1.md` 和 `FD-027-test-authorization-r2.md`，均绑定实现事件、修订、摘要和 Tester 会话。

首次执行退出码 1，5.004 秒。原始失败摘录：

```text
test_conflict_moves_resolution_to_fd_tree_and_needs_explicit_retry ... FAIL
test_successful_local_merge_targets_recorded_parent ... FAIL
AssertionError: '?? .ai/\n?? .wt/' != ''
AssertionError: 2 != 0 : wt: parent worktree has uncommitted changes; no merge was started
Ran 9 tests in 5.004s
FAILED (failures=2)
```

修正仅在临时仓库初始提交中加入忽略规则 `.ai/`、`.wt/`。修订后测试文件 SHA-256 为 `594DB3EAFC9E223286597516D34CD7F8362494D1E68DF71F0AB7FACD1A4933FC`。第二次执行退出码 0，原始摘要：

```text
test_add_records_exact_worktree_coordinates ... ok
test_add_rejects_dirty_parent ... ok
test_conflict_moves_resolution_to_fd_tree_and_needs_explicit_retry ... ok
test_fd_plugin_does_not_dispatch_worktree ... ok
test_rejects_malformed_record_before_commit ... ok
test_rejects_task_and_unknown_fd_without_creating_tree ... ok
test_rejects_wrong_branch_and_dirty_worktree_before_merge ... ok
test_status_and_commit_use_recorded_fd_tree ... ok
test_successful_local_merge_targets_recorded_parent ... ok
Ran 9 tests in 5.774s
OK
```

建议 PM 接受本轮有范围说明的测试证据，并把 5 个未覆盖场景及未测量的业务分支覆盖率交给 Reviewer 判断。未执行其他测试、覆盖率、构建、lint 或网络命令。
