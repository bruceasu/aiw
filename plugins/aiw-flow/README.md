# aiw-flow

`aiw-flow` 是一个专注于执行 AI 编程任务的 Python CLI。它负责管理 Codex Session、Prompt、Memory、Thread 和执行结果，不负责任务分支、Git Worktree 或仓库生命周期。

Git 任务和 Worktree 应由 aiw / aiw-wt 管理；aiw-flow 只接收一个已经存在的 `--workspace` 作为 Codex 的执行目录。

## 安装与前置条件

```bash
python -m pip install -e .
```

需要 Python 3.9+ 和可用的 `codex exec` CLI。通过 aiw 插件运行时，下面的 `aiw-flow` 可以替换为 `aiw flow`。

## 状态目录

默认状态目录是当前工作目录下的 `.ai/`，也可以通过全局参数 `--root` 指定：

```text
.ai/
├── sessions/<session-id>/
│   ├── status.json
│   ├── instructions.md
│   ├── memory.md
│   ├── events.jsonl
│   ├── prompts/
│   ├── outputs/
│   └── artifacts/
├── locks/
├── logs/
└── archive/
```

```bash
aiw-flow --root D:/aiw-state new \
  --id BUG-1001-login \
  --title "Fix login timeout" \
  --workspace D:/repos/web \
  --instructions examples/coding-agent-instructions.md
```

`--root` 必须放在子命令之前。`.ai/` 也可能被 aiw 的其他功能使用，团队应统一约定状态文件布局，或为 aiw-flow 指定独立的 `--root`。

## Session 生命周期

```text
new → run → continue（可多次）→ finish → archive
  └→ loop（可多次交互）
grill → loop（可选）
                                      └→ delete
```

- `new`：登记一个 AI 任务并保存持久化 Instructions。
- `grill`：创建需求澄清 Session，采集受限的 Workspace 摘要并立即开始第一轮访谈。
- `run`：执行第一轮 Codex Prompt，创建并记录 Thread ID。
- `continue`：复用已有 Thread 执行下一阶段。
- `finish`：将 Session 标记为完成，可生成执行产物。
- `archive`：归档已完成的 Session。
- `delete`：删除 aiw-flow 自己保存的 Session 状态。

## 创建任务：`new`

```text
aiw-flow new --id ID --title TITLE --workspace PATH --instructions FILE
  [--ephemeral] [--loop] [--phase PHASE] [--timeout SECONDS]
```

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `--id SESSION_ID` | 是 | Session 唯一 ID，例如 `GWJ-1234-order-slip`。只用于标识 AI 任务和状态目录。 |
| `--title TITLE` | 是 | 任务标题。 |
| `--workspace PATH` | 是 | 已存在的 AI 执行目录。aiw-flow 不创建、删除、修复或切换 Worktree。 |
| `--instructions FILE` | 是 | UTF-8 持久化执行规则文件，会保存为 Session 的 `instructions.md`。 |
| `--ephemeral` | 否 | 标记为临时 AI 任务。 |
| `--loop` | 否 | 创建 Session 后立即进入交互式 Loop，第一条输入会创建并绑定 Codex Thread。 |
| `--phase PHASE` | 否 | `--loop` 使用的阶段；未指定时为 `interactive`。 |
| `--timeout SECONDS` | 否 | Loop 中每一轮 Codex 执行的超时时间。 |

示例：先由 aiw-wt 准备工作区，再交给 aiw-flow：

```bash
aiw wt add FEAT-204-export main

aiw-flow new \
  --id FEAT-204-export \
  --title "Add CSV export" \
  --workspace .wt/FEAT-204-export \
  --instructions examples/coding-agent-instructions.md
```

如果不使用 Worktree，也可以直接把已有仓库目录作为工作区：

```bash
aiw-flow new \
  --id BUG-1001-login \
  --title "Fix login timeout" \
  --workspace D:/repos/web \
  --instructions examples/coding-agent-instructions.md
```

## 需求澄清：`grill`

```text
aiw-flow grill --id ID --title TITLE --workspace PATH
  (--requirement TEXT | --requirement-file FILE)
  [--timeout SECONDS] [--ephemeral]
```

`grill` 使用内置的 Easy English 访谈规则创建普通 aiw-flow Session，并立即启动第一轮 Codex 执行。规则要求 Codex：

- 先检查 Workspace，不询问可以从本地文件确认的事实。
- 每轮最多提出一个需要用户决策的问题。
- 每个问题同时给出推荐答案和理由。
- 只有用户明确结束 Grill 时，才输出 `SUCCESS: Ready to execute.` 和最终规格。
- Grill 阶段只澄清需求，不实现代码。

```bash
aiw-flow grill \
  --id FEAT-204-export \
  --title "Clarify export requirement" \
  --workspace .wt/FEAT-204-export \
  --requirement "Add an export workflow for operations users." \
  --loop
```

带 `--loop` 时，首轮问题完成后会直接等待回答。不带 `--loop` 时保持单发行为，回答上一轮问题可以继续使用 `continue`：

```bash
aiw-flow continue FEAT-204-export \
  --phase grill \
  --prompt "CSV is sufficient for the first release."
```

首次启动时会生成 `artifacts/workspace-context.md`。该摘要：

- 只读取明确允许的项目元数据文件，例如 `README.md`、`AGENTS.md`、`go.mod` 和 `pyproject.toml`。
- 跳过隐藏目录、版本控制目录、依赖目录、缓存和构建目录。
- 限制目录深度、条目数量、单文件字节数和总字节数。
- 在保存和发送给 Codex 前，替换常见的密码、Token、Secret 和 API Key 赋值。
- 不读取 `.env`、私钥或任意业务文件内容。

这些限制用于减少意外暴露，但不是完整的秘密扫描器。不要在允许读取的元数据文件中保存真实凭证。

## 执行第一轮：`run`

```text
aiw-flow run SESSION_ID --phase PHASE [--prompt TEXT] [--prompt-file FILE] [--timeout SECONDS] [--force-new-thread]
```

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `SESSION_ID` | 是 | `new` 创建的 Session ID。 |
| `--phase PHASE` | 是 | 阶段名称，例如 `analyze`、`implement`、`fix-tests`。 |
| `--prompt TEXT` | 否 | 直接提供 Prompt。 |
| `--prompt-file FILE` | 否 | 从 UTF-8 文件读取 Prompt。 |
| `--timeout SECONDS` | 否 | 本轮 Codex 执行的超时时间。 |
| `--force-new-thread` | 否 | 忽略已有 Thread ID，重新创建上下文。仅在原 Thread 不可用时使用。 |

Prompt 至少需要来自 `--prompt`、`--prompt-file` 或 stdin 之一。多个来源会按命令行、文件、stdin 的顺序拼接：

```bash
aiw-flow run BUG-1001-login \
  --phase analyze \
  --prompt "Find the root cause and propose a minimal fix."

aiw-flow run BUG-1001-login \
  --phase analyze \
  --prompt-file examples/analyze.md

Get-Content .\task.md | aiw-flow run BUG-1001-login --phase analyze
```

## 继续任务：`continue`

```text
aiw-flow continue SESSION_ID --phase PHASE [--prompt TEXT] [--prompt-file FILE] [--timeout SECONDS]
```

`continue` 要求 Session 已经有 Thread ID，不支持重新指定 Thread。推荐按阶段推进：

```bash
aiw-flow run GWJ-1234-order-slip \
  --phase analyze \
  --prompt-file examples/analyze.md

aiw-flow continue GWJ-1234-order-slip \
  --phase implement \
  --prompt-file examples/implement.md

aiw-flow continue GWJ-1234-order-slip \
  --phase fix-tests \
  --prompt-file examples/fix-tests.md
```

每轮发送给 Codex 的内容由四部分组成：持久化 Instructions、Session Memory、阶段名称和当前 Prompt。每轮 Prompt 会保存到 `prompts/`，最终输出保存到 `outputs/`，事件保存到 `events.jsonl`。

## 交互式 Session：`loop`

Loop 是单发命令之外的可选交互外壳：

```text
aiw-flow loop SESSION_ID [--phase PHASE] [--timeout SECONDS]
```

可以从三种位置进入：

```bash
# 1. 创建普通 Session 后立即交互；第一条输入执行首轮
aiw-flow new \
  --id FEAT-300-refactor \
  --title "Interactive refactor" \
  --workspace .wt/FEAT-300-refactor \
  --instructions examples/coding-agent-instructions.md \
  --loop \
  --phase analyze

# 2. 创建 Grill，执行首轮问题后持续交互
aiw-flow grill \
  --id FEAT-301-export \
  --title "Clarify export" \
  --workspace .wt/FEAT-301-export \
  --requirement "Add export support." \
  --loop

# 3. 恢复已有 Session
aiw-flow loop FEAT-301-export --phase grill
```

如果 `loop` 没有显式 `--phase`，它会使用 Session 当前阶段；Session 也没有当前阶段时使用 `interactive`。

```text
Interactive loop for FEAT-301-export (phase: grill). Type /help for commands.
You> CSV is sufficient for the first release.
...
You> /done
SUCCESS: Ready to execute.
...
```

Loop 支持以下本地控制和 Skill 命令。除 `/skill` 与 `/done` 外，本地命令不会执行 Codex Turn：

| 命令 | 行为 |
| --- | --- |
| `/help` | 显示 Loop 帮助。 |
| `/status` | 显示 Session 状态。 |
| `/memory` | 显示 Session Memory。 |
| `/handoff` | 生成 `artifacts/handoff.md`。 |
| `/fork` | 生成 handoff，以 handoff 作为新 Thread 的业务上下文，执行一次新 Thread 后退出 Loop。 |
| `/skills` | 按项目和用户作用域列出可发现的 Codex Skills，不执行 Turn。 |
| `/skill NAME MESSAGE` | 使用 Codex 原生 `$NAME` 语法调用一个已发现的 Skill，并执行一个普通 Turn。 |
| `/done` | 仅在 `grill` 阶段发送 `Grill Done`，显示最终响应后退出。 |
| `/exit` | 不发送新 Turn，直接退出。 |
| `//text` | 发送以 `/` 开头的普通消息，例如 `//review` 会发送 `/review`。 |

Skill 发现不需要额外配置，候选目录为：

- 从 Session workspace 到 Git 仓库根目录的各级 `.agents/skills`。
- Git 仓库根目录的 `.codex/skills`；非 Git workspace 使用 workspace 自身。
- 用户目录的 `~/.agents/skills`。
- 有效 Codex Home 下的 `skills`；默认是 `~/.codex/skills`。

每个候选 Skill 必须是包含 `SKILL.md` 的直接子目录，且 frontmatter 必须提供有效的 `name` 和 `description`。`/skills` 会显示作用域、来源路径、无效候选警告和重名标记。同名 Skill 出现在多个位置时，`/skill` 会报告所有冲突路径并拒绝猜测优先级。

```text
You> /skills
Project Skills:
  metrics-review - Review financial metric definitions.
    D:\repos\demo\.agents\skills\metrics-review

You> /skill metrics-review Review the revenue metrics
```

也可以直接输入 Codex 原生调用，例如 `$metrics-review Review the revenue metrics`；aiw-flow 会把它当作普通消息原样发送。`/skill` 不复制、安装或持久激活 Skill，完整 `SKILL.md` 及关联资源仍由 Codex 按需加载。

空输入会被忽略。EOF、输入等待期间的 `Ctrl+C` 和 `/exit` 都会正常退出，不改变 Session 状态。`running`、`completed`、`archived` 或 `deleted` Session 不能进入 Loop。

Loop 保持一个 aiw-flow 进程，但每条普通输入仍通过现有执行路径启动一次 `codex exec`，因此 Thread、Prompt、Output、Event、超时和错误行为与 `run/continue` 一致。当前版本采用单行输入；长 Prompt 仍建议使用单发命令的 `--prompt-file` 或 stdin。

## 查看任务状态

### `status`

```text
aiw-flow status SESSION_ID [--json]
```

查看状态、Thread ID、执行阶段、最近退出码、最后输出和错误信息。`--json` 适合脚本或 CI：

```bash
aiw-flow status GWJ-1234-order-slip
aiw-flow status GWJ-1234-order-slip --json
```

### `list`

```text
aiw-flow list [--state STATE]
```

列出 AI Session，可按状态过滤：

```bash
aiw-flow list
aiw-flow list --state active
```

### `inspect`

```text
aiw-flow inspect SESSION_ID
```

输出完整状态、最近事件和 Memory 摘要，适合排查某轮执行失败或 Thread 未绑定。

## Memory 管理

Memory 只记录 AI 任务上下文，不管理仓库或分支：

```text
aiw-flow memory show SESSION_ID
aiw-flow memory append SESSION_ID --text TEXT
aiw-flow memory replace SESSION_ID --file FILE
```

```bash
aiw-flow memory append BUG-1001-login \
  --text "Confirmed: timeout occurs only when the refresh token is expired."

aiw-flow memory show BUG-1001-login
aiw-flow memory replace BUG-1001-login --file notes/confirmed-findings.md
```

## 跨 Session 交接：`handoff`

```text
aiw-flow handoff create SESSION_ID [--focus TEXT]
aiw-flow handoff show SESSION_ID
```

`handoff create` 不调用模型。它从 Session 的状态、Memory、最近输出和已有 Artifact 路径生成确定性的 `artifacts/handoff.md`：

```bash
aiw-flow handoff create FEAT-204-export \
  --focus "Continue from validation and resolve the encoding decision."

aiw-flow handoff show FEAT-204-export
```

交接文档包含 Goal、Current State、Confirmed Findings、Decisions、Modified Files、Validation State、Open Issues、Recommended Next Action、Suggested Skills 和 Artifact References。最近输出只保存受限长度的摘录，完整内容仍通过 `outputs/` 路径引用。

Handoff 写入使用 Session 锁和原子替换，因此不需要依赖系统临时目录或 `next-agent` shell 函数，也不会误读其他 Workspace 的“最新文件”。

## 完成与归档

```text
aiw-flow finish SESSION_ID [--create-patch]
aiw-flow archive SESSION_ID
aiw-flow delete SESSION_ID --yes
```

`finish` 标记 AI 任务完成；`--create-patch` 只生成当前工作区的审查产物，不创建或清理 Worktree。Git 分支和 Worktree 的清理由 aiw-wt 负责：

```bash
aiw-flow finish FEAT-204-export --create-patch
aiw archive FEAT-204-export --cleanup-wt --delete-branch
```

`delete` 只删除 aiw-flow 的 Session 状态，不删除 Workspace、分支或 Worktree。

## 配置文件

全局配置文件位于：

- Windows：`%APPDATA%\aiw-flow\config.toml`
- Linux/macOS：`$XDG_CONFIG_HOME/aiw-flow/config.toml`，未设置时为 `~/.config/aiw-flow/config.toml`

```toml
model = "gpt-5-codex"
profile = "default"
sandbox = "workspace-write"
codex_home = "D:/codex-home"
additional_codex_args = ["--color", "never"]
```

## 安全约束

- aiw-flow 不执行 `commit`、`push`、`git reset --hard` 或 `git clean -fd`。
- 不使用 `shell=True`，Codex 命令参数以参数数组传递。
- 不要把 API Key、密码等秘密写入 Instructions、Prompt、Memory 或事件日志。
- 并行 AI 任务应由 aiw-wt 提供独立 Workspace，然后为每个 Workspace 创建独立 Session。

## 测试

```bash
python -m pytest
```

测试不要求真实 Codex 安装，后端执行会使用 fake backend 或 mock process。
