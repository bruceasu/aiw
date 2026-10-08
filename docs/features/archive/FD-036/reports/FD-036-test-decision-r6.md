# FD-036 PM 测试报告决策 r6

<!-- aiw-data: FD-036-test-decision-r6.json -->

## 决策

**接受本轮测试证据并进入独立 Reviewer，带风险。** 本决策绑定 FD-036-000033-test-report-ready、FD revision 33、digest 54862b4294872e269be7ea94f4b8dd4d3467b9e2e4b50a85d69776f306dd05e0。r6 是经批准的 S04 单用例诊断：1 项通过、退出码 0。r5 完整套件中同一用例的 ERROR 未复现，但其原因仍未知；r6 不是完整套件重跑，也不改写 r5 的历史结果。

三位独立评估者 A（验收影响）、B（技术证据）、C（交付运维）均投 ccept-with-risk：

- A：docs/features/reports/FD-036-test-risk-assessment-r6-a.md
- B：docs/features/reports/FD-036-test-risk-assessment-r6-b.md
- C：docs/features/reports/FD-036-test-risk-assessment-r6-c.md

采用 daptive-v1/escalated，升级原因是重大证据缺口：当前报告只覆盖 S04 的单例诊断；r5 的 S04 错误仍未归因，S08/S09/S14/S15 仍未覆盖，branch coverage 未测量。三票接纳达到升级流程门槛。本决定只允许进入 Reviewer，并不将未运行或未覆盖行为视为通过。

后续风险仍包括 r5 ERROR 的未知原因、完整套件未在当前实现修订重跑、真实 API 与跨平台 profile 路径未验证，以及翻译质量未评估。

**Disposition:** accepted
**Tester report:** docs/features/reports/FD-036-test-report-r6-diagnostic.md
**FD revision:** 33
**FD digest:** 54862b4294872e269be7ea94f4b8dd4d3467b9e2e4b50a85d69776f306dd05e0
**Requirements coverage:** 100.00%
**Branch coverage:** not measured
**Failed behavior tests:** 0
**Assessment policy:** adaptive-v1
**Assessment mode:** escalated
**Escalation reason:** material-evidence-gap
**Escalation detail:** r6 is one approved diagnostic case, not a full-suite rerun; the prior r5 suite ERROR remains unexplained, and multiple acceptance scenarios remain uncovered.
**Coverage gap disposition:** material
**Coverage gap reason:** The current evidence covers only S04 diagnostic. It does not establish a full-suite result; S08/S09/S14/S15 and branch coverage remain uncovered or unmeasured.
**Assessments:** ["docs/features/reports/FD-036-test-risk-assessment-r6-a.md","docs/features/reports/FD-036-test-risk-assessment-r6-b.md","docs/features/reports/FD-036-test-risk-assessment-r6-c.md"]
**Rationale:** Three independent assessors voted accept-with-risk. The r6 authorized S04 diagnostic passed (1/1, exit code 0), showing the prior r5 ERROR did not reproduce in that run. This does not explain the earlier ERROR or substitute for a full-suite rerun. Accept only to proceed to independent review, retaining all stated evidence limitations.
**Exceptions:** S04 historical r5 ERROR cause unresolved; no full suite rerun on current revision; S08/S09/S14/S15 not covered; branch coverage not measured; real API, cross-platform profile paths, and translation quality not validated.
**Residual risk:** Potential intermittent or environment-related S04 issue remains unknown. The single-case diagnostic cannot establish full-FD behavior. Existing coverage, platform, API, and translation-quality risks remain.
**PM identity:** fd036-pm-20261008-r6-host
**Decision time:** 2026-10-08T09:52:44+00:00
