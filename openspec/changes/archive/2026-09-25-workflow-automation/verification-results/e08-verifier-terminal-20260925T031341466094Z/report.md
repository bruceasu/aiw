# E08 修复后重跑结果

Task workflow-automation；wi-0012 / 2.2；实现来源 E08。用户在正常终端重跑原三项一次，退出 0，三个顶层用例及 22 个子场景全部 passed。

| 测试 | 结果 |
| --- | --- |
| TestVerifierCoverageRequiresEvidenceAndConsistentOutcome | passed；10 个子场景 |
| TestVerifierFrozenReportRejectsMismatchedInput | passed；10 个子场景 |
| TestVerifierNegativePublicationPreservesDevelopmentState | passed；failed/inconclusive 两个子场景 |

实际命令：

```text
go test ./internal/workflow -run ^(TestVerifierCoverageRequiresEvidenceAndConsistentOutcome|TestVerifierFrozenReportRejectsMismatchedInput|TestVerifierNegativePublicationPreservesDevelopmentState)$ -count=1 -timeout=60s -vet=off -json
```

执行时间 2026-09-25T03:13:41.598953Z 至 03:13:45.480527Z；Go 1.25.1 windows/amd64。GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local。执行期间 changed_inputs 为空，stderr 为空。精确命令/环境/输出哈希见 [run.json](run.json)，逐项结果见 [stdout.jsonl](stdout.jsonl)。

[inputs.json](inputs.json) SHA256：`4639ef8248424a459e3d599a0bc2ddc0c7b9010ad3ea8bdb532cc0a68b857dab`；冻结 [approved-plan.md](approved-plan.md) SHA256：`c21ee91dbc21f62deb755dd0a3292c4e43dc204fa469c87ae3dc992546ddd69d`。原始输出、计划和输入摘要已核对，未改写执行前计划状态。

[首跑失败](../e08-verifier-terminal-20260925T015608050801Z/report.md) 保留。唯一相关修复是 verifierTestRow 将空 Evidence/Gaps 初始化为 []，避免 null；生产解析器及原有断言不变。本次结果证明修复后的测试输入可通过严格解析，并已执行此前未到达的版本/引用校验和负面报告发布断言。

覆盖证据三态、固定 Task/Work Item/Attempt/snapshot 和引用范围、迟到负面报告的持久发布及现有开发状态隔离。使用临时 Store 和固定测试快照；未经过生产接受路径建立已完成 Task，未验证真实 Git 快照封存、模型语义审查或独立宿主。不得据此关闭完整 AC31/AC32、2.2 或联合启用 Gate。
