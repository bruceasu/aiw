# FD-116: Git worktree helper with optional change transfer

**Status:** Complete
**Revision:** 2
**Priority:** Medium
**Review policy:** Self
**Evidence policy:** Markdown

## Problem

使用者在当前 worktree 新增或修改文件后，有时发现这些改动应放到独立 worktree。原生 `git worktree add` 只从提交检出内容，不会搬运当前工作区改动；手工 stash、创建 worktree、恢复改动容易漏掉未跟踪文件或误删恢复依据。

## Solution

在 `aiw-git` 增加自动发现的 `new-wt` Python 子命令。`aiw git new-wt <branch> <path>` 创建新分支和 worktree；`--existing` 使用已有本地分支；`--base REV` 选择新分支起点。`--move-changes` 显式将当前已跟踪和未跟踪改动 stash，再应用到新 worktree。只创建分支而不创建目录仍使用原生 `git switch -c`，本命令不包装分支切换，也不占用 `aiw git worktree` 名称。

先创建目标 worktree，再 stash 和恢复，避免 worktree 创建失败时改变当前目录。恢复成功后保留 stash 作为备份并显示其引用；不自动删除 stash。忽略文件不纳入移动。

## Scope

- 新增单个 `src/plugins/aiw-git/git-new-wt.py` 子命令，沿用 `META`、`main(argv)` 和脚本自动发现约定。
- 支持新建分支或使用已有本地分支创建 worktree；可显式将当前 tracked/untracked 改动迁入。
- 不修改 `git-wt.py` 的 FD 工作树流程，不包装 `git switch -c`，不搬运 ignored 文件，不删除 stash，不添加依赖。

## Work items

- [x] 1.1 实现并静态检查可发现的 worktree 子命令、参数校验和安全的改动迁移／恢复。
- [x] 1.2 完成一次 Python compile-only 检查，记录未运行的测试和剩余边界。

## Acceptance

- `aiw git new-wt <branch> <path>` 创建指定的新分支和 worktree；`--existing` 将已有本地分支检出到新 worktree；`--base` 仅用于新分支。
- 不提供 `--move-changes` 时保留当前目录改动；提供时连同未跟踪文件移动，成功后源目录干净且目标目录含改动，stash 仍保留。
- 分支、起点或目标路径无效，或 Git 创建 worktree 失败时报告原因，不 stash 当前改动。
- stash 应用失败时保留 stash，不自动删除部分恢复的目标 worktree，并打印核对与恢复依据。
- 不创建或切换源目录的分支来代替 worktree；ignored 文件不移动。

## Verification

- `git diff --check` passed before the final `--existing` branch correction.
- `python -c "from pathlib import Path; p=Path(r'src/plugins/aiw-git/git-new-wt.py'); compile(p.read_text(encoding='utf-8'), str(p), 'exec')"` passed after all source changes; it created no bytecode artifact.
- No tests or Git worktree/stash operation were run. `--move-changes` remains unverified against a real repository; the user should inspect both worktrees and the retained stash before dropping it.

## TODO

1.1 已实现；1.2 编译检查待执行。Git stash/worktree 的真实操作尚未验证，用户可在目标仓库试用。

## Sources

- Issue: none

Link relevant specs and evidence used for decisions.

**Completed:** 2026-11-15
