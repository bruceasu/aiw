# FD-030 独立审查 R2

<!-- aiw-data: FD-030-review-r2.json -->

## 发现

1. **P2 — `ls` 是未记录的公开别名。** `plugins/aiw-git/git-wt.py:504` 接受 `aiw git wt ls`，但同一文件的 `META.commands`、usage/help、README 和稳定规格只列出 `list`。这违反 `openspec/specs/command-help-consistency/spec.md` 的命令别名须与实际 dispatch 一致的要求，也超出 FD-030 列明的命令面。建议删除 `ls` 分支，或把别名明确加入 metadata、help、README 和规格；为保持 FD-030 已选的唯一命令面，优先删除别名。

## 审查范围与证据

- FD：revision 13，SHA-256 `64494c4d174ae31402591f64bb48fa66c670998a18b16ce2a63ce659bf839f6b`
- Reviewer handoff：`FD-030-000013-review-requested`；Reviewer session：`fd030-reviewer-20261008-a8c6d1`
- Worker：`FD-030-implementation-r2.md`；Worker session：`codex-fd030-worker-r2-20261007`
- Tester：`FD-030-test-report-r2.md`；Tester session：`fd030-tester-20261008-a71c9e`
- PM decision：`FD-030-test-decision-r2.md`，接受测试报告仅用于继续独立审查，并记录用户对本 FD 的测试豁免。
- 审查 commit：`c455294e431ef052e54cd6e08b4920c7bdf5e810`
- Diff base：`bf53cf1936d7553ee34b37a9b467c9d76059d848`
- R1 的三项测试迁移 finding 已静态核对：FD-027/FD-029 测试改经 aiw-git dispatcher；FD-029 现在断言成功清理终态；旧 Task-only 插件测试已删除并由 FD 冲突恢复用例替代。
- 当前 dispatch、清理和共享收据路径已静态检查。安全清理在删除前核验 worktree/branch 记录、clean 状态、父分支历史、单父 squash、来源 SHA 和 `.ai` 链接目标；`aiw-fd` 从 Git worktree 列表定位共享主工作区的 `.ai/fd`。
- Tester R2 报告为 0/20（0%），branch coverage 未测量；五个模块因 Python 导入路径错误未运行任何行为断言。PM 明确记录用户豁免后续测试。本审查遵循该豁免，不把它描述为测试通过。
- Junction 清理场景仍没有运行时证据；Tester 报告将其标为 blocked。用户豁免测试不豁免对应行为要求。

## 执行的命令

- `aiw fd show FD-030`
- `aiw fd resume FD-030`
- `aiw fd claim FD-030 FD-030-000013-review-requested --session fd030-reviewer-20261008-a8c6d1`
- `git status --short --branch`
- `git log --oneline --decorate -6`
- `git diff --stat develop...HEAD`
- `git diff --name-status develop...HEAD`
- `git diff develop...HEAD -- plugins/aiw-git/git-wt.py plugins/aiw-fd.py cmd/aiw/main.go README.md docs/usage/aiw-fd.md`
- `git diff develop...HEAD -- skills/fd-workflow/SKILL.md skills/fd-workflow/references/portable-operations.md skills/fd-workflow/roles/*.md skills/work-management.md AGENTS.md openspec/specs/fd-workflow/spec.md`
- `rg -n --glob '!docs/features/archive/**' --glob '!**/.git/**' 'plugins/aiw-wt\.py|aiw-wt\.py|\baiw wt\b|aiw git wt' README.md docs openspec skills tests plugins cmd`
- `git rev-parse HEAD`
- `git merge-base develop HEAD`

Reviewer 未运行测试、构建或其他运行时检查。没有把未运行的检查记为通过。

## 结论与剩余风险

结论：**changes-requested**。修复未记录的 `ls` 别名后，应对当前 revision 发起新一轮独立审查。

剩余风险：测试覆盖仍为 0/20，branch coverage 未测量。Windows junction/reparse 清理、冲突恢复、部分清理失败和 linked-worktree handoff 尚无运行时证据；这些风险已由 PM 记录并按用户决定豁免本次测试。
