# E05 持久化聚焦结果

Task workflow-automation；wi-0012 / 2.2；实现来源 E05。用户正常终端执行 run-e05-durable-terminal.py，退出 0，以下三项全部 passed：

- TestAuxiliaryRecoverySurvivesRestartAndConsumerReuse
- TestAuxiliaryQueueLimitPreservesSourceCursor
- TestAuxiliaryLateMemoryPublicationAndStop

实际命令：

```text
go test ./internal/workflow -run ^(TestAuxiliaryRecoverySurvivesRestartAndConsumerReuse|TestAuxiliaryQueueLimitPreservesSourceCursor|TestAuxiliaryLateMemoryPublicationAndStop)$ -count=1 -timeout=60s -vet=off -json
```

2026-09-25T01:30:46.726239Z 至 01:30:49.328972Z；Go 1.25.1 windows/amd64；GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local。changed_inputs 为空，stderr 为空。精确参数、环境、输出哈希见 [run.json](run.json)，事件见 [stdout.jsonl](stdout.jsonl)。

输入清单 [inputs.json](inputs.json) SHA256：`5f126ca3dd93b6a4146541c4894aec59ef757380b4de2b31ef49e6082f3edd3d`；计划快照 [approved-plan.md](approved-plan.md) SHA256：`ee94291bcaafc01af874c973df27bafad299f2d87fce84c933683e058236acc6`。快照中的待执行状态保留为执行前记录。

覆盖临时 Store 重开后保留未知预约/恢复额度、跨消费者来源复用、队列满不推进游标、旧摘要迟到保留历史、新摘要投影及 Stop 拒绝新派发。能力、授权、盘点和容量是测试替身；没有模型/网络/真实宿主或实际项目迁移。

仍缺进程崩溃窗口、模型与保存失败共享恢复、赞助 Task Stop、原始工件损坏及全部资源边界。不代表完整 E05 或 AC/AX 验收，不生成产品 Runner grant，不解除 2.2 Gate。
