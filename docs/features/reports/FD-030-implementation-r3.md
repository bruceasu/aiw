# FD-030 Worker 修复报告 R3

<!-- aiw-data: FD-030-implementation-r3.json -->

## 结果

已处理 Reviewer R2 的 finding：`aiw git wt` 不再接受未记录的 `ls` 别名，列表操作只保留 metadata、usage/help 和规格中声明的 `list`。

## 改动范围

- `plugins/aiw-git/git-wt.py`：删除 `ls` dispatch 别名。
- `docs/features/FD-030_FD_WORKTREE_AIW_GIT.md`：新增 Work Item 1.10、验收条件和 R2/R3 记录。

## 验证与风险

- Python 内存编译：`python -c "from pathlib import Path; name='plugins/aiw-git/git-wt.py'; compile(Path(name).read_text(encoding='utf-8'), name, 'exec')"`。
- 静态确认命令 dispatch 仅保留 `list`，`usage()` 的声明命令一致。
- 未运行测试，遵循用户对 FD-030 的测试豁免。此 finding 属于静态命令面一致性修复，不增加行为验证声明。
- 继承 R2 的残余风险：自动清理、Windows junction、冲突恢复和共享 handoff 仍无运行时证据；Tester 覆盖为 0/20。

## 来源

- Source event：`FD-030-000014-changes-requested`。
- Reviewer R2：`docs/features/reviews/FD-030-review-r2.md`。
