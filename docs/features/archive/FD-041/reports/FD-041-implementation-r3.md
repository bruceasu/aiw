# FD-041 Worker 修复报告 r3

<!-- aiw-data: FD-041-implementation-r3.json -->

## 修复

针对 Reviewer R2 的 finding，移除了通过包含 `256color` 或 `-color` 后缀识别能力的宽松判断。当前只在 stdout 为 TTY、未设置 `NO_COLOR`、`TERM` 不是 `dumb`，并且 TERM 属于明确识别的终端前缀、`TERM_PROGRAM` 属于已识别程序或 Windows ANSI 环境标记存在时输出颜色；未知 TERM 默认纯文本。

ANSI 列宽修复保留：按完整颜色起始码和重置码的字符串长度计算补偿。R1、R2 findings 及其报告均保留为历史证据。

## 验证

- Compile-only 命令已通过：`python -c "from pathlib import Path; p=Path('plugins/aiw-fd.py'); compile(p.read_text(encoding='utf-8'), str(p), 'exec')"`。
- 修复提交前的 `git diff --cached --check` 通过。
- 未运行测试、smoke、lint、完整构建或运行时终端检查。未知/已知 TERM、TTY、重定向、`NO_COLOR` 和 Windows 控制台的实际行为没有运行时证据。

## 来源

- Worker handoff：`FD-041-000013-changes-requested`
- Worker session：`fd041-worker-20261009-3c5b1a`
- Reviewer finding：`FD-041-000012-implementation-ready` 的 R2 review
- 修复：仅缩紧 `plugins/aiw-fd.py` 的 `supports_ansi_color()` 能力信号判定。
