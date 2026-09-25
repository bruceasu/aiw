# E07 通知状态机聚焦验证计划

Task workflow-automation；登记 wi-0012 / 2.2；实现来源 E07 / wi-0009。状态：用户正常终端首跑四项及五个子场景全部 passed，退出 0，见 [执行报告](verification-results/e07-terminal-20260924T093425416837Z/report.md)。依据 design.md、R4 和 specs/workflow-notifications/spec.md；不启用生产通知，不改变 E05 宿主依赖。

选择 internal/workflow/notification_test.go 的四个已有用例：

| 用例 | 验证边界 |
| --- | --- |
| TestNotificationNetworkRetryKeepsDeadlineAndMessageBudget | 首次网络失败至少等 60 秒；重启保留期限；总共最多两次，独立消息预算 |
| TestNotificationOtherResultsNeverRetry | 未知、权限、服务、配置及 attempted 不自动重发，也不冒称送达 |
| TestNotificationDisabledAndFrozenTarget | 缺配置不消耗次数；冻结目标不被新配置替换；禁用后不补发历史完成通知 |
| TestNotificationInterruptedResultIsUnknown | 已预约但结果中断保持 unknown，不释放次数或重复派发 |

使用临时 Store 与 notificationDispatcherStub；修改临时保存的到期时间代替等待，无真实 60 秒 sleep。替身不执行 aiw-notify、send_teams_msg.py、网络、console 外发或模型；不读取真实凭据/配置。

## 命令与取证

操作员在正常终端显式执行一次：

```powershell
python C:\Users\svictor\workspace\tools\aiw\.wt\workflow-automation\openspec\changes\workflow-automation\run-e07-terminal.py
```

脚本固定运行：

```powershell
go test ./internal/workflow -run '^(TestNotificationNetworkRetryKeepsDeadlineAndMessageBudget|TestNotificationOtherResultsNeverRetry|TestNotificationDisabledAndFrozenTarget|TestNotificationInterruptedResultIsUnknown)$' -count=1 -timeout=60s -vet=off -json
```

预计含编译 1–3 分钟，测试阶段限时 60 秒；GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local，Go 编译同包测试源码但只运行指定四项。环境和代码不变时不重跑；相关修复后最多同范围重跑一次。权限失败停止对应路径，不提权、扩范围或绕过检查。E01/E02/E03 的批准不自动扩展到本组。

独立目录保存计划快照、输入和脚本摘要、argv/cwd、工具链/环境、时间、stdout/stderr、退出码及输入前后比较。只由操作者显式调用脚本执行，不自动重试、生成产品 grant、创建 Attempt 或解除 Gate。

## 覆盖与缺口

提供 SW28/SW29/SW30、AC28/AC29/AC30 及按 R4 修订 AX04 的局部状态机证据；不证明实际插件协议、TLS/HTTP 分类、Teams 发送、真实进程中断或并发宿主行为。缺少 E05/模型能力不妨碍替身单元验证，但不能据此启用生产通知。

%% REMAINING: 本组局部状态机测试已通过；实际适配器、宿主与完整 AC/AX 验收仍待证据。2.2/3.1/3.2 与整体 Gate 保留。
