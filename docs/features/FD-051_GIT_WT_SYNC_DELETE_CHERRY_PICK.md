# FD-051: 扩展 `aiw git wt` 的分支操作

**Status:** Open  
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

- [x] 为 `delete`、`sync`、`cherry-pick` 实现参数处理、预检和 Git 操作。
- [x] 更新帮助/命令元数据及稳定 CLI 规格场景。
- [x] 静态检查变更并运行 Python compile-only 检查；未运行测试。

## Verification

- 已静态检查最终 diff、路径/分支校验及错误处理；`python -m py_compile src/plugins/aiw-git/git-wt.py` 与 `git diff --check` 均通过。
- 未执行测试或运行时命令。

## 风险与待确认

`delete` 使用非强制 worktree 移除；如果 Git 拒绝移除，命令保留 worktree 和分支。`sync` 与 `cherry-pick` 在 FD worktree 有未提交更改或 Git 操作进行中时拒绝启动。
