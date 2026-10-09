# Requirement 管理与发现

## Requirements
### Requirement: 需求工件与版本

系统 MUST 将新 ISSUE 记录保存在 `docs/issues/<id>/`，使用 `issue.toml` 记录状态、revision、批准、推广、对话关联和工件摘要。旧 REQ 记录继续保存在 `docs/requirements/<id>/requirement.toml`；MUST NOT 自动迁移或改写历史 FD 引用。新需求 MUST 以 DRAFT、revision=1、批准 PENDING 和推广 NOT_STARTED 创建。

#### Scenario: 创建重复需求

- **WHEN** 相同 ID 已存在于活动、archive 或 cancelled 目录
- **THEN** 系统拒绝创建，不覆盖已有需求。

#### Scenario: 捕获正式工件

- **WHEN** capture 接受支持的工件类型和可读源文件
- **THEN** 系统写入对应工件、记录路径和内容摘要并递增 revision。
- **AND** 普通首次捕获将 DRAFT 推进为 DISCOVERED；捕获 issue-plan 或 requirement-plan 可将 DRAFT / DISCOVERED 推进为 DECIDED。新 ISSUE 的 Plan 文件为 issue-plan.md，旧 REQ 保留 requirement-plan.md，内部工件键保留 requirement-plan。

### Requirement: Issue 拆分关系

系统 MUST 允许尚未批准的子 Issue 记录一个已存在的父 Issue ID，并从父 Issue 查询直接子项。系统 MUST 拒绝自身关联、循环和覆盖已有父项；旧 REQ 记录缺少 `parent_id` 时仍作为根 Issue 读取。关联变更 MUST 递增子 Issue revision，已批准或关闭的子 Issue 不得改写来源关系。

#### Scenario: 拆分后查询

- **WHEN** 子 Issue 在批准前关联父 Issue
- **THEN** `show` 展示父 ID，`children` 能从父 ID 查询该子 Issue。

### Requirement: 人工决定与直接 CLI 兼容

approve MUST 接受 APPROVED、DEFERRED 或 REJECTED，并要求决定人及原因。直接 CLI 的 APPROVED MUST 要求当前状态为 DECIDED。系统 MUST 保存决定记录；聊天就绪评估不能替代这项显式决定。

#### Scenario: 正式批准

- **WHEN** 对 DECIDED 需求提交包含决定人与原因的 APPROVED 命令
- **THEN** 系统更新批准信息、状态、revision 和 decision-log。

#### Scenario: 仅有发现报告

- **WHEN** 发现对话生成建议批准的报告
- **THEN** 该报告本身不执行 approve，不建立 Task，也不进入实现。

### Requirement: 推广与可恢复关联

`issue promote <id>` 和兼容别名 `req promote <id>` MUST 要求 Issue 与批准记录均为 APPROVED，并通过 `fd new` 使用 Issue 标题创建编号 FD、关联来源 Issue、请求 Planner handoff。已关联的 Issue MUST NOT 重复创建 FD。推广 MUST NOT 创建 Task、旧 handoff 或 OpenSpec change，也 MUST NOT 修改既有 `[promotion]` 元数据；旧 `SPEC_DRAFTED` 记录 MUST 保持可读，不自动迁移或覆盖。AIW MUST NOT 伪造批准范围、工程决策或稳定 spec。

批准与推广仍各自要求原有显式授权。promote 只创建 FD 并记录 Planner handoff，不开始实现或验收；后续 FD 角色按 FD workflow 执行。

#### Scenario: 已批准 Issue 执行 promote

- **WHEN** 已批准 Issue 执行 promote
- **THEN** 系统以该 Issue 的标题和 ID 调用 `aiw fd new`，创建唯一编号 FD 并产生 Planner handoff。
- **AND** Requirement 的 `[promotion]` 状态和 `task_id` 保持原值。

#### Scenario: Issue 未批准或不存在

- **WHEN** Issue 缺失或其正式批准状态不是 APPROVED
- **THEN** promote 返回错误，不创建 FD 或 handoff。

#### Scenario: Issue 已关联 FD

- **WHEN** 已有 FD 关联同一 Issue，再次执行 promote
- **THEN** promote 报告重复关联，不创建第二个 FD。

### Requirement: 来源内容完整性

系统 MUST 在生成需求工件快照时计算实际内容摘要，并拒绝已捕获工件的实际摘要与登记摘要不一致的情况。

#### Scenario: 捕获后工件被直接改写

- **WHEN** 读取快照发现实际工件内容与捕获时摘要不符
- **THEN** 系统返回来源变化错误，不把它当作原捕获版本继续推广。

### Requirement: 终止与列表

archive / cancel MUST 要求操作人与原因，并将需求移动至对应终止目录。archive MUST 限制来源状态为 DECIDED、APPROVED 或 PROMOTED。终止后的需求 MUST 拒绝继续 capture、approve 或重新绑定对话。

#### Scenario: 需求归档

- **WHEN** 对允许归档的需求执行 archive
- **THEN** 系统记录终止信息和决定日志，将目录移动到原记录根目录的 `archive/<id>/`；新 ISSUE 使用 docs/issues，旧 REQ 使用 docs/requirements。

#### Scenario: 查询活动与终止需求

- **WHEN** 使用默认 list 或显式 archived / cancelled / all 过滤
- **THEN** 系统按相应范围返回需求，默认不包含终止需求。

### Requirement: 自动生成稳定编号

默认 `new <slug> [title]` MUST 创建 `ISSUE-001` 格式的完整 ID，数字至少三位且递增；slug MUST 为小写英文或数字片段，以单连字符连接，但不作为 ID 后缀。省略 title 时使用 slug 作为标题。生成后的 ID 不随标题变化。ISSUE 编号与旧 REQ 序列独立，活动、归档、取消和已预留编号均不得复用。

#### Scenario: 顺序创建

- **WHEN** 同一本地仓库连续创建两个新需求
- **THEN** 后一个数字大于前一个，输出、目录和元数据使用同一完整 ID。

#### Scenario: 超过五位

- **WHEN** 编号达到 1000
- **THEN** 数字宽度自然扩展，不回绕。

### Requirement: 精确 ID 兼容

`new --id <id> [title]` 和领域 Create MUST 保持精确 ID 创建语义。旧工件 MUST 保留原 ID，读取、捕获、批准和推广关联不自动迁移。

#### Scenario: 旧名称继续使用

- **WHEN** 显式创建不带数字前缀的合法旧 ID
- **THEN** 系统使用该名称，不追加前缀。

### Requirement: 持久化且互斥的编号分配

同一本地仓库 MUST 通过共享高水位记录和排他创建锁分配编号，并在创建目录前持久化序号。系统 MUST 考虑可见的活动、归档和取消目录中的已有编号；正常计数不倒退，显式数字 ID 不得复用已知编号。计数缺失或格式损坏时 MUST 持锁按现存需求目录最大编号恢复，无编号时从 1 开始；损坏原件 MUST 先备份并同步写盘，安全替换计数后明确输出恢复信息。计数丢失且历史需求已删除时，不保证历史编号永不复用。

#### Scenario: 取消或归档后创建

- **WHEN** 既有编号已被取消或归档
- **THEN** 后续创建仍使用更大的编号。

#### Scenario: 计数损坏后恢复

- **WHEN** 计数损坏且现存需求最大编号为 12
- **THEN** 系统保留损坏原件备份，重建计数并分配 13，输出恢复原因、最大编号、预留及下一个编号和备份路径。

#### Scenario: 没有已编号工件

- **WHEN** 计数缺失或损坏且成功扫描后没有已编号需求
- **THEN** 系统重建计数并从 1 分配，不改无编号需求的 ID。

#### Scenario: 恢复失败或创建锁占用

- **WHEN** 读取、备份或替换失败、编号溢出或创建锁被占用
- **THEN** 创建明确失败，不将 I/O 错误解释为空仓库，不删除原计数以强行替换，也不覆盖需求。

### Requirement: 正式工件与运行文件分离

系统 MUST 使用 docs/issues 和 docs/requirements 及各自的 archive、cancelled 目录读写对应记录，MUST NOT 回退旧根目录 requirements 或提供自动迁移。新 ISSUE 编号、锁、恢复备份和临时文件位于 .ai/issues，旧 REQ 使用 .ai/requirements；持久计数 MUST NOT 作为临时缓存清理。Session 历史存储位置不变；来源路径变化 MUST 使历史候选重新复核，不改写历史证据。按需单条迁移留给后续工作，本变更不提供迁移命令。

#### Scenario: 新路径生命周期

- **WHEN** 创建、读取、捕获、归档或取消需求
- **THEN** 正式工件位于对应 docs/issues 或 docs/requirements 范围内，运行文件不写入正式文档目录。

### Requirement: 统一身份定位

所有 Issue 生命周期命令及 FD 来源读取 MUST 使用同一定位规则：完整 ID 优先且大小写不敏感，REQ 数字前缀只在唯一匹配时解析。无匹配或多匹配 MUST 报错，不能任意选择记录或改写目标。父子关系和新 FD 来源 MUST 保存规范完整 ID，旧 FD 引用不重写。终止记录仍可读，但不能因此重新审批、修改或推广。

#### Scenario: 简写来源交接

- **WHEN** 唯一已批准 REQ 以 req00008 作为 FD 来源输入
- **THEN** FD 来源使用 REQ00008 的完整记录 ID，审批和重复关联检查使用同一身份。

#### Scenario: 结构化来源信息

- **WHEN** 调用 issue show <id> --json
- **THEN** 返回规范 id、status、approval_status 并校验已捕获工件摘要；FD 创建使用该接口而不是自行拼接 Requirement 目录。

#### Scenario: 不回退旧路径

- **WHEN** 需求只存在于旧 requirements 目录
- **THEN** 读取和列表不加载它，编号扫描不考虑旧目录，程序不自动移动文件。

#### Scenario: 移动后加载来源

- **WHEN** 工件已直接移动到新路径但元数据仍记录历史路径
- **THEN** 按已注册工件类型解析真实新路径，保留原文件内容、身份和 revision，历史候选按实际来源路径重新复核。

### Requirement: 聊天使用实际创建身份

聊天 prepare MUST 不占号；confirm MUST 返回实际完整 ID，后续上下文、Session 绑定与 memory 使用该 ID。历史 pending action 未声明自动编号时 MUST 保持精确 ID 语义。

#### Scenario: 确认自动编号创建

- **WHEN** 用户确认以 slug 准备的新需求
- **THEN** 系统分配完整 ID，并用该 ID 继续读取和捕获需求。

#### Scenario: 尚未确认

- **WHEN** 创建动作仅处于 prepare 阶段
- **THEN** 不生成 Requirement，也不消耗编号。

## 实现依据

- [存储、状态与摘要](../../../internal/issue/store.go)：Create、Capture、Approve、StartPromotion、CompletePromotion、ArtifactSnapshot、moveTerminal。
- [编号分配](../../../internal/issue/numbering.go)、[创建参数](../../../cmd/aiw-req/requirement_creation.go)。
- [命令与推广编排](../../../cmd/aiw-req/requirement.go)：DispatchRequirement、promoteRequirement、OpenSpec delegation。


---

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

策略选择 SHOULD 根据已有证据与用户细微偏好选择明显较优方案，并在 Issue Plan 记录理由。方案接近或缺少关键事实时才询问人类；真实来源冲突仍需呈现证据并请人类裁决。Issue 可覆盖 bug、feature、modification，并可按独立结果拆分为子 Issue。

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

- [上下文](../../../internal/issue/context.go)、[路径与预算](../../../internal/issue/context_sources.go)、[方法选择](../../../internal/issue/context_methods.go)。
- [覆盖校验](../../../internal/issue/coverage.go)、[问题排序](../../../internal/issue/questions.go)、[对话编排](../../../internal/issue/conversation.go)。
- [恢复](../../../internal/issue/conversation_history.go)、[就绪检查](../../../internal/issue/readiness.go)。
- [聊天接入与捕获检查点](../../../cmd/aiw-req/requirement_discovery.go)、[批准检查点](../../../cmd/aiw-req/requirement_readiness.go)、[promote 命令](../../../cmd/aiw-req/issue-promote.go)。

%% 引用有效、JSON 合法和问题排序正确，不证明模型找全了业务缺口；专业质量仍需人工评审。

