# FD-051: 扩展 `aiw git wt` 的分支操作

**Status:** Complete
**Revision:** 5
**Priority:** Medium

## 背景

`aiw git wt` 当前提供 FD worktree 的创建、状态查看、提交和 squash 交付。实现过程中还需要从主工作区或指定分支同步更改、挑选指定提交，以及在 FD worktree 和对应分支不再需要时清理它们。

## 目标

为现有 `aiw git wt` 命令增加 `delete`、`sync` 和 `cherry-pick`，沿用登记的 FD workspace 元数据及 `feature/<fd-id>`、`.wt/<fd-id>` 约定，并在帮助信息和稳定 CLI 规格中说明行为。

## 非目标

- 不修改 `aiw fd` 生命周期或 `local-merge` 的交付策略。
- 不执行 fetch、push 或其他网络操作。
- 不改变既有子命令行为。

## 验收标准

1. `aiw git wt delete <fd-id>` 清理对应登记的 worktree 和 `feature/<fd-id>` 分支；任一约定资源不存在时跳过该项并输出明确的“不存在”信息。
2. `aiw git wt sync <fd-id> [branch]` 将主工作区分支合并到 FD worktree；省略 `branch` 时使用主工作区当前分支，提供时使用指定分支。
3. `aiw git wt cherry-pick <fd-id> <commit-id>` 在对应 FD worktree 中挑选指定提交。
4. 命令拒绝不匹配登记元数据的路径或分支，并遵循 Git 对脏 worktree、冲突和无效 ref 的错误处理；失败信息保留恢复所需上下文。
5. 更新命令元数据、用法帮助与 `openspec/specs/cli-and-plugins/spec.md` 中的稳定要求。

## Work Items

- [x] 1.1 Remove only the registered FD worktree and feature branch. (Size: S; Difficulty: Medium; Depends: none; Done when: only the conventional worktree and feature branch are targeted, and Git refusal preserves both.)
- [x] 1.2 Sync a selected local branch into the recorded FD worktree. (Size: S; Difficulty: Medium; Depends: none; Done when: source ref and target worktree are checked, and dirty/conflicted state is preserved.)
- [x] 1.3 Cherry-pick a resolved commit into the recorded FD worktree. (Size: S; Difficulty: Medium; Depends: none; Done when: commit and target are checked, and conflict recovery state remains available.)
- [x] 1.4 Update CLI help and stable CLI scenarios for the three operations. (Size: S; Difficulty: Low; Depends: 1.1-1.3; Done when: commands and failure/recovery behavior are documented.)
- [x] 1.5 Complete static review and the compile-only check. (Size: S; Difficulty: Low; Depends: 1.1-1.4; Done when: actual evidence is recorded; tests are not run.)

Historical mapping: the original first item maps to 1.1-1.3, the second to 1.4, and the third to 1.5. The original items were already checked; current evidence is recorded in the Worker report.

## Verification

- Static review: `src/plugins/aiw-git/git-wt.py` path, branch, clean-worktree, and Git recovery checks; compared with `openspec/specs/cli-and-plugins/spec.md`.
- Compile-only: `python -m py_compile src/plugins/aiw-git/git-wt.py` passed.
- Tests were not run.
- Independent Reviewer: passed for `FD-051-000004-implementation-ready`; see `docs/features/reviews/FD-051-review-r1.md`. The Reviewer inspected the baseline implementation because this FD branch has no source or stable-spec diff. Tests, builds, and compile-only were not rerun.
## 风险与待确认

`delete` 使用非强制 worktree 移除；如果 Git 拒绝移除，命令保留 worktree 和分支。`sync` 与 `cherry-pick` 在 FD worktree 有未提交更改或 Git 操作进行中时拒绝启动。

**Completed:** 2026-10-10
