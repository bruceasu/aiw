# FD-030 独立审查 R1

<!-- aiw-data: FD-030-review-r1.json -->

## 发现

1. **P2 — FD-027 黑盒测试引用已删除的插件。** **tests/test_fd027_wt_blackbox.py:12** 将 WT_PLUGIN 指向 plugins/aiw-wt.py；该实现已迁移至 plugins/aiw-git/git-wt.py，旧文件已删除。此测试不能再通过旧目标验证当前公开入口。请将 fixture 改为经 aiw-git dispatcher 调用 aiw git wt，并保留在根 tests/ 下。

2. **P2 — FD-029 squash 测试仍调用已删除的插件，并从已清理的目录重试。** **tests/test_fd029_wt_squash.py:12** 指向已删除的 plugins/aiw-wt.py；测试在第 121 行首次成功调用 local-merge 后，第 123 行还从原 tree 路径启动第二次调用。FD-030 的成功清理会删除该 worktree，因此原调用路径与新生命周期不符。请迁移到 aiw git wt，并将重复交付场景改为模拟清理未完成但登记仍在的重试，或改测成功清理后的终态。

3. **P2 — 插件目录中的冲突恢复测试也引用已删除文件。** **plugins/aiw_wt_test.py:10** 将插件路径设为同目录的 aiw-wt.py。目标文件已删除，该 fixture 无法导入待测插件。请迁移该用例到根 tests/，通过 aiw git wt 测试当前接口；若其行为不适用于 FD worktree，应改为对应的 FD 场景。

## 审查范围与证据

- FD：修订版 8，SHA-256 0854b89fb86af2c365dac7890ebcc0f3eb4e5156c619c4e11f5a960435ee0c32。
- Reviewer handoff：FD-030-000008-test-accepted；Reviewer session：fd030-reviewer-20261007-29a71c。领取前确认事件为 pending，绑定摘要与当前 FD 相同。
- 审查差异：基线 bf53cf1 至分支 HEAD ea07314；主要实现提交 155158a。
- 已静态检查 plugins/aiw-git/git-wt.py 的入口、metadata 校验、squash 来源校验、清理顺序和错误保留路径；检查 plugins/aiw-fd.py 的共享 .ai/fd 根解析；并搜索活跃 README、docs、spec、Skills 与测试中的旧入口引用。
- 现有活跃文档和稳定规格使用 aiw git wt。三处测试引用属于明确迁移遗漏；FD Verification 已记录相同风险。

## 独立测试证据核对

Tester 报告列出 20 个适用场景，覆盖 0/20（0%），行为测试执行数为 0，branch coverage 未测量。两次获批的命令 python -B -m unittest tests.test_fd030_worktree_blackbox 均在 setUpClass 创建临时目录时因 PermissionError [WinError 5] 退出，结果均为 Ran 0 tests。这不是行为测试失败，也不是行为测试通过；报告将 20 个场景记为未运行或未覆盖。

两份 Planner 授权都绑定 Worker event FD-030-000006-implementation-ready、修订版 6、摘要 5bb6333ca24b65f58d6fb708312166fecd7367108c54ecdcfdce1f8741474793、Tester session fd030-tester-20261007-7c56c2 和相同精确命令。PM R1 决策绑定 Tester event FD-030-000007-test-report-ready 与修订版 7 / 摘要 8f445bb96e7c9c8c4068b0d758a9a2178b01d3127e026ee60c25c363ceed69ae，只接受 0% 覆盖例外以继续静态审查。该例外没有使测试变为通过，也没有解除上述代码与测试资产问题。

## 静态核对结论

FD worktree 子命令由 plugins/aiw-git/git-wt.py 提供，旧独立插件已删除。清理函数在删除 worktree 和分支前会核对干净状态、来源 HEAD、父分支历史、单父提交与 FD-Source；Windows reparse link 会先核对类型和目标。FD 收据路径从 Git worktree 列表取主工作区，并写入其 .ai/fd。上述仅是代码路径静态证据。Tester 未运行任何行为测试；Windows junction、清理失败、冲突恢复和 linked-worktree 收据行为均无运行时证据，测试失败后可能遗留临时目录；本 Reviewer 未检查或清理它们。

## Reviewer 实际执行的命令

- aiw fd resume FD-030：报告最新 Reviewer handoff pending。
- aiw fd claim FD-030 FD-030-000008-test-accepted --session fd030-reviewer-20261007-29a71c：领取指定事件成功。
- aiw fd --help 与 aiw fd emit --help：核对 emit 接口。
- git status --short --branch、git log -5 --oneline --decorate、git rev-parse HEAD：确认分支、提交和工作区状态。
- git diff --stat bf53cf1936d7553ee34b37a9b467c9d76059d848 HEAD、git diff --name-status bf53cf1936d7553ee34b37a9b467c9d76059d848 HEAD、git diff -M bf53cf1936d7553ee34b37a9b467c9d76059d848 HEAD -- plugins/aiw-wt.py plugins/aiw-git/git-wt.py：检查实现差异。
- rg -n --glob '!docs/features/archive/**' --glob '!**/.git/**' 'plugins/aiw-wt\.py|aiw-wt\.py|\baiw wt\b|aiw git wt|git wt' README.md docs openspec skills tests plugins cmd：查找旧接口引用。
- Get-FileHash docs/features/FD-030_FD_WORKTREE_AIW_GIT.md -Algorithm SHA256：结果与 event 8 摘要一致。

Tester 命令是其报告记录的证据；本 Reviewer 未运行测试、构建或其他运行时验证。结论为 **changes-requested**，原因是三项可修复的旧测试引用及 FD-029 重试场景未随接口和清理生命周期迁移。