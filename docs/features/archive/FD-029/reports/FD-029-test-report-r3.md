# FD-029 独立测试报告，第 3 轮

<!-- aiw-data: FD-029-test-report-r3.json -->

## 结论

- 来源事件：`FD-029-000015-implementation-ready`；FD revision 15；摘要 `8430f6b55b4dcdc8f41850215880546228dfb3826f77776b6275cccd37672c6b`；Tester 会话 `fd029-tester-r3-4e7b19c2`。
- 根据 [第 3 轮授权](FD-029-test-authorization-r3.md)，在 FD worktree 只运行一次指定命令：**6 个测试通过、0 个失败，耗时 7.117 秒**。新增的重复交付用例通过，父分支保持首笔 squash 提交且工作区干净。
- 13 个适用行为场景中，11 个有本轮运行证据；需求场景覆盖率为 **11/13，即 84.62%**。业务代码分支覆盖率未测量。归档后清理的 S11、S12 未执行，不计为通过。

## 场景与证据

测试经公开 `aiw wt` CLI 在系统临时 Git 仓库中观察提交图和工作区状态。R2 的通过记录仅为历史证据；本表依据 R3 单次执行结果。

| ID | 可观察行为 | R3 结果 |
| --- | --- | --- |
| S01 | 多个 Work Item 提交在 FD 分支保留 | 通过：`test_multiple_work_items_become_one_single_parent_delivery` |
| S02 | 父分支只新增含 FD ID 与 `FD-Source` 的单父 squash 提交 | 通过：同上 |
| S03 | Work Item 提交不成为父分支祖先 | 通过：同上 |
| S04 | 只向 `workspace.json` 记录的父分支交付 | 通过：`test_delivery_uses_recorded_parent_branch` |
| S05 | 父工作区脏时，在父分支改变前拒绝 | 通过：`test_dirty_parent_or_fd_tree_is_rejected_before_delivery` |
| S06 | FD worktree 脏时，在父分支改变前拒绝 | 通过：同上 |
| S07 | 父或 FD 工作区处于错误分支时拒绝 | 通过：`test_wrong_parent_or_fd_branch_is_rejected` |
| S08 | squash 内容冲突后父 HEAD 不变且父工作区干净 | 通过：`test_conflict_restores_parent_and_requires_explicit_retry` |
| S09 | 内容冲突转入 FD worktree 供解决 | 通过：同上 |
| S10 | 解决并提交冲突后仅显式重试才交付 | 通过：同上 |
| S11 | `FD-Source` 与当前 FD HEAD 不同时保留 worktree 与分支 | 未覆盖：没有执行归档清理 |
| S12 | 来源 SHA 相同且归档后清理 worktree 与分支 | 未覆盖：没有执行归档生命周期 |
| S13 | 对同一 FD HEAD 重复执行 `local-merge` 不产生第二笔提交 | 通过：`test_repeated_delivery_keeps_the_first_squash_commit` |

纯静态要求另列：取消自动 rebase 的技能、规格和文档，以及 legacy Task 命令的兼容性，应由独立 Reviewer 静态检查，未计入上述行为场景。

## 命令与风险

工作目录：`D:\03_projects\AI-tools\aiw\.wt\FD-029`。实际命令仅一次：`python -B -m unittest tests.test_fd029_wt_squash -v`。测试文件 SHA-256：`E7146235127D79DFDC6C020F6874AEF638E8037F4CF41C97EDB49441FA5D2A0F`。授权：`docs/features/reports/FD-029-test-authorization-r3.md`，绑定本轮事件、revision、摘要和 Tester 会话。原始输出：

```text
test_conflict_restores_parent_and_requires_explicit_retry (tests.test_fd029_wt_squash.SquashDeliveryBlackBox.test_conflict_restores_parent_and_requires_explicit_retry) ... ok
test_delivery_uses_recorded_parent_branch (tests.test_fd029_wt_squash.SquashDeliveryBlackBox.test_delivery_uses_recorded_parent_branch) ... ok
test_dirty_parent_or_fd_tree_is_rejected_before_delivery (tests.test_fd029_wt_squash.SquashDeliveryBlackBox.test_dirty_parent_or_fd_tree_is_rejected_before_delivery) ... ok
test_multiple_work_items_become_one_single_parent_delivery (tests.test_fd029_wt_squash.SquashDeliveryBlackBox.test_multiple_work_items_become_one_single_parent_delivery) ... ok
test_repeated_delivery_keeps_the_first_squash_commit (tests.test_fd029_wt_squash.SquashDeliveryBlackBox.test_repeated_delivery_keeps_the_first_squash_commit) ... ok
test_wrong_parent_or_fd_branch_is_rejected (tests.test_fd029_wt_squash.SquashDeliveryBlackBox.test_wrong_parent_or_fd_branch_is_rejected) ... ok

----------------------------------------------------------------------
Ran 6 tests in 7.117s

OK
```

测试仅在系统临时目录创建、提交并清理隔离 Git 仓库/worktree，禁用系统及全局 Git 配置和 hooks；没有网络、下载、提权或发布产物。未运行第二次测试、覆盖率工具、构建、lint 或格式化。剩余风险是 S11–S12 无本轮运行证据，以及业务代码分支覆盖率未测量。
