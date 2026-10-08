# FD-XXX 测试风险评估，第 N 轮，评估者 A

<!-- aiw-data: FD-XXX-test-risk-assessment-rN-a.json -->

## 独立结论

建议填写 `accept-with-risk` 或 `repair`。依据当前 Tester 报告与原始证据，说明失败、未运行和未覆盖项的严重性；不能把带风险接纳写成测试通过。
评估者 A 侧重用户和验收影响；需要追加评估时，B 侧重工程证据与修复，C 侧重交付与运行风险。每位评估者仍填写全部风险字段并独立投票，JSON `assessment_focus` 分别使用 `acceptance-impact`、`technical-repair`、`delivery-operations`。

## 影响与时间

说明影响范围、预计修复时间、对交付时间的影响及不确定性。无法估算时写明 `unknown` 和原因，不能凭空给出日期。

## 证据与风险

列出报告和事件、关键证据、理由及接纳后仍存在的风险。评估者不得阅读其他评估者的草稿或投票，也不得代写 PM 决策。

同名 JSON 按 `TEST_RISK_ASSESSMENT_DATA_TEMPLATE.json` 编写。每份评估使用独立 session，并与 Worker、Tester、PM session 不同。追加的评估必须引用同一个当前 Tester 事件、FD 修订和摘要。
