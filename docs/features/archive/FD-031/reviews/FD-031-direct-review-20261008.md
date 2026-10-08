# FD-031 独立直接审查

<!-- aiw-data: FD-031-direct-review-20261008.json -->

## 发现

1. **P1：Issue 标题中的引号不能原样进入 FD。** `internal/issue/store.go:204` 用 Go `%q` 写入标题，因此 `批准的 Issue "alpha beta"` 在 TOML 中包含转义的 `\"`。`internal/issue/store.go:167,407` 的 `unquote` 只裁掉两端的引号和反斜杠，不还原字符串内部的 `\"`；`cmd/aiw-req/issue-promote.go:31` 又直接把 `meta.Title` 作为 `fd new` 的标题参数。于是生成的 FD 标题会保留多余反斜杠，违反 FD-031 的标题保真验收和 `openspec/specs/requirement/spec.md` 的 promote 约定。Tester 的 S06 标题断言失败与此静态路径吻合。应修复元数据读取的转义还原，并确认兼容旧记录；随后用新的授权和运行证据复核。Tester 输出中中文字符的乱码成因仍未由静态证据确认，不能把它归到同一原因。
2. **P2：FD 创建失败用例没有注入实际失败。** `tests/test_fd031_promote_blackbox.py:193-197` 删除 `TEMPLATE.md` 后预期 promote 失败，但 `plugins/aiw-fd.py:1073-1075` 明确在模板缺失时写回 `DEFAULT_TEMPLATE`。Tester 的 S12 因此失败，只证明夹具假设错误，不证明产品的错误传播有缺陷或已通过。应改用确实使 `fd new` 返回非零的隔离条件，再检查 promote 的退出码、诊断和无 Task/元数据副作用。

## 范围与证据

- 审查结论：**需要修改**。这是用户要求的直接独立审查；当前 FD 为 `In Progress`，最新 `FD-031-000006-test-rejected` 是已领取的 Worker handoff。**没有 Reviewer 来源事件**，本报告不表示正式 `changes-requested` 或 `verification-passed` 交接，也不改变 FD 状态。
- Reviewer session：`fd031-reviewer-direct-20261008-c8e41b`；不同于 Worker `issue-promote-20261008-01`、`fd031-worker-repair-20261008-01` 及 Tester `fd031-tester-20261008-a3f7c9`。
- 审查 HEAD：`9fe06436042cbdf742c6b85d5269566a4b68eafe`；`develop...HEAD` 的 merge base：`de304afce52a5956f807024d5ac526fde70a07da`。检查了 19 个文件的差异统计，重点读取 promote dispatch、元数据编解码、FD 创建逻辑、黑盒测试和文档/规格差异。`git diff --check develop...HEAD` 无输出。
- Worker 实现报告称 `git diff --check` 与 `go build -o NUL ./cmd/aiw-req` 通过，并注明当时未运行测试；这仅是 Worker 当轮证据。后续 Tester 在 FD 修订 4、`FD-031-000004-implementation-ready`、digest `f2574ae2f23e1605895b823ffce3251351c18133e64be443bab21146f7e2adcc` 上，按 Planner 授权执行 `python -B -m unittest tests.test_fd031_promote_blackbox -v` 两次。首次 3/8 通过，修正临时二进制路径后 6/8 通过，仍有 S06 与 S12 失败。授权记录绑定该命令、worktree、Tester session 和实现版本；报告未把覆盖率命令伪称已执行。
- Tester 场景覆盖 9/13（69.2%），分支覆盖率未测。S08 的 Planner 回执实体未检验；S13 的 FD CLI 不可用未注入。PM 在 `FD-031-000005-test-report-ready` 后明确退回测试报告，未批准低覆盖率例外。当前不具备通过独立验证的证据。
- 文档、帮助和稳定规格的 promote 入口与 FD 方向总体一致。`[promotion]` 保持不写的代码路径已静态核对。以上判断不代替未执行场景的运行结果。

## 实际命令与限制

本 Reviewer 仅执行 PowerShell 只读文件读取、`rg`、`git status --short`、`git branch --show-current`、`git log -1`、`git rev-parse HEAD`、`git merge-base develop HEAD`、`git diff --stat develop...HEAD`、`git diff --numstat develop...HEAD`、指定文件的 `git diff develop...HEAD -- ...` 和 `git diff --check develop...HEAD`。读取 FD、规格、Worker/Tester/PM/Planner 证据及审查技能。未运行测试、构建、格式化、lint、vet、覆盖率、网络或命令行行为验证；未领取 Worker event，未发出 Reviewer event。

剩余风险：标题的 Unicode 乱码归因待验证；Planner 回执实体、CLI 不可用及真实 FD 创建失败路径尚缺运行证据。当前 Worker handoff 已领取，因此本次仅写独立报告，不修改 FD 正文或事件回执，以免改变进行中的交接依据。修复后应先完成新的独立 Tester 与 PM 决策，再由正式 Reviewer handoff 审查。
