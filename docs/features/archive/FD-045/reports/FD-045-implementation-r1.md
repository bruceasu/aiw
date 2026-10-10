# FD-045 实现报告 r1

<!-- aiw-data: FD-045-implementation-r1.json -->


**状态：** Implementation Ready，等待独立 Reviewer  
**FD revision：** 9  
**工作树：** `.wt/FD-045`，分支 `feature/FD-045`

## 实现摘要

Gateway 后端已切换到 Codex App Server stdio JSON-RPC。新实现按 `global_concurrency` 建立惰性启动的有界进程池，每个请求使用新的 ephemeral thread 和独立临时 cwd；健康进程可复用。每次借用池槽都会执行 effective config 与 MCP 状态 preflight。进程启动时通过明确的 feature overrides 关闭本地执行工具、apps、plugins、hooks、web、浏览器/电脑、图像生成和多 agent 能力。MCP 启用或状态不明、协议异常及不确定执行状态都会 fail closed，且不自动重试。

Responses 映射保留文本、无工具文本 SSE、`json_object`、顺序 function call 及后续 function call 历史重放。`strict:true` 参数通过有界 JSON Schema 子集校验；输出经 schema、工具名 allowlist 和 JSON object 检查后才生成 Responses `function_call`。turn usage 仅取当前 turn，缺失或无效时保持 unknown。quota 仅在 `turn/start` 成功响应后记录 started；请求状态不明时保留 reservation。

认证复用既有 `CODEX_HOME` 和登录状态，不复制凭据、不修改用户配置。进程 stderr、stdout/JSONL 行与通知队列均有上限；取消/超时发送 `turn/interrupt` 并淘汰该池槽。Gateway shutdown 会并行回收池进程；树清理失败会保留 state lock 并返回错误。

## Work Item 与提交

| Work Item | 结果 | 提交 |
|---|---|---|
| 1.1–1.2 | 已完成：锁定版本化协议来源、Responses 映射与拒绝矩阵 | 计划提交 `891156b` |
| 1.3 | 已完成：有界 JSONL、request ID 路由、通知和 server request 分类 | `2473fb6` |
| 1.4–1.5 | 已完成：受限进程启动、认证环境复用、initialize 与 MCP/config preflight | `ba8ea4a`, `e7e5e09`, `b1619e2` |
| 1.6–1.8、1.11 | 已完成：ephemeral thread/turn、取消、终态/usage、Responses 文本和 function call 映射 | `204e4fd` |
| 1.9 | 已完成：strict function 参数 schema 子集和边界校验 | `678086b` |
| 1.10 | 已完成：真实 App Server 文本 delta 的 SSE 映射 | `48b12c8` |
| 1.12 | 已完成：有界池、启动失败替换、坏槽淘汰和安全 preflight | `b1619e2` |
| 1.13 | 已完成：取消、进程树清理和 shutdown 回收 | `204e4fd`, `83fd948` |
| 1.14 | 已完成：稳定规格、README、FD Verification 与本报告 | 本报告提交 |

## 静态证据与验证

- 静态检查覆盖 JSONL line/total 限制、JSON-RPC ID 匹配、initialize 顺序、config/read 与 `mcpServerStatus/list` fail-closed、thread/start 和 turn/start 参数、interrupt、终态与 usage、工具参数 schema、Responses item/SSE、quota reservation、池并发和 shutdown 清理。
- compile-only：`python -B scripts/compile.py` 在 `program/agent-gateway` 执行并通过。首次通过后，进程清理错误处理有源码修正；按规则重跑一次后仍通过。
- 未运行 Go tests、最终构建、formatters/linters、Codex SDK/App Server、真实账户请求或其他运行时检查。未下载依赖。

## 未验证事项与剩余风险

- 编译不验证安装中的 Codex 版本、App Server 双向 RPC 实际行为、登录状态、MCP 状态响应形状或 feature override 的运行时生效情况；这些运行时验证仍需单独授权。
- 复用现有 `CODEX_HOME` 保留了用户认证材料。显式关闭工具并做 MCP preflight 限制后端能力，但 Codex 本身的 ephemeral thread 不代表进程内存或 rollout 数据必然清零；目标版本的持久化/清理行为仍需运行时证据。
- feature flag 和 JSON-RPC 字段根据 FD 锁定的 Codex `rust-v0.160.1` 协议源码静态对照。部署版本不匹配时会出现 preflight/协议失败并拒绝请求。

## 独立复审

尚未执行。此报告是 Worker 实现证据，不代表 Reviewer 已通过或 FD 已完成归档。
