# FD-005 独立评审报告

- FD：`FD-005`
- 结论：`CHANGES_REQUESTED`
- 复核轮次：本轮确认上一评审后的实现提交未发生变化；FD 文档状态曾被错误改为 `implementation-ready`，已在评审结论下纠正为 `Pending Verification`。
- 来源事件：未发现有效的 Worker `implementation-ready` 移交；可见的
  `FD-005-000002-design-requested` 是旧 Planner 事件，不能作为本次 Reviewer
  handoff 的 `source-event`。
- 评审基线：`origin/main..HEAD`，HEAD=`ef22144eb9633a943dd33e8889134cbd1d1ed4b7`
- 实施报告：未找到 `docs/features/reports/FD-005-implementation.md`

## Findings

1. **高：`--retry` 未保持既有 CLI 语义。**
   `plugins/aiw-cz/aiw-cz.py:46-53` 在 `--retry` 时仍先调用 `candidates()`；只有 provider 没有返回候选时才读取上一条提交。旧实现 `plugins/aiw-cz/src/index.ts:174-188` 则在 `--retry` 时直接使用上一条提交，完全跳过 LLM。当前实现会在配置了 LLM 时改写用户明确要求“重试上一条草稿”的行为，也可能触发网络或 CLI provider。

2. **高：可配置 CLI 参数可以绕过 FD 要求的安全和输出契约。**
   `plugins/aiw-cz/cz_providers.py:80-103` 在存在 `CZ_CODEX_ARGS` 或 `CZ_COPILOT_ARGS` 时直接采用环境参数；此路径不强制 Codex 使用 `exec --sandbox read-only --json -`，也不强制 Copilot 使用 `--output-format text`。因此调用者可配置出会读写仓库、等待交互输入或输出非结构化结果的命令，静态的前后 `git status` 检查不足以提供执行层只读保证。

3. **中：配置文件选择语义与原实现不兼容。**
   `plugins/aiw-cz/cz_config.py:175-182` 会在同一位置同时读取并依次合并 `cz.toml` 与 `.cz.toml`（以及 `aiw.toml` 与 `.aiw.toml`）。原实现 `plugins/aiw-cz/src/config.ts:96-101` 使用 `firstExisting`，每个位置只选择一个文件。两份文件同时存在时，Python 版本可能改变 provider、locale、候选类型和消息的最终值，违背 FD 要求的现有配置兼容性。

4. **高：验收证据仍不完整。**
   FD Verification 明确记录未运行发布安装、真实 CLI provider、TUI、提交流程和 OpenAI 请求；同时没有 Worker 实施报告或可引用的 `implementation-ready` 事件。因而不能把 1.1-1.8 的静态实现声明当作全部验收条件已被真实证据支持。

5. **流程：FD 状态与事件记录不一致。**
   当前 `.ai/fd/FD-005/events/` 只有 `design-requested`，没有 Worker 的
   `implementation-ready` 事件，但 FD 文档曾标记为 `implementation-ready`。
   在本轮仍为 `CHANGES_REQUESTED` 且缺少有效 handoff 的情况下，该状态不能成立。

## Evidence reviewed

- `docs/features/FD-005_AIW_CZ_TYPESCRIPT_PYTHON.md`
- `docs/requirements/REQ00004-aiw-cz-python-cli/`
- `openspec/specs/cz-configuration-priority/spec.md`
- `openspec/specs/cz-python-runtime/spec.md`
- `plugins/aiw-cz/aiw-cz.py`, `cz_config.py`, `cz_core.py`, `cz_llm.py`, `cz_openai.py`, `cz_providers.py`, `cz_ui.py`
- 原 TypeScript 实现及 `internal/plugin/discover.go`、`internal/plugin/exec.go`
- `build.bat` 与当前 Git 提交差异
- `.ai/fd/FD-005/events/000002-design-requested.json`

## Commands actually run

- `Get-Content`：读取仓库规则、共享 `work-management.md`、FD、Issue、稳定规格、实现和既有报告
- `Select-String` / `rg`：定位 Work Items、Verification、事件、provider、配置和入口调用
- `git status --short`
- `git log --oneline --decorate -8`
- `git diff --stat origin/main..HEAD -- ...`
- `git diff --name-status origin/main..HEAD -- ...`
- `git show`：检查最近 CZ 修复提交
- `git rev-parse HEAD` 与 `git rev-parse origin/main`

## Skipped checks

未运行测试、Python 插件、CLI provider、TUI、OpenAI 网络请求、构建、安装或最终产物验证；也未执行 `aiw fd claim`/`emit`，因为不存在可认领的有效 Worker handoff。未执行 Git 写操作。

## Residual risk

目标环境中的 Codex/Copilot 版本、非交互参数、JSON/JSONL 输出、Python 安装路径和 TUI/配置保留行为仍缺少运行证据。修复上述问题并由 Worker 通过有效 `implementation-ready` 事件重新移交后，需进行新的独立 Reviewer 复核。
