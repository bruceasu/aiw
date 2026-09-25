## ADDED Requirements

### Requirement: 通用监督 Git 查询指令

通过当前工作区预检的每个 supervised Work Item 请求 MUST 携带限定于该工作区的只读 Git 查询指令，不依赖标题、语言或嵌套工具是否继承环境变量。指令 MUST 使用命令级 safe.directory 和相同目录的 -C，并保持 shell 字面路径正确。

#### Scenario: 普通实现项

- **WHEN** 中文标题的普通实现项通过预检并准备派发
- **THEN** 最终 provider 请求包含限定目录 Git 查询指令
- **AND** 不要求标题包含 no unrelated changes，也不把实现范围缩减为只修改检查框

#### Scenario: 范围审查项

- **WHEN** 现有规则识别出 scope-review 项
- **THEN** 请求包含通用查询指令及独立的审查编辑限制
- **AND** 原有只能修改所选检查框的限制仍然生效

#### Scenario: 路径包含特殊字符

- **WHEN** 规范工作树路径包含空格、单引号或美元符号
- **THEN** 对应 shell 中的命令参数保持完整字面路径，safe.directory 与 -C 指向同一已验证目录

### Requirement: 信任依据及失败边界

监督查询指令 MUST 仅来自当前请求对应的成功工作区预检。预检失败、信任目录缺失或不匹配时 MUST 明确停止派发。系统 MUST NOT 修改持久 Git 配置、信任任意目录或因此扩大 Agent 写入权限。

#### Scenario: 缺少当前预检依据

- **WHEN** 当前请求没有有效且匹配的信任目录
- **THEN** 在 provider 调用前返回明确诊断，并沿用准备失败的状态清理规则
- **AND** 不以空提示、全局 Git 配置或通配信任继续执行

#### Scenario: 旧 handoff 引用其他工作树

- **WHEN** 旧 handoff 中存在不同路径或此前的 Git 阻塞说明
- **THEN** 新请求仅使用当前预检核验的目录，明确限定新的查询授权
- **AND** 不改变其他执行、文件编辑或 Git 写操作限制

#### Scenario: 限定查询仍失败

- **WHEN** Agent 使用限定目录查询仍遇到访问错误
- **THEN** 保留具体错误并报告阻塞，不扩大目录信任或自动循环重试
- **AND** 保持现有 Core 与 Agent outcome 分类职责

### Requirement: 历史阻塞项显式恢复

修复生效 MUST NOT 自动关闭已有 Gate、重开 Work Item 或重新启动 supervise。恢复文档 MUST 说明确认修复版本与实际查询结果后，先解决 Gate，再 reopen，最后显式启动监督执行，并保留原失败证据。

#### Scenario: 存在旧的访问阻塞 Gate

- **WHEN** 修复交付但历史 Work Item 仍为 blocked
- **THEN** 该项保持阻塞直到显式恢复，不因代码更新或清单同步自动完成
