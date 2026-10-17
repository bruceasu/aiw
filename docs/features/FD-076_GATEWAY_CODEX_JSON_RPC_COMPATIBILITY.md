# FD-076: Gateway Codex JSON-RPC compatibility

**Status:** Design
**Revision:** 2
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户通过本地 gateway 调用 `aiw say --model gpt-6-luna`，请求在模型执行前返回 HTTP 502 / `app_server_preflight`。
本会话已用实际配置、相同特性覆盖参数和 Codex 0.162.1 完成一次初始化采集：响应仅包含 `id`、`result`，没有 `jsonrpc`；四个初始化必需字段均存在。当前解析器强制要求 `message.JSONRPC == "2.0"`，因此拒绝这一响应。

## Options and decision

选择兼容 Codex 省略协议版本字段的消息，同时严格检查显式版本。
替代方案：更换 Codex 版本需要额外环境操作，且未解决解析器兼容性；完全删除版本检查会接受显式错误版本，故不采用。
这是内部协议解析修复，不更改 HTTP API、权限、持久化或并发策略，无需额外独立设计审查；实施后仍必须由独立 Reviewer 审查。

## Solution

在 `appRPC.readLoop` 中区分字段缺失与字段存在：缺失 `jsonrpc` 可以继续解析；存在时必须是字符串 `"2.0"`。显式空字符串、null、其他版本或错误类型必须拒绝。
保留消息结构、响应 ID 匹配、服务端请求拒绝、JSONL 长度和缓冲上限等原有检查。
初始化成功不代表后续 MCP 检查或模型执行成功，不扩展本 FD 去绕过后续安全检查。

## Scope

范围仅含 `src/programs/agent-gateway/appserver.go` 的兼容解析及本 FD 的交付证据。
不增加通用日志系统、不改变错误码、不修改用户配置或 README 既有修改，不创建测试，不更新依赖，不替换 `C:/green/aiw` 中程序。
稳定能力要求没有变化，`openspec/specs/agent-gateway/spec.md` 保持不变；无需新增 Issue 或 OpenSpec change。

## Work items

- [ ] 1.1 修复可选 `jsonrpc` 字段兼容性，保留显式版本及现有协议防护；完成静态追踪、compile-only，并提交双格式实施报告。规模：小；难度：低；依赖：计划提交且父工作区干净后创建专用 worktree；完成证据：实现 diff、编译结果、独立审查。

## TODO

- 父分支 `develop` 存在用户已有 README 修改，先解决工作区门禁，再提交计划并使用 `aiw git wt add FD-076`。
- 专用分支 `feature/FD-076`，工作区 `.wt/FD-076`；完成实现、独立审查、本地 squash 交付、清理与 Complete 归档。

## Acceptance

1. 缺失 `jsonrpc` 且消息结构有效的响应或通知可以继续原有分派流程。
2. 显式 `jsonrpc: "2.0"` 保持兼容；显式错误值、null、空字符串或非字符串被拒绝。
3. 非法 JSON、无响应 ID、未知或乱序 ID、意外服务端请求继续走原有失败路径。
4. 不改变初始化字段校验、MCP 预检、限流、配额、安全沙箱与进程清理行为。

## Verification

- 已有运行证据：本会话一次受限 `initialize` 采集，返回键为 `id/result`，无 `jsonrpc`，四个必需字段均非空；未发送模型请求；诊断进程树清理退出码 0。这是修复前证据，不作为修复后通过结果。
- 实施后计划运行一次 `python src/programs/agent-gateway/scripts/compile.py`（在专用 worktree 中）。该脚本使用离线 Go 配置，将 Windows/Linux 编译输出写入空设备，不保留分发产物。
- 静态核对字段缺失/显式值分支及原有消息校验、ID 路由和安全调用路径，最多一次静态验证命令。
- 不运行测试、端到端调用、联网检查、格式化、lint 或最终产物构建；既有握手授权不扩大为本 FD 的额外运行预算。
- 尚未实现、编译或独立审查，不宣称问题已修复。部署程序版本及后续 MCP/模型执行仍未验证。

## Sources

- Issue: none

- 用户日志及本会话诊断采集（2026-10-17，Asia/Tokyo）。
- `src/programs/agent-gateway/appserver.go`：启动、握手和 `readLoop`。
- `src/programs/agent-gateway/app_runner.go`：预检失败映射。
- `openspec/specs/agent-gateway/spec.md`：Bounded Codex App Server backend。
- 阻塞记录：[父工作区未清理](reports/FD-076-blocker-parent-dirty-c81d.md)。
