# FD-035 独立测试报告，第 1 轮

<!-- aiw-data: FD-035-test-report-r1.json -->

## 结论

Tester session `fd035-tester-20261008-8f4ac2` 认领 `FD-035-000004-implementation-ready`；测试绑定 FD 修订 4、摘要 `031564c0783d1959717aeec8cf1e3e37b085013c3e9a032a75ea11b7f31ddc97`。两次执行均有各自的 Planner 授权。首次 7 个方法中 6 个通过，1 个在第二个子场景准备临时 FD 时失败，尚未运行该子场景的决策断言。拆分测试方法以隔离临时目录后，唯一一次获批重跑的 8 个方法全部通过。

按独立可观察行为统计，25 个场景中 19 个有通过的黑盒断言，需求场景覆盖率为 **19/25（76%）**。最终一轮 8 个行为测试通过，0 个行为断言失败。首次的失败是测试夹具复用临时仓库的错误，不能归为产品行为失败；其原始结果仍保留。业务代码分支覆盖率**未测量**，授权不包含覆盖率命令。

## 场景与证据

| 场景 | 可观察行为 | 结果 | 用例或缺口 |
| --- | --- | --- | --- |
| S01 | 无失败、缺口可控时，单份接纳票交 Reviewer | 通过 | `test_clean_single_vote_routes_and_keeps_reviewer_independent` |
| S02 | 单份修复票不能被 PM 改成接纳 | 通过 | `test_clean_single_repair_vote_cannot_be_accepted` |
| S03 | 单份修复票使 FD 返回 Worker | 通过 | `test_clean_single_repair_vote_cannot_be_accepted` |
| S04 | 有失败行为测试时拒绝单份决策 | 通过 | `test_failed_behavior_requires_three_distinct_focuses` |
| S05 | 升级三份要求互补侧重点，重复侧重点被拒绝 | 通过 | `test_failed_behavior_requires_three_distinct_focuses` |
| S06 | PM 记录重大证据缺口时三份评估可交 Reviewer | 通过 | `test_material_gap_uses_three_votes` |
| S07 | PM 与首份修复票分歧时三份评估可交 Reviewer | 通过 | `test_pm_disagreement_uses_three_votes` |
| S08 | 失败测试和 0% 场景覆盖仍保留为事实，三份中两票接纳可交 Reviewer | 通过 | `test_failed_behavior_requires_three_distinct_focuses`、`test_two_accept_votes_route_failed_low_coverage_report_to_reviewer` |
| S09 | 三份中仅一票接纳时，接纳处置被拒绝 | 通过 | `test_one_accept_vote_returns_to_worker_and_mismatched_pm_vote_fails` |
| S10 | 三份中仅一票接纳时，修复处置返回 Worker | 通过 | `test_one_accept_vote_returns_to_worker_and_mismatched_pm_vote_fails` |
| S11 | 缺失评估文件不能发出 PM 事件 | 通过 | `test_missing_duplicate_stale_forged_and_conflicting_assessments_fail` |
| S12 | 重复评估路径不能发出 PM 事件 | 通过 | 同 S11 |
| S13 | 评估摘要过期不能发出 PM 事件 | 通过 | 同 S11 |
| S14 | 伪造评估来源事件不能发出 PM 事件 | 通过 | 同 S11 |
| S15 | 评估者 session 重复不能发出 PM 事件 | 通过 | 同 S11 |
| S16 | 评估者与 Worker、Tester 或 PM session 冲突不能发出 PM 事件 | 通过 | 同 S11 |
| S17 | Reviewer claim 与所有评估者及 Worker、Tester session 隔离 | 通过 | `test_clean_single_vote_routes_and_keeps_reviewer_independent`、`test_two_accept_votes_route_failed_low_coverage_report_to_reviewer` |
| S18 | PM 接纳风险不会改写 Tester 报告和失败事实 | 通过 | `test_two_accept_votes_route_failed_low_coverage_report_to_reviewer` |
| S19 | 不带新策略字段的旧三份式决策仍可交 Reviewer | 通过 | `test_two_accept_votes_route_failed_low_coverage_report_to_reviewer` |
| S20 | 新策略的无效策略值、模式或升级原因被拒绝 | 未覆盖 | 本轮未构造这些无效字段组合 |
| S21 | 评估修订号不符被拒绝 | 未覆盖 | 本轮只构造过期摘要，未独立改修订号 |
| S22 | 评估缺少完整风险字段时被拒绝 | 未覆盖 | 本轮只验证完整字段的有效夹具 |
| S23 | 重大缺口却提交单份决策时被拒绝 | 未覆盖 | 本轮验证了重大缺口的合法三份路径 |
| S24 | 新三份模式缺少侧重点时被拒绝 | 未覆盖 | 本轮验证了重复侧重点被拒绝和正确三份被接纳 |
| S25 | 无失败但低覆盖率且 PM 认为缺口可控时，单份路径不因覆盖率自动升级 | 未覆盖 | 本轮单份路径使用 100% 场景覆盖夹具 |

评估者之间不共享草稿、旧归档文件保持原样、文档说明一致性以及 Reviewer 对未披露缺陷的审查权属于流程或静态要求，本轮没有把它们记为运行通过。上述未覆盖场景也未推断为通过。

## 命令与风险

工作目录：`C:\Users\svictor\workspace\tools\ai-tools\aiw\.wt\FD-035`。两次均执行精确命令：

```powershell
python -B -m unittest tests.test_fd032_risk_decision_blackbox -v
```

- 首次授权：`FD-035-test-authorization-r1.md`。退出码 1；原始摘要 `Ran 7 tests in 18.149s` / `FAILED (failures=1)`。`pm-disagreement` 子场景在 `ready_worker` 夹具准备时返回 `fd: design-ready requires a preceding planner handoff`，其余 6 个测试方法为 `ok`。
- 修正：仅将重大缺口与 PM 分歧拆为两个 `unittest` 方法，各自得到独立 `setUp` 和 `TemporaryDirectory`。测试文件 SHA-256 为 `65e4cab7308bf2e48f54489d1e44894761e695d95e948965590ff02785d8272f`；未改实现源码。
- 第二次授权：`FD-035-test-authorization-r2.md`。退出码 0；原始摘要 `Ran 8 tests in 19.725s` / `OK`，8 个方法均为 `ok`。

测试文件为 `tests/test_fd032_risk_decision_blackbox.py`，调用 `tests/test_fd014_blackbox.py` 的临时仓库夹具。运行在系统临时目录复制 CLI 和模板，初始化临时 Git 仓库，在该目录生成合成 FD 文件与收据并清理；没有预期的工作区写入。没有执行覆盖率测量、其他测试、构建、lint、格式化或网络命令。剩余风险是 S20–S25 无行为证据，业务代码分支覆盖率未知；建议 PM 结合独立风险评估判断。
