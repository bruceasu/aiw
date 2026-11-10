# FD-115 独立审查报告

- **结论：** Changes requested
- **Reviewer session：** `reviewer-fd115-20261111-01`
- **Source event：** `FD-115-000003-review-requested`
- **审查提交：** `f8dc46f`（基于 `4e36b87`）
- **工作区：** `.wt/FD-115`，分支 `feature/FD-115`

## 发现

1. **P2 — AIW 桥接的 stdout 上限小于声明的 Provider 文本上限。** [`src/internal/commands/aiprovider/command.go:106`](../../../../src/internal/commands/aiprovider/command.go) 允许 `text` 到 4 MiB，随后 JSON encoder 将其写入 stdout；[`src/plugins/aiw-ai/aiw-ai.mjs:351`](../../../../src/plugins/aiw-ai/aiw-ai.mjs) 的 `spawnSync.maxBuffer` 却只有 4 MiB + 64 KiB。JSON 会转义换行和控制字符，序列化结果可明显大于原文本。例如，低于 4 MiB、含大量换行的 Provider 响应通过 Go 字节上限后，仍可能被 Node 以 `ENOBUFS` 拒绝。请让子进程缓冲上限覆盖 4 MiB 文本的最坏 JSON 膨胀及元数据，或统一并明确序列化后的有效上限。

2. **P2 — “65536 个字符”被按 UTF-16 code unit 计数。** Node 在 [`src/plugins/aiw-ai/aiw-ai.mjs:177,200,420`](../../../../src/plugins/aiw-ai/aiw-ai.mjs) 使用 `String.length` 检查 system/prompt；Go 桥接则在 [`src/internal/commands/aiprovider/command.go:53,56`](../../../../src/internal/commands/aiprovider/command.go) 使用 Unicode rune 数。比如 40,000 个 emoji 是 40,000 个 Unicode code point，UTF-8 文件约 160 KiB，低于文件和字符上限，却会因 JS 长度为 80,000 而被拒绝。请让 Node 与 Go 按同一种字符定义计数，并保持规格中的 65536 字符上限一致。

## 已覆盖内容

静态检查确认默认路由为共享 AIW Provider、显式 Gateway 及仅 `--url` 的兼容选择，且失败不跨路由回退；Gateway 配置只在 Gateway 分支解析。Go 桥接使用共享 Provider 配置、只读请求选项和超时 context，并限制 stdin、未知字段、尾随 JSON 和响应文本。文件输入在本机按严格 UTF-8 或指定 TextDecoder 标签解码，限制原始字节数、与位置提示词合并并在请求前校验；文件路径不进入 Provider 请求。README、manifest、环境变量文档及相关稳定规格与主要实现方向一致。未发现超出 FD 范围的源码改动。

## 证据与限制

- 已读取批准的 `ISSUE-00013`、FD-115 验收与 Worker Verification、两份相关 OpenSpec 稳定规格、实现 diff，以及 Node/Go Provider 调用链和错误路径。
- Worker Verification 记录 `python scripts/compile.py` 与 `node --check src/plugins/aiw-ai/aiw-ai.mjs` 成功；这是 Worker 记录，本 Reviewer 未运行这些命令。
- 本 Reviewer 实际命令：`aiw fd show FD-115`；`aiw fd claim FD-115 FD-115-000003-review-requested --session reviewer-fd115-20261111-01`；`aiw fd emit --help`；`git status --short`、`git log -4 --oneline`、`git show`、`git diff`；针对 FD、Issue、稳定规格与 Node/Go 调用链的 `rg` 和 `Get-Content` 静态读取。
- 本 Reviewer 未运行测试、compile/build、Node check、真实 Provider/Gateway 请求、格式化、lint、vet 或验证脚本。运行时 Provider 行为及 Windows/Unix 实际子进程边界未验证。

两个发现均涉及已写入稳定规格的边界行为，当前不能通过本次 Verification；修复后应针对这两个边界重新审查。