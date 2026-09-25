# E04 首轮聚焦验证结果

Task workflow-automation；登记项 wi-0012 / 2.2；范围 E04 的两个现有单元用例。

用户在本会话回复“confirm, continue”批准 [原计划](approved-plan.md)。[authorization.json](authorization.json) 保留精确范围及计划摘要，这是工程会话的用户授权记录，不是产品受管 Runner 的 grant。没有执行迁移、模型调用、通知或 Git 交付。

结果：**2 项通过，0 项失败，命令退出码 0**。测试输出报告包执行 0.059 秒；含编译与工具链记录约 5.86 秒（01:07:10.559481Z 至 01:07:16.419929Z）。Go 1.25.1，Windows/amd64。stderr 为空，输入前后摘要无变化。

| 测试 | 结果 | 证据边界 |
| --- | --- | --- |
| TestGenerationBudgetDeduplicatesDelayedDefectsAcrossRestart | passed | 内存状态 JSON 往返后的同一生成缺陷去重与第二次有效失败升级；不是真实进程崩溃或磁盘持久化验收 |
| TestGenerationBudgetReservationsAndEscalationLimits | passed | 预约/未知/确认未派发的计次边界、每 Actor 两次升级上限及 Actor 之间隔离 |

命令在既有 worktree 中执行：

```text
go test ./internal/workflow -run ^(TestGenerationBudgetDeduplicatesDelayedDefectsAcrossRestart|TestGenerationBudgetReservationsAndEscalationLimits)$ -count=1 -timeout=60s -vet=off -json
```

GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local；缓存位于 worktree/.ai/compile-cache/go。禁用下载，没有启用 vet；只运行这两个用例。参数数组、目录、宿主、起止时间、工具链及输出摘要见 [run.json](run.json)。完整输出见 [stdout.jsonl](stdout.jsonl)、[stderr.txt](stderr.txt)，源码、模块和相关规格/计划的版本清单见 [inputs.json](inputs.json)。未运行全包用例、完整 AC/AX、集成、故障注入或真实外部服务验证；没有重跑。

这两项仅提供 SW08/SW15/SW16 的部分单元证据。2.2 与 3.1 仍未完成。现有 authorization Gate 的范围涉及剩余验证，不能因有限授权和两项通过整体解除。Core 的 command Evidence 入口要求相关授权 Gate 已 resolved；本轮保留 Gate，先登记限定范围的 approval 引用，将实际测试证据保存在此处，不改用其他 Evidence 类型冒充已获完整 Runner 授权。后续扩大测试需新的明确授权。
