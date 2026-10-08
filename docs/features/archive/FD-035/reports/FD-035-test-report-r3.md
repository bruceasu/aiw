# FD-035 独立测试报告，第 3 轮

<!-- aiw-data: FD-035-test-report-r3.json -->

## 结论

Tester session `fd035-tester-20261008-a7d92e` 认领 `FD-035-000011-implementation-ready`。本轮证据绑定 FD revision 11、摘要 `359ffa2cb9c8e6e4307a4189549da31a66bb69d6f9aca12bcb736fd6492b7017`。修复 S20 测试夹具后，获批重跑的 14 个黑盒测试方法全部通过；coverage runner 随后再次运行同一模块，14 个方法也全部通过。25 个独立可观察场景均有通过证据，需求场景覆盖率为 **25/25（100%）**。

获批 coverage runner 收集到实际 CLI 子进程 `plugins/aiw-fd.py` 的 **210/548 分支（38.3%）**。这表示本轮实测分支占比；未覆盖的 338 个分支仍是证据缺口，不据此推断其行为正确。

## 场景与证据

| 场景 | 可观察行为 | 结果 | 用例 |
| --- | --- | --- | --- |
| S01 | 无失败且 PM 判断缺口可控时，一份接纳票交 Reviewer | 通过 | `test_clean_single_vote_routes_and_keeps_reviewer_independent` |
| S02 | 单份修复票不能被 PM 改成接纳处置 | 通过 | `test_clean_single_repair_vote_cannot_be_accepted` |
| S03 | 单份修复票使 FD 返回 Worker | 通过 | `test_clean_single_repair_vote_cannot_be_accepted` |
| S04 | 失败行为测试时拒绝单份决策 | 通过 | `test_failed_behavior_requires_three_distinct_focuses` |
| S05 | 升级三份要求互补侧重点，缺失侧重点被拒绝 | 通过 | `test_escalated_decision_requires_all_assessment_focuses` |
| S06 | PM 记录重大证据缺口后三份评估可交 Reviewer | 通过 | `test_material_gap_uses_three_votes` |
| S07 | PM 与首份票分歧后三份评估可交 Reviewer | 通过 | `test_pm_disagreement_uses_three_votes` |
| S08 | 失败与低覆盖率事实保留，三份中两票接纳可交 Reviewer | 通过 | `test_failed_behavior_requires_three_distinct_focuses`、`test_two_accept_votes_route_failed_low_coverage_report_to_reviewer` |
| S09 | 三份中仅一票接纳时接纳处置被拒绝 | 通过 | `test_one_accept_vote_returns_to_worker_and_mismatched_pm_vote_fails` |
| S10 | 三份中仅一票接纳时修复处置返回 Worker | 通过 | `test_one_accept_vote_returns_to_worker_and_mismatched_pm_vote_fails` |
| S11 | 缺失评估文件不能发出 PM 事件 | 通过 | `test_missing_duplicate_stale_forged_and_conflicting_assessments_fail` |
| S12 | 重复评估路径不能发出 PM 事件 | 通过 | 同 S11 |
| S13 | 评估摘要过期不能发出 PM 事件 | 通过 | 同 S11 |
| S14 | 伪造评估来源事件不能发出 PM 事件 | 通过 | 同 S11 |
| S15 | 评估者 session 重复不能发出 PM 事件 | 通过 | 同 S11 |
| S16 | 评估者与 Worker、Tester 或 PM session 冲突不能发出 PM 事件 | 通过 | 同 S11 |
| S17 | Reviewer claim 与 Worker、Tester 和评估者 session 隔离 | 通过 | `test_clean_single_vote_routes_and_keeps_reviewer_independent`、`test_two_accept_votes_route_failed_low_coverage_report_to_reviewer` |
| S18 | 接纳风险不改写 Tester 报告和失败事实 | 通过 | `test_two_accept_votes_route_failed_low_coverage_report_to_reviewer` |
| S19 | 不带新策略字段的旧三份式决策仍可读取并交 Reviewer | 通过 | `test_two_accept_votes_route_failed_low_coverage_report_to_reviewer` |
| S20 | 新策略值、模式或升级原因无效时拒绝 PM 决策 | 通过 | `test_invalid_adaptive_policy_mode_or_reason_is_rejected`；三种非法字段组合均被拒绝，PM event 保持不变 |
| S21 | 评估修订号不符时拒绝 PM 决策 | 通过 | `test_assessment_revision_mismatch_is_rejected` |
| S22 | 评估缺少完整风险字段时拒绝 PM 决策 | 通过 | `test_assessment_missing_required_risk_field_is_rejected` |
| S23 | 重大证据缺口下单份决策被拒绝 | 通过 | `test_material_gap_cannot_use_single_assessment` |
| S24 | 新三份模式缺少评估侧重点时被拒绝 | 通过 | `test_escalated_decision_requires_all_assessment_focuses` |
| S25 | 无失败、低场景覆盖率且缺口可控时，单份路径不因覆盖率自动升级 | 通过 | `test_low_scenario_coverage_does_not_force_escalation` |

评估者间不共享草稿、旧归档文件保持原样、文档与模板一致性以及 Reviewer 对未披露缺陷的审查权属于流程或静态要求；这些不在本轮运行场景覆盖率内。

## 命令与风险

工作目录均为 `C:\Users\svictor\workspace\tools\ai-tools\aiw\.wt\FD-035`。

1. `python -B -m unittest tests.test_fd032_risk_decision_blackbox -v`，授权 `FD-035-test-authorization-r3.md`。原始摘要：`Ran 14 tests in 31.808s; FAILED (failures=2)`。两个失败是 `mode`、`reason` 子场景的临时夹具 claim 阶段错误，未到达行为断言。修复将一个 PM handoff、评估和决策复用于三个子场景，分别变异同一决策 JSON 并确认拒绝后事件不变。
2. 同一 unittest 命令，授权 `FD-035-test-authorization-r4.md`。原始摘要：`Ran 14 tests in 32.016s; OK`。14 个方法全部通过。
3. `$env:FD035_COVERAGE_PACKAGE_DIR='C:\Users\svictor\AppData\Local\Temp\aiw-fd035-coverage-py'; python -B -m tests.fd035_coverage`，授权 `FD-035-test-authorization-r5.md`。原始摘要：`Ran 14 tests in 49.445s; OK`，随后输出 `Branch coverage: plugins\aiw-fd.py: 210/548 branches (38.3%).` coverage.py 7.10.7 位于用户批准的一次性系统临时目录；runner 的临时仓库、配置与覆盖率数据按设计自动清理。

此前 r2 报告交接曾被 CLI 两次拒绝：先因覆盖率字段使用了 `25/25 (100%)` 而非百分比格式，修正为 `100%` 后仍收到 `fd: machine evidence JSON is invalid`。r2 文件保留这些历史状态；本报告是新的 r3 evidence。未执行其他测试、构建、lint、格式化或网络命令。剩余风险是本轮仅覆盖 38.3% 的 CLI 分支；未覆盖分支未被行为验证。
