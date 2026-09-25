# ai-routing：自动化开发工作流变更

来源：[已批准 Requirement Plan](../../../../../docs/requirements/REQ00001-workflow-automation/requirement-plan.md)。以下是拟交付规范与待执行验收；按 Plan 后续确认覆盖旧来源中的人工计划审批和通知重试规则。未列出的稳定规格条款保持不变。

## MODIFIED Requirements

### Requirement: 路由计划与实际调度范围

系统 MUST 保存角色映射和 Compile Plan，按 Work Item、Actor、需求上下文和有效失败为新请求路由。固定配置推荐模型只可选择已配置 Profile；失败/无效推荐使用配置默认且不递归。Coder、Tester、Verifier 实际消费各自选择；Compiler/Runner 不使用 LLM。

#### Scenario: 无效推荐
- **WHEN** 推荐超出允许 Profile 或调用失败
- **THEN** 保存失败理由并使用配置默认，不递归调用推荐或要求用户逐次选择。

### Requirement: 请求绑定模型快照

新阶段请求 MUST 固定 Profile、provider/model 和无凭据摘要；单次覆盖在创建时应用，历史请求和在途请求不得回写。升级仅作用于下一次确需且允许派发的生成请求，按实际 provider/model 去重。

#### Scenario: 配置变化与重启
- **WHEN** 已派发请求恢复且当前配置改变
- **THEN** 使用已有快照核对该请求；不会回写模型或重置失败账。

## ADDED Requirements

### Requirement: SW15 路由

系统 MUST 满足以下规则：固定配置的路由模型结合 Work Item、角色、需求、上下文和失败选择已配置 Profile，记录理由；失败或无效推荐使用配置默认档位，不递归路由或逐次人工选择。Compiler/Test Runner 不选模型；名称不代表能力，相同实际 provider/model 的别名不算升级，历史请求快照不可回写。

追踪：SW15 / AC15。

#### Scenario: AC15 验收行为

- **WHEN** 推荐无效
- **THEN** 配置默认；Profile 别名映射同实际模型 → 不计升级；Compiler 不路由，旧请求模型不回写

### Requirement: SW16 角色失败与升级

系统 MUST 满足以下规则：接受前按 Work Item、责任 Actor、当前模型档位累计有效失败；同一生成执行的多条缺陷/重复读取不重复计数，延迟发现归到生成该产物的请求。中间成功、其他角色活动、环境故障均不清零。每档两次有效失败后，在确需且允许派发时升级到下一不同模型，每 Actor 最多两次；无更高模型或耗尽且达到失败阈值则暂停。新档从零、历史保留，Work Item 接受后结束计数。

追踪：SW16 / AC16。

#### Scenario: AC16 验收行为

- **WHEN** Coder 缺陷重复读取/多条错误
- **THEN** 同生成请求只计一次；编译通过不清零；两次有效失败升级，每 Actor 独立最多两次，耗尽阈值后停止

### Requirement: SW17 修复上限与优先级

系统 MUST 满足以下规则：测试失败回交后的修复派发由 Coder/Tester 共享最多 6 次；第 6 次结果满足条件仍接受，不允许第 7 次自动修复。已归因实现/测试缺陷以新规则替换旧三次连续编译失败及同类普通重试；报告/诊断/辅助/通知各自更小或专门上限优先，不叠加基础设施额度。先判断已有结果是否成功，再判断是否允许下一次派发，不虚扣未发生的升级。

追踪：SW17 / AC17。

#### Scenario: AC17 验收行为

- **WHEN** Coder/Tester 共用第六次修复成功
- **THEN** 接受；仍失败 → 不派第七次；第三次编译缺陷不再被旧阈值提前截停；专门上限不叠加
