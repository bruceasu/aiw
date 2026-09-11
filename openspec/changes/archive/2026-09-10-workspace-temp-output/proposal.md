## Why

AIW 工具和仓库内 Skill 在生成可丢弃的临时输出时会使用操作系统临时目录，导致同一开发工作产生的文件分散在项目外且不便发现或清理。将此类输出统一放在工作区的 `.ai/tmp`，可降低对其他目录的污染并保持项目级可追溯性。

## What Changes

- 为 AIW 的可丢弃临时输出定义工作区级目录 `.ai/tmp`，并在需要时自动创建。
- 将 `cz` 的提交消息和外部编辑器草稿迁移到该目录，并在流程结束时照常清理。
- 更新会生成临时文件的内置 Skill 指引，使其使用 `.ai/tmp`，而不是操作系统临时目录。
- 保持用于原子替换的目标文件同目录临时文件不变。

## Capabilities

### New Capabilities

- `workspace-temporary-output`: 将 AIW 可丢弃临时输出限定到工作区 `.ai/tmp`，并保持其清理语义。

### Modified Capabilities

无。

## Impact

- `internal/commands/cz` 的临时文件创建位置。
- `skills/handoff` 与 `skills/improve-codebase-architecture` 的输出路径指引。
- 不新增依赖、不改变公开命令参数，也不迁移持久化状态。
