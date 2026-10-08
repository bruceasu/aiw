# FD-030 Worker 修复报告 R2

<!-- aiw-data: FD-030-implementation-r2.json -->

## 结果

已处理独立 Reviewer R1 的三项 changes-requested finding：活跃 FD worktree 测试不再引用已删除的 `plugins/aiw-wt.py`；FD-027/FD-029 测试经 `plugins/aiw-git/aiw-git.py wt` dispatcher 调用；FD-029 重复交付场景现在断言首次成功清理的终态，并从仍存在的父工作区确认登记已移除、交付提交保持不变。

原 `plugins/aiw_wt_test.py` 覆盖的是已移除的 Task-only `pull/apply_merge_resolution` API。该 fixture 已删除，并转换为根 `tests/test_fd030_conflict_recovery.py` 中的 FD 冲突恢复场景：父端 squash 冲突后在 FD worktree 解决，显式重试交付并核对清理、单父 squash 与 `FD-Source`。

## 改动范围

- `tests/test_fd027_wt_blackbox.py`：改用 aiw-git dispatcher；冲突恢复场景移至专用 FD 测试。
- `tests/test_fd029_wt_squash.py`：改用 aiw-git dispatcher；重复交付改断言成功清理后的终态。
- `tests/test_fd030_conflict_recovery.py`：新增当前 FD 接口的冲突恢复场景。
- `plugins/aiw_wt_test.py`：删除失效且属于旧 Task API 的测试 fixture。
- `docs/features/FD-030_FD_WORKTREE_AIW_GIT.md`：记录 Work Item 1.9、Reviewer R1 finding 修复和剩余风险。

## 静态证据与检查

- 执行 Python 内存编译与静态断言：编译四个根测试模块；断言它们不包含 `plugins/aiw-wt.py` 路径、包含 aiw-git 路由，且旧插件测试文件已移除。命令退出 0。
- 未运行测试。R1/R2 Tester 授权绑定 FD revision 6 与旧实现 digest，不适用于当前修订和迁移后的测试代码；需新的 revision-bound Planner 授权。
- 未运行构建、格式化、lint、网络或部署。测试场景的运行时行为仍未验证。

## 来源与剩余风险

- Source event：`FD-030-000009-changes-requested`。
- Reviewer R1：`docs/features/reviews/FD-030-review-r1.md`。
- Tester R1 报告为 0/20 场景覆盖、0 个行为测试执行，branch coverage 未测量；本次修复没有把该结果改写为通过。
- Windows junction、共享收据运行行为、冲突恢复及成功清理仍需要在新 Tester handoff 下评估，并按当前 revision 单独授权测试命令。
