# agent-session：自动化开发工作流变更

来源：[已批准 Requirement Plan](../../../../../docs/requirements/REQ00001-workflow-automation/requirement-plan.md)。以下是拟交付规范与待执行验收；按 Plan 后续确认覆盖旧来源中的人工计划审批和通知重试规则。未列出的稳定规格条款保持不变。

## MODIFIED Requirements

### Requirement: 可追踪的 prompt 组合

执行入口 MUST 保留 instructions、memory、phase、task 的实际组合，并装载角色所需需求、验收、Task 来源和依赖正文。派发前保存最终 prompt、来源身份/版本/读取状态、选择和降级理由。必需输入缺失或预算不足停止相关派发；摘要不可用时可使用完整原始工件。

#### Scenario: 摘要不可用
- **WHEN** 必需原始来源完整而摘要不可用
- **THEN** 记录降级、加载正文继续；历史 prompt 不被后来摘要覆盖。

### Requirement: Task handoff 与监督指令

Task Agent 入口 MUST 解析 handoff 精确来源并实际装载必需内容，按 Coder/Tester/Verifier 角色组织监督边界和输入；只传路径不满足装载要求。handoff 不自行增加验收约束，候选知识不能替代主需求。

#### Scenario: 必需精确版本缺失
- **WHEN** 主需求精确纳入的来源版本无法取得
- **THEN** 阻止受影响派发；普通可选知识不可读则登记降级，不伪装零命中。

## ADDED Requirements

### Requirement: SW25 实际输入与缺口

系统 MUST 满足以下规则：每轮提供需求、任务上下文、handoff 和相关知识的实际内容，按角色组织；保存实际 prompt、来源身份/版本/状态、读取错误、选择/跳过和降级理由。核心主来源、角色边界、必需依赖缺失或版本无法解释则阻塞受影响派发；完整原始工件可替代不可用摘要。普通项目知识可选，读取失败不冒充零命中；只有被已确认主来源精确纳入的知识版本成为必需约束，handoff 不自行加约束。历史输入不被 latest 回写。

追踪：SW25 / AC25。

#### Scenario: AC25 验收行为

- **WHEN** 必需正文缺失或超预算
- **THEN** 不派发；普通知识不可读 → 记录降级而非零命中；主来源精确引用版本缺失 → 阻塞；历史 prompt 不被 latest 改写
