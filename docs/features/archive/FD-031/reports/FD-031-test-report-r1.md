# FD-031 独立测试报告，第 1 轮

<!-- aiw-data: FD-031-test-report-r1.json -->

## 结论

针对修订 4、事件 `FD-031-000004-implementation-ready` 执行了授权的同一条聚焦命令两次。首次因临时 `aiw-req.exe` 放置路径错误，8 例中 3 例通过、5 例失败；修正夹具路径后按规则重跑，8 例中 6 例通过、2 例失败。首次失败没有提供产品成功路径的有效证据。第二次失败之一是无效的模板缺失注入；另一个是 Issue 标题未按夹具原文保留，产品行为与 Windows 编码/夹具处理的归因尚未确定。不得将这两项写成通过。

需求场景覆盖为 **9/13（69.2%）**。业务代码分支覆盖率**未测量**；没有授权覆盖率命令。建议 PM 暂不接受测试报告，并安排诊断标题传递及修正无效失败注入。已执行的失败行为测试不能按通过处理。

## 场景与证据

| ID | 可观察场景 | 状态 | 证据或缺口 |
| --- | --- | --- | --- |
| S01 | `issue --help` 列出 promote | 通过 | `test_both_help_entries_name_promote` |
| S02 | `req --help` 列出 promote | 通过 | 同一用例的 `req` 子场景 |
| S03 | 不存在的 Issue 被拒绝且不创建 FD | 通过 | `test_missing_issue_is_rejected` |
| S04 | Issue 状态未批准时被拒绝 | 通过 | `test_unapproved_issue_is_rejected` |
| S05 | 批准记录未批准时被拒绝 | 通过 | `test_issue_and_approval_must_both_be_approved` |
| S06 | `issue promote` 创建 FD 且完整保留标题和 ID | 受阻 | 创建了 FD，但标题断言失败；ID 后续断言未执行。见下方原始片段。 |
| S07 | `req promote` 兼容入口创建 FD | 通过 | `test_req_alias_creates_fd` |
| S08 | 成功创建后有可核验的 Planner 回执 | 未覆盖 | S06 在检查回执前失败；stdout 出现 handoff 文本，但未核验回执 JSON。 |
| S09 | 旧 `[promotion]` 元数据保持原样 | 通过 | S07 用 `SPEC_DRAFTED` 夹具做字节比较。 |
| S10 | promote 不创建 Task | 通过 | S07、拒绝路径检查临时项目没有 Task 目录。 |
| S11 | 已关联 Issue 拒绝第二次创建 | 通过 | `test_duplicate_link_is_rejected` |
| S12 | FD 创建失败时传回错误 | 未覆盖 | `test_fd_creation_failure_is_reported` 删除 `TEMPLATE.md` 后 FD 仍成功，说明该注入没有触发失败；这是夹具错误，不能据此判定产品缺陷。 |
| S13 | FD CLI 不可用时传回错误 | 未覆盖 | 未注入此条件，避免离开隔离 CLI 路径。 |

S06 夹具预期标题是 `批准的 Issue "alpha beta"`。第二次输出中，生成 FD 的标题片段显示为 `# FD-001: ��y�I Issue \\"alpha beta\\`，断言报告为 `AssertionError: '��y�I Issue "alpha beta"' not found in ...`。测试文件中的预期 Unicode 码点经只读核对为 `\u6279\u51c6\u7684`；错误文本经 Windows 命令输出显示，不能仅凭该显示断定损坏发生在产品、命令输出编码还是夹具的转义处理。引号的反斜杠差异也需要单独诊断。当前保留为**行为失败且归因未决**。

第二次执行的相关原始错误输出（长 FD 模板正文省略）：

```text
FAIL: test_fd_creation_failure_is_reported
AssertionError: 0 == 0 : handoff FD-001-000002-design-requested pending planner
created FD-001: ...\docs\features\FD-001_ISSUE_ALPHA_BETA.md
FAIL: test_issue_alias_creates_fd_and_planner_handoff
AssertionError: '��y�I Issue "alpha beta"' not found in '# FD-001: ��y�I Issue \\"alpha beta\\...'
Ran 8 tests in 17.878s
FAILED (failures=2)
```

文档与稳定规格一致性属于静态审查事项，本轮未将其计入运行时场景；Tester 设计黑盒用例时未读取实现源码。没有运行覆盖率工具，无法推断业务分支覆盖。

## 命令与风险

- 授权：`docs/features/reports/FD-031-test-authorization-r1.md`，绑定 FD 修订 4、digest `f2574ae2f23e1605895b823ffce3251351c18133e64be443bab21146f7e2adcc`、本 Tester session `fd031-tester-20261008-a3f7c9`。
- 实际命令，两次相同：`python -B -m unittest tests.test_fd031_promote_blackbox -v`；工作目录为 FD-031 worktree。首次 `Ran 8 tests in 16.867s; FAILED (failures=5)`；原因是夹具二进制路径。修正路径后第二次 `Ran 8 tests in 17.878s; FAILED (failures=2)`。
- 测试代码：`tests/test_fd031_promote_blackbox.py`。Go 构建缓存、临时 CLI、Git 项目及生成 FD/回执均在系统 `%TEMP%\fd031-promote-*` 内，结束后由 `TemporaryDirectory` 清理。命令设置 `GOPROXY=off`、`GOSUMDB=off`、`GOTOOLCHAIN=local`。
- 未执行额外测试、覆盖率、最终构建、格式化、lint、vet 或网络命令。本轮一次修正后的重跑额度已用完。
- 剩余风险：标题原文保真、Planner 回执实体、FD CLI 不可用与实际 FD 创建失败的传播尚未证实。当前测试文件保留无效模板缺失用例，后续修正需要新的版本绑定授权和新执行证据。

本报告来源事件：`FD-031-000004-implementation-ready`；独立 Tester session：`fd031-tester-20261008-a3f7c9`。
