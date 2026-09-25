## ADDED Requirements

### Requirement: Explicit per-turn requirement context

系统 SHALL 在每轮需求讨论前装载当前需求身份、版本、已保存工件内容、已确认结论、未决问题、本轮输入及选定讨论方法，并区分确认事实与候选内容。系统 MUST NOT 仅依赖后端线程历史或工件路径。

#### Scenario: Resume without backend history
- **WHEN** 一个已有 Requirement 在没有可用模型历史的会话中恢复
- **THEN** 本轮输入包含当前已保存需求的实际内容和来源
- **AND** 未保存的历史不被捏造为已恢复事实

#### Scenario: New requirement has no artifacts
- **WHEN** 用户开始新需求讨论
- **THEN** 系统以原始请求和可获得的项目背景开始发现
- **AND** 不虚构需求 ID、已有决定或已捕获工件

### Requirement: Bounded and traceable source selection

系统 SHALL 按需求引用和相关模块选择允许范围内的本地背景，记录来源标识、版本或指纹、用途和装载结果，并采用有界预算。系统 MUST NOT 无差别加载整个仓库、凭据或项目范围外资料。

#### Scenario: Required source is unavailable
- **WHEN** 必需需求工件或当前方法不可读，或无法在预算内装载
- **THEN** 系统明确指出缺失内容及其影响
- **AND** 不声称上下文完整或建议需求已就绪

#### Scenario: Optional background is omitted
- **WHEN** 可选背景超出预算
- **THEN** 系统保留关键需求和方法，记录被省略来源与原因

#### Scenario: Referenced path escapes allowed scope
- **WHEN** 工件引用试图读取允许项目范围外的文件
- **THEN** 系统拒绝读取并记录原因，不将内容传给模型

### Requirement: Applicable discussion methods are actually loaded

系统 SHALL 使用通用需求发现基线，并依据需求领域与当前缺口选择允许的支持 Skill、装载其实际内容和记录版本。系统 MUST NOT 默认把非金融需求强制路由到金融流程，或声称应用了未读取的方法。

#### Scenario: Developer tool requirement
- **WHEN** 用户讨论开发工具通知且不存在金融业务规则
- **THEN** 系统使用通用发现方法而非强制金融指标讨论

#### Scenario: Domain-specific metric gap
- **WHEN** 金融指标口径是需求的重要未决问题
- **THEN** 系统选择可用的指标讨论方法并提供其内容
- **AND** 若方法不可用，明确报告而不伪称已经应用

### Requirement: Evidence-linked discovery coverage

系统 SHALL 评估角色、问题、现状、目标、范围、规则、异常、数据、权限、依赖及验收等维度。
每项 MUST 标为已明确、待确认、存在冲突或不适用，并提供来源、影响或不适用理由。模型推断 MUST NOT 自动成为人类已确认事实。

#### Scenario: Conflicting rules
- **WHEN** 两个来源给出相互冲突的关键规则
- **THEN** 系统标识双方来源和受影响决定，并优先澄清
- **AND** 不静默选择其中一方

#### Scenario: Invalid model assessment
- **WHEN** 模型评估使用无效结构、状态或不存在的来源
- **THEN** 系统保留诊断信息但不接受该评估为推进或批准依据

### Requirement: Focused professional questions

系统 SHALL 根据缺口对目标、范围和正确性的影响选择问题，每轮有问题时提出 1～3 个，并说明已知依据、待决事项和影响；有真实备选方案时说明取舍。系统 MUST NOT 无依据补造业务事实或无理由重复询问已确认问题。

#### Scenario: Vague notification request
- **WHEN** 用户仅提出“任务完成后通知我”
- **THEN** 系统识别并询问完成定义、人工介入或失败处理中的高影响缺口
- **AND** 不直接把这句话视为足够批准的完整需求

#### Scenario: Settled scope
- **WHEN** 通知仅限 Task 完成或需人工处理已被确认
- **THEN** 系统引用该约束并询问尚未明确内容，除非出现新冲突才重新讨论范围

### Requirement: Gap-driven phase refresh

系统 SHALL 根据新回答和确认后的最新工件重新决定讨论阶段。系统 MUST NOT 仅因某文件存在就判定内容充分，也不得在确认改变需求后继续使用过期阶段。

#### Scenario: Brief exists but critical rule is missing
- **WHEN** Problem Brief 已保存但关键业务规则仍缺失
- **THEN** 系统继续相关澄清而不直接建议批准

#### Scenario: Capture changes the next discussion
- **WHEN** 用户确认捕获工件并解决当前缺口
- **THEN** 下一轮重新读取工件并选择剩余最高优先级缺口或汇总阶段

### Requirement: Recoverable confirmed and candidate conclusions

系统 SHALL 以正式需求工件和决策记录恢复已确认事实，以版本绑定的 Session 记录辅助恢复未确认问题。
系统 MUST 区分过期候选与当前有效事实，不引入第二套权威生命周期或自动改写旧记录。

#### Scenario: Candidate belongs to an older revision
- **WHEN** 会话候选依据的版本与当前需求不一致
- **THEN** 系统标记候选需要复核并重新评估，不覆盖现有正式结论

#### Scenario: Legacy session lacks structured assessment
- **WHEN** 旧会话没有新版覆盖评估
- **THEN** 系统从已有正式工件重新构造上下文，明确说明无法恢复的未保存讨论

### Requirement: Draft capture and readiness are distinct

系统 SHALL 允许经确认保存不完整草稿；建议批准前 SHALL 检查目标、范围、关键规则和可验证验收条件等适用业务维度，不存在未解决的关键冲突。
Requirement Plan MUST 明示事实、假设、非目标、验收实例、来源和剩余决定。

#### Scenario: Incomplete plan is captured
- **WHEN** 用户确认保存包含阻塞问题的 Requirement Plan
- **THEN** 系统保存草稿但不因保存成功就建议批准

#### Scenario: Business context is sufficient
- **WHEN** 适用关键维度已明确且无阻塞业务冲突
- **THEN** 系统可以提出汇总及人工批准建议
- **AND** 可延后的工程设计问题仍被显式保留

#### Scenario: Engineering postponement needs human confirmation
- **WHEN** 模型建议将设计问题留给后续工程阶段
- **THEN** Plan 明示问题及延后理由，系统展示并要求人类确认原文片段
- **AND** 未确认前不能将该建议视为已接受的非阻塞延后
- **AND** 已确认延后也不能豁免未解决的业务缺口或冲突

#### Scenario: Approval evidence changes after display
- **WHEN** 聊天批准检查点展示后 Requirement 版本、来源或确认记录发生变化
- **THEN** 系统拒绝依据旧检查点批准，要求重新评估和展示
- **AND** 直接 approve CLI 的原有契约不受此对话门槛影响

### Requirement: Human authority and compatibility are preserved

系统 MUST 保留创建、捕获、批准、延期、拒绝和晋升的显式人工确认。
自动评估 SHALL 只影响对话建议，不修改历史批准状态或暗中改变直接 approve CLI 的既有语义。

#### Scenario: Model recommends approval
- **WHEN** 模型认为需求已就绪
- **THEN** 系统展示建议并等待确认，不自动批准或晋升

#### Scenario: Old approved requirement has a discovered gap
- **WHEN** 恢复旧批准需求时发现内容缺口
- **THEN** 系统提示风险并请求相关决定，不自动撤销批准

### Requirement: Conversation evidence remains inspectable

系统 SHALL 使用现有 Session 记录机制保存本轮实际输入、所选方法及来源清单、候选评估和相关 turn / Requirement revision，使维护者可以区分输入缺失与模型未利用输入。

#### Scenario: Investigate a shallow question
- **WHEN** 维护者检查某轮提问依据
- **THEN** 能定位当时装载的来源版本、方法及缺失信息
- **AND** 后续文件变化不会被误认为当时模型已获得的新内容
