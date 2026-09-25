# E06 提取与草稿验证结果

Task workflow-automation；wi-0012 / 2.2；实现来源 E06。用户正常终端执行一次，退出 0，三个测试及六个子场景全部 passed：

- TestKnowledgeExtractionRequiresExplicitNoNewEvidence（六个子场景）
- TestKnowledgePartialDraftPreservesCoverageAndHistory
- TestKnowledgeGeneratedVersionsPreservePriorReview

实际命令：

```text
go test ./internal/workflow -run ^(TestKnowledgeExtractionRequiresExplicitNoNewEvidence|TestKnowledgePartialDraftPreservesCoverageAndHistory|TestKnowledgeGeneratedVersionsPreservePriorReview)$ -count=1 -timeout=60s -vet=off -json
```

2026-09-25T04:17:41.972784Z 至 04:17:44.196415Z；Go 1.25.1 windows/amd64；GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local。执行期间输入未变，stderr 为空。精确参数与环境见 [run.json](run.json)，逐项结果见 [stdout.jsonl](stdout.jsonl)。

[inputs.json](inputs.json) SHA256：`47bacb1ddf3884462338968d38d28bee3acc8c95c7f9d2b11a0a72e5bf7241b2`；冻结 [approved-plan.md](approved-plan.md) SHA256：`1b80aa92a682185d14c2256e20be5ba501f5b57acaae52c48e8352f02b1c4a69`。原始证据不改写。

支持无新增必须有明确依据、部分草稿保留失败/运行中/缺失覆盖、草稿版本不可变，以及新正文/接受引用生成候选且保留旧确认。测试直接调用内存语义与发布转换，人工确认作为既有状态构造；不证明后台接受登记、重启登记去重、真实 Store 发布、宿主或模型执行。完整 AC20/AC21/AX05、2.2 与 Gate 保留。
