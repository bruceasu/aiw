# E08 首次运行失败与 fixture 修复

Task workflow-automation；wi-0012 / 2.2；实现来源 E08。用户正常终端执行 run-e08-verifier-terminal.py，退出 1，失败证据原样保留。

| 测试 | 实际结果 |
| --- | --- |
| TestVerifierCoverageRequiresEvidenceAndConsistentOutcome | passed；十个子场景全部通过 |
| TestVerifierFrozenReportRejectsMismatchedInput | failed；合法基线报告解析失败，尚未进入十个变异子场景 |
| TestVerifierNegativePublicationPreservesDevelopmentState | failed；failed/inconclusive 两个子场景均在合法报告解析处失败，尚未进入发布断言 |

原始错误：`null is not a contract value`，见 [stdout.jsonl](stdout.jsonl)。`verifierTestRow` 只填写当前结论需要的切片，另一切片默认 nil，`encoding/json` 因此输出 `null`。生产指令要求 evidence/gaps 为数组，strictJSON 拒绝 null；这是 fixture 不符合报告契约，不是放宽生产解析的理由。

最小修复：将 helper 中 Evidence、Gaps 初始化为非 nil 空数组，保留后续证据/缺口内容和所有断言，未修改生产代码。修复后的结果必须另行取证；本报告不把首跑改记为 passed。

实际命令：

```text
go test ./internal/workflow -run ^(TestVerifierCoverageRequiresEvidenceAndConsistentOutcome|TestVerifierFrozenReportRejectsMismatchedInput|TestVerifierNegativePublicationPreservesDevelopmentState)$ -count=1 -timeout=60s -vet=off -json
```

2026-09-25T01:56:08.161241Z 至 01:56:09.977516Z；Go 1.25.1 windows/amd64；执行期间 changed_inputs 为空，stderr 为空。精确环境及哈希见 [run.json](run.json)。

[inputs.json](inputs.json) SHA256：`9904f7c9e33ac848a4663f4b2d6d8de4f1ce2c3820880cf1cf0dac2e994369fd`；冻结 [approved-plan.md](approved-plan.md) SHA256：`96a63664d8d8f64f1db42254e2f8ec6d28464ab05c415cf76ea9a53fddb545c0`。修复使测试源码版本变化，旧执行仅证明旧输入。

后续只重跑原三项一次，不扩大范围。此前工具宿主已有 Windows 临时 Store 路径权限失败记录，因此继续使用用户正常终端取证，不更改 ACL、安全路径检查或申请提权。2.2 与 Gate 保留；真实宿主/模型、Git 封存及完整验收仍待证据。
