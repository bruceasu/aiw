# E06 登记与持久发布结果

Task workflow-automation；wi-0012 / 2.2；实现来源 E06。用户正常终端一次运行，退出 0，两项全部 passed：

- TestKnowledgeAcceptedRegistrationDeduplicatesAfterRestart
- TestKnowledgePartialDraftPublishesDurablyThenAdvances

实际命令：

```text
go test ./internal/workflow -run ^(TestKnowledgeAcceptedRegistrationDeduplicatesAfterRestart|TestKnowledgePartialDraftPublishesDurablyThenAdvances)$ -count=1 -timeout=60s -vet=off -json
```

2026-09-25T04:22:35.009920Z 至 04:22:41.689677Z；Go 1.25.1 windows/amd64；GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local。执行期间输入未变，stderr 为空。精确命令/环境见 [run.json](run.json)，事件见 [stdout.jsonl](stdout.jsonl)。

[inputs.json](inputs.json) SHA256：`322623c3f1c156d61fc37da3033c7efe78426c7f278473739fa7c66d8eaf0601`；冻结 [approved-plan.md](approved-plan.md) SHA256：`c90d618ad9ce9f11e28ffa4433546a3904b1a3a72f44dc90cca02dd660593dd0`。原始证据保持不变。

证明临时 Store 中：未接受进展不登记知识作业；构造的接受事实可登记提取及汇总，重开 Store 不重复，后续进展复用原提取；部分草稿可持久发布，提取完成后新草稿保留旧版本及明确无新增事实，不改变现有接受/Gate/交付状态。

边界：接受事实由 fixture 构造，能力/授权/盘点为替身；未执行完整 Coder/Tester/Runner 接受链、真实宿主/模型、提交中断或磁盘故障。完整 AC20/AC21/AX05、2.2 与 Gate 保留。
