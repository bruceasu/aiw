# E02 持久恢复聚焦验证计划

Task：workflow-automation；工作项：wi-0012 / authored 2.2。
状态：待本命令的明确执行授权。既有 E04 和调度恢复授权不扩展到本轮。

## 输入与范围

工作目录：`C:/Users/svictor/workspace/tools/aiw/.wt/workflow-automation`。
依据：design.md 的 R1、specs/workflow-supervision/spec.md 的请求归属、阶段写入权与 Stop 契约，以及 internal/workflow/protocol_windows_test.go。

| 用例 | 本轮判据 | 证据局限 |
| --- | --- | --- |
| TestDurableUnknownAndStopRetainWriter | 未知派发拒绝第二执行器；重新打开 Store 后保留 Stop、请求写租约和 revision；旧 supervisor 不绕过 Stop | 不是进程崩溃或真实子进程终止测试 |
| TestDurableResultReplayConsumesOnceAndKeepsAttempt | 同一结果重复消费不增加 revision/预算；Coder 结果释放阶段写权但不关闭 Attempt/整项；拒绝另一 turn 覆盖结果 | 授权、接受及预算服务使用测试替身，不证明 E01–E04 生产联合启用 |
| TestDurablePendingTailRecoveryKeepsRevision | 半条事件尾部恢复完成原提交；重复恢复不再次推进 revision | 构造截断文件，不证明断电、磁盘介质或每个持久化断点 |

为 SW05/SW06/SW08 和相关 AC 提供局部证据；不能据此宣称完整 AC/AX 通过。

## 精确命令与预算

```powershell
$env:GOPROXY = 'off'
$env:GOSUMDB = 'off'
$env:GOTOOLCHAIN = 'local'
go test ./internal/workflow -run '^(TestDurableUnknownAndStopRetainWriter|TestDurableResultReplayConsumesOnceAndKeepsAttempt|TestDurablePendingTailRecoveryKeepsRevision)$' -count=1 -timeout=60s -vet=off -json
```

预计含编译 1–3 分钟；60 秒为测试阶段限时。只运行以上三个用例，Go 会编译包内其余测试源码。允许本地临时目录、Go 缓存和证据工件写入；不下载依赖、不调用模型/Teams、不启动 supervisor、不迁移真实 Task、不执行 Git 交付。关闭 Go 下载不等同于证明全局网络隔离。

执行一次；相关修复或环境变化后最多同命令重跑一次。扩大范围另行授权。

## 取证与完成边界

授权后先保存本计划快照及会话授权，记录 go.mod/go.sum、包内生产与测试源码、适用设计/规格的摘要、工具链、工作目录、实际 argv 和环境限制。保存起止时间、退出码、完整 JSON 输出、stderr 与执行前后输入摘要。若失败，保留首跑结果，不覆盖历史。

会话授权不能冒充产品 Runner 的 grant.md。通过受管 Runner 时另满足其精确计划与 grant 契约；本地结果如实作为会话授权的局部证据保存，不修改授权 Gate 来绕过 Core 记录前提。

运行后更新 tasks.md、coverage-review.md 和结果引用。只有完整当前版本证据足以满足 2.2 时才推进该项，随后处理 3.1、3.2；本轮三个用例通过本身不解除整个 Gate。

%% REMAINING: E01–E04 联合启用、真实平台故障恢复、E05–E08 能力与通知/模型集成、其他 AC/AX 仍须分别取得适用证据；不凭测试替身开启 schema 10。
