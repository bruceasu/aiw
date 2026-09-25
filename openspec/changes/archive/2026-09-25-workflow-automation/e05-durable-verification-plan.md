# E05 持久队列、恢复与 memory 验证计划

Task workflow-automation；选定工作项 wi-0012 / 2.2；实现来源 E05 / wi-0007。状态：用户正常终端执行三项全部通过，退出 0，见 [结果](verification-results/e05-durable-terminal-20260925T013046528023Z/report.md)。

依据 task-memory/spec.md 的 SW09/SW18/SW19、R1 来源账及 R3 持久累计，新增 Windows 用例 auxiliary_durable_windows_test.go。使用真实临时 Store/系统锁/持久文件，复用 durableFixture；能力、授权、容量和初始盘点是明确测试替身，不是生产启用证据。观察结果先保存不可变工件，再复核内容一致性；不调用模型、网络、真实后台宿主或当前项目的 Core 存储。

| 用例 | 覆盖 |
| --- | --- |
| TestAuxiliaryRecoverySurvivesRestartAndConsumerReuse | 未知结果保留项目槽和全额预约；重开 Store 不允许重复派发；终态重放不再提交；仅一次追加恢复，另一个消费者复用原来源账和赞助者，不能获得第三次调用；未知 usage 保留累计 |
| TestAuxiliaryQueueLimitPreservesSourceCursor | 队列已满时新实质来源仍保留，游标不推进，不新增队列预约或派发；重开 Store 后仍成立 |
| TestAuxiliaryLateMemoryPublicationAndStop | 迟到旧摘要保存为历史，当前视图回退到原始事实；新版本摘要可发布和投影，重复发布不新增提交；Stop 后不允许新派发，重开 Store 保留 Stop |

## 执行边界

一次执行，预计 1–3 分钟：

```powershell
python C:\Users\svictor\workspace\tools\aiw\.wt\workflow-automation\openspec\changes\workflow-automation\run-e05-durable-terminal.py
```

固定命令：

```powershell
go test ./internal/workflow -run '^(TestAuxiliaryRecoverySurvivesRestartAndConsumerReuse|TestAuxiliaryQueueLimitPreservesSourceCursor|TestAuxiliaryLateMemoryPublicationAndStop)$' -count=1 -timeout=60s -vet=off -json
```

GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local；编译同包测试，但仅运行以上三项。输入、计划、命令、环境、工具链和原始输出按独立目录保存，不自动重试。若工具环境再次发生此前已知的 Windows 路径权限失败，保留结果，交由用户正常终端执行；不改安全检查、提权或换壳重试。

授权来自本组用户确认或显式执行；不生成产品 Runner grant、不清除 Gate、不操作实际 Task 租约。新测试运行前，不把生产 compile.py 的成功作为测试源码通过证据。

## TODO 与 Verification

2026-09-25：离线执行 `python scripts/compile.py`，退出 0，临时编译产物由脚本清理。仅编译生产代码，不覆盖新增 `_test.go`；三项测试尚未执行。使用直接补丁回退，沿用既有 aiw patch 内部 Git apply 不可用的限制，不执行 Git 写入。

- [x] 按已批准规格编写三项持久化用例及单次取证脚本。
- [x] 记录本组实际运行结果与输入版本：e05-durable-terminal-20260925T013046528023Z，三项 passed、退出 0；未改写冻结计划和原始输出。

%% REMAINING: 尚未验证模型重调与保存失败共用恢复额度、进程崩溃窗口、真实宿主/模型能力、全部资源硬上限、赞助 Task Stop 及原始工件损坏。即使本组通过，也不等于完整 E05、AC09/18/19/27 或 AX02/04 验收。
