# FD-041 独立复审 R2

<!-- aiw-data: FD-041-review-r2.json -->

## 结论

**changes-requested**。ANSI 对齐修复的长度补偿公式在当前映射的颜色码上成立；但终端能力检测仍会仅凭未识别的 `TERM` 名称后缀启用 ANSI，未满足不支持或未知终端必须输出纯文本的验收条件。

## Finding

1. **中等：未识别的 TERM 名称仍可能开启 ANSI** — `plugins/aiw-fd.py:228-229`。当 stdout 是 TTY、未设置 `NO_COLOR` 且 `TERM` 不是 `dumb` 时，`"256color" in term` 或 `term.endswith("-color")` 会让任意匹配这些字符串的 TERM 值启用颜色，即使该终端不在已知终端前缀列表中。例如 `TERM=mystery-256color` 和 `TERM=mystery-color` 在当前布尔条件中会命中。这里是对条件表达式的静态推导，并非运行这些终端配置所得。仅凭名称后缀不能证明终端支持 ANSI，因此违反 Acceptance 中“不支持颜色的终端不含 ANSI 控制码”，也没有完整落实 R1 要求的未知能力 fail-closed。请收紧为已识别或有明确能力依据的终端条件，令未知值保持纯文本。

## 验收覆盖

- **易读表格：部分满足。** `render_fd_list` 输出分组标题、FD/STATUS/PRIORITY/TITLE 列，未复用 Markdown 表格分隔语法。已知映射颜色的行宽补偿按完整起始与重置转义字符串的长度计算；公式静态上与 `ansi_color` 输出相符。未运行终端进行视觉检查。
- **活动与归档来源：满足（静态）。** `fd_list_rows` 调用 `all_files`；后者合并 `active_files` 和 `archived_files`。行字段从 FD 文档提取状态、优先级和标题。
- **颜色仅作冗余提示：满足（静态）。** 颜色只传给状态和优先级字段；禁用颜色时 `ansi_color` 原样返回字段文本。
- **纯文本回退：部分满足。** `NO_COLOR`、`TERM=dumb` 和非 TTY 分支会关闭颜色；未识别的 TERM 后缀仍可能启用 ANSI，见上述 finding。
- **标题安全渲染：满足（静态）。** `terminal_text` 移除 `Cc`/`Cf` 字符，并通过 `backslashreplace` 处理 stdout 编码无法表示的字符。
- **只读且无新依赖：满足（静态）。** `list` 分支只调用渲染并打印；新增的 `unicodedata` 属于 Python 标准库。未发现列表渲染路径写入 FD、索引、receipt 或归档文件。

## 审查依据

- FD：FD-041，Revision 12；source event：`FD-041-000012-implementation-ready`。
- Reviewer session：`fd041-reviewer-r2-20261009-6e42cd`。
- Diff base：`4a445b27de935b39d45cfb1bd4276258abf33d36`（`office-dev` 与当前 HEAD 的 merge-base）。
- Reviewed HEAD：`fd70cc64f2e736c62fd86de694f370b0893fb281`；ANSI 修复提交：`2151d091`。
- 阅读了上一轮 `FD-041-review-r1.md`、Worker `FD-041-implementation-r2.md` 及其 JSON、FD 当前验收与 Verification、CLI 使用说明、FEATURE_INDEX、`openspec/specs/fd-workflow/spec.md` 相关段落和实现 diff。

## 执行命令与跳过项

- 已执行：`aiw fd resume FD-041`、`aiw fd claim FD-041 FD-041-000012-implementation-ready --session fd041-reviewer-r2-20261009-6e42cd`、`git status --short --branch`、`git merge-base office-dev HEAD`、`git rev-parse HEAD`、`git log --format="%h %s" office-dev..HEAD`、`git diff --stat office-dev...HEAD`、相关 `git diff`/`rg`/文件读取命令。
- 曾尝试直接读取 worktree 内 `.ai/fd/FD-041/receipt.json`，该路径不存在；通过 `aiw fd resume` 确认 handoff，并成功认领精确 source event。未手工改写 receipt。
- 未运行测试、smoke、lint、format、完整构建、运行时 CLI/终端矩阵或 Reviewer compile-only 检查。Worker 报告记录 compile-only 检查通过；本轮未重跑。

## 剩余风险

未知 TERM 名称可绕过颜色能力检测；需要 Worker 修复后再审。ANSI 行宽公式仅经静态检查，真实终端渲染、Unicode 显示宽度及 Windows 编码矩阵未运行验证。
