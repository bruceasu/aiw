# FD-034 独立测试报告，第 1 轮

<!-- aiw-data: FD-034-test-report-r1.json -->

## 结论

Tester session `fd034-tester-20261008-8b85b9d1` 认领 `FD-034-000014-test-requested`，测试 FD 修订 14，SHA-256 为 `43863ae0df4bac7bf829db5b00443950638c8b64f2b810b793143cb456ad5ffc`。按 Planner 授权 `FD-034-test-authorization-r1.md`，仅执行同一聚焦命令两次。首次 9 例中 5 例通过、4 例失败；修正测试断言后，允许的唯一一次重跑为 9/9 通过。首轮失败由测试的 Git 对照输出缩写对象 ID、以及把英文“已应用”提示误断言为中文引起；没有据此认定产品缺陷。

按独立可观察验收行为统计，13 个场景中 12 个已覆盖并通过，需求场景覆盖率为 12/13（92.3%）。未测量业务代码分支覆盖率；本次授权不包含覆盖率命令。测试在临时 Git 仓库中执行，未调用网络或外部服务。

## 场景与证据

| 场景 | 可观察行为 | 结果 | 测试用例 |
| --- | --- | --- | --- |
| S01 | 默认补丁含已跟踪暂存、未暂存及二进制变化，不含未跟踪文件；说明有摘要、应用命令与限制 | 通过 | `test_default_exports_tracked_staged_unstaged_and_binary_with_guide` |
| S02 | `--staged` 只导出暂存区 | 通过 | `test_staged_and_worktree_select_independent_diffs` |
| S03 | `--worktree` 只导出未暂存区 | 通过 | `test_staged_and_worktree_select_independent_diffs` |
| S04 | 两个分叉分支 ref 导出直接 A 到 B 的树差异，与未提交工作区编辑无关；说明记录两端 ref 和完整提交 ID | 通过 | `test_two_refs_use_direct_tree_diff_and_record_fixed_ids` |
| S05 | 两个提交 ID 可用作 ref，产出同样的直接差异 | 通过 | `test_two_refs_use_direct_tree_diff_and_record_fixed_ids` |
| S06 | 单独指定 ref、与范围参数混用或无效 ref 均非零退出且不写输出 | 通过 | `test_invalid_ref_options_fail_without_output` |
| S07 | 空差异、Git 失败、patch 或说明已存在时非零退出，不覆盖现有内容 | 通过 | `test_empty_diff_git_failure_and_existing_outputs_do_not_overwrite` |
| S08 | 成功应用只改变目标工作区，暂存区与 HEAD 不变 | 通过 | `test_apply_changes_only_worktree_and_recognizes_already_applied` |
| S09 | 重复应用失败时提示可能已应用，目标状态不变 | 通过 | `test_apply_changes_only_worktree_and_recognizes_already_applied` |
| S10 | 不匹配的预检失败显示 Git 错误与恢复建议，目标文件、暂存区和 HEAD 不变 | 通过 | `test_failed_precheck_preserves_target_and_reports_recovery` |
| S11 | 空补丁与不存在的补丁被拒绝，仓库状态不变 | 通过 | `test_empty_and_missing_patch_are_rejected` |
| S12 | 帮助入口提供创建、应用与来源范围参数 | 通过 | `test_help_exposes_patch_creation_and_application` |
| S13 | 预检通过后实际应用步骤又失败时，返回非零并说明可能的局部改动 | 未覆盖 | 需要在两次 Git 调用之间制造可控变化；现有黑盒夹具未模拟该竞态 |

使用文档与稳定规格已按公开文本阅读；“用户无需查 Git 参数即可完成操作”的易用性不是本次自动化断言。没有对“绝不启动 AI/网络”或所有 `--3way`、`--reject` 路径作运行时拦截证明。上述限制不计作通过。

## 命令与风险

唯一授权命令和工作目录：

```text
cwd: C:\Users\svictor\workspace\tools\ai-tools\aiw\.wt\FD-034
python -B -m unittest tests.test_fd034_git_patch_blackbox -v
```

- 首次执行：退出码 1，耗时 8.998 秒。原始摘要为 `Ran 9 tests in 8.997s` / `FAILED (failures=4)`。失败用例为默认导出、范围导出、双 ref 导出、重复应用提示；其余 5 例为 `ok`。三个导出用例的实际 patch `index` 行含完整对象 ID，测试对照 `git diff --binary` 只有缩写 ID；重复应用的实际输出含 `Suggestion: the patch may already be applied. Inspect git diff and git status --short before retrying.`，测试误期望中文提示。
- 测试修正：仅将 Git 对照命令加 `--full-index`，并检查实际英文 `already be applied` 提示。未修改实现源码或临时仓库之外的工作区内容。
- 唯一重跑：退出码 0，耗时 9.116 秒。原始摘要为 `Ran 9 tests in 9.116s` / `OK`，全部 9 个用例为 `ok`。

测试代码位于 `tests/test_fd034_git_patch_blackbox.py`；临时目录在系统临时位置自动清理。未运行构建、lint、格式化、Go 测试、覆盖率测量或额外测试命令。剩余风险为应用前后发生外部改动时的实际应用失败分支尚无行为证据，业务代码分支覆盖率未知。建议后续审查关注这一分支；测试事实不代替 PM 的风险决定。
