# 需求管理与开发工作流设计

## 状态

提案；尚未实现 `aiw requirement` 命令或修改相关 Skill。

## 目标

将需求讨论、工程实施和发布决策拆成具有单一所有者的阶段，使 AIW 能在未来自动衔接完整开发流程，同时不与 OpenSpec 或实施类 Skill 争夺文件和状态的写入权。

## 所有权边界

| 层 | 负责的事实 | 不负责的事实 |
|---|---|---|
| 人机需求讨论 Skill | 问题、价值、指标意图、技术选项、未知项和建议 | Task 生命周期、实现清单、分支、worktree、发布许可 |
| AIW Requirement Management（待实现） | 需求记录、讨论产物版本、人工批准、晋升记录 | OpenSpec 文档正文、实施过程状态 |
| AIW Task / Workflow | Task 状态、Session、worktree、handoff、执行编排 | proposal、design、spec、`tasks.md` 的业务正文 |
| OpenSpec | proposal、design、长期 spec、实施 checklist | AIW Task 生命周期与运行时租约 |
| 发布门禁 | 已实现变更的发布证据和上线结论 | 早期需求批准 |

任何需求讨论 Skill 都不得直接创建或修改 `task.toml`、`tasks.md`、`design.md`、长期 spec、分支或 worktree。

## 生命周期

```text
Raw request
  -> Requirement discussion artifacts
  -> Requirement record
  -> Human approval
  -> promotion
  -> AIW Task + OpenSpec change
  -> implementation and verification
  -> release gate
```

需求阶段的结论只说明需求是否足以由人类决定是否进入开发；不代表可以编码或发布。

当需求中的业务术语、参与者、实体关系或领域边界不稳定时，需求流程应先使用
`domain-modeling` 澄清并固化领域语言，再继续需求阶段产物。需求阶段只把确认后的
词汇、关系和边界返回 Requirement Artifact；不在该阶段创建 ADR 或直接修改工程
设计文件。进入 Task/OpenSpec 设计后，如果仍有工程领域模型缺口，再以 Engineering
mode 使用 `domain-modeling`，并将结果记录到 Task/OpenSpec 所属位置。

```text
Requirement:
DRAFT -> DISCOVERED -> DECIDED -> APPROVED -> PROMOTED
                         |              |
                         v              v
                     DEFERRED        REJECTED

Task / implementation:
TODO -> NEEDS_DECISION -> READY -> IN_PROGRESS -> DONE -> ARCHIVED

Release:
NOT_STARTED -> GO | GO_WITH_RISK | NO_GO
```

## Skill 重命名与职责

不保留旧 Skill 名称的兼容层。

| 旧名称 | 新名称 | 生命周期阶段 | 逻辑产物 |
|---|---|---|---|
| `office-hours-finance` | `finance-requirement-intake` | 需求发现 | `Problem Brief` |
| `business-review` | `finance-value-assessment` | 价值决策 | `Business Case` |
| `metrics-review` | `finance-metric-brief` | 指标意图与口径讨论 | `Metric Brief` |
| `eng-review-finance` | `finance-engineering-options` | 技术可行性与方案选项 | `Engineering Options` |
| `autoplan-finance` | `finance-requirement-synthesis` | 汇总与人工决策准备 | `Requirement Plan` |
| `release-review` | `finance-release-gate` | 实现后的发布阶段 | `Release Review` |

`finance-release-gate` 不属于默认需求讨论链路，也不能由 `finance-requirement-synthesis` 自动触发。

## Requirement Artifact Contract

需求阶段 Skill 以统一的逻辑产物交接。当前阶段返回结构化草案；未来由 AIW Requirement Management 持久化、版本化和关联。Skill 不决定最终文件路径。

```markdown
# Requirement Artifact

## Metadata
- Artifact Type:
- Requirement ID: %% OPTIONAL / NOT_ASSIGNED
- Stage:
- Status:
- Based On:
- Human Approval Required: yes

## Facts
## Assumptions
## Decision / Recommendation
## Open Questions
## Suggested Next Stage
```

规则：

- 未决问题、外部依赖、风险和待确认事实必须使用 `%%`，不使用 `TODO`。
- `Based On` 只引用上游产物标识，不复制上游全文。
- 任何 Skill 都可以提出建议，但不得把需求标记为 `APPROVED`，也不得创建工程任务。
- 不得将 blocker 自动转换为 OpenSpec `tasks.md` 项。
- 事实、假设与建议必须分开表达。

## 局部状态模型

各阶段状态不可互相推导，尤其不能把需求通过误认为工程或发布批准。

| Skill | 可用状态 |
|---|---|
| `finance-requirement-intake` | `CLEAR` / `NEEDS_INPUT` / `OUT_OF_SCOPE` |
| `finance-value-assessment` | `SUPPORTED` / `VALIDATE_FIRST` / `NOT_SUPPORTED` |
| `finance-metric-brief` | `DEFINED` / `PARTIAL` / `CONFLICT` |
| `finance-engineering-options` | `FEASIBLE` / `CONSTRAINED` / `NEEDS_DECISION` |
| `finance-requirement-synthesis` | `READY_FOR_HUMAN_DECISION` / `BLOCKED` / `DEFERRED` |
| `finance-release-gate` | `GO` / `GO_WITH_RISK` / `NO_GO` |

`READY_FOR_HUMAN_DECISION` 只表示需求材料完整到足以请求人类决定；它不等同于 `Task READY`。

## 需求阶段编排

`finance-requirement-synthesis` 支持下列讨论模式：

| 模式 | 行为 |
|---|---|
| `focused` | 只处理用户指定的一个阶段或问题 |
| `resume` | 基于已有逻辑产物补齐最早的阻塞阶段 |
| `synthesize` | 汇总已有产物，生成 `Requirement Plan` |
| `full` | 用户明确要求时，依次覆盖需求发现、价值、指标和技术选项 |

`full` 不执行发布门禁。它只在计划中记录发布尚未开始及其未来触发条件。

## Requirement 存储与最小元数据

Requirement 是需要跨 session、跨人员追溯的业务资产，因此默认位于版本控制的仓库根目录，而不是 `.ai/` 运行时目录：

```text
requirements/<requirement-id>/
  requirement.toml          # AIW Requirement Management 所有
  problem-brief.md
  business-case.md
  metric-brief.md
  engineering-options.md
  requirement-plan.md
  decision-log.md
```

`requirement.toml` 只存 AIW 拥有的身份、状态、批准和 promotion 摘要；讨论内容继续由各 Markdown 产物拥有。

```toml
id = "daily-withdrawal-report"
title = "Daily withdrawal report"
status = "DECIDED"
created = "2026-09-08"
updated = "2026-09-08"

[approval]
status = "PENDING" # PENDING | APPROVED | DEFERRED | REJECTED
by = ""
at = ""

[promotion]
status = "NOT_STARTED" # NOT_STARTED | TASK_CREATED | SPEC_DRAFTED | COMPLETE
task_id = ""
```

## `aiw requirement` 的最小命令集

该命令组是未来实现目标，不是本文档创建的现有 CLI。

| 命令 | 作用 | 写入边界 |
|---|---|---|
| `aiw requirement new <id>` | 创建 Requirement 目录与最小元数据 | 仅 `requirements/<id>/` |
| `aiw requirement show <id>` | 查看状态、产物索引、批准和 promotion | 只读 |
| `aiw requirement capture <id> <artifact>` | 保存经用户确认的 Requirement Artifact | 仅对应 Requirement Markdown 文件 |
| `aiw requirement approve <id>` | 记录人工批准、延期或拒绝 | 仅 `requirement.toml` 和 `decision-log.md` |
| `aiw requirement promote <id> --task <task-id>` | 创建或关联受管 Task，并生成交接输入 | Requirement 元数据、AIW Task 和交接工件 |

`capture` 不运行 Skill，也不解释需求内容；它只保存明确给出的产物。Skill 的输出与 Requirement 文件之间不应存在隐式写入。

## Promotion 事务与状态

promotion 是唯一跨越 Requirement 层和工程层的适配器。它只在 Requirement 的人工批准为 `APPROVED` 时开始。

```text
APPROVED Requirement
  -> validate identity and promotion state
  -> create or resolve AIW Task
  -> record Task link and source-artifact digests
  -> write requirement handoff for the Task
  -> request OpenSpec proposal/spec generation
  -> record the resulting OpenSpec artifact links
```

完成后的需求状态是 `PROMOTED`；Task 的实现和交付状态仍只由 AIW Workflow 管理。`SPEC_DRAFTED` 与 `COMPLETE` 描述 promotion 交接的完成度，不描述代码是否完成。

### Promotion 的幂等和失败恢复

- 同一 Requirement 默认只能关联一个活动 Task；再次 promote 必须返回已关联的 Task，而不是创建第二个 Task。
- 如果 Task 已创建但 OpenSpec 工件尚未生成，重复执行必须从 `TASK_CREATED` 恢复。
- 如果交接已生成但 OpenSpec 生成失败，保留交接与失败原因，并停在 `TASK_CREATED` 或 `SPEC_DRAFTED`；不得删除 Task 或覆盖已有 OpenSpec 内容。
- Requirement 产物在 promotion 后发生变化时，记录为新的 Requirement 版本；不得静默覆写已经交给 Task 的版本。后续同步应创建显式的变更请求或新的 promotion revision。
- `REJECTED`、`DEFERRED` 或未批准的 Requirement 不得 promotion，除非人类先改变批准状态并留下 decision-log 记录。

## Promotion 的预留接口

未来的 AIW Requirement Management 应是唯一跨越需求层与工程层的适配器：

```text
Requirement artifacts
  -> AIW Requirement record
  -> explicit human approval
  -> AIW-managed promotion
  -> AIW Task creation
  -> OpenSpec proposal / design / spec / tasks generation
```

promotion 必须先建立受管 AIW Task，再由 AIW 协调 OpenSpec 后端。它不能允许某个需求讨论 Skill 直接写入 OpenSpec 工件。

建议映射：

| Requirement 产物 | promotion 后的 OpenSpec 输入 |
|---|---|
| `Problem Brief` 与 `Business Case` | proposal 的问题、范围与预期价值 |
| `Metric Brief` | capability spec 变更的候选输入 |
| `Engineering Options` | design 的输入，不是已批准设计 |
| `Requirement Plan` | proposal 的验收条件、风险与交接上下文 |

具体权限模型、OpenSpec 生成器选择和 Requirement 版本摘要算法仍属于后续 `aiw requirement` 的独立设计。

## Requirement 到 Task 的交接契约

promotion 创建的交接工件应当是 Task 可消费的引用清单，而不是 Requirement 文档的副本。建议由 AIW 写入该 Task 的 `artifacts/requirement-handoff.md`：

```markdown
# Requirement Handoff

## Source
- Requirement ID:
- Requirement revision:
- Approved by:
- Approved at:
- Promotion at:

## Referenced Artifacts
| Artifact | Path | Revision / Digest | Status |
|---|---|---|---|

## Approved Scope
## Explicit Non-Goals
## Accepted Risks
## Open Decisions Carried Into Engineering
## Required OpenSpec Outputs
## Suggested Next Workflow Action
```

Task 后续的 agent 或 `aiw flow` 必须读取该交接文件和被引用的权威 Requirement 工件；它们不得把交接摘要视为可自由修改的需求原文。

## 自动化边界

AIW 可以自动推进机械性、可验证的转换，但不得自动做商业或发布授权：

| 阶段转换 | 可自动执行 | 必须保留人工决定 |
|---|---|---|
| Skill 草案 → Requirement capture | 否；先展示草案 | 是否保存为 Requirement 事实 |
| Requirement → `APPROVED` | 否 | 是否进入工程投入 |
| `APPROVED` → Task 创建 | 是，作为显式 promote 的一部分 | Task ID 与 promotion 指令本身 |
| Task → OpenSpec 草案 | 是，可由受控 flow 启动 | 方案、范围或未决项有实质变化时的确认 |
| OpenSpec → 实施 | 是，但仅在 Task 已受管且工程前置条件满足时 | 高风险变更、依赖、权限、迁移或部署授权 |
| 实施 → 发布 | 否 | 发布窗口、残余风险接受与最终 GO/NO_GO |

自动流程应按明确 phase 写入 Evidence 和 Gate，而不是依赖模型在自然语言中声称“已完成”：

```text
requirement-capture
  -> requirement-approval
  -> promotion
  -> openspec-proposal
  -> implementation
  -> verification
  -> release-gate
```

任何 phase 遇到 `%%` 阻塞项、缺少授权、缺少外部证据或高风险操作时，都必须停在对应 Gate 并把下一步交回人类。自动化不得提交代码、发布、执行迁移、创建外部工单或扩大验证范围，除非当前 Task 获得了明确授权。

## Skill 改造顺序

1. 创建共用 Requirement Artifact Contract，并移除所有直接 OpenSpec 写入说明。
2. 改造并重命名前五个需求阶段 Skill；统一阶段、输入、输出、局部状态与 handoff。
3. 将发布 Skill 移出需求链路，仅保留已实现变更的发布门禁职责。
4. 为新 Skill 集补充静态结构校验与迁移说明。
5. 在 Skill 契约稳定后，再设计 `aiw requirement` 的数据模型、CLI 和 promotion 适配器。

## 非目标

- 本设计不创建新的 AIW 命令。
- 本设计不改变现有 AIW Task、Session、worktree 或 OpenSpec 所有权。
- 本设计不自动生成 implementation tasks、代码、迁移或发布脚本。
- 本设计不让发布门禁提前参与需求批准。
## Requirement history and retention

Requirement records are independent from AIW Task and OpenSpec implementation status. Keep active records under `requirements/<id>/`; move completed records to `requirements/archive/<id>/` and cancelled records to `requirements/cancelled/<id>/`. Archive is allowed for `DECIDED`, `APPROVED`, or `PROMOTED` Requirements. Cancellation requires an actor and reason. Default listing excludes both historical roots, while explicit filters include them. All artifacts, approval data, promotion links, and decision logs are retained.
