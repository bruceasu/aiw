# FD-005 独立评审报告

- FD：`FD-005`
- 结论：`CHANGES_REQUESTED`
- 来源事件：`FD-005-000005-review-requested`
- 认领会话：`codex-review-fd005-r4`
- 评审基线：HEAD=`61acf474abf222e87e6278ee21226692b309428`，并包含当前工作区的 Python 修复及 Node/TypeScript 清理差异
- Worker 报告：`docs/features/reports/FD-005-implementation.md`

## Findings

1. **高：验收证据仍不足，不能通过 Verification。**
   Worker 报告及 FD Verification 明确记载尚未验证目标环境中的 Codex/Copilot
   CLI 参数和输出、OpenAI HTTP、TUI 提交、Python 发布安装及配置保留行为；FD
   也将 Work Items 1.3–1.7 保持未完成。静态代码修复本身不能替代这些验收证据。

## Static review result

- `plugins/aiw-cz/aiw-cz.py`：`--retry` 现在在候选生成前直接复用上一条草稿。
- `plugins/aiw-cz/cz_providers.py`：拒绝 `CZ_*_ARGS` 覆盖，并固定 Codex/Copilot 的
  只读与结构化输出调用契约。
- `plugins/aiw-cz/cz_config.py`：每个配置层恢复为 `first-existing` 选择语义。
- `plugins/aiw-fd.py`：Revision 正则和 FD digest 对 CRLF/LF 进行规范化；本轮事件已成功 claim。
- Node/TypeScript 文件删除与 `build.bat` 的 Python 发布路径符合 FD 的迁移范围；未发现超出范围的代码变更。

## Evidence reviewed

- FD-005、Issue 计划、`openspec/specs/cz-configuration-priority/spec.md`、`openspec/specs/cz-python-runtime/spec.md`
- `docs/features/reports/FD-005-implementation.md`
- `plugins/aiw-cz/aiw-cz.py`, `cz_config.py`, `cz_providers.py`, `cz_llm.py`, `cz_openai.py`, `cz_ui.py`
- `plugins/aiw-fd.py`、`build.bat`、原 TypeScript 实现删除差异
- `.ai/fd/FD-005/events/000005-review-requested.json`

## Commands actually run

- `python plugins/aiw-fd.py claim FD-005 FD-005-000005-review-requested --session codex-review-fd005-r4`
- `python plugins/aiw-fd.py show FD-005`
- `Get-Content`、`rg`、`Select-String`：读取 FD、报告、规格和关键实现
- `git log --oneline --decorate -10`
- `git status --short`
- `git diff --stat` / `git diff --name-status` / `git diff`：核对实现与清理差异

## Skipped checks

未运行测试、完整构建、Python 发布安装、真实 CLI provider、TUI、OpenAI 网络请求或 Git 提交流程；这些检查在仓库规则下未获运行授权，且 Worker 报告也明确列为未验证。

## Residual risk

在目标环境证据补齐并由 Reviewer 重新复核前，不能确认 CLI 版本兼容性、非交互行为、TUI 提交和发布安装路径。`changes-requested` 事件已将 FD 返回 Worker，当前状态为 `In Progress`。
