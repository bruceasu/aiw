# FD-041 独立复审 R3

<!-- aiw-data: FD-041-review-r3.json -->

## 结论

**changes-requested**。实现静态上修复了 R2 的未知 TERM 后缀匹配问题，ANSI 列宽补偿也与实际起止转义串长度相符。仍有一项文档验收缺口：CLI 使用说明暗示任意 TTY（除 `TERM=dumb` 或设置 `NO_COLOR` 外）都会输出颜色，与当前 fail-closed 能力判定不一致。

## Finding

1. **低：颜色回退说明没有反映 allowlist 条件** — `docs/usage/aiw-fd.md:25-26`。文档称 TTY 会使用 ANSI，除非 `TERM=dumb` 或设置 `NO_COLOR`；但 `supports_ansi_color()` 还要求 TERM 命中已识别的终端前缀、TERM_PROGRAM 命中已识别程序，或存在 Windows ANSI 能力信号。未知 TERM 且没有这些信号的 TTY 实际输出纯文本。请说明未知终端也会 fail closed，并准确列出着色所需的能力依据。

## 验收核对

- **表格和列展示：满足（静态）。** `render_fd_list()` 输出 FD、状态、优先级和标题列，不生成 Markdown 表格语法。
- **活动与归档 FD 数据：满足（静态）。** `fd_list_rows()` 使用 `all_files()`，并从 FD 文档提取状态、优先级和标题。
- **颜色仅作冗余提示：满足（静态）。** ANSI 包装仅作用于状态和优先级；关闭颜色时 `ansi_color()` 返回原文本。
- **颜色回退：实现满足（静态）。** 非 TTY、存在 `NO_COLOR` 或 `TERM=dumb` 时关闭颜色；未知 TERM 只有在匹配已识别终端前缀、TERM_PROGRAM 或 Windows ANSI 信号时才可能启用。使用说明存在上述 finding。
- **标题安全渲染：满足（静态）。** `terminal_text()` 移除 `Cc`/`Cf` 字符，并以 `backslashreplace` 转义 stdout 编码无法表示的字符。
- **只读与依赖：满足（静态）。** 列表路径读取 FD 文件并渲染；新增 `unicodedata` 属于 Python 标准库，未发现写入 FD、索引、receipt 或归档文件的路径。

## 审查依据

- FD：FD-041，Revision 15；source event：`FD-041-000015-implementation-ready`。
- Reviewer session：`fd041-reviewer-r3-20261009-b7d24c`。
- Diff base：`4a445b27de935b39d45cfb1bd4276258abf33d36`；reviewed HEAD：`371f2bfe2d571b1476fdb91f4c2450d0dcfd29ba`。
- 已核对 event 15 为待认领 handoff，且当前 FD SHA-256 与 receipt 一致；随后认领该精确事件。
- 阅读 FD-041、R1/R2 评审、Worker r3 报告及 JSON、稳定 fd-workflow spec 相关段落、CLI 使用说明和从 diff base 到 HEAD 的实现差异。

## 命令与跳过项

- 已执行：`aiw fd resume FD-041`、FD SHA-256 与 event receipt 比对、`aiw fd claim FD-041 FD-041-000015-implementation-ready --session fd041-reviewer-r3-20261009-b7d24c`、`git status --short --branch`、`git log`、`git diff --stat`、`git diff --name-status`、相关 `git diff`、`git rev-parse HEAD`、`git merge-base office-dev HEAD` 和 `rg` 静态定位。
- Worker r3 报告记录 Python in-memory compile-only 检查通过；Reviewer 未重复执行。
- 未运行测试、smoke、lint、格式化、完整构建、compile-only 或运行时终端检查。

## 剩余风险

- 当前复审的唯一待修项是使用说明与实现的颜色能力条件不一致。
- TERM 前缀及 TERM_PROGRAM/Windows 环境信号只做静态审查；没有实际终端矩阵证据。Worker 报告也注明运行时 ANSI 行为未经验证。
