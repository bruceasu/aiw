# E06 知识聚焦验证结果

Task workflow-automation；wi-0012 / 2.2；实现来源 E06。用户正常终端执行 run-e06-knowledge-terminal.py 一次，退出 0；三个测试及两个历史额度子场景全部 passed：

- TestKnowledgeVersionReviewAndExplicitRecheck
- TestKnowledgeInjectionDeterministicAndPreservesRequiredInput
- TestKnowledgeHistoryLimitsSurviveRestart（source-count、byte-total）

实际命令：

```text
go test ./internal/workflow -run ^(TestKnowledgeVersionReviewAndExplicitRecheck|TestKnowledgeInjectionDeterministicAndPreservesRequiredInput|TestKnowledgeHistoryLimitsSurviveRestart)$ -count=1 -timeout=60s -vet=off -json
```

2026-09-25T01:37:54.409391Z 至 01:37:57.811378Z；Go 1.25.1 windows/amd64；GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local。执行期间输入未变化，stderr 为空。命令、环境及输出摘要见 [run.json](run.json)，事件见 [stdout.jsonl](stdout.jsonl)。

[inputs.json](inputs.json) SHA256：`a5ef4d8375333d64d2adbbd599839ac2af6335b6bc4a0be595780f7d1779f14d`；冻结 [approved-plan.md](approved-plan.md) SHA256：`714fb69b89ef7c7cbf2243def70b91ca2b55cb17345c8be016984d7633c84d7e`。保留执行前计划状态和原始输出。

覆盖五态审阅、旧 revision 拒绝、编辑新候选、读取绑定文件变化及人工复核、重复拒绝防绕过、确定性可信排序/主输入保护，以及 64 来源/4 MiB 历史累计跨重开 Store 保留。真实临时 Store 配合明确的能力/容量/盘点替身；计数器为确定性字节长度替身，不证明真实模型 token 计量。

异步提取、部分覆盖草稿、生成版本、跨 Task 选择及全部注入硬边界仍待验证。不代表完整 E06/AC20–AC27/AX05，不生成产品 grant、不解除 Gate、不改实际项目启用状态。
