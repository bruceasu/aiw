# FD-049 独立审查 r2

<!-- aiw-data: FD-049-review-r2.json -->

结论：静态验收通过，无剩余阻断发现。Reviewer 会话 `fd049-reviewer-20261010-a613` 已 claim `FD-049-000007-implementation-ready`；审查 HEAD `be852f2`、计划基点 `f653509`，本轮重点差异 `a3e5b23..be852f2`。r1 报告保留原始请求修改结论。

## R1 修复复核

`chat.go` 在 typed decode 后对顶层原始键精确允许 model/messages/stream/n，并在处理指令或对话前对每条消息原始键精确允许 role/content。大小写别名及与标准键同时出现的覆盖组合均在返回内部 Request 前拒绝。原有 exact-key 重复检测、UTF-8/尾随 JSON 检测仍先执行。两次 messages 解码使用同一经白名单验证的数组，索引长度对应；null 消息仍因空内容拒绝，字段 null 显式拒绝。R1 已解决，Responses decoder 未变。

## 验收依据

- Say 的 model/system/user/stream:false 输入与单 assistant 字符串 choice/stop 读取匹配。
- 支持字段/角色、指令顺序、null、stream/n、未知字段和模型及大小边界均在模型启动前由 decoder/共享 handler 校验。
- Chat 与 Responses 共用单次授权、acquire、executeApp、Reserve、取消/清理，无新增重试；Responses 成功对象及 SSE 分支保持原代码。
- 已知 usage 重命名，nil 保持 null；失败沿用共享错误分支。response ID 与审计 ID 共用随机后缀。
- route loader 扩展并继续接受旧 route；文档明确旧二进制可能拒绝新 route，未声称可直接降级。
- Work Items 1.1–1.3 有代码、稳定 spec、接入文档及 Dual Worker 证据；FD-048 上游代理和完整计费不属于本 FD 验收。

## 命令与限制

本轮实际运行 `aiw fd show FD-049`、`aiw fd claim FD-049 FD-049-000007-implementation-ready --session fd049-reviewer-20261010-a613`、Get-Content 定向读取、`git diff --stat a3e5b23..be852f2`、`git diff a3e5b23..be852f2 -- ...`、`git diff f653509..be852f2 -- program/agent-gateway/server.go program/agent-gateway/observation.go`。报告编辑后运行一次 `git diff --check`。

Worker 报告两次 compile-only（源码修正后一次）退出码均为 0；Reviewer 未重跑。未运行测试、HTTP/Say/App Server、SDK、最终构建、格式化/lint/vet 或网络。实际接口、Codex 安装/登录、RPC 兼容性仍未经运行验证；本次通过仅表示修订版符合 FD 已批准的静态验收范围。
