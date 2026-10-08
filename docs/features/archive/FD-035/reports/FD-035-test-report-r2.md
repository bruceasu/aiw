# FD-035 独立测试报告，第 2 轮

<!-- aiw-data: FD-035-test-report-r2.json -->

## 结论

Tester session `fd035-tester-20261008-a7d92e` 认领 `FD-035-000011-implementation-ready`。本轮绑定 FD revision 11、摘要 `359ffa2cb9c8e6e4307a4189549da31a66bb69d6f9aca12bcb736fd6492b7017`。首次获批执行运行 14 个测试方法，其中 13 个通过；1 个方法的 `mode` 和 `reason` 子用例在临时 FD 准备/claim 阶段失败，错误为 `human and PM handoffs cannot be claimed by an agent session`，未到达产品行为断言。修复夹具后，经单独授权重跑同一命令，14 个方法全部通过（`Ran 14 tests in 32.016s; OK`），包括 S20 的 policy、mode、reason 拒绝断言及每次拒绝后 PM event 未变化的断言。首次失败作为测试夹具问题保留，不是产品行为失败。

按 25 个独立可观察场景统计，最终获批重跑中 S01–S25 均有通过证据，需求场景覆盖率为 **25/25（100%）**。分支覆盖率**未测量**：coverage 命令未获授权，且当前环境缺少 coverage.py；未安装依赖或访问网络。因此用户要求的分支覆盖率证据仍未完成，本报告的 recommendation 为 `blocked`。

## 场景与证据

| 场景 | 可观察行为 | 结果 | 用例或缺口 |
| --- | --- | --- | --- |
| S01 | 无失败且 PM 判断缺口可控时，一份接纳票交 Reviewer | 通过 | `test_clean_single_vote_routes_and_keeps_reviewer_independent` |
| S02 | 单份修复票不能被 PM 改成接纳处置 | 通过 | `test_clean_single_repair_vote_cannot_be_accepted` |
| S03 | 单份修复票使 FD 返回 Worker | 通过 | `test_clean_single_repair_vote_cannot_be_accepted` |
| S04 | 失败行为测试时拒绝单份决策 | 通过 | `test_failed_behavior_requires_three_distinct_focuses` |
| S05 | 升级三份要求互补侧重点，缺失所需侧重点被拒绝 | 通过 | `test_escalated_decision_requires_all_assessment_focuses` |
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
| S20 | 新策略值、模式或升级原因无效时拒绝 PM 决策 | 通过 | `test_invalid_adaptive_policy_mode_or_reason_is_rejected`；修复夹具后验证三种非法字段组合均被拒绝，且 PM event 不变 |
| S21 | 评估修订号不符时拒绝 PM 决策 | 通过 | `test_assessment_revision_mismatch_is_rejected` |
| S22 | 评估缺少完整风险字段时拒绝 PM 决策 | 通过 | `test_assessment_missing_required_risk_field_is_rejected` |
| S23 | 重大证据缺口下单份决策被拒绝 | 通过 | `test_material_gap_cannot_use_single_assessment` |
| S24 | 新三份模式缺少评估侧重点时被拒绝 | 通过 | `test_escalated_decision_requires_all_assessment_focuses` |
| S25 | 无失败、低场景覆盖率且缺口可控时，单份路径不因覆盖率自动升级 | 通过 | `test_low_scenario_coverage_does_not_force_escalation` |

评估者之间不共享草稿、旧归档文件保持原样、文档与模板一致性以及 Reviewer 对未披露缺陷的审查权属于流程或静态要求；本轮未将其记为运行通过。

## 命令与风险

工作目录：`C:\Users\svictor\workspace\tools\ai-tools\aiw\.wt\FD-035`。

执行命令（两次，分别经授权）：

```powershell
python -B -m unittest tests.test_fd032_risk_decision_blackbox -v
```

首次授权：`FD-035-test-authorization-r3.md`，绑定当前实现事件、revision 11、摘要及本 Tester session。原始摘要：`Ran 14 tests in 31.808s; FAILED (failures=2)`。13 个方法为 `ok`；S20 的 `mode` 与 `reason` 子用例在临时仓库夹具调用 `claim` 时失败，未到达行为断言。修复将一次 `begin_pm`、assessor 和 decision 复用于三个 subTest；每个子用例从原始 decision JSON 变异，并确认拒绝后 PM event 未变化。Planner 审核后发出 `FD-035-test-authorization-r4.md`，授权唯一一次相关修复重跑。重跑原始摘要：`Ran 14 tests in 32.016s; OK`，14 个方法均通过。

分支覆盖率命令 `python -B -m tests.fd035_coverage` 未运行、也未获授权。Planner 提供的信息为当前 Python 环境缺少 coverage.py；是否允许临时安装仍待用户决定，Tester 未安装依赖或使用网络。因此报告不提供分支总数、命中数或百分比。未执行其他测试、构建、lint、格式化或网络命令。剩余风险为 CLI 分支覆盖率未知，指定的覆盖率要求尚未完成。
