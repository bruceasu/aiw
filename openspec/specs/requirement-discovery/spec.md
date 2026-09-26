# Requirement 发现

## Purpose

固定已接入 Requirement 聊天入口的上下文装载、覆盖评估、问题选择和人工确认行为。机器校验保证来源与结构约束，不保证模型的专业判断正确。

## Requirements

### Requirement: 每轮重建有来源的上下文

系统 MUST 根据当前输入、需求元数据、已捕获工件正文和摘要构造上下文，并区分正式来源、已确认片段与未确认候选。新需求入口 MUST 可在没有 Requirement ID 或历史模型线程时构造通用发现上下文。

#### Scenario: 恢复已有需求

- **WHEN** 用户继续一个已有 Requirement
- **THEN** 系统重新读取当前 revision 和登记工件正文，不仅传递文件路径。
- **AND** 来源清单记录路径、用途、摘要、装载状态及失败或省略原因。

#### Scenario: 必需来源失效

- **WHEN** 已登记工件缺失、摘要变化或无法在预算内装载
- **THEN** 本轮评估被阻断，不能静默把不完整上下文标为已装载。

### Requirement: 有界本地来源

上下文读取 MUST 限制在项目允许的本地路径内，拒绝路径穿越与符号链接。默认正文预算 MUST 为 64 KiB，调用方只能降低该上限。可选背景或模块来源被省略时 MUST 保留原因，不把可选省略伪装成必需来源成功。

#### Scenario: 输入超出预算

- **WHEN** 当前用户输入已经超出上下文预算
- **THEN** 系统在模型评估前返回错误。

#### Scenario: 可选来源不可用

- **WHEN** 必需来源完整，但某个可选本地来源无法装载
- **THEN** 系统记录省略信息；该可选来源不成为评估证据。

### Requirement: 实际装载发现方法

系统 MUST 装载内置 generic-discovery 基线。领域方法建议 MUST 带有当前已装载内容的有效摘要和原文引用。通过校验的金融领域建议可选择项目 `.agents/skills/` 中对应方法；术语不明确可增加 domain-modeling。方法正文 MUST 实际进入上下文。

#### Scenario: 无法证明领域建议

- **WHEN** 方法建议无效、过期或没有有效引用
- **THEN** 系统明确降级至通用发现方法。

#### Scenario: 已选方法文件缺失

- **WHEN** 已选方法的文件不可读或为空
- **THEN** 系统记录 blocked，不能宣称已使用该方法。

### Requirement: 带证据的覆盖评估

覆盖输出 MUST 是单个有效 JSON，绑定版本、Requirement ID 和 revision，并且恰好包含 roles、problem、current_workflow、goals、scope、rules、exceptions、data、permissions、dependencies、acceptance 十一维度。系统 MUST 校验来源摘要和原文片段，拒绝重复维度、未知字段、失效引用及无效状态。

#### Scenario: 判定已明确

- **WHEN** 模型将某维度标为 resolved
- **THEN** 该项必须包含当前有效的人工确认片段，且不能仍带开放问题。
- **AND** 当前用户输入、候选文本和方法说明不能自行充当已确认业务事实。

#### Scenario: 判定冲突或不适用

- **WHEN** 模型输出 conflict 或 not_applicable
- **THEN** conflict 必须包含至少两个不同证据片段和开放问题；not_applicable 必须包含理由且没有开放问题。

#### Scenario: 模型响应期间来源变化

- **WHEN** 第二次读取的需求或来源快照与评估输入不同
- **THEN** 系统拒绝该评估并要求重新评估，保留诊断信息。

### Requirement: 有限问题与动态阶段

系统 MUST 从有效覆盖评估中选择去重后的最多三个待确认问题，优先冲突，再按不可逆影响、正确性、范围、目标及其他影响排序。阶段 MUST 从当前评估派生，而不是仅依据文件是否存在。

问题 MUST 说明已知依据、待决事项与影响；有真实备选方案时说明取舍，不补造业务事实。已确认内容 MUST 复用，只有出现新的冲突或变化证据时才重新打开相关问题。

#### Scenario: 模糊请求与已确认范围

- **WHEN** 用户仅提出“任务完成后通知我”，或进一步确认只在任务完成及需要人工处理时通知
- **THEN** 系统询问尚未明确的完成定义、失败处理等高影响问题，不把短请求视为可批准需求，也不无依据重复询问已确认范围。

#### Scenario: 尚有多个缺口

- **WHEN** 评估包含 needs_input 或 conflict
- **THEN** 系统展示一至三个优先问题；存在 conflict 时阶段为 deep-discovery，仅有普通缺口时为 discovery。

#### Scenario: 无开放覆盖项

- **WHEN** 当前覆盖项都没有开放缺口
- **THEN** 阶段可成为 synthesis，但空问题列表本身不构成批准。

### Requirement: Session 恢复与过期候选

系统 MUST 使用 Session 保存本轮上下文、原始评估、诊断与 turn 关联。恢复 MUST 重读来源并重验覆盖；历史候选只能作为未确认讨论，不能恢复成批准、事实确认或待执行动作。

Session 证据 MUST 保留实际装载的方法、来源版本和缺失清单，使维护者能够区分输入缺失与模型未使用输入，不把后来修改的正文当作当轮模型已读到的内容。

#### Scenario: 历史候选已经失效

- **WHEN** revision、正式来源或方法正文发生变化，或者最新讨论记录未完成
- **THEN** 系统不复用旧候选，并解释需要重新评估的原因。

#### Scenario: 没有结构化历史

- **WHEN** 旧会话没有有效讨论证据
- **THEN** 系统从当前正式工件重建上下文，不承诺恢复未保存的讨论。

### Requirement: 草稿捕获与事实确认分离

聊天 capture MUST 先展示检查点。事实确认 MUST 绑定被展示的动作、需求 revision、草稿摘要和原文片段。没有明确事实片段的捕获 MUST 只保存草稿。

#### Scenario: 展示后草稿被更改

- **WHEN** 用户确认时 pending action、revision 或草稿摘要与检查点不同
- **THEN** 系统拒绝继续使用旧检查点。

#### Scenario: 替换正式来源

- **WHEN** capture 替换某工件
- **THEN** 对被替换正文的旧确认不会自动迁移；上一 revision 中仍有效的其他原文确认可以保留。

### Requirement: 有证据的批准建议

就绪评估 MUST 同时检查业务覆盖和 Requirement Plan 正文。Plan MUST 覆盖事实、假设、目标、范围、非目标、规则、验收实例、来源和剩余决定；事实、目标、范围、规则与验收声明需要人工确认依据。未确认的设计延后不能使报告就绪，设计延后也不能豁免业务缺口。

#### Scenario: Plan 不充分

- **WHEN** Plan 缺失、只提供标题、包含 TODO / TBD 等占位符或缺少有效正文证据
- **THEN** 系统报告内容缺口，不建议批准，但仍允许草稿捕获。

#### Scenario: 确认聊天中的批准动作

- **WHEN** 用户确认 APPROVED 检查点
- **THEN** 系统重新读取历史、确认记录及来源，重新评估就绪并比较检查点摘要，之后才调用正式 approve。
- **AND** 后续讨论不就绪不会自行撤销此前的正式批准。

### Requirement: 人工确认边界

系统 MUST 保留创建、捕获、批准、延期、拒绝和推广的显式人工确认。对话评估 MUST NOT 暗中修改直接 approve CLI 的既有语义、自动执行批准或推广，或撤销历史批准。经确认可以保存不完整草稿，但 MUST NOT 因文件存在或 capture 成功而宣称需求已充分。

#### Scenario: 保存草稿或建议批准

- **WHEN** 用户确认保存不完整 Plan，或模型建议批准当前需求
- **THEN** 草稿可保存；批准和推广仍等待独立人工确认，未解决的关键业务缺口不能被工程设计延后建议豁免。

## 实现依据与归档同步

本规格已按 improve-requirement-management 的增量要求合并，保留代码基线中的具体校验规则；运行证据及未验证项见对应归档清单。

- [上下文](../../../internal/req/domain/context.go)、[路径与预算](../../../internal/req/domain/context_sources.go)、[方法选择](../../../internal/req/domain/context_methods.go)。
- [覆盖校验](../../../internal/req/domain/coverage.go)、[问题排序](../../../internal/req/domain/questions.go)、[对话编排](../../../internal/req/domain/conversation.go)。
- [恢复](../../../internal/req/domain/conversation_history.go)、[就绪检查](../../../internal/req/domain/readiness.go)。
- [聊天接入与捕获检查点](../../../cmd/aiw-req/requirement_discovery.go)、[批准检查点](../../../cmd/aiw-req/requirement_readiness.go)。
- [领域回归材料](../../../internal/req/domain/conversation_regression_test.go)、[命令回归材料](../../../cmd/aiw-req/requirement_regression_test.go)；本次未执行。

%% 引用有效、JSON 合法和问题排序正确，不证明模型找全了业务缺口；专业质量仍需人工评审。
