# FD-112 关停规范聚焦复审（r3）

<!-- aiw-data: FD-112-review-r3.json -->

## 结论

**此前 residual finding 已解决，发出 `verification-passed`。** Stable spec 移除了 App Server 进程树清理语义，同时保留 Gateway 当前关停流程需要的请求取消、HTTP 关闭、代理连接关闭、内容清理等待和状态锁释放语义。前两项文档修正仍然准确。

Reviewer session：`fd112-review-shutdown-fix-20261108`。来源事件：`FD-112-000011-implementation-ready`。FD revision 11。

## 核对结果

- **Shutdown spec：** `Local graceful shutdown control` 现要求取消活动后端请求、优雅停止 HTTP 处理、关闭活动代理连接、等待定期内容清理退出并释放状态锁；停止 CLI 仍要求有界等待、不跟随重定向、不重试，只有监听器停止且锁释放后才报告成功。该段不再要求进程树清理，也未恢复 App Server 进程池、JSON-RPC、MCP、sandbox 或 workspace 生命周期要求。
- **调用链与实现一致：** `server.go` 为直接后端请求把服务 context 取消关联至执行 context；proxy 请求 context 同样关联服务取消。`main.go` 关停时调用 HTTP `Shutdown`、关闭追踪的代理连接、等待内容清理 goroutine 结束；正常返回后通过 deferred Store close 释放状态锁，stop 命令等待锁释放。源码相较上一轮无变化；此次只核对相关调用点，没有执行运行验证。
- **development/shared：** Deployment listener and peer boundaries 仍单独规定 development 只使用 loopback listener/peer，shared 必须显式配置 `listen`；不包含 App Server 生命周期约束。
- **README 持久化范围：** `openai_codex` 的 `state_dir/requests` 和本地 `/v1/usage` 统计范围明确；`openai_proxy` 明确不持久化 HTTP 请求元数据且无本地 usage 统计，与现存模式代码一致。
- **旧 finding：** r2 中的 process-tree cleanup finding 已被上述规范修订解决。README 与 deployment listener 两项 r1 finding 也继续保持已解决状态。

## Findings

无。

## 命令与未运行检查

仅运行本地只读检查：`aiw fd show FD-112`、`aiw fd claim FD-112 FD-112-000011-implementation-ready --session fd112-review-shutdown-fix-20261108`、`git diff --stat`、`git diff`（stable spec/README/FD）、`Get-Content`、`Select-String`。Worker 在 FD Verification 中记录 Windows/Linux `python scripts/compile.py` 成功；本 Reviewer 未重跑编译。

未运行测试、构建、编译、网络或运行时命令；未修改实现代码。依 Go 子树规则，未创建或运行 Go 测试。

## 剩余风险

配置迁移、旧进程 stop 与运行时关停行为未在本次复审中执行；本结论基于源码和规范静态一致性，以及 FD 已记录的 compile-only 结果。
