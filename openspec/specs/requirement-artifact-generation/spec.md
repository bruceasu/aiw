# requirement-artifact-generation Specification

## Purpose
TBD - created by archiving change requirement-artifact-generation. Update Purpose after archive.
## Requirements
### Requirement: 绑定批准来源的生成输入

系统 MUST 以批准的 Plan、捕获正文、已确认结论及明确修订关系建立生成快照，绑定 Requirement、revision、批准记录、Task、相关稳定规格和输入/目标摘要。未确认讨论 MUST NOT 作为批准事实，系统 MUST NOT 声称读取未提供的聊天内容。来源缺失、冲突、摘要不符或批准范围变化 MUST 阻止接受候选。

#### Scenario: 已确认修订取代旧规则

- **WHEN** 当前快照包含有效的已确认替代关系
- **THEN** 生成使用替代后的规则并保留来源追溯，不由旧正文覆盖新结论。

#### Scenario: 仅有未保存会话

- **WHEN** 所需规则只存在于 CLI 未收到的聊天中
- **THEN** 系统报告缺少来源，不推测规则或伪造确认。

### Requirement: 生成真实业务工件

系统 MUST 生成表达实际目标、范围、规则、约束、异常与验收的 proposal、design、specs 和 tasks。capability MUST 来自业务功能与既有规格，支持多个新增/修改能力。未知工程细节 MUST 如实记录，不得臆造。

#### Scenario: 不同业务需求

- **WHEN** 输入分别为订单 CSV 导出与库存低水位提醒
- **THEN** 各套工件分别包含其字段/权限/异常/验收和相应任务，不得仅替换标题而复用生成器说明。

#### Scenario: 一个需求涉及多个能力

- **WHEN** 来源同时要求新增能力并修改既有能力
- **THEN** proposal 与对应 delta specs 一致标明新增及修改内容，不固定使用 requirement-management。

### Requirement: 冻结已批准的工件目标

系统 MUST 从批准的 Requirement Plan 中读取明确的 OpenSpec Targets，并在 Generation Request 中冻结唯一允许的目标清单。每个 Target MUST 包含 capability ID 与 `new` 或 `modified` 意图；基础文件为 proposal.md、design.md 和 tasks.md，每个 capability 只允许对应 specs/<capability-id>/spec.md。候选 MUST NOT 增加、重命名、删除或写入清单外目标。系统 MUST 为已有目标保存摘要，并为预期新增目标保存 `absent` 基线；候选接续时 MUST 重验这些基线。

#### Scenario: 已声明的新增和修改能力

- **WHEN** 批准计划声明一个 `new` capability 和一个 `modified` capability，且两者的冻结基线均有效
- **THEN** 请求只允许基础文件及这两个对应的 spec 路径，并允许候选在同一清单内生成完整工件。

#### Scenario: 候选临时增加 capability

- **WHEN** 候选包含未列入冻结清单的 specs/<capability>/spec.md，或遗漏已声明的 capability spec
- **THEN** 系统拒绝候选，不从候选路径补充批准范围或目标基线。

#### Scenario: 新增目标的基线已变化

- **WHEN** 清单中的 `new` capability 在请求创建后、候选接受前已被第三方创建
- **THEN** 系统报告目标基线冲突并保留现有文件，不写入候选或接受请求。

### Requirement: Provider 无关且有界的生成

系统 MUST 使用统一候选契约，并只尝试本功能明确配置的模型候选。每项 MUST 按其 provider 独立解析 model、endpoint、credentials 和 command；密钥 MUST NOT 写入审计。每个去重配置候选在同一请求最多调用一次，调用必须可取消且有有限超时。

#### Scenario: 主模型失败后使用备用

- **WHEN** 主候选不可用、超时或返回无效生成结果且有后续配置候选
- **THEN** 系统记录原因后尝试下一项，不携带主 provider 的模型名、endpoint 或凭据。

#### Scenario: 配置候选耗尽

- **WHEN** 无明确配置或全部候选失败
- **THEN** 系统进入交接，不自动探测未选择的云服务/本地模型，不无限重试。

#### Scenario: 输入或写入冲突

- **WHEN** 失败来自来源版本、权限或人工内容冲突
- **THEN** 系统停止接受并报告缺口，不通过换模型隐藏问题。

### Requirement: 可恢复的会话 Agent 交接

系统 MUST 输出包含输入身份、来源、允许文件、候选格式和恢复入口的交接文件，并显示实际路径。会话 Agent 候选 MUST 通过与模型候选相同的检查。CLI MUST NOT 宣称自动调用宿主聊天 Agent；无人接续时 MUST 保持明确未完成。

#### Scenario: 无配置但有交互式 Agent

- **WHEN** 用户已批准 promotion 且当前 Agent 按交接生成候选
- **THEN** 用户无需为同一范围重新批准或重跑 promote，prepare-spec 接续入口验证并完成生成。

#### Scenario: 候选已过期

- **WHEN** 提交的 request_id、输入摘要或目标基线不匹配当前状态
- **THEN** 拒绝接受并解释过期原因，不使用旧候选覆盖新内容。

### Requirement: 内容检查与来源覆盖

系统 MUST 验证必需文件、至少一个 capability spec、delta 结构及必要来源项到实际要求/场景的覆盖；MUST 拒绝空壳、通用生成器模板、不可解析引用和缺失业务要求。模型自述成功及 JSON 合法 MUST NOT 代替检查。报告 MUST 区分结构、覆盖、工程就绪和实现验收。

#### Scenario: 结构合法但遗漏禁止事项

- **WHEN** 候选满足章节和 JSON 结构但未覆盖来源中的必要禁止事项
- **THEN** 系统拒绝接受并报告遗漏来源项。

#### Scenario: 工程细节仍待深化

- **WHEN** 业务范围与规则完整而允许延期的工程决定仍未确定
- **THEN** 记录 TODO/Verification 与不确定事项，不伪造设计决定或宣称实现就绪。

### Requirement: 保留人工内容并安全恢复

系统 MUST 在写入前检查所有目标路径与基线摘要，只替换本生成器拥有且未变更的内容或完整匹配的已知生成占位。人工内容、已有清单身份/完成标记及 Core-owned 区域 MUST 保留。系统 MUST 记录部分写入进度，只有全部写入和受管清单同步成功后才接受请求。

#### Scenario: 新 Task 已有通用清单

- **WHEN** newTask 创建了完整匹配的通用 tasks.md
- **THEN** 将其转换为实际需求清单并由受管同步维护映射，不因文件已存在跳过生成，也不手写 Core 状态。

#### Scenario: 已有人工修改

- **WHEN** 目标基线变化或无法证明占位内容可安全替换
- **THEN** 报告冲突，保留人工内容，不盲目覆盖或删除。

#### Scenario: 写入后同步失败

- **WHEN** 正式文件已部分或全部写入但清单同步失败
- **THEN** 请求保持未接受并可恢复；恢复核对原/已写摘要后接续，不创建重复 Task 或重复清单项。

#### Scenario: 候选路径越界

- **WHEN** 候选使用父目录路径、绝对路径、越界链接或受保护目标
- **THEN** 拒绝写入并报告非法目标，不修改生命周期或批准记录。

