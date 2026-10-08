# FD-XXX PM 测试报告决策，第 N 轮

<!-- aiw-data: FD-XXX-test-decision-rN.json -->

## 决策

用中文列出独立风险评估及各自票数，说明严重性、影响范围、预计修复时间、
交付影响、异议和剩余风险。新决策填写 `assessment_policy: adaptive-v1`：
默认 `single`，仅在失败行为测试数为零、PM 判定证据缺口可控且决定与唯一
评估票一致时使用。失败测试、PM 与首份意见分歧或 PM 判定证据缺口重大时
使用 `escalated`，写明升级原因并引用三份互补视角的评估；至少两票
`accept-with-risk` 才接纳。两项覆盖率和失败测试数必须保留为事实；
它们不自动代表测试通过或失败。不得把失败或未运行的测试描述为通过。

JSON 中填写 `assessment_mode`、`escalation_reason`、`escalation_detail`、
`coverage_gap_disposition` 和 `coverage_gap_reason`。单份时升级原因是
`none`、缺口判断是 `bounded`；三份时升级原因是 `failed-tests`、
`material-evidence-gap` 或 `pm-disagreement`，并具体说明。若需要三份，
将 B、C 的报告路径加入 `assessments`，按实际票数更新计数。

同名 JSON 按 `TEST_DECISION_DATA_TEMPLATE.json` 编写。事件仍以这份
Markdown 为 `--artifact`，CLI 从 JSON 核对机器字段。
