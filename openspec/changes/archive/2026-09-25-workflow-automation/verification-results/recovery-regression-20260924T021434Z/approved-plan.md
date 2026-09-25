# 恢复路径修复与聚焦验证计划

范围：workflow-automation 的 wi-0012 验证支持；只处理已有恢复故障，不改变 E01–E08 需求或当前运行授权。

## 实现

前次修复已让 RepairWorkflowChecklist 传递 DependsOn。本次为其增加真实 Store/清单边界的回归用例，确认清单依赖、稳定 ID、验收顺序和一次性修复语义保留。

直接 turn/takeover 的 startManagedAttempt 以前未刷新清单即选择新工作项。本次保留既有 prepared/running Attempt 的复用路径，只在选择新的已映射工作项前调用现有 SyncWorkflowChecklist，读取 Task 绑定工作区的当前清单；同步失败返回错误，不猜测工作项。增加回归用例，模拟“验收项先创建、后补实现依赖”，断言真正获得写租约的是实现项。

## 待批准的测试命令

工作目录为既有 worktree，禁用 GOPROXY/GOSUMDB 和工具链下载，缓存使用 worktree/.ai/compile-cache/go。

```text
go test ./internal/taskx ./internal/commands/task -run ^(TestRepairWorkflowChecklistPreservesAuthoredDependencies|TestStartManagedAttemptRefreshesAuthoredDependencies)$ -count=1 -timeout=60s -vet=off -json
```

只运行两个指定回归测试，不运行两个包的其他测试。用例在 t.TempDir 中建立 Task/Store/清单，不调用 Git、模型或外部网络，不修改真实 Task。Go 会编译两个包的测试源码；预计 1–3 分钟，60 秒超时只覆盖测试阶段。当前状态：尚未取得扩大至此命令的授权，尚未执行；这不是产品 Runner grant。

本轮实现后按仓库规则执行一次 compile-only。安装版 AIW 不由编译自动更新，不构建或安装最终制品，不启动 Supervisor，不迁移真实 Task。原两个 E04 用例已有成功证据，本次不因修改了无关恢复代码而重复运行它们。完整 AC/AX、联合启用和验收仍未完成。

## 实际验证

2026-09-24 已执行一次 `python scripts/compile.py`，GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local，退出码 0；脚本清理临时可执行文件。一次编辑后静态检查核对了 startManagedAttempt 的复用/同步/选择顺序、repair 与 sync 均传递依赖以及两个回归用例位置，确认 tasks.md 保留 12 个正式编号且没有重复。没有运行新的回归测试、格式化、vet、网络、Git 或最终制品构建。compile-only 不编译 `_test.go`，因此两个新测试的编译及运行结果仍未知。测试授权问题已向用户提出，未得到答复前保持未执行。
