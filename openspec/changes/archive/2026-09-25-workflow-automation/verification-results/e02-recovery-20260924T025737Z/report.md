# E02 恢复验证：首跑失败

Task：workflow-automation；工作项：wi-0012 / 2.2。

用户批准 approved-plan.md 的三个指定测试。实际命令、输入摘要、环境、时间、退出码见 run.json；授权见 authorization.json；原始输出为 stdout.jsonl 和 stderr.txt。

```text
go test ./internal/workflow -run ^(TestDurableUnknownAndStopRetainWriter|TestDurableResultReplayConsumesOnceAndKeepsAttempt|TestDurablePendingTailRecoveryKeepsRevision)$ -count=1 -timeout=60s -vet=off -json
```

Go 1.25.1 windows/amd64；GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local；worktree 本地缓存。2026-09-24T02:57:38Z 至 02:57:39Z，退出码 1。三个测试均在 fixture 初始化阶段返回 `Access is denied.`，stderr 为空，运行期间已登记输入未改变。恢复行为断言尚未获得有效结果。

## 诊断与修改

原 fixture 使用 t.Helper 和没有步骤名的 t.Fatal(err)，输出只定位到调用 fixture 的测试行，不能区分 Store 创建、锁准备/获取、工件持久化和迁移。静态读取这些调用及 Windows 锁/替换实现，未能确定是哪一系统调用失败；也不能据此认定是沙箱限制或生产缺陷。

已在 worktree 的 internal/workflow/protocol_windows_test.go 为两个 fixture 的九处错误出口补充操作名称，保留原始错误、断言、锁及持久化实现。不跳过失败、不降级同步、不修改 ACL 或真实 Task。沿用直接补丁方式，未重试此前受阻的 Git 补丁路径。

修改后执行离线 `python scripts/compile.py`，退出 0；临时可执行文件由脚本清理。该命令仅编译生产代码，不编译或运行修改后的 _test.go，因此不证明诊断改动已经过运行验证。

## 诊断重跑：仍失败

用户另行回复 confirm，明确批准权限失败后的同环境、同命令诊断重跑一次。记录保存于 rerun-1/：authorization.json、inputs.json、run.json、stdout.jsonl、stderr.txt。命令与离线环境同首跑；2026-09-24T03:26:40Z 至 03:26:45Z，退出码 1，三个用例均返回 `migrate fixture execution: Access is denied.`，stderr 为空，执行期间输入摘要未变化。

新增的测试错误上下文已编译并运行。运行路径证实 Store.Create、PrepareDurableTaskLock、fixture 的锁获取和激活工件持久化均已成功，错误发生在 MigrateDurableExecution 返回处。静态追踪该方法包含再次取锁、读取/核验来源、保存迁移工件、持久状态与事件提交；现有错误未标明迁移内部具体操作，不能据此认定是 ACL、沙箱、锁竞争或同步写入缺陷。

本轮只登记结果和同步文档，没有修改生产协议或继续运行第三次测试。诊断修改后的运行结果仍为失败，不将成功进入迁移阶段作为恢复验收通过。

## 后续边界（首跑时的记录）

未执行测试重跑。AGENTS.md 要求权限失败后停止该路径，不尝试其他 shell、提权或更宽命令。本次是子进程返回的 Windows 错误，没有自动审批拒绝记录。

如明确授权权限失败后的诊断重跑，可在同一 worktree、相同离线环境下执行上述同一命令一次，以获取带步骤名的错误；预计 1–3 分钟，仍仅临时 Store，无提权、ACL 修改、网络或真实 Task 迁移。首跑结果保留，重跑另存。若仍被拒绝则停止该执行路径。

%% BLOCKED: 一次诊断重跑已用尽，失败缩小到 MigrateDurableExecution 内部，但具体系统调用及根因未确认。后续应先补足该方法内部的分步错误上下文，再另行决定是否运行；不重复无新增信息的命令，不绕过权限或关闭持久化保证。E02 恢复验收未通过；会话批准不替代产品 Runner grant，授权 Gate、2.2/3.1/3.2 保留。不能将编译通过或其他四个测试通过作为这三个测试通过的证据。
