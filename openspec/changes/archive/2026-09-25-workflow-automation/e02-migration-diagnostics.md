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

后续实际结果：用户明确批准后执行一次，三个用例均定位到 migration: read activation artifact，退出码 1。详见 [迁移内部诊断结果](verification-results/e02-recovery-20260924T025737Z/migration-diagnostic-run/report.md)。上述“本轮未运行测试”为诊断代码实现时的记录。

最新运行已确认根目录 EvalSymlinks 返回 Access is denied；默认 Temp 与 worktree 临时目录都失败，见 [根路径诊断及正常终端取证方式](verification-results/e02-recovery-20260924T025737Z/path-diagnosis-report.md)。

最新正常终端结果：[一项通过、两项 fixture 失败](verification-results/e02-terminal-20260924T073650196536Z/report.md)。路径拒绝在用户终端未复现；已补齐冻结输入的模型选择，等待修复后的原三项运行结果。

修复后结果：用户正常终端重跑原三项全部 passed，见 [通过报告](verification-results/e02-terminal-20260924T074441498470Z/report.md)。本次 fixture 修复与局部恢复取证已完成；以上待运行描述作为历史记录保留。

%% REMAINING: 工具环境路径权限限制仍存在；完整 AC/AX 和生产联合启用尚有证据缺口，Gate 及 2.2/3.1/3.2 保持未完成。局部三项通过不等于整体验收。
