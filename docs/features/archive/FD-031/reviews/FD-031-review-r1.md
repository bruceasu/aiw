# FD-031 独立审查，第 1 轮

<!-- aiw-data: FD-031-review-r1.json -->

## 发现

1. **P2：`recover-worker` 的索引读取异常会绕过回滚。** `plugins/aiw-fd.py:809-813` 先写入新 FD 修订和已取消的旧回执，再调用 `update_index`。索引生成在 `plugins/aiw-fd.py:526` 读取所有 FD 文件；若另一份编号 FD 文件含无效 UTF-8，`read_text` 会抛出 `UnicodeDecodeError`。此异常不属于 `OSError` 或 `FDError`，因此跳过 `recover_worker` 的回滚块：旧回执仍为 `cancelled`，FD 修订已变化，新回执停在不可认领的 `preparing`。这违反 FD-031「失败时恢复旧 FD 和回执」的验收及 `openspec/specs/fd-workflow/spec.md` 的失败恢复要求。请让写入后的异常进入回滚路径，并确保回滚本身的异常被明确报告；可保存原索引内容以免在恢复时再次依赖同一批 FD 的解码。修复后由独立 Tester 对新修订取得授权并验证相应失败路径。

## 审查依据

- 来源事件：`FD-031-000011-test-accepted`；Reviewer session：`fd031-reviewer-r1-20261008-69bc92e1`，与 Worker `fd031-worker-repair-20261008-4e367cb0` 和 Tester `fd031-tester-r2-20261008-4433f97b` 不同。
- 审查实现 HEAD：`dd99dcbcca36fda805ba8bd107df644401c90687`；相对 `develop` 的 merge base：`de304afce52a5956f807024d5ac526fde70a07da`。核对了实际差异、FD 修订 11、Requirement 与 FD workflow 稳定规格、Worker r3 报告、Tester r2 报告及 PM r2 接纳决定。
- promote 路径静态核对了 `%q` 写入与 `strconv.Unquote` 读取、两个别名的命令分派、标题和 ID 到 `fd new` 的参数传递、审批拒绝、重复关联与旧 `[promotion]` 不写入。当前 Tester 对该修订执行 12 个黑箱测试，12 个均通过；历史 S06 与 S12 已覆盖。
- Tester 命令 `python -B -m unittest tests.test_fd031_promote_blackbox tests.test_fd031_recover_worker_blackbox -v` 与 Planner 授权 r2 的事件、FD 修订 9、摘要、Tester session 和工作目录一致。测试报告将 18/21（85.7%）场景标为覆盖，分支覆盖率未测量；PM 明确接纳该例外。S13、S20、S21 仍未运行，未计为通过。本发现属于 S21 中可静态推导的真实缺陷，PM 的测试例外不豁免行为要求。

## 结论与限制

**需要修改**。请修复上述失败回滚路径，再提交新的 Worker 实现和证据，重新完成独立测试与 PM 决策。本轮没有运行测试、编译、最终构建、覆盖率、格式化、lint、vet 或网络命令。实际命令仅涉及 `aiw fd claim`、`aiw fd emit --help`、只读 PowerShell 文件读取、`rg`、`git status`、`git log`、`git rev-parse`、`git merge-base` 及定向 `git diff`。本审查不能把未运行的文件系统失败注入描述为通过。
