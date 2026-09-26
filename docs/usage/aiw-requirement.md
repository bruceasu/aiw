# AIW Requirement Management 使用手册

## 适用场景

Requirement Management 用于保存需求讨论的结果，并在人工明确批准后，受控地把它交接给 AIW Task。它位于需求讨论与 OpenSpec 工程规格之间：讨论 Skill 产出草案，Requirement 保存捕获内容和人工决策；保存草稿不代表其中每条断言已确认，promotion 才会创建或复用工程 Task。

它不会自动批准需求、自动开始实现或发布。promotion 会创建或复用 Task，写入 handoff，并强制委托 OpenSpec CLI 创建对应 change。AIW 不生成 proposal、design、spec 或 tasks；OpenSpec CLI 不可用时 promotion 失败并保留可恢复状态。

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
- `promote` 创建或复用一个 AIW Task，调用 OpenSpec CLI 创建 change，并建立 handoff 与 Workflow 映射。OpenSpec CLI 失败时不得宣称 `SPEC_DRAFTED`。
- `DEFERRED` 和 `REJECTED` 不可直接 promotion；当前版本没有重新打开命令，需等待未来的显式 revision 流程。

直接 CLI 的创建、capture 和 approve 将正式工件写入 `docs/requirements/<requirement-id>/`，计数、备份和临时文件写入 `.ai/requirements/`，不会创建 Task、OpenSpec change、Session、worktree 或分支。聊天适配器还会保存 Session 证据及确认检查点。

## 快速开始

人类用户从对话入口开始，不需要记忆 artifact 类型、文件路径或命令参数：

```powershell
# 新需求：创建可恢复的 Requirement Conversation
aiw req chat

# 已有需求：恢复其 Conversation
aiw req chat daily-withdrawal-report
```

Conversation 默认使用通用发现基线，只有金融领域的相关缺口才选择金融方法，不强制跑完固定阶段。每轮装载实际来源与方法，生成覆盖评估，再选择最多三个高影响问题；冲突优先。每次创建、capture、审批或 promotion 前，宿主展示目标、内容摘要和写入范围。AI 只能准备待执行动作；人类在会话中输入 `confirm` 或“确认”后，适配器才执行。

### 如何看待阶段、确认和批准建议

- 每轮通常有两次模型调用：方法选择和覆盖评估。每次回答或确认后重新读取来源，不仅依赖后端历史。
- 覆盖分为已明确、待确认、冲突、不适用；本轮回答先作为候选，不能自己证明“已确认”。
- capture 可保存不完整草稿。聊天可通过可选 --facts-json 展示拟确认的原文片段；只有人类确认且版本/草稿摘要匹配后，这些片段才记为确认。无需用户手工构造参数或编辑 Session JSON。
- synthesis、零个问题或 DECIDED 状态都不等于可以批准。就绪报告分开列出业务阻塞、Plan 缺项、待确认延后与已确认延后。
- Plan 必须明示事实、假设、目标、范围、非目标、规则、验收实例、来源和剩余决定，并给出正文依据。工程设计延后必须有理由和人类确认，不能豁免业务缺口。
- 聊天批准在展示与确认时复核当前证据。直接 approve CLI 保持原兼容语义，不是语义校验工具，也不应被 AI 用来绕过聊天门槛。发现旧批准记录的新风险只提示，不自动撤销批准。

### 恢复与排查

已有需求按 ID 恢复。Session 的来源正文、方法摘要、候选评估和 turn 编号位于
`.ai/sessions/<session-id>/artifacts/requirement-discussion-*.json`，
latest 指针指向最新轮；实际 prompt/output 保存在同一 Session 下。
这些是运行证据，不是第二套正式需求或批准记录。

候选 revision、来源或方法变化时要求复核。缺少结构化历史时从正式工件重建，
不会把 memory 或未保存的讨论推定为已确认事实，也不从失败的最新记录回退到旧成功结论。
无效模型输出保留原文和诊断，不自动推进或无限重试。

方法只加载项目 `.agents/skills/<允许名称>/SKILL.md`，不回退个人目录或递归加载链接。
缺少已选方法会阻塞；可选背景省略会在来源清单说明。不要因为传入路径就认为正文已被读取。
定位浅问题时先核对该轮实际 prompt 是否包含所需资料，再检查原始回答与宿主诊断。

提问质量的固定案例和人工评分方法见[专业提问人工评审](requirement-discovery-review.md)。
机器测试通过不等于案例已通过；运行真实模型需另行授权。

### 自动编号与旧 ID

新建需求默认只需提供小写英文短名，例如 `aiw req new add-chat-support "Chat support"`。
首次分配会输出 `REQ00001-add-chat-support`；后续 show、capture、approve、chat、promote 使用输出的完整 ID。
短名由小写字母或数字片段以单个连字符连接。数字至少五位，超过五位时自然扩展；标题变化不改变 ID。

编号在实际创建时分配，聊天 prepare 不占号，确认后才生成。取消、归档和创建失败不回收已经分配的编号，因此允许跳号。
同一本地仓库的 linked worktrees 共享 `.ai/requirements/sequence` 高水位记录和创建锁；不同克隆之间不保证全局唯一。
已有需求不重命名。依赖固定 ID 的脚本应使用 `new --id <id> [title]`；显式指定数字 ID 时，该数字必须大于已知高水位。
计数缺失或格式损坏时，在创建锁内扫描活动、archive、cancelled 需求目录名，按最大编号加一恢复；没有带编号需求时从 1 开始。正常计数不倒退。损坏原件先备份到 `.ai/requirements/backups/`，替换成功才创建需求，并输出恢复原因、目录最大编号、已预留编号、下一个编号及备份位置。
读取或备份失败、编号溢出及锁占用仍会报错；不要把删除计数器或活动锁作为重试方法。计数丢失且历史工件已被删除时，不能保证历史编号永不复用。创建已成功而 Session 后续写入失败时，先按报错中的完整 ID 检查工件，避免重复创建。

正式工件仅使用 `docs/requirements`，不回退到旧根目录 `requirements`，不提供迁移命令。旧项目需在停用旧版 aiw 后直接移动工件；旧计数如需保留，应一并移动到 `.ai/requirements/sequence`。不要同时运行新旧版本。
`.ai/requirements/sequence` 是持久高水位，`sequence.lock` 是活动锁，`backups` 保存恢复证据；它们不是可随意清理的缓存。正式工件写入暂存文件放在 `temporary`，恢复计数的短期暂存文件也在 `.ai/requirements` 内。Session 历史位置不变，旧候选在来源路径变化后需重新复核。

以下命令使用兼容的精确 ID 创建，适用于依赖固定名称的脚本：

```powershell
# 1. 创建 Requirement
aiw req new --id daily-withdrawal-report "Daily withdrawal report"

# 2. 查看当前状态
aiw req show daily-withdrawal-report

# 3. 捕获已由人确认的需求方案
aiw req capture daily-withdrawal-report requirement-plan --file .\drafts\requirement-plan.md

# 4. 由明确负责人记录批准决定
aiw req approve daily-withdrawal-report APPROVED --by svictor --reason "Scope and metrics are confirmed"

# 5. 提升为工程 Task
aiw req promote daily-withdrawal-report --task daily-withdrawal-report-implementation
```

如果 promotion 需要新建 Task，且工作区存在未提交改动，AIW 会检查脏路径。目标 Task/Change 目录有脏改动时始终拒绝；全部为无关改动时默认继续。不要手动编辑 Requirement 元数据绕过该保护。

## OpenSpec 委托

Requirement promotion 不接受 AIW 内置生成器或模型 profile。`aiw req promote`
会通过 `new --backend openspec` 委托 OpenSpec 创建 change；随后直接使用
OpenSpec 的命令和 Skill 完成 proposal、design、spec 和 tasks。OpenSpec CLI
不可用或创建失败时，AIW 保留 Requirement、Task 和 handoff，并返回可恢复的错误。

## 命令参考

| 命令 | 作用 | 写入范围 |
| --- | --- | --- |
| `aiw req chat [id]` | 创建或恢复 Requirement Conversation | AIW Session；已有 Requirement 会保存 Session 引用 |
| `aiw req new <slug> [title]` | 自动编号并创建 Requirement | `docs/requirements/<完整ID>/`、共享编号记录 |
| `aiw req new --id <id> [title]` | 精确创建兼容 ID | `docs/requirements/<id>/`，数字 ID 同时更新编号记录 |
| `aiw req show <id>` | 输出 Requirement、审批和 promotion 状态 | 无 |
| `aiw req capture <id> <artifact> --file <path>` | 从明确给定的文件复制一个讨论产物 | Requirement 目录及元数据 |
| `aiw req approve <id> <decision> --by <actor> --reason <reason>` | 记录批准、延期或拒绝，并追加决策日志 | Requirement 目录 |
| `aiw req promote <id> --task <task-id>` | 创建或复用 Task，委托 OpenSpec 创建 change，并写入交接文件 | Requirement、Task 交接工件 |

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
docs/requirements/<requirement-id>/
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

handoff 只引用已批准产物及其摘要，包含批准范围、非目标、风险和待决问题的结构化段落；随后由 OpenSpec 负责 proposal、design、spec 和 tasks 内容。

## 重试与变更处理

- 同一 Requirement 再次 promote 到相同 Task 是幂等的：会复用 Task，并补齐缺失的 handoff，不会创建第二个 Task。
- 已有 handoff 不会被覆盖，保护后续工程人员添加的内容。
- 如果创建 Task 后写 handoff 失败，Requirement 保持可恢复的 promotion 状态；使用同一 Task ID 重试。
- capture 后若 artifact 被直接修改，摘要校验会拒绝将未记录的内容交接给工程。
- 当前版本没有 Requirement revision/synchronization 命令。需要改变已 promotion 的需求时，保留原记录，并在人工确认后等待显式 revision 流程；不要静默修改旧交接的语义。

## 与 Requirement Skills 的协作

`finance-requirement-intake`、`finance-value-assessment`、`finance-metric-brief`、`finance-engineering-options` 和 `finance-requirement-synthesis` 负责讨论、澄清和输出结构化草案。它们不直接写 Requirement 记录，也不创建 AIW Task。

独立使用金融方法时可采用以下显式流程；普通用户优先使用 chat 自动准备草稿和检查点：

1. 用对应 Skill 讨论并得到草案。
2. 人工确认草案内容。
3. 用 `aiw req capture` 保存文件；直接 capture 本身不创建聊天的片段确认记录。
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
docs/requirements/<id>/
docs/requirements/archive/<id>/
docs/requirements/cancelled/<id>/
```

```powershell
aiw req archive <id> --reason "Decision is complete"
aiw req cancel <id> --reason "No longer needed"

`--by <actor>` is optional, may appear in either order, and defaults to the current OS user.
aiw req list
aiw req list --archived
aiw req list --cancelled
aiw req list --all
```

Archive is available for `DECIDED`, `APPROVED`, and `PROMOTED` Requirements. The original artifacts and decision log are preserved, and read operations continue to accept the Requirement ID.
