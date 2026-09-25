# E02 迁移分步错误诊断

Task：workflow-automation；Work Item：wi-0012 / 2.2；2026-09-24。

用户确认继续补足 MigrateDurableExecution 内部错误上下文。变更位于既有 worktree 的 internal/workflow/protocol_store.go。

## 修改

为十四处下层错误返回增加具体步骤：服务检查、取锁、源文件读取、已迁移状态加载、旧状态加载、激活工件读取、激活核验、来源工件保存、预算迁移、候选状态校验、待提交状态保存、事件追加、已确认状态保存和最终状态加载。

使用 fmt.Errorf 的 %w 保留原始错误链，便于 errors.Is/errors.As 识别；激活工件错误附上既有相对路径，不输出工件正文。保留返回状态、调用次数、执行顺序、schema/预算规则及提交协议，不增加重试或异常降级。错误文本现在包含步骤前缀；依赖原始错误字符串完全相等的外部消费者可能受到影响。

## Verification

实际执行离线 `python scripts/compile.py`，GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local，退出 0；临时产物由脚本清理。通过直接补丁修改源码，沿用已知 Git 补丁路径受阻时的回退方式；未执行 Git 写入。

本轮未运行测试、网络、最终制品构建、格式化或迁移真实 Task。此前两次测试均失败，其输入早于本诊断改动；本次编译不表示 Access is denied 已解决。代码只改善定位信息，运行根因仍未确认。

下一次有明确授权时，在同一 worktree 和离线环境仅执行原三个用例一次，预计 1–3 分钟，保留新输入快照和输出，不扩大范围或提权：

```powershell
go test ./internal/workflow -run '^(TestDurableUnknownAndStopRetainWriter|TestDurableResultReplayConsumesOnceAndKeepsAttempt|TestDurablePendingTailRecoveryKeepsRevision)$' -count=1 -timeout=60s -vet=off -json
```

%% BLOCKED: 新诊断尚未取得运行证据，不能确定迁移内部哪个步骤被拒绝；Gate 及 2.2/3.1/3.2 保持未完成。不得凭诊断代码编译通过解除授权或验收约束。
