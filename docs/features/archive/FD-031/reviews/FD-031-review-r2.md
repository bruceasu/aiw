# FD-031 独立审查，第 2 轮

<!-- aiw-data: FD-031-review-r2.json -->

## 结论

**通过验证。** 本轮没有阻断发现。Reviewer outcome 累计 2 次：第 1 轮 `changes-requested` 1 次，本轮 `verification-passed` 1 次。来源事件为 `FD-031-000015-test-accepted`；Reviewer session 为 `fd031-reviewer-r1-20261008-69bc92e1`，与 Worker 和 Tester session 均不同。

## 审查依据

- 审查实现 HEAD `d345eb501d3f3d98583cf4e5460781e667f4115c`，相对 `develop` 的 merge base 为 `de304afce52a5956f807024d5ac526fde70a07da`；重点复查修复提交 `97c5de6f84957cafc7a8eb9ac8baf290d7f3a9bb`，并核对整个当前差异中 promote、元数据读取、FD 创建与 `recover-worker` 的调用路径及规格。
- r1 的 P2 已修复：`recover_worker` 在修改前读取索引原始字节；写入阶段的 `UnicodeDecodeError` 属于所捕获的 `Exception`，随后逐项删除新事件并恢复 FD、旧回执和索引。索引恢复调用 `atomic_bytes` 写回快照，不再读取损坏的其他 FD。每项回滚单独尝试，失败时报告步骤、异常类型和原因。索引原本不存在时删除新索引。普通成功路径仍在 FD、旧回执和索引写入后才把新事件设为 `pending`。
- Tester r3 的命令 `python -B -m unittest tests.test_fd031_promote_blackbox tests.test_fd031_recover_worker_blackbox -v` 与 Planner r3 授权的实现事件、修订 13、摘要、Tester session 和工作目录一致。报告记录一次执行，13/13 通过、19/21 场景覆盖（90.5%），分支覆盖率未测量。新增 S21 用例在临时项目中让另一编号 FD 含无效 UTF-8，命令失败后逐字节确认目标 FD、全部旧事件回执、索引和事件文件集合与调用前相同。
- PM r3 明确接纳 S13（FD CLI 不可用）、S20（归档 FD 拒绝）及分支覆盖率未测量的例外。静态代码中，`aiwCLI` 找不到 CLI 时返回错误；`recover_worker` 要求目标 FD 位于活动目录。未把这两项当作已运行测试。promote 的两个入口、审批校验、标题反转义、重复 Issue 关联保护、无 Task 写入及 `[promotion]` 不变由当前代码路径和已有黑箱证据支持。

## 限制与命令

S21 运行证据只覆盖索引解码失败，不代表所有磁盘故障均已注入。Reviewer 本轮仅执行 `aiw fd claim`、只读 PowerShell 文件读取、`rg`、`git status`、`git log`、`git rev-parse`、`git show` 和定向 `git diff`；没有运行测试、编译、构建、覆盖率、格式化、lint、vet 或网络命令。
