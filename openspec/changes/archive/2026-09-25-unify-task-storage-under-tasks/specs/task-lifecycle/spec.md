## MODIFIED Requirements

### Requirement: Task 活动存储与旧位置诊断

系统 MUST 将活动 Task 元数据和运行信息归于 `.ai/tasks/<id>/`，将计划与清单归于 `openspec/changes/<id>/`。系统 MUST 保留旧元数据文件名 `tasks.toml` 的读取兼容性。旧活动目录 `.ai/<id>/` MUST NOT 被正常读取、写入、自动迁移或用于同 ID 的新建/补建；发现旧目录或迁移标记时 MUST 返回可操作的手工迁移诊断。Task 归档位置 MUST 保持 `.ai/archive/<date>-<id>/`。

#### Scenario: 创建和读取活动 Task

- **WHEN** Task 位于 `.ai/tasks/<id>/`
- **THEN** 元数据、Workflow 状态及运行记录均从该目录解析。

#### Scenario: 发现旧活动目录

- **WHEN** Task 只存在于 `.ai/<id>/` 或新位置含有迁移标记
- **THEN** 正常运行停止并说明旧目录及目标路径，等待操作者在所有 AIW 写入者停止后调整完整目录。
- **AND** 系统不自动搬动、合并、覆盖或重建该 Task。

#### Scenario: 归档位置不变

- **WHEN** Task 被归档
- **THEN** 运行数据仍存放于 `.ai/archive/<date>-<id>/`。
