# aiw

轻量级 OpenSpec 风格任务工作流 CLI（Go 实现）。

## 功能概览

- 初始化 OpenSpec-lite 目录和默认指令文件
- 自动创建或追加 `.wt/` 到 `.gitignore`
- 从 `docs/agent-templates/` 创建或合并 AI prompts
- 创建/查看/更新任务
- 为任务创建独立 git worktree
- 按任务输出上下文提示
- 创建长期 spec 文档
- 归档完成任务
- 生成任务注册表 `openspec/registry.json`

## 目录结构

执行 `aiw init` 后会创建以下结构（如果不存在）：

```text
repo/
├── openspec/
│   ├── changes/
│   ├── specs/
│   ├── archive/
│   └── registry.json
├── .wt/
├── AGENTS.md
└── .github/
    └── copilot-instructions.md
```

说明：

- `AGENTS.md` 与 `.github/copilot-instructions.md` 仅在文件不存在时创建。
- `registry.json` 来自 `openspec/changes/*/task.toml` 的汇总。

## 安装与构建

```bash
go build -o aiw .
```

Windows 可执行：

```powershell
.\aiw.exe init
```

## 命令

```text
aiw init [--prompts] [--merge] [--force] [--template <name>]
aiw new <task-id>
aiw list
aiw show <task-id>
aiw status <task-id> <status>
aiw done <task-id>
aiw archive <task-id> [--push] [--cleanup-wt] [--delete-branch] [--finalize]
aiw wt <task-id> [base-branch]
aiw wt-rm <task-id> [--delete-branch] [--force]
aiw wt-list [--porcelain]
aiw wt-prune [--dry-run]
aiw wt-lock <task-id> [reason]
aiw wt-unlock <task-id>
aiw wt-repair
aiw context <task-id>
aiw decision <task-id>
aiw spec <spec-id>
aiw registry
aiw prompts list
aiw prompts [template] [--merge] [--force]
aiw ignore-wt
```

### 命令行为细节

1. `aiw init`
- 创建 `openspec/`、`.wt/` 等目录。
- 写入默认模板文件（仅缺失时）。
- 创建或追加 `.wt/` 到 `.gitignore`。
- 生成/刷新 `openspec/registry.json`。
- 默认不会自动合并 `docs/agent-templates/` 下的实践模板。
- 可选参数：
  - `--prompts`：初始化完成后立即执行 prompts 同步
  - `--merge`：仅在与 `--prompts` 一起使用时有效，合并到现有 prompts 文件
  - `--force`：仅在与 `--prompts` 一起使用时有效，覆盖现有 prompts 文件
  - `--template <name>`：仅在与 `--prompts` 一起使用时有效，显式指定模板目录，如 `go` / `java` / `python`

2. `aiw new <task-id>`
- 在 `openspec/changes/<task-id>/` 创建：
  - `task.toml`
  - `task.md`
  - `notes.md`
- 默认元数据：
  - `type = "task"`
  - `status = "TODO"`
  - `branch = "feature/<task-id>"`
  - `worktree = ".wt/<task-id>"`

3. `aiw decision <task-id>`
- 为任务创建 `design.md`（若已存在则不覆盖）。

4. `aiw spec <spec-id>`
- 在 `openspec/specs/<spec-id>/` 创建：
  - `spec.toml`
  - `spec.md`

5. `aiw status <task-id> <status>`
- 更新 `task.toml` 的 `status`（会转成大写）和 `updated`。

6. `aiw done <task-id>`
- 等价于 `aiw status <task-id> DONE`。
- 不会自动归档。

7. `aiw archive <task-id>`
- 移动目录：
  - `openspec/changes/<task-id>`
  - -> `openspec/archive/<YYYY-MM-DD>-<task-id>`
- 可选参数：
  - `--push`：先执行 `git push -u origin feature/<task-id>`
  - `--cleanup-wt`：先执行 `git worktree remove .wt/<task-id>`
  - `--delete-branch`：尝试执行 `git branch -d feature/<task-id>`
  - `--finalize`：等价于同时开启 `--push --cleanup-wt --delete-branch`

8. `aiw wt <task-id> [base-branch]`
- 默认基线分支：`origin/main`
- 实际执行：
  - `git fetch origin`
  - `git worktree add .wt/<task-id> -b feature/<task-id> <base-branch>`
- 同步更新 `task.toml` 中的 `branch/worktree/updated`。

9. `aiw wt-rm <task-id> [--delete-branch]`
- 默认执行 `git worktree remove .wt/<task-id>`（安全模式，不强制）。
- 带 `--force` 时执行强制删除。
- 会清空 `task.toml` 中的 `worktree` 字段并更新时间。
- 带 `--delete-branch` 时还会执行 `git branch -d feature/<task-id>`，并清空 `branch` 字段。

10. `aiw wt-list [--porcelain]`
- 对应 `git worktree list`。
- 带 `--porcelain` 时输出脚本友好格式。

11. `aiw wt-prune [--dry-run]`
- 对应 `git worktree prune`。
- 带 `--dry-run` 时执行预演（`-n -v`）。

12. `aiw wt-lock <task-id> [reason]`
- 对应 `git worktree lock`。
- 可附原因，适合保护 hotfix 工作区。

13. `aiw wt-unlock <task-id>`
- 对应 `git worktree unlock`。

14. `aiw wt-repair`
- 对应 `git worktree repair`。

15. `aiw context <task-id>`
- 打印该任务建议优先阅读的文件列表和执行约束提示。

16. `aiw registry`
- 重新生成 `openspec/registry.json`。

17. `aiw prompts [template] [--merge] [--force]`
- `aiw prompts list` 可列出 `docs/agent-templates/` 下当前可用模板目录。
- 从 `docs/agent-templates/` 生成或合并仓库级 AI 提示文件。
- 支持的模板目录按当前仓库内容自动识别：
  - `go`：检测到 `go.mod`
  - `java`：检测到 `pom.xml` / `build.gradle` / `build.gradle.kts`
  - `python`：检测到 `pyproject.toml` / `requirements.txt` / `setup.py`
- 也可以显式指定模板，例如：`aiw prompts go --merge`
- 文件落点：
  - `AGENTS.md`
  - `.github/copilot-instructions.md`
  - `CODEX.md`
- 行为规则：
  - 默认：仅创建缺失文件，已存在则跳过
  - `--merge`：把模板内容追加/更新到现有文件中的 AIW 标记区块
  - `--force`：直接覆盖目标文件
- 执行完成后会输出统一摘要，说明哪些文件被 `created` / `merged` / `wrote` / `skipped existing`

18. `aiw ignore-wt`
- 创建 `.gitignore`，或在现有 `.gitignore` 中追加 `.wt/`。
- 如果已经存在 `.wt` 或 `.wt/` 规则，则不会重复追加。

## task.toml 格式（当前实现）

`task.toml` 为简化的键值格式（非完整 TOML 解析器），当前程序会读写以下字段：

```toml
id = "payment-retry"
type = "task"
status = "TODO"
created = "2026-05-28"
updated = "2026-05-28"
branch = "feature/payment-retry"
worktree = ".wt/payment-retry"
```

## task-id / spec-id 规则

允许字符：

- 英文字母 `a-z` `A-Z`
- 数字 `0-9`
- `-` `_` `.`

其他字符会被判定为非法 ID。

## 快速上手

```bash
aiw init
aiw init --prompts --merge
aiw init --prompts --template go
aiw new payment-retry
aiw wt payment-retry
aiw context payment-retry
aiw status payment-retry IN_PROGRESS
aiw done payment-retry
aiw archive payment-retry --finalize
aiw prompts list
aiw prompts go --merge
aiw ignore-wt
```
