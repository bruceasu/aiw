# AIW Requirement Management 使用手册

## 适用场景

Requirement Management 用于保存需求讨论的结果，并在人工明确批准后，受控地把它交接给 AIW Task。它位于需求讨论与 OpenSpec 工程规格之间：讨论 Skill 产出草案，Requirement 保存已确认的内容，promotion 才会创建或复用工程 Task。

它不会自动批准需求、自动开始实现或发布。promotion 会使用共享的 schema-aware 生成器准备 OpenSpec proposal/design/spec/tasks；OpenSpec CLI 仅用于可选的额外校验。

## 生命周期与边界

```text
DRAFT -> DISCOVERED -> DECIDED -> APPROVED -> PROMOTED
                         |              |
                         v              v
                     DEFERRED        REJECTED
```

- `new` 创建 `DRAFT` Requirement。
- 首次 capture 将 `DRAFT` 变为 `DISCOVERED`。
- capture `requirement-plan` 后，`DISCOVERED` 变为 `DECIDED`。
- 只有 `DECIDED` Requirement 可以被人工 `APPROVED`。
- `promote` 创建或复用一个 AIW Task，生成并校验 OpenSpec artifacts，成功后把 promotion 状态推进到 `SPEC_DRAFTED`。
- `DEFERRED` 和 `REJECTED` 不可直接 promotion；当前版本没有重新打开命令，需等待未来的显式 revision 流程。

创建、capture 和 approve 只写入 `requirements/<requirement-id>/`，不会创建 Task、OpenSpec change、Session、worktree 或分支。

## 快速开始

人类用户从对话入口开始，不需要记忆 artifact 类型、文件路径或命令参数：

```powershell
# 新需求：创建可恢复的 Requirement Conversation
aiw requirement chat

# 已有需求：恢复其 Conversation
aiw requirement chat daily-withdrawal-report
```

Conversation 会按缺口推进 intake、价值、指标、工程选项或 synthesis；只有关键歧义、冲突、不可逆决策或 `%% NEEDS_INPUT` 才进入 deep discovery。每次创建、capture、审批或 promotion 前，它都会展示目标、内容摘要和写入范围。AI 只能准备待执行动作；人类在会话中输入 `confirm` 或“确认”后，适配器才会执行该动作。

以下命令适用于脚本、自动化，或已经知道精确参数的用户：

```powershell
# 1. 创建 Requirement
aiw requirement new daily-withdrawal-report "Daily withdrawal report"

# 2. 查看当前状态
aiw requirement show daily-withdrawal-report

# 3. 捕获已由人确认的需求方案
aiw requirement capture daily-withdrawal-report requirement-plan --file .\drafts\requirement-plan.md

# 4. 由明确负责人记录批准决定
aiw requirement approve daily-withdrawal-report APPROVED --by svictor --reason "Scope and metrics are confirmed"

# 5. 提升为工程 Task
aiw requirement promote daily-withdrawal-report --task daily-withdrawal-report-implementation
```

如果 promotion 需要新建 Task，且工作区存在未提交改动，AIW 会检查脏路径。目标 Task/Change 目录有脏改动时始终拒绝；全部为无关改动时默认继续。不要手动编辑 Requirement 元数据绕过该保护。

## 命令参考

| 命令 | 作用 | 写入范围 |
| --- | --- | --- |
| `aiw requirement chat [id]` | 创建或恢复 Requirement Conversation | AIW Session；已有 Requirement 会保存 Session 引用 |
| `aiw requirement new <id> [title]` | 创建 Requirement 和最小元数据 | `requirements/<id>/` |
| `aiw requirement show <id>` | 输出 Requirement、审批和 promotion 状态 | 无 |
| `aiw requirement capture <id> <artifact> --file <path>` | 从明确给定的文件复制一个讨论产物 | Requirement 目录及元数据 |
| `aiw requirement approve <id> <decision> --by <actor> --reason <reason>` | 记录批准、延期或拒绝，并追加决策日志 | Requirement 目录 |
| `aiw requirement promote <id> --task <task-id>` | 创建或复用一个 AIW Task，并写入交接文件 | Requirement、Task 交接工件 |

`id` 和 `task-id` 只允许字母、数字、`-`、`_`、`.`。

### 支持的 artifact 类型

| 参数 | 目标文件 |
| --- | --- |
| `problem-brief` | `problem-brief.md` |
| `business-case` | `business-case.md` |
| `metric-brief` | `metric-brief.md` |
| `engineering-options` | `engineering-options.md` |
| `requirement-plan` | `requirement-plan.md` |

capture 必须指定 `--file`。源文件不能就是 Requirement 目录中的目标 artifact 文件；这样可避免把文件覆盖成自身或绕过 capture 记录。

## 文件结构

```text
requirements/<requirement-id>/
  requirement.toml
  problem-brief.md
  business-case.md
  metric-brief.md
  engineering-options.md
  requirement-plan.md
  decision-log.md
```

`requirement.toml` 保存 identity、状态、revision、审批、promotion，以及每个已 capture artifact 的路径和 SHA-256 摘要。Markdown artifact 保持可人工阅读。

promotion 成功后，Task 目录会有：

```text
openspec/changes/<task-id>/artifacts/requirement-handoff.md
```

handoff 只引用已批准产物及其摘要，包含批准范围、非目标、风险和待决问题的结构化段落；随后共享生成器会准备 OpenSpec 的 proposal、design、spec 和 tasks 内容，并保留已有人工文件。

## 重试与变更处理

- 同一 Requirement 再次 promote 到相同 Task 是幂等的：会复用 Task，并补齐缺失的 handoff，不会创建第二个 Task。
- 已有 handoff 不会被覆盖，保护后续工程人员添加的内容。
- 如果创建 Task 后写 handoff 失败，Requirement 保持可恢复的 promotion 状态；使用同一 Task ID 重试。
- capture 后若 artifact 被直接修改，摘要校验会拒绝将未记录的内容交接给工程。
- 当前版本没有 Requirement revision/synchronization 命令。需要改变已 promotion 的需求时，保留原记录，并在人工确认后等待显式 revision 流程；不要静默修改旧交接的语义。

## 与 Requirement Skills 的协作

`finance-requirement-intake`、`finance-value-assessment`、`finance-metric-brief`、`finance-engineering-options` 和 `finance-requirement-synthesis` 负责讨论、澄清和输出结构化草案。它们不直接写 Requirement 记录，也不创建 AIW Task。

推荐流程：

1. 用对应 Skill 讨论并得到草案。
2. 人工确认草案内容。
3. 用 `aiw requirement capture` 保存确认后的文件。
4. 用 `approve` 记录决策人和原因。
5. 用 `promote` 创建工程交接。
6. 在 Task 中按既有 OpenSpec workflow 创建 proposal、design、spec 和实现清单。

这样可保留人类决策链，同时避免需求讨论 Skill 越权修改工程规格或实施状态。

## 使用 Requirement Management Skill

当你希望从当前 Chat 直接进入需求工作流时，调用
`$requirement-management`。它会启动或恢复 Requirement Conversation，自动选择下一轮讨论，并在确认 checkpoint 前汇总待写入的内容。

若当前项目尚未安装该 Skill，先运行：

```text
aiw skills install requirement-management
```

例如：

```text
$requirement-management 我需要一个每日提现报告。
$requirement-management 继续 requirement daily-withdrawal-report。
```

该 Skill 会在每个 durable action 前请求明确确认。只有人类输入 `confirm` 或“确认”后，Conversation 才调用已有 Requirement 操作完成写入；它不会自动批准、自动 promotion、创建 OpenSpec 文档、开始实现或做发布决定。
## Requirement history

Completed and cancelled Requirements are stored separately from active discussion records:

```text
requirements/<id>/
requirements/archive/<id>/
requirements/cancelled/<id>/
```

```powershell
aiw requirement archive <id> --reason "Decision is complete"
aiw requirement cancel <id> --reason "No longer needed"

`--by <actor>` is optional, may appear in either order, and defaults to the current OS user.
aiw requirement list
aiw requirement list --archived
aiw requirement list --cancelled
aiw requirement list --all
```

Archive is available for `DECIDED`, `APPROVED`, and `PROMOTED` Requirements. The original artifacts and decision log are preserved, and read operations continue to accept the Requirement ID.
