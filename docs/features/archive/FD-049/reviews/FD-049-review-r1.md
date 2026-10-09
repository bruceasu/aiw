# FD-049 独立审查 r1

<!-- aiw-data: FD-049-review-r1.json -->

结论：请求修改。Reviewer 会话 `fd049-reviewer-20261010-a613`，已 claim `FD-049-000005-implementation-ready`。本轮审查基点 `f653509`，Worker HEAD `e72fc27`，检查全部十个文件差异及相关调用链。

## 发现

- **R1，中等，阻断：Chat 字段白名单未精确匹配。** `program/agent-gateway/chat.go:27` 使用 `strictDecode` 解码 struct；`program/agent-gateway/protocol.go:73` 的 `DisallowUnknownFields` 仍允许 Go JSON decoder 不区分大小写匹配标签。因此 `MODEL`、`Messages`、消息内 `ROLE`、`Content` 会被接受；同时出现 `model` 和 `MODEL` 还能绕过 `rejectDuplicates` 的原始 key 比较并覆盖模型。FD Solution 与 Acceptance 2，以及稳定 spec 要求仅接受明确列出的字段，并在执行前拒绝未知/重复字段。请在 Chat 顶层及每个消息对象按原始 JSON key 校验精确白名单，拒绝大小写变体；保持 Responses 解码契约不变。

## 其他静态证据

- Say 仅发送 model/system/user/stream:false；新成功对象的单 choice、字符串 content、stop 与其读取方式一致。
- null 顶层字段被显式拒绝；消息 null/content null/role null 因空字段检查拒绝。流式、n 非 1、工具、数组内容、不支持角色和对话后的指令被拒绝。映射复用 decodeRequest 的 64 KiB 总量及既有 1 MiB HTTP 主体限制。
- 两个接口仅切换 decoder/最终对象，共用一次 acquire、executeApp、Reserve、取消和清理；Responses SSE 分支和成功对象保持原代码。异常由共享失败分支返回，不伪造成功。
- 已知 usage 直接重命名，nil 经 interface 零值编码为 null。新 route 纳入 loader 枚举；文档明确旧二进制降级可能拒绝新记录。
- 修改未涉及 FD-048 上游代理、依赖或新增计费。Worker 报告记录 compile-only 通过，脚本使用离线配置及系统空设备；Reviewer 未重跑编译。

## 命令与限制

实际执行了 Get-Content/rg 定向读取、`git diff --stat f653509..e72fc27`、`git diff f653509..e72fc27 -- ...`、`git status --short`、`aiw fd show FD-049`、`aiw fd emit --help` 与上述 claim。首次误用 `aiw fd status FD-049` 返回无效子命令，已改用 show；未影响仓库。

未运行测试、SDK/App Server/Say/HTTP、编译、格式化、lint/vet、最终构建或网络下载。实际接口、Codex 安装与登录及 RPC 运行行为仍未验证；这些不是本轮另加的验收条件。
