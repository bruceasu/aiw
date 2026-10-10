# FD-041 独立评审 R1

<!-- aiw-data: FD-041-review-r1.json -->

## 结论

请求修改。静态审查发现两项验收缺口：代码无法识别部分不支持 ANSI 的交互终端；启用颜色时列宽计算会让行内容与表头错位。

## 发现

1. **中：不支持 ANSI 的 TTY 仍会输出转义码** — plugins/aiw-fd.py:210-214。supports_ansi_color 只检查 stdout 是否为 TTY、NO_COLOR 是否存在和 TERM 是否为 dumb。TERM 为空或其他不支持 ANSI 的终端仍会返回 True，随后状态与优先级会被 ANSI 转义码包裹。这不满足“无颜色能力的终端不含 ANSI 控制码”的验收。请加入可靠的终端能力判定；能力未知或不支持时回退为纯文本。
2. **低：ANSI 序列长度被多计，表格列错位** — plugins/aiw-fd.py:252-254。ansi_color 添加前缀 ESC[<code>m 和后缀 ESC[0m，实际宽度开销为 len(code)+7；列宽却增加 len(code)+9。当前颜色码为两位，状态列多出两个空格，优先级列也多出两个空格，正文列与表头不齐。请按实际 ANSI 序列长度补齐，或按去除转义码后的可见宽度计算。

## 验收核对

- **终端表格与列展示：部分满足。** 输出由表头和纯文本行组成，没有 Markdown 分隔语法；ANSI 开启时存在列错位，见发现 2。
- **活动及归档 FD 数据：满足静态审查。** active_files、archived_files 和 all_files 枚举活动与归档文档；状态、优先级、标题来自 FD 文档。
- **颜色作为冗余提示：满足静态审查。** 仅状态和优先级值使用颜色，纯文本路径保留字段内容。
- **禁用颜色及不支持颜色的终端：部分满足。** NO_COLOR、TERM=dumb 和非 TTY 分支关闭颜色；未知或不支持 ANSI 的 TTY 未检测，见发现 1。
- **标题安全显示：满足静态审查。** Cc/Cf 字符被过滤，当前 stdout 编码不能表示的字符通过 backslashreplace 转义。
- **只读及依赖范围：满足静态审查。** 新列表逻辑读取 FD 文档并在内存中渲染；新增 unicodedata 为 Python 标准库。未发现对索引、FD、receipt 或归档文件的写入路径。

## 审查范围与证据

- FD：FD-041，Revision 8；Source event：FD-041-000008-implementation-ready。
- Reviewer session：fd041-reviewer-20261009-ce272b。
- 审查基线：4a445b27de935b39d45cfb1bd4276258abf33d36（office-dev / merge-base）。
- 审查提交：4e8bdacb1d9886b495189daed8c82f2800a144d6。
- 已确认 handoff 中的 FD SHA-256 与当前 FD 正文匹配。
- 检查了 FD-041、fd-workflow 稳定 spec、Worker Markdown/JSON 报告，以及从 office-dev 到审查提交的实际差异。FD 未设置 Independent test policy，因此该 handoff 直接路由 Reviewer。

## 实际执行的命令

- aiw fd claim FD-041 FD-041-000008-implementation-ready --session fd041-reviewer-20261009-ce272b
- git status --short --branch
- git log --format="%h %s" office-dev..HEAD
- git rev-parse HEAD
- git merge-base office-dev HEAD
- git diff --stat office-dev...HEAD
- git diff --name-status office-dev...HEAD
- git diff --unified=40 office-dev...HEAD -- plugins/aiw-fd.py docs/usage/aiw-fd.md
- git diff --unified=8 office-dev...HEAD -- docs/features/FEATURE_INDEX.md
- rg -n "fd list|FEATURE_INDEX|terminal|ANSI|color" openspec/specs/fd-workflow/spec.md
- rg -n "def terminal_text|def supports_ansi_color|def ansi_color|def render_fd_list|state_width|priority_width|fd_list_rows|def active_files|def archived_files|def all_files" plugins/aiw-fd.py
- Python UTF-8 只读检查：比较 FD 正文 SHA-256 与 implementation-ready handoff digest；读取 Worker sidecar 和相关源码片段。

## 未执行检查与剩余风险

- 未运行测试、smoke、lint、格式化、完整构建或运行时 CLI 验证。
- Worker 报告记录 compile-only 检查通过；Reviewer 未重复执行。
- 未运行时验证不同终端、重定向输出、NO_COLOR、TERM=dumb 或 Windows 编码行为；终端兼容性仍有上述静态发现及运行时风险。
