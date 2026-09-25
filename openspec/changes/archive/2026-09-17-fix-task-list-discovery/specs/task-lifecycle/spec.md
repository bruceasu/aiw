## ADDED Requirements

### Requirement: Task 列表按元数据发现身份

`aiw list` MUST 仅将受支持位置中具有 Task 元数据的记录识别为 Task。系统 MUST 忽略没有 task.toml 或兼容 tasks.toml 的普通目录，MUST NOT 通过硬编码内部目录名黑名单实现识别，MUST NOT 为这些目录输出 UNKNOWN 行或推导的 Change 路径。

#### Scenario: Task 与内部运行目录混合

- **WHEN** `.ai` 同时包含有效 Task 和 compile-cache、issue、locks、requirements、sessions、tasks、tmp 等无 Task 元数据的目录
- **THEN** 正常列表只包含有效 Task，普通目录不会被展示为 Task。

#### Scenario: 未来新增运行目录

- **WHEN** 任意未预先列入规则的新目录不包含 Task 元数据
- **THEN** 系统同样忽略它，无需更新名称排除表。

#### Scenario: 没有 Task

- **WHEN** 可读运行根只含内部目录或普通文件
- **THEN** 列表为空且成功，不生成 UNKNOWN 记录。

### Requirement: 列表保留元数据兼容与唯一性

Task 列表 MUST 支持现有规范路径、旧 `.ai/tasks/<id>/` 位置及 tasks.toml 文件名，遵循既有规范位置和文件优先级，按身份去重并稳定排序。系统 MUST NOT 递归发现任意运行子目录，也 MUST NOT 因 OpenSpec 工件缺失而隐藏具有有效元数据的 Task。

#### Scenario: 仅有旧记录

- **WHEN** Task 仅有旧位置或兼容文件名的有效元数据
- **THEN** 系统列出该 Task，而不把旧容器目录自身列为 Task。

#### Scenario: 同 ID 存在两份记录

- **WHEN** 规范与旧位置均存在同 ID Task
- **THEN** 仅按规范路径优先级处理一次，不重复列出，不修改任一记录。

### Requirement: 列表区分无记录与元数据错误

不存在元数据 MUST 被作为非 Task 跳过；元数据不可读、路径类型错误、缺失身份、非法身份或 ID 与候选目录不符 MUST 被报告为错误，包含实际路径及原因，MUST NOT 作为正常 UNKNOWN Task 输出。系统 MUST 继续展示其他有效 Task，并在存在元数据错误时返回非零结果。根枚举失败 MUST 报错。

#### Scenario: 有效 Task 与损坏记录混合

- **WHEN** 一项元数据缺 ID，而另一项有效
- **THEN** 有效 Task 正常输出，错误记录在 stderr 被诊断，最终命令返回错误。

#### Scenario: 读取权限或 I/O 错误

- **WHEN** 候选元数据检查或读取遇到非不存在错误
- **THEN** 系统报告原因，不静默跳过或回退到旧副本。

### Requirement: 正常列表保留 Workflow 摘要

有效 Task 行 MUST 保留现有身份、Workflow 派生状态和 Change 路径的展示格式；Workflow 摘要失败 MUST 保留现有 RUNTIME_ERROR 行语义。列表修复 MUST NOT 启动 Attempt、派发 Agent 或修改任务完成状态。

#### Scenario: 有效 DRAFT 与 DONE Task

- **WHEN** 两项有效 Task 的 Workflow 分别派生 DRAFT 和 DONE
- **THEN** 列表仍分别显示对应状态，不以元数据静态状态替代。

#### Scenario: Workflow 摘要读取失败

- **WHEN** Task 身份有效但 Workflow 摘要无法读取
- **THEN** 保留该 Task 的 RUNTIME_ERROR 行，不误分类为普通目录。
