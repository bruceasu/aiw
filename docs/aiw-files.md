# AIW 文件生成清单

本文档归纳当前程序及其主要插件会生成的持久化文件。除用户目录路径外，以下路径均以项目根目录为基准。

## 1. 项目初始化文件

执行 `aiw init` 或项目设置插件后，可能生成：

| 文件/目录 | 作用 |
|---|---|
| `openspec/` | OpenSpec 工作区根目录 |
| `openspec/changes/` | 可选的活跃 OpenSpec change |
| `openspec/changes/archive/` | 已归档 Change |
| `openspec/specs/` | 长期维护的规格说明 |
| `.wt/` | Git worktree 默认目录 |
| `.ai/` | AIW Task 和 Workflow 的运行时数据 |
| `AGENTS.md` | 项目级 Agent 工作规则 |
| `.github/copilot-instructions.md` | GitHub Copilot 项目指令 |
| `docs/agents/work-management.md` | AIW 工作管理规则 |
| `docs/agents/domain.md` | 项目领域约定 |
| `.gitignore` | 自动加入 `.wt/` 忽略规则 |

已有文件通常不会覆盖；`--merge`、`--force` 或交互式设置流程可能更新已有内容。

## 2. Task、FD 和 OpenSpec 文件

执行 `aiw new <task-id>` 后：

```text
docs/features/<task-id>.md

.ai/tasks/<task-id>/
├── task.toml
├── state.json
└── events.jsonl
```

作用：

- `docs/features/<task-id>.md`：工程决策、人工工作项和验证意图。
- `.ai/tasks/<task-id>/task.toml`：Task 元数据、分支、worktree、Session 等信息。
- `state.json`：Workflow Core 的运行状态、Work Items、Attempts、Gates 和 Evidence。
- `events.jsonl`：Workflow 状态事件日志。

其他 Task 命令可能生成：

| 命令 | 文件 |
|---|---|
| 旧版 `aiw decision <task-id>` | 已存在的 OpenSpec change 下的 `design.md`；新 Task 决策写入 FD |
| `aiw spec <spec-id>` | `openspec/specs/<spec-id>/spec.toml`、`spec.md` |
| `Workflow Runner` | `.ai/tasks/<task-id>/artifacts/handoff.md` |
| `aiw archive` | 将 Task、FD 和存在的 Change 移到各自的归档位置 |

这里的 Workflow Runner handoff 是运行时生成的 Attempt-bound managed context，属于 `.ai/tasks/<task-id>/`；不要与 Session handoff（`.ai/sessions/<session-id>/artifacts/handoff.md`）或 OpenSpec 的业务交接产物混淆。
### 当前活动 Task 的一次性手工路径调整

本次改版完成后，仅对当前唯一活动 Task 做一次调整；已归档 Task 不移动。按此顺序操作：

1. 先停止旧版 AIW/`go run` 程序，再停止该 Task 的 Supervisor；确认没有活动 Attempt、写租约或其他 AIW 写入者。
2. 确认 `.ai/<task-id>/` 是唯一源记录，`.ai/tasks/<task-id>/` 除 `migrated-to` 标记外没有 Task 数据；身份或内容冲突时停止，不覆盖任何一侧。
3. 将目标目录的 `migrated-to` 标记备份到目标目录之外，确认目标目录为空后移除空目录，再把整个 `.ai/<task-id>/` 移到 `.ai/tasks/<task-id>/`。
4. 核对 `task.toml` 的 Task ID、`state.json`、`events.jsonl` 的身份和事件序号，以及 reports、artifacts 等目录均完整；用新版 AIW 做只读状态核对。核验通过前保留标记备份。

不重建 Task、不改写运行状态或事件历史。归档位置继续使用 `.ai/archive/<date>-<task-id>/`；这不是通用迁移命令或批量迁移流程。

显式选择 OpenSpec 后端或关联 change 时，可能存在以下可选工件：

```text
openspec/changes/<task-id>/
├── proposal.md
├── design.md
├── tasks.md
└── specs/<capability>/spec.md
```

原生 Task 与 Issue promotion 不要求这些工件。

## 3. Issue 文件（兼容 Requirement 记录）

执行 `aiw issue new <slug>` 后，当前兼容程序默认生成：

```text
docs/requirements/<requirement-id>/
└── requirement.toml
```

捕获需求产物后，可能增加：

```text
docs/requirements/<requirement-id>/
├── problem-brief.md
├── business-case.md
├── metric-brief.md
├── engineering-options.md
├── requirement-plan.md
└── decision-log.md
```

作用：

- `requirement.toml`：Issue 身份、可选 `parent_id`、状态、审批、promotion、revision 和工件摘要。
- 各类 Markdown：需求讨论和决策产物。
- `decision-log.md`：审批、延期、拒绝、归档或取消记录。

归档或取消后，整个目录会被移动到：

```text
docs/requirements/archive/<requirement-id>/
docs/requirements/cancelled/<requirement-id>/
```

Issue promotion 后，还会在目标 Task 下生成：

```text
.ai/tasks/<task-id>/artifacts/requirement-handoff.md
```

## 4. Session 文件

AIW Session 默认保存在项目的 `.ai/sessions/<session-id>/`：

```text
.ai/sessions/<session-id>/
├── status.json
├── instructions.md
├── memory.md
├── events.jsonl
├── prompts/
├── artifacts/
└── outputs/
```

作用：

- `status.json`：Session 状态、后端、模型、工作区和结果状态。
- `instructions.md`：本次 Session 的执行说明。
- `memory.md`：Session 持久记忆。
- `events.jsonl`：交互和状态事件。
- `prompts/`：发送给 Agent 的提示。
- `artifacts/`：handoff 等 Session 工件。
- `outputs/`：每次 Agent 执行的结果。

单次执行还可能生成：

```text
.ai/sessions/<session-id>/outputs/
├── 0001-final.txt
├── 0001-events.jsonl
└── 0001-stderr.log
```

`.ai/sessions/locks/<session-id>.lock` 是运行期间的临时互斥锁，正常完成后会删除。

## 5. Focused Test 文件

Focused Test 试验功能使用以下文件：

```text
openspec/changes/<task-id>/artifacts/verification-plan.json

.ai/tasks/<task-id>/artifacts/
├── focused-test-selection.json
├── focused-test-result.json
└── focused-test-output.txt
```

作用：

- `verification-plan.json`：人工定义的验证计划。
- `focused-test-selection.json`：测试 Agent 选择的受控检查项。
- `focused-test-result.json`：受控执行结果和输出摘要。
- `focused-test-output.txt`：有大小限制的命令输出。

## 6. Ask 会话文件

`aiw ask` 会将私有会话写入用户目录，而不是项目目录：

```text
~/.aiw/ask/YYYY-MM-DD/<timestamp>-<sha256>.md
```

文件保存问题、回答、结构化数据或错误状态，权限为用户私有。

## 7. Skills 文件

执行 `aiw skills install` 后，项目级 Skill 默认安装到：

```text
.agents/skills/
├── <skill-name>/
└── .aiw-skills.json
```

使用 `--scope user` 时，位置为：

```text
~/.agents/skills/
├── <skill-name>/
└── .aiw-skills.json
```

作用：

- `<skill-name>/`：复制后的 Skill 内容。
- `.aiw-skills.json`：托管清单、来源、版本信息、复制模式和 SHA-256 摘要。

`discover`、`adopt` 和 `sync` 主要更新或读取该托管清单；`--dry-run` 不写入目标文件。

## 8. Codex Session 辅助文件

`aiw cxs` 会在当前工作区生成：

```text
sessions/
├── index.json
└── cache.json
```

作用：

- `index.json`：Session 别名和索引。
- `cache.json`：Session 文件扫描缓存。

如果使用 `-o/--output-last-message`，还会按用户指定路径生成最终模型消息文件。

## 9. AI 代码索引文件

`aiw-ai-gen-index` 默认生成：

```text
.ai/
├── PROJECT_MAP.md
├── API_INDEX.md
├── symbols.jsonl
├── apis.jsonl
├── files.jsonl
└── metadata.json
```

作用：

- `PROJECT_MAP.md`：项目模块和符号索引。
- `API_INDEX.md`：API 路由索引。
- `symbols.jsonl`：符号记录。
- `apis.jsonl`：API 记录。
- `files.jsonl`：文件记录。
- `metadata.json`：生成器版本、时间和统计信息。

`aiw-ai-code-index` 还会在用户目录维护：

```text
~/.ai-code-index/
├── repos.json
└── db.sqlite
```

分别用于已索引仓库登记和代码索引数据库。

## 10. Shell 补全配置

项目初始化或执行 `setup-project` 时，可能修改用户 Shell 配置：

| Shell | 文件 |
|---|---|
| PowerShell | `~/Documents/PowerShell/Microsoft.PowerShell_profile.ps1` 或 `~/Documents/WindowsPowerShell/Microsoft.PowerShell_profile.ps1` |
| Bash | `~/.bashrc` |
| Zsh | `~/.zshrc` |
| Fish | `~/.config/fish/conf.d/aiw-completion.fish` |

写入内容位于：

```text
# >>> aiw completion >>>
...
# <<< aiw completion <<<
```

## 说明

- `.ai/` 下的文件主要是 AIW 的运行时状态，不属于人工编写的 OpenSpec 输入。
- `openspec/`、`requirements/` 和 `docs/` 下的文件通常是可审阅、可人工维护的业务或工程文档。
- `aiw exec-java` 执行 Maven 后产生的 `target/`、编译产物和 Maven 缓存由 Maven/JDK 生成，不是 AIW 自己维护的文件。
- 运行期间还可能出现临时文件、`.lock` 文件和原子写入临时文件；这些不属于稳定文件清单。
