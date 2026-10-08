# FD-040 独立评审 R1

<!-- aiw-data: FD-040-review-r1.json -->

## 结论

`verification-passed`。审查范围内未发现阻断问题；FD-040 验收条件均有实现与静态证据支持。

## 验收核对

- 14 个原模板文件均以 100% 内容迁移至 `docs/templates/`；`docs/features/` 顶层不再包含这些模板。
- `plugins/aiw-fd.py` 的 `new_fd` 从 `docs/templates/TEMPLATE.md` 读取；缺失时创建新目录并将内置模板写入新路径。
- `skills/fd-workflow/`、`skills/fd-test/`、`skills/work-management.md`、可移植操作说明和 `docs/usage/aiw-fd.md` 已更新模板路径。模板路径检索在活跃文件中未发现旧位置引用；FD-038/039 记录中的旧路径属于本 FD 明确排除的历史 FD 文本。
- `scripts/fd_smoke.py` 与 3 个黑盒 fixture 均从 `docs/templates/` 复制模板；缺失模板场景也指向新位置。
- FD Work Items 与 TODO 均已勾选，状态为 Pending Verification；模板迁移未改变 FD/evidence 目录约定。

## 审查范围与证据

- FD：`FD-040`，Revision 4。
- Source event：`FD-040-000004-implementation-ready`。
- Reviewer session：`fd040-reviewer-20261008-c6a23d`。
- 审查提交：`47c5959..2c84c64`，HEAD 为 `2c84c64`。
- 检查了 FD、`openspec/specs/fd-workflow/spec.md`、Worker Markdown/JSON 报告及实际差异。
- Worker 报告记录 5 个 Python 文件内存 compile-only 成功；本 Reviewer 未重跑该命令。报告称执行静态路径扫描，但 JSON `commands` 未记录该扫描的具体命令；本次独立检索和差异检查仍支持路径迁移验收。

## 实际执行的命令

- `aiw fd show FD-040`
- `aiw fd claim FD-040 FD-040-000004-implementation-ready --session fd040-reviewer-20261008-c6a23d`
- `git log --oneline --decorate --max-count=6`
- `git diff --find-renames --name-status 47c5959..HEAD`
- `git diff --find-renames --unified=18 47c5959..HEAD --`（程序、技能、用法文档、smoke 与 3 个 fixture）
- `rg -n "docs/features/[^\\s`]*TEMPLATE|docs/features/(TEMPLATE|.*_TEMPLATE)|features/TEMPLATE\\.md" plugins scripts skills docs/usage tests`
- `rg -n "docs/features/[^\\s]*TEMPLATE|docs/features/.*_TEMPLATE|features/TEMPLATE\\.md"`（排除历史 reports/reviews/archive）
- `rg -n "TEMPLATE\\.md|_TEMPLATE\\.md"`（限定于 smoke 与 3 个调整后的 fixture）
- `aiw fd --help`；`aiw fd emit --help`

## 未执行检查与剩余风险

- 未运行测试、smoke、最终构建、lint、格式化或运行时 CLI 验证；仓库规则和本 FD 均未授权这些执行。
- Worker 的 compile-only 结果来自其实施报告，Reviewer 未独立复跑。
- 因未执行 CLI，缺失模板时的默认模板生成行为仍只有静态代码证据。
