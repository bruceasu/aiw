# 恢复回归测试结果

Task workflow-automation；验证登记项 wi-0012 / 2.2。用户回复“confirm, continue”批准 [两个指定回归用例](approved-plan.md)。授权范围与计划摘要见 [authorization.json](authorization.json)，不是产品 Runner grant。

## 首跑与修复

首跑退出 1，两个目标用例均未执行，因为测试包编译失败：

- internal/taskx/checklist_test.go 的 range 错误地解构四个变量；改为 name、tc，并保留原来的内容、编号、错误类型和不修改清单的断言。
- internal/commands/task/archive_test.go 对 session.Store 调用了不存在的 SetDelivery；改为测试项目中 workflow.NewStore(".ai").SetDelivery，保留原有交付状态、元数据与归档断言。

首跑的 [run.json](run.json)、[stdout.jsonl](stdout.jsonl)、[stderr.txt](stderr.txt)、[输入摘要](inputs.json) 均保留，未用成功记录覆盖失败记录。

## 修复后的唯一重跑

只修改上述两处编译阻塞后，使用同一参数重跑一次，**退出 0，两个目标用例全部通过**：

| 用例 | 结果 | 验证内容 |
| --- | --- | --- |
| TestRepairWorkflowChecklistPreservesAuthoredDependencies | passed | repair 保留依赖及 Work Item ID，不提前选择验收项，重复 repair 不再产生转换 |
| TestStartManagedAttemptRefreshesAuthoredDependencies | passed | 直接创建新 Attempt 前读取更新后的清单依赖，正确选择实现项并绑定写租约 |

命令（工作目录为现有 worktree）：

```text
go test ./internal/taskx ./internal/commands/task -run ^(TestRepairWorkflowChecklistPreservesAuthoredDependencies|TestStartManagedAttemptRefreshesAuthoredDependencies)$ -count=1 -timeout=60s -vet=off -json
```

Windows/amd64，Go 1.25.1；GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local，缓存为 worktree/.ai/compile-cache/go。重跑从 02:15:57.454307Z 至 02:15:59.143303Z，约 1.69 秒。stderr 为空，运行期间记录的输入摘要无变化。完整元数据和原始结果见 [rerun-1/run.json](rerun-1/run.json)、[rerun-1/stdout.jsonl](rerun-1/stdout.jsonl)、[rerun-1/inputs.json](rerun-1/inputs.json)。

## 边界

本次验证编译了两个包的测试源码，但仅执行上述两个用例。修正编译错误的清单拒绝/归档用例本身未执行；未跑整包、完整 AC/AX、真实网络、通知、模型、迁移或 Git 交付。生产源码未在本轮再次改动；其主程序 compile-only 已于前轮通过，本次没有重复最终构建或安装可执行文件。

用例只在临时目录中调用现有 Store 与适配器，未派发真实 Agent，也不代表真实平台崩溃恢复与联合启用通过。wi-0012 的完整验证 Gate 和正式 2.2 保持未完成。Core 登记本次有限用户授权，运行结果在此保留；不通过伪造 grant 或更换 Evidence 类型绕过整个 authorization Gate。
