# Requirement 管理

## Purpose

固定需求从正式捕获、人工决定到 Task 推广的当前行为，以及它与发现对话的权限边界；通过来源摘要、版本及终止状态场景，明确已有需求工件何时可继续使用。
## Requirements
### Requirement: 需求工件与版本

系统 MUST 将活动需求保存在 `docs/requirements/<id>/`，使用 `requirement.toml` 记录状态、revision、批准、推广、对话关联和工件摘要。新需求 MUST 以 DRAFT、revision=1、批准 PENDING 和推广 NOT_STARTED 创建。

#### Scenario: 创建重复需求

- **WHEN** 相同 ID 已存在于活动、archive 或 cancelled 目录
- **THEN** 系统拒绝创建，不覆盖已有需求。

#### Scenario: 捕获正式工件

- **WHEN** capture 接受支持的工件类型和可读源文件
- **THEN** 系统写入对应工件、记录路径和内容摘要并递增 revision。
- **AND** 普通首次捕获将 DRAFT 推进为 DISCOVERED；捕获 requirement-plan 可将 DRAFT / DISCOVERED 推进为 DECIDED。

### Requirement: 人工决定与直接 CLI 兼容

approve MUST 接受 APPROVED、DEFERRED 或 REJECTED，并要求决定人及原因。直接 CLI 的 APPROVED MUST 要求当前状态为 DECIDED。系统 MUST 保存决定记录；聊天就绪评估不能替代这项显式决定。

#### Scenario: 正式批准

- **WHEN** 对 DECIDED 需求提交包含决定人与原因的 APPROVED 命令
- **THEN** 系统更新批准信息、状态、revision 和 decision-log。

#### Scenario: 仅有发现报告

- **WHEN** 发现对话生成建议批准的报告
- **THEN** 该报告本身不执行 approve，不建立 Task，也不进入实现。

### Requirement: 推广与可恢复关联

promote MUST 要求有效的 APPROVED 决定，并把一个 Requirement 关联到一个 Task。已有关联只允许继续同一 Task 的推广。推广 MUST 生成 handoff，并通过共享需求工件生成流程准备 proposal、design、capability specs 和 tasks。系统 MUST 在工件通过来源、结构、需求覆盖、保护式写入及清单同步检查后才完成新的推广并记录 SPEC_DRAFTED；MUST NOT 把通用模板、失败调用或等待会话 Agent 的状态报告为完成。

系统 MUST 报告路由建议结果或其不可用诊断；无模型时可使用确定性配置或明确未配置状态，MUST NOT 因可选模型建议不可用而使已经通过交接验证的工件永远无法完成推广。后续执行的路由前置条件不因推广完成而被豁免。批准和推广仍各自要求原有正式授权，已授权范围内的工件补齐不要求重复批准。

#### Scenario: 推广批准需求

- **WHEN** 已批准需求通过创建预检并完成候选内容检查
- **THEN** 系统复用或创建目标 Task，写入表达实际需求的工件，同步清单并记录 SPEC_DRAFTED。
- **AND** 人工内容、清单身份、批准记录与来源关联保持完整。

#### Scenario: 无模型时等待接续

- **WHEN** 没有明确可用的生成模型或配置候选全部失败
- **THEN** 系统保留 Task 关联并输出交接文件实际路径与恢复入口，不把该次推广记录为 SPEC_DRAFTED。

#### Scenario: 恢复中断的推广

- **WHEN** 需求已关联 Task 且生成、写入或同步中断
- **THEN** 继续同一请求的可恢复步骤，拒绝不同 Task 关联，不重复创建 Task 或清单映射。

#### Scenario: 会话 Agent 完成交接

- **WHEN** Agent 提交绑定当前请求的候选且通过统一检查和同步
- **THEN** 系统完成同一推广，不要求为已批准范围再次批准，不启动实现。

#### Scenario: 历史完成记录缺少内容证据

- **WHEN** 显式恢复旧 SPEC_DRAFTED 需求且缺少生成接受证据
- **THEN** 系统保留历史状态与审批审计，报告需要复核，不把旧状态等同于本次内容检查通过。

### Requirement: 来源内容完整性

系统 MUST 在生成需求工件快照时计算实际内容摘要，并拒绝已捕获工件的实际摘要与登记摘要不一致的情况。

#### Scenario: 捕获后工件被直接改写

- **WHEN** 读取快照发现实际工件内容与捕获时摘要不符
- **THEN** 系统返回来源变化错误，不把它当作原捕获版本继续推广。

### Requirement: 终止与列表

archive / cancel MUST 要求操作人与原因，并将需求移动至对应终止目录。archive MUST 限制来源状态为 DECIDED、APPROVED 或 PROMOTED。终止后的需求 MUST 拒绝继续 capture、approve 或重新绑定对话。

#### Scenario: 需求归档

- **WHEN** 对允许归档的需求执行 archive
- **THEN** 系统记录终止信息和决定日志，将目录移动到 `docs/requirements/archive/<id>/`。

#### Scenario: 查询活动与终止需求

- **WHEN** 使用默认 list 或显式 archived / cancelled / all 过滤
- **THEN** 系统按相应范围返回需求，默认不包含终止需求。

### Requirement: 自动生成稳定编号

默认 `new <slug> [title]` MUST 创建 `REQ00001-slug` 格式的完整 ID，数字至少五位且递增；slug MUST 为小写英文或数字片段，以单连字符连接。生成后的 ID 不随标题变化。

#### Scenario: 顺序创建

- **WHEN** 同一本地仓库连续创建两个新需求
- **THEN** 后一个数字大于前一个，输出、目录和元数据使用同一完整 ID。

#### Scenario: 超过五位

- **WHEN** 编号达到 100000
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

系统 MUST 仅使用 docs/requirements 及其 archive、cancelled 目录读写正式工件与扫描需求编号，MUST NOT 回退旧根目录 requirements 或提供迁移命令。编号、锁、恢复备份与临时文件 MUST 位于 .ai/requirements，持久计数 MUST NOT 作为临时缓存清理。Session 历史存储位置不变；来源路径变化 MUST 使历史候选重新复核，不改写历史证据。

#### Scenario: 新路径生命周期

- **WHEN** 创建、读取、捕获、归档或取消需求
- **THEN** 正式工件始终位于 docs/requirements 范围内，运行文件不写入正式文档目录。

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

- [存储、状态与摘要](../../../internal/requirement/store.go)：Create、Capture、Approve、StartPromotion、CompletePromotion、ArtifactSnapshot、moveTerminal。
- [编号分配](../../../internal/requirement/numbering.go)、[创建参数](../../../internal/commands/task/requirement_creation.go)。
- [命令与推广编排](../../../internal/commands/task/requirement.go)：DispatchRequirement、promoteRequirement、prepareOpenSpecArtifactsShared。
- [回归材料](../../../internal/requirement/store_test.go)；本次未执行。
