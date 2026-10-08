# FD-030 独立测试报告 R1

<!-- aiw-data: FD-030-test-report-r1.json -->

## 结论

两次获批执行均在 unittest 的 `setUpClass` 创建临时 CLI 根目录时遇到 `WinError 5`，均报告 `Ran 0 tests`。没有构建 Go CLI、启动插件或运行 Git fixture，因此没有验收场景通过或失败；本报告将 3 个准备好的测试方法映射到的 6 个验收行为标为 blocked，其余场景标为 uncovered。推荐状态：**blocked**。

## 验收场景与覆盖

| ID | 可观察行为 | 状态 | 证据/未覆盖原因 |
| --- | --- | --- | --- |
| FD030-S01 | `aiw git wt add` 创建对应 FD worktree 和 feature 分支 | blocked | 准备于 `test_successful_delivery_cleans_worktree_and_branch_but_keeps_receipts`；两次均未进入测试方法 |
| FD030-S02 | `aiw git wt status` 报告管理状态 | blocked | 同一准备测试包含 status 调用；未执行 |
| FD030-S03 | `aiw git wt commit` 提交 worktree 中的变更 | blocked | 同一准备测试包含 commit 调用；未执行 |
| FD030-S04 | `local-merge` 成功交付单父 squash，`FD-Source` 匹配交付前 FD HEAD，随后清理 worktree/分支并保留 `.ai/fd` 收据 | blocked | 准备于成功交付测试；未执行 |
| FD030-S05 | `aiw git wt list` 列出 FD worktree | uncovered | 测试模块未准备此场景 |
| FD030-S06 | 独立 `aiw wt` 入口不可用 | blocked | 准备于 `test_standalone_aiw_wt_entrypoint_is_not_available`；未执行 |
| FD030-S07 | 未完成 squash 时保留 worktree 和分支 | blocked | 准备于 `test_failed_delivery_keeps_worktree_and_branch`；未执行 |
| FD030-S08 | 冲突恢复未完成时保留 worktree 和分支 | uncovered | 未准备/未执行 |
| FD030-S09 | `FD-Source` 与当前 FD HEAD 校验失败时拒绝清理并保留资源 | uncovered | 未准备/未执行 |
| FD030-S10 | 记录的 worktree 路径不匹配时拒绝清理 | uncovered | 未准备/未执行 |
| FD030-S11 | 记录的 feature 分支不匹配时拒绝清理 | uncovered | 未准备/未执行 |
| FD030-S12 | workspace metadata 不匹配时拒绝清理 | uncovered | 未准备/未执行 |
| FD030-S13 | FD 或父工作区不满足 clean 前置条件时拒绝清理 | uncovered | 未准备/未执行 |
| FD030-S14 | `.ai` junction 已验证指向主工作区时只移除链接本体，保留目标及 `.ai/fd` 收据 | uncovered | Windows junction 行为未准备/未执行 |
| FD030-S15 | 未知 reparse 类型或非预期 junction 目标时拒绝清理并保留资源 | uncovered | 未准备/未执行 |
| FD030-S16 | 部分清理失败不回滚已交付 squash，并报告已完成步骤和恢复信息 | uncovered | 未准备/未执行 |
| FD030-S17 | 从 linked worktree 执行 FD 命令时，handoff/锁使用主工作区共享 `.ai/fd` | uncovered | 未准备/未执行 |
| FD030-S18 | 从 linked worktree 执行时，FD 文档仍从当前项目树读取 | uncovered | 未准备/未执行 |
| FD030-S19 | README、help、稳定规格与文档一致描述唯一入口和清理行为 | uncovered | 本 Tester 未执行此静态一致性场景 |
| FD030-S20 | Tester、Reviewer 阶段及其提示保持角色分离与独立会话约束 | uncovered | 本 Tester 未执行此静态一致性场景 |

需求场景覆盖率为 **0/20（0%）**；其中 6 个验收行为由 3 个准备好的测试方法覆盖意图，但因 setup 错误而 blocked，14 个未准备或未执行。执行的行为测试为 **0**，通过 **0**，失败 **0**，未运行 **20**。分支覆盖率未测量：命令未通过 setUpClass，Go CLI 未构建，未进入实现代码。

## 命令与原始证据

两次命令均为 `python -B -m unittest tests.test_fd030_worktree_blackbox`，工作目录均为 `D:\03_projects\AI-tools\aiw\.wt\FD-030`，均绑定 FD revision 6、digest `5bb6333ca24b65f58d6fb708312166fecd7367108c54ecdcfdce1f8741474793` 和 Tester session `fd030-tester-20261007-7c56c2`。

1. Planner 授权 R1：`docs/features/reports/FD-030-test-authorization-r1.md` 及同名 JSON。运行在 setUpClass 创建 `C:\Users\suk\AppData\Local\Temp\fd030-aiw-runtime-16wpl8lr\aiw-root` 时失败，原始异常为 `PermissionError: [WinError 5] Access is denied`。`TemporaryDirectory` 清理也因 WinError 5 失败，路径为 `C:\Users\suk\AppData\Local\Temp\fd030-aiw-runtime-16wpl8lr`。结果：`Ran 0 tests in 0.030s`、`FAILED (errors=2)`。
2. Planner 授权 R2：`docs/features/reports/FD-030-test-authorization-r2.md` 及同名 JSON。将临时目录移入 FD worktree 后重跑，仍在创建 `D:\03_projects\AI-tools\aiw\.wt\FD-030\.fd030-aiw-runtime-e8uil59b\aiw-root` 时失败，原始异常同为 `PermissionError: [WinError 5] Access is denied`。清理也因 WinError 5 失败，路径为 `D:\03_projects\AI-tools\aiw\.wt\FD-030\.fd030-aiw-runtime-e8uil59b`。结果：`Ran 0 tests in 0.003s`、`FAILED (errors=2)`。

两次均未运行 `go build`、插件、Git 命令或验收测试。Go module cache、网络、系统 Git 配置和插件执行均未触及。原始输出表明两个临时目录的自动清理失败；可能存在残留目录，本 Tester 未探查或清理。

## 授权与剩余风险

- R1 和 R2 授权均只允许上述精确命令、当前 worktree cwd 和三项黑盒 CLI 场景；本报告不扩大其范围。
- R1 可能遗留 `C:\Users\suk\AppData\Local\Temp\fd030-aiw-runtime-16wpl8lr`；R2 可能遗留 `D:\03_projects\AI-tools\aiw\.wt\FD-030\.fd030-aiw-runtime-e8uil59b`。均未探查或清理。
- FD-030 的全部运行时行为仍无测试证据；Windows junction、linked-worktree handoff、冲突恢复、来源校验失败及部分清理失败均未覆盖。
- 未发出 `test-report-ready` handoff；需 PM 审阅本报告并决定后续流程。
