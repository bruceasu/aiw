## ADDED Requirements

### Requirement: 工作区临时输出目录
AIW 工具在当前工作区内创建可丢弃的临时输出时，系统 SHALL 使用 `.ai/tmp` 并在目录不存在时创建它。该目录 SHALL 保持在版本控制之外。

#### Scenario: 工具需要提交消息临时文件
- **WHEN** `cz` 为 Git 提交准备临时消息文件
- **THEN** 该文件 SHALL 在当前项目根目录的 `.ai/tmp` 中创建，并在提交流程结束时删除

#### Scenario: 工具需要编辑器临时目录
- **WHEN** `cz` 在外部编辑器中收集多行文本
- **THEN** 编辑器文件 SHALL 位于当前项目根目录的 `.ai/tmp` 下的唯一临时目录中，并在编辑流程结束时删除该目录

### Requirement: Skill 临时输出约定
会输出临时文件的内置 Skill SHALL 指示将文件保存到当前工作区 `.ai/tmp`，并使用唯一名称避免覆盖不相关的输出。

#### Scenario: 架构评审生成 HTML 报告
- **WHEN** `improve-codebase-architecture` 生成自包含 HTML 报告
- **THEN** 它 SHALL 将报告写入 `.ai/tmp`，向用户报告绝对路径，并不写入仓库的受版本控制目录

#### Scenario: 无 Session 存储时生成 handoff
- **WHEN** `handoff` 无法使用 AIW Session artifact store
- **THEN** 它 SHALL 将临时 handoff 写入 `.ai/tmp`，而不是操作系统临时目录

### Requirement: 原子写入例外
用于原子替换目标文件的内部临时文件 SHALL 保持在目标文件所在目录；该文件不属于工作区临时输出目录约定。

#### Scenario: 持久化状态原子写入
- **WHEN** AIW 原子写入持久化状态文件
- **THEN** 临时文件 SHALL 与目标文件位于同一目录，以保持原子替换语义
