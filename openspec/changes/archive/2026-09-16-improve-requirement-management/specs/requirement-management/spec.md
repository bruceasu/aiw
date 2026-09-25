## ADDED Requirements

### Requirement: Automatic stable requirement numbering

系统 MUST 为默认新建需求分配 `REQ00001-slug` 格式的递增 ID，编号至少五位，slug 为小写英文或数字片段以单连字符连接。完整 ID MUST 在创建后保持稳定。

#### Scenario: Sequential creation
- **WHEN** 在同一本地仓库连续确认创建两个新需求
- **THEN** 第二个编号大于第一个，返回的完整 ID 与目录及元数据一致

#### Scenario: Number width grows
- **WHEN** 下一个编号超过五位
- **THEN** 系统扩展数字宽度，不回绕或复用编号

### Requirement: Explicit and historical ID compatibility

系统 MUST 保留现有需求的 ID 和原有读取、捕获、批准及推广关联。CLI MUST 支持 `new --id <id> [title]` 精确创建，领域 Create MUST 保持精确 ID 语义。

#### Scenario: Legacy ID
- **WHEN** 显式创建或读取不带编号的合法旧 ID
- **THEN** 系统不自动重命名或添加前缀

### Requirement: Serialized durable allocation

系统 MUST 在同一本地仓库内互斥分配编号，先保存序号高水位再创建需求，允许跳号但 MUST NOT 因取消或归档回收编号。计数缺失或格式损坏时 MUST 在锁内扫描现存活动、归档及取消需求的目录名最大编号并恢复；无编号时下一编号为 1。正常计数 MUST NOT 因目录最大值较小而倒退。损坏原件 MUST 先备份，恢复 MUST 明确提示；读取、备份或替换失败、编号溢出及锁占用时 MUST 报错，不覆盖已有工件。

#### Scenario: Number exists in terminal artifacts
- **WHEN** 已有编号位于归档或取消目录
- **THEN** 新编号仍大于该编号

#### Scenario: Concurrent creation
- **WHEN** 创建锁已占用
- **THEN** 该次创建明确失败，不创建新需求或偷取锁

#### Scenario: Missing or damaged counter
- **WHEN** 序号文件缺失或损坏且现存需求最大编号为 12
- **THEN** 系统备份损坏原件（若有），安全重建计数并从 13 分配，输出恢复提示

#### Scenario: No numbered artifacts
- **WHEN** 序号文件缺失或损坏且扫描成功但没有带编号的需求
- **THEN** 系统修复计数后从 1 开始，不给旧 ID 自动重命名

#### Scenario: Failed recovery
- **WHEN** 目录读取、损坏原件备份或计数替换失败
- **THEN** 创建停止并报告错误，不将失败解释为空仓库

### Requirement: Canonical requirement storage

系统 MUST 将正式需求工件统一保存到 docs/requirements，将需求运行状态和临时文件保存到 .ai/requirements，并区分持久计数和可清理临时文件。系统 MUST NOT 回退读写旧 requirements 路径或提供迁移命令。编号扫描 MUST 使用新根目录及其 archive、cancelled 位置。当前项目现存工件直接移动时 MUST 保留 ID、revision、状态、关联和正文，不覆盖目标。路径变化时 MUST 重新复核旧候选，不篡改历史证据。

#### Scenario: New repository layout
- **WHEN** 在新仓库创建需求
- **THEN** 正式工件位于 docs/requirements，编号状态位于 .ai/requirements

#### Scenario: No legacy fallback
- **WHEN** 需求仅存在于旧 requirements 路径
- **THEN** 系统不从该位置加载或原位更新，也不自动移动该需求

#### Scenario: Existing project documents
- **WHEN** 按用户授权直接移动当前项目现存需求且目标没有冲突
- **THEN** 工件内容及身份保持不变，后续由新目录实现读取，不引入迁移命令

### Requirement: Confirmation binds the generated identity

聊天 prepare MUST 不占用序号或创建需求；确认创建后 MUST 使用返回的完整 ID 更新 Session 绑定、后续上下文和确认记录。旧 pending action MUST 维持原精确 ID 语义。

#### Scenario: Confirm automatic creation
- **WHEN** 用户确认已展示的自动编号创建动作
- **THEN** 系统分配完整 ID 并继续使用该 ID 读取需求，不能继续把原 slug 当作需求 ID

#### Scenario: Cancel before confirmation
- **WHEN** 用户没有确认准备中的创建动作
- **THEN** 不生成需求，也不消耗编号
