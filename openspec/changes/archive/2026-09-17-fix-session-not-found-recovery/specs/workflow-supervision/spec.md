## ADDED Requirements

### Requirement: 首次请求准备识别 Session 真实缺失

系统 MUST 识别 Session 存储的明确缺失错误及其包装形式。对于满足现有执行前置条件且 Session ID 等于 Task ID 的新任务，系统 MUST 在创建 Attempt 和 prepared request 前创建缺失 Session 并重新读取。系统 MUST 保留已有工作区绑定。

#### Scenario: 首次启动同名 Session 不存在
- **WHEN** Task 已有合法工作树、可执行 Work Item、同名 Session 绑定且存储确认 Session 不存在
- **THEN** 系统创建并读取 Session，再准备绑定该 Session 的请求，不因错误包装跳过创建。

#### Scenario: 重复准备已有请求
- **WHEN** 合法 Session 和匹配的 prepared request 已存在
- **THEN** 系统沿用现有归属，不创建重复 Session 或 Attempt。

#### Scenario: Session 创建失败
- **WHEN** Session 创建或重新读取失败
- **THEN** 系统报告错误，不据此创建新的 Attempt 或 prepared request，不删除已有工作树。

### Requirement: Session 非缺失错误不得触发自动重建

系统 MUST 区分真正缺失、损坏、不可读、身份冲突与归档记录。系统 MUST NOT 将后四者视为可自动创建的新 Session，MUST 保留记录与原始诊断。异名 Session 绑定缺失 MUST NOT 触发同名 Session 创建。

#### Scenario: Session 目录存在但状态文件损坏或缺少
- **WHEN** Session 已有目录但状态文件缺失或无法解析
- **THEN** 系统返回记录损坏或读取错误，不覆盖目录或创建替代 Session。

#### Scenario: Session 不可读或身份冲突
- **WHEN** 存储返回权限错误、重复记录或身份不匹配
- **THEN** 请求准备失败并保留原记录，不把错误转换为缺失。

#### Scenario: Session 已归档
- **WHEN** 绑定 Session 位于合法归档位置
- **THEN** 系统保持归档只读和不可运行约束，不在活动目录重建同 ID Session。

#### Scenario: 异名绑定缺失
- **WHEN** Task 显式绑定其他 Session ID 且该记录不存在
- **THEN** 系统返回缺失诊断，不猜测绑定或创建同名 Session。

### Requirement: 缺失 Session 的恢复保留 Attempt 审计

既有 Workflow repair MUST 使用相同的 Session 缺失分类，并仅在原恢复资格满足时调用已有缺失 Session Attempt 恢复操作。系统 MUST 保留 Attempt 历史，MUST NOT 为损坏、不可读或归档记录执行缺失恢复。

#### Scenario: 满足原恢复资格的缺失 Session Attempt
- **WHEN** 写租约存在、无 prepared request 且对应 Attempt 的 Session 确认不存在
- **THEN** repair 执行原有恢复转换并保留 Attempt 审计，不因包装错误漏掉恢复。

#### Scenario: 恢复读取到非缺失错误
- **WHEN** 对应 Session 存储返回损坏或读取错误
- **THEN** repair 报告错误，不执行缺失 Session 的 Attempt 转换。
