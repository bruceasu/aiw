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

# TCC 包装器
aiw tcc hello.c -o hello.exe
aiw tcc dll hello.c -o hello.dll
aiw tcc x86_64 hello.c -o hello.exe
aiw tcc run hello.c
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
aiw wt add <task-id> [base-branch]
aiw wt rm  <task-id> [--delete-branch] [--force]
aiw wt list [--porcelain]
aiw wt prune [--dry-run]
aiw wt lock <task-id> [reason]
aiw wt unlock <task-id>
aiw wt repair
aiw wt ignore
aiw context <task-id>
aiw decision <task-id>
aiw spec <spec-id>
aiw registry
aiw prompts list
aiw prompts [template] [--merge] [--force]
aiw tcc [args...]           # TCC wrapper with auto include/lib defaults
aiw git <subcommand>           # run: aiw git help
```

## 插件扩展（Plugins）

`aiw` 支持通过外部可执行程序/脚本扩展子命令。若调用的子命令不是内置命令，`aiw` 会按约定位置搜索并尝试执行名为 `aiw-<plugin-name>` 的可执行文件或脚本。

- 搜索路径（优先级顺序）：
  1. 程序目录下的 `plugins` 子目录（即与 `aiw` 可执行同目录的 `plugins/`）
  2. `$HOME/.config/aiw/plugins`
  3. 系统 `PATH`（仅匹配路径下的可执行文件）

- 命名规则：
  - 文件基础名须为 `aiw-<plugin-name>`，允许扩展名：`.exe`, `.py`, `.sh`, `.bat`, `.cmd`, `.ps1`, `.js` 或无扩展（Linux ELF 或带 shebang 的脚本）。
  - `plugins` 目录内若存在子目录，`aiw` 会向下一层递归查找符合命名规则的文件。

- 执行优先级（同名多文件时）：
  1. `.bat` / `.cmd` / `.sh`
  2. `.py`
  3. 无扩展（带 shebang 的脚本）
  4. 原生二进制（ELF / .exe）

- 解释器与 shebang：
  - 对无扩展或脚本文件，若第一行为 `#!`（shebang），`aiw` 会使用 shebang 指定的解释器运行。
  - 对 `.js` 文件，优先尝试 `bun`（若可用），否则使用 `node`。

- 环境变量（`aiw` 会注入到插件进程）：
  - `AIW_PLUGIN_NAME`：插件名称（不含 `aiw-` 前缀）
  - `AIW_PLUGIN_PATH`：被执行插件的绝对路径
  - `AIW_CMDLINE`：从子命令开始的原始命令行（示例：`hello one two`）
  - `AIW_HOME` / `AIW_ROOT`：`aiw` 的配置或可执行所在根目录

- 使用示例：

  将 `plugins/aiw-hello.sh` 放入仓库根目录的 `plugins/`，运行：

  ```bash
  aiw hello arg1 arg2
  ```

  插件应把参数作为常规 argv 处理，并将输出写到 stdout/stderr；`aiw` 会把插件退出码作为自身的退出码返回。

- 安全提示：
  - 插件由外部代码执行，存在安全风险。仅放置受信任脚本或可执行文件；必要时在运维流程中引入签名或白名单策略。


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

8. `aiw wt <subcommand>` — worktree 管理（`aiw wt help` 查看完整列表）

| 子命令 | 说明 |
|--------|------|
| `add <task-id> [base]` | 创建 worktree，分支 `feature/<task-id>`，默认基线 `origin/main` |
| `rm  <task-id> [--delete-branch] [--force]` | 移除 worktree；`--delete-branch` 同时删除本地分支 |
| `list [--porcelain]` | 列出所有 worktree（对应 `git worktree list`）|
| `prune [--dry-run]` | 清理失效 worktree 元数据（`--dry-run` 预演）|
| `lock <task-id> [reason]` | 锁定 worktree，防止误删（适合 hotfix）|
| `unlock <task-id>` | 解锁 worktree |
| `repair` | 修复路径移动后的 worktree 链接 |
| `ignore` | 在 `.gitignore` 中添加 `.wt/` 规则 |

`add` 执行步骤：`git fetch origin` → `git worktree add .wt/<task-id> -b feature/<task-id> <base>`，并同步更新 `task.toml` 中的 `branch/worktree/updated`。

9. `aiw context <task-id>`
- 打印该任务建议优先阅读的文件列表和执行约束提示。

10. `aiw registry`
- 重新生成 `openspec/registry.json`。

11. `aiw prompts [template] [--merge] [--force]`
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

12. `aiw ignore-wt`（已合并为 `aiw wt ignore`）
- 创建 `.gitignore`，或在现有 `.gitignore` 中追加 `.wt/`。
- 如果已经存在 `.wt` 或 `.wt/` 规则，则不会重复追加。

13. `aiw git <subcommand>` — Git 快捷命令（`aiw git help` 查看完整列表）

命令按 9 个类别分组：

| 分组 | 包含命令 |
|------|----------|
| **Snapshot & Commit** | `save`, `cz`, `undo`, `ca`, `caf`, `change-author` ⚠, `rm-from-commit` ⚠, `revert` ⚠ |
| **History & Status** | `st`, `log`, `whatchanged`, `unpushed`, `unpulled` |
| **Sync & Remote** | `sync`, `update`, `outstanding`, `get`, `set-remote-branch`, `set-remote`, `add-remote`, `add-mirror` |
| **Branch** | `delete-branch` ⚠, `rename-branch`, `track`, `mv-to-branch` ⚠, `change-branch-base` ⚠ |
| **File** | `un-add`, `rm-keep`, `restore-file`, `get-file-from`, `rename` |
| **Conflicts** | `conflicts` |
| **Tags & Export** | `delete-tag` ⚠, `export` |
| **Recovery** | `rollback`, `find-commit-back`, `gc` ⚠, `detach` ⚠⚠, `clean-all-histories` ⚠⚠, `subdir-to-root` ⚠⚠ |
| **Guides** | `how-to-split`, `help` |

⚠ = 需要确认，可用 `--force` 跳过。⚠⚠ = 重写全部历史，协作者需重新 clone。

### 危险命令说明

#### `change-author <Name> <email> [--all --filter-name <old> ...] [--force]`

```bash
# 修改最后一个 commit 的作者
aiw git change-author "Bob" bob@example.com

# 重写全部历史，替换匹配旧名字的所有提交
aiw git change-author "Bob" bob@example.com --all \
  --filter-name "oldname" --filter-name "old.name"
```

- 无 `--all`：只 amend 最后一个 commit。
- 有 `--all`：使用 `git filter-branch --env-filter` 重写全部历史，同时替换 author 和 committer。

#### `rm-from-commit <file> [--history] [--from <commit>] [--force]`

```bash
# 只从最后一个 commit 删除文件
aiw git rm-from-commit passwords.txt

# 从整个历史删除
aiw git rm-from-commit passwords.txt --history

# 只从某个 commit 之后删除（之前的提交不受影响）
aiw git rm-from-commit passwords.txt --history --from abc1234
```

底层使用 `git filter-branch --tree-filter 'git rm -f --ignore-unmatch <file>'`。

#### `detach <ref> [--branch <name>] [--message <msg>] [--force]`

```bash
# 删除 abc1234 之前的所有历史，保留 abc1234..HEAD
aiw git detach abc1234
```

等价于：`checkout --orphan` → `commit` → `rebase --onto` → 删除临时分支。

#### `clean-all-histories [--branch <name>] [--remote <name>] [--message <msg>] [--force]`

```bash
# 将整个仓库历史压缩为单个 commit，并 force-push 到远程
aiw git clean-all-histories
aiw git clean-all-histories --branch main --remote origin
```

#### `subdir-to-root <subdir> [--force]`

```bash
# 将 trunk 子目录变为新的仓库根目录（适合 SVN 迁移后清理）
aiw git subdir-to-root trunk
```

底层使用 `git filter-branch --subdirectory-filter`。不包含该目录的 commit 会被丢弃。

常用示例：

```bash
aiw git save                          # stage all + commit "wip"
aiw git cz                            # 向导式提交（默认不启用 LLM）
aiw git cz --llm -N 5                # 启用 LLM，生成 5 条候选供选择
aiw git cz --emoji                   # 启用 emoji 提升可辨识度
aiw git save "fix typo"               # commit with message
aiw git st                            # short status
aiw git log                           # graph log (last 20)
aiw git sync                          # fetch origin, rebase, push
aiw git sync main upstream            # sync branch main against upstream
aiw git update main                   # update local main without checkout
aiw git delete-branch old-feat --remote  # delete local + remote branch
aiw git rename-branch new-name        # rename current branch
aiw git restore-file src/main.go      # discard working-tree changes
aiw git rollback                      # show reflog + recovery guide
aiw git export v2.0.0                 # archive tag to zip
aiw git change-author "Bob" b@x.com   # amend last commit's author ⚠
aiw git change-author "Bob" b@x.com --all --filter-name "old"  # rewrite all history ⚠⚠
aiw git rm-from-commit secret.txt --history  # scrub file from all history ⚠⚠
aiw git detach abc123                 # drop history before abc123 ⚠⚠
aiw git clean-all-histories           # squash all history + force-push ⚠⚠
aiw git subdir-to-root trunk          # make trunk/ the new repo root ⚠⚠
aiw git help                          # grouped help
aiw git help --alphabet               # alphabetical listing
```

### `aiw git cz`（Conventional Commit 向导）

- 默认行为：不启用 LLM，进入双语向导填写 type/scope/subject/body/breaking/footer。
- 长文本编辑：在 body/breaking/footer 输入 `/edit` 或 `^e`（Ctrl+E 文本快捷）可启动外部编辑器。
- 可选 LLM：仅在 `--llm` 时启用，直连 OpenAI Chat Completions API（不再依赖 `codex exec` 输出解析）。

```bash
set OPENAI_API_KEY=your_api_key
# optional
set OPENAI_MODEL=gpt-4o-mini
set OPENAI_BASE_URL=https://api.openai.com/v1
```

- 配置优先级：`CLI` > `项目根目录配置` > `程序目录配置`。
- 配置文件：`aiw.toml` 或 `.aiw.toml`（TOML）。
- `EDITOR`：可选，配置后优先于环境变量编辑器（`GIT_EDITOR`/`VISUAL`/`EDITOR`）。
- OpenAI 配置读取优先级：`配置文件 [cz]` > `外部环境变量` > `当前目录 .env` > `程序目录 .env` > 默认值。
- 可配置项：`model` / `base_url` / `api_key`（分别对应 `OPENAI_MODEL` / `OPENAI_BASE_URL` / `OPENAI_API_KEY`）。

示例配置：

```toml
[cz]
llm = false
candidates = 3
emoji = false
EDITOR = "code --wait"
model = "gpt-4o-mini"
base_url = "https://api.openai.com/v1"
api_key = ""

[cz.messages]
type = "选择你要提交的类型 / Select commit type:"
scope = "选择一个提交范围（可选）/ Scope (optional):"
subject = "填写简短精炼的变更描述 / Subject:"
confirmCommit = "是否提交或修改 commit? / Commit or modify?"

[[cz.types]]
value = "feat"
name = "feat:     新增功能 | A new feature"

[[cz.types]]
value = "fix"
name = "fix:      修复缺陷 | A bug fix"
```

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
# 初始化仓库
aiw init
aiw init --prompts --merge
aiw init --prompts --template go

# 任务工作流
aiw new payment-retry
aiw wt add payment-retry
aiw context payment-retry
aiw status payment-retry IN_PROGRESS
aiw done payment-retry
aiw archive payment-retry --finalize

# Worktree
aiw wt list
aiw wt prune --dry-run
aiw wt lock payment-retry "in review"
aiw wt rm payment-retry --delete-branch
aiw wt ignore

# Git 快捷命令
aiw git st
aiw git save "feat: add retry"
aiw git sync
aiw git update main
aiw git log
aiw git help

# TCC 包装器
aiw tcc hello.c -o hello.exe
aiw tcc dll hello.c -o hello.dll
aiw tcc x86_64 hello.c -o hello.exe
aiw tcc run hello.c

# Prompts
aiw prompts list
aiw prompts go --merge
```
