## Why

活动 Task 直接存放在 `.ai/<task-id>/`，与 sessions、requirements、locks、缓存等内部目录混杂，增加浏览、发现和维护成本。统一活动 Task 路径后，当前仓库中唯一仍活动的 Task 记录需要在开发完成后调整到新位置。

## What Changes

- **BREAKING**：活动 Task 唯一正式目录改为 `.ai/tasks/<task-id>/`，不再将 `.ai/<task-id>/` 作为正常读写回退路径。
- 统一 Task 元数据、Workflow Core、发现、提示词/工件和归档入口的路径解析；旧布局未调整时明确诊断，不静默创建另一份记录。
- 开发完成、Supervisor 停止后，由操作者仅调整当前活动 Task 的完整运行目录；归档 Task 保持在 `.ai/archive/<date>-<id>/`。
- 不新增通用 storage migrate 命令、批量迁移、迁移日志或 rollback 功能。
- 用户确认不存在其他活动 Task；本 change 的单次目录调整不包括已归档 Task。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `task-lifecycle`: 活动 Task 的唯一存储位置、旧布局诊断和既有归档位置。

## Impact

涉及 Task/Workflow 存储及依赖 Task 路径的创建、列表、运行、修复和归档调用方，包含脚本、文档与测试夹具。不新增依赖或迁移 CLI，不改变 Task/Session 的所有权关系。历史路径引用和事件正文保持原样。

## User Stories

1. 作为使用者，我希望所有活动 Task 集中在 tasks 目录，从而快速定位任务记录。
2. 作为维护者，我希望 Task 和 Session 分别存放，通过身份引用关联，从而不要求先有会话才能有任务。
3. 作为操作者，我希望开发完成后能将当前活动 Task 的完整记录调整到新目录，并保持状态、事件和证据不变。
4. 作为操作者，我希望已归档 Task 和归档目录保持原样。

## Out of Scope

通用或批量迁移 CLI、自动迁移、迁移 journal、rollback、历史工件路径重写、Session/OpenSpec/归档根搬迁、自动交付、删除工作树、启动 supervisor 或重新激活已归档任务。
