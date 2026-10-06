# FD-029 独立测试报告，第 2 轮

<!-- aiw-data: FD-029-test-report-r2.json -->

## 结论

- 来源事件：`FD-029-000009-implementation-ready`；FD revision 9；摘要 `e11364415ebf4e1198d65cfa508a38948e6d5cb0ae46d0d12d71649e7c01ad05`；Tester 会话 `fd029-tester-r2-9f52347690eb`。
- 依据 [第 2 轮授权](FD-029-test-authorization-r2.md)，在 FD worktree 仅运行一次指定命令：**5 个测试通过、0 个失败，耗时 5.790 秒**。12 个适用行为场景中 10 个有本轮通过证据；需求场景覆盖率 **10/12（83.33%）**。
- 业务代码分支覆盖率**未测量**：授权命令未采集分支覆盖数据。归档后清理的 2 个场景未执行。本轮建议 PM 按已测范围与残余风险决定是否接受；不将未覆盖场景写为通过。

## 场景与证据

测试仅从 FD 验收与公开 CLI 契约编写；未阅读 `plugins/aiw-wt.py` 实现。所有测试在系统临时 Git 仓库执行，通过公开 `aiw wt` CLI 与 Git 提交图、工作区状态观察结果。

| ID | 可观察行为 | R2 结果 |
| --- | --- | --- |
| S01 | 多个 Work Item 提交仍保留在 FD 分支 | 通过：`test_multiple_work_items_become_one_single_parent_delivery` |
| S02 | 父分支只新增含 FD ID、`FD-Source` 的单父 squash 提交 | 通过：同上 |
| S03 | Work Item 提交不成为父分支祖先 | 通过：同上 |
| S04 | 只向 `workspace.json` 记录的父分支交付 | 通过：`test_delivery_uses_recorded_parent_branch` |
| S05 | 父工作区脏时在交付前拒绝 | 通过：`test_dirty_parent_or_fd_tree_is_rejected_before_delivery` |
| S06 | FD worktree 脏时在交付前拒绝 | 通过：同上 |
| S07 | 父或 FD 工作区处于错误分支时拒绝 | 通过：`test_wrong_parent_or_fd_branch_is_rejected` |
| S08 | squash 内容冲突后父 HEAD 不变且父工作区干净 | 通过：`test_conflict_restores_parent_and_requires_explicit_retry` |
| S09 | 内容冲突转入 FD worktree 供解决 | 通过：同上 |
| S10 | 解决并提交后仅显式重试才交付 | 通过：同上 |
| S11 | `FD-Source` 与当前 FD HEAD 不同时保留 worktree 与分支 | 未覆盖：归档清理不是本次 `aiw wt` 命令范围。 |
| S12 | 来源 SHA 一致且归档后清理 worktree 与分支 | 未覆盖：未执行归档生命周期。 |

纯静态要求另列：工作流、规格和文档中取消自动 rebase，以及 legacy Task 命令不变，应由独立 Reviewer 静态审查，未计入 12 个行为场景。

## 命令与风险

工作目录：`D:\03_projects\AI-tools\aiw\.wt\FD-029`。实际命令：`python -B -m unittest tests.test_fd029_wt_squash -v`；退出码 0。授权：`docs/features/reports/FD-029-test-authorization-r2.md`，绑定本次事件、revision 9、摘要、Tester 会话与精确命令。原始输出：

```text
test_conflict_restores_parent_and_requires_explicit_retry (tests.test_fd029_wt_squash.SquashDeliveryBlackBox.test_conflict_restores_parent_and_requires_explicit_retry) ... ok
test_delivery_uses_recorded_parent_branch (tests.test_fd029_wt_squash.SquashDeliveryBlackBox.test_delivery_uses_recorded_parent_branch) ... ok
test_dirty_parent_or_fd_tree_is_rejected_before_delivery (tests.test_fd029_wt_squash.SquashDeliveryBlackBox.test_dirty_parent_or_fd_tree_is_rejected_before_delivery) ... ok
test_multiple_work_items_become_one_single_parent_delivery (tests.test_fd029_wt_squash.SquashDeliveryBlackBox.test_multiple_work_items_become_one_single_parent_delivery) ... ok
test_wrong_parent_or_fd_branch_is_rejected (tests.test_fd029_wt_squash.SquashDeliveryBlackBox.test_wrong_parent_or_fd_branch_is_rejected) ... ok
Ran 5 tests in 5.790s
OK
```

测试代码：`tests/test_fd029_wt_squash.py`，位于 FD worktree。运行中仅在系统临时目录创建、提交并清理独立本地 Git 仓库/worktree；禁用系统及全局 Git 配置和 hooks；没有网络、下载、提权或发布产物。未运行第二次测试、覆盖率工具、构建、lint 或格式化。剩余风险是归档清理路径尚无运行证据、业务代码分支覆盖率未测量；R1 的 0/12 报告保留为历史证据，不计入本轮统计。
