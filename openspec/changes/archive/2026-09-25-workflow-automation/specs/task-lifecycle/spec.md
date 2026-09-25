# task-lifecycle：自动化开发工作流变更

来源：[已批准 Requirement Plan](../../../../../docs/requirements/REQ00001-workflow-automation/requirement-plan.md)。以下是拟交付规范与待执行验收；按 Plan 后续确认覆盖旧来源中的人工计划审批和通知重试规则。未列出的稳定规格条款保持不变。

## MODIFIED Requirements

### Requirement: 状态由 Workflow Core 映射与派生

status/done MUST 由 Core 事实派生并分开 planning、execution、validation、delivery、workspace。Work Item 接受服从 SW14；缺必需验证时不能以 checkbox/旧 metadata/waived 派生完成。开发完成事实保存后立即登记通知和汇总，Git、知识、Verifier 的状态单独展示。

#### Scenario: 辅助结果未结束
- **WHEN** 所有实现项已按当前证据接受而辅助队列仍有缺口
- **THEN** 开发完成保持成立，辅助失败不撤销它；交付仍可 pending。

## ADDED Requirements

### Requirement: SW34 生命周期及设计闭合

系统 MUST 满足以下规则：本批维持 REQ00001-workflow-automation → workflow-automation 的一个 Requirement、一个 Task/change 生命周期，按 E01–E08 分组。推广后的 proposal、design、specs 和 tasks 须基于已批准目标、验收和授权政策形成实质规划，不能以通用模板冒充语义完成。工程在对应能力实施前闭合状态机、存储、迁移、恢复等设计，不逐字段或为补全工件再次请求用户确认；只有改变已确认行为、权限或成本边界的决定才重新提问。需求批准、实现、验证及交付各有其适用授权，不以产品中的 AI 计划审批取代 Requirement 的用户决定。

追踪：SW34 / AC34。

#### Scenario: AC34 验收行为

- **WHEN** 推广已批准的本批需求
- **THEN** 一个 Task/change 含有需求驱动的 proposal/design/specs/tasks 与 E01–E08 追踪；普通工件补全不新设用户确认，设计未知留给对应工程项
