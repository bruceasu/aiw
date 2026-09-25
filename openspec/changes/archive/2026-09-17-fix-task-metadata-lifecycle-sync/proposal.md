## Why

已完成并交付的 Task 在 worktree 清理后，task.toml 仍保留 TODO、pending 和失效的 isolated 路径。archive 使用 Core 判断完成，却使用旧绑定同步清单，导致读取已删除工作树失败。

## What Changes

- 将 task.toml 的 status、delivery 定义为 Core 权威状态的派生快照，在生命周期写操作后同步，禁止反向覆盖已有 Core。
- 成功清理工作树后解除当前绑定；保留历史分支、父分支及交付证据，不自动绑定主工作区。
- 归档已交付且已清理的 Task 时，从核验后的主工作区工件读取清单，兼容重复操作和部分清理失败。
- 对历史已完成 Task 提供幂等元数据修复，包含活动和归档位置；跳过状态、身份或清理证据不充分的记录。
- 本次按用户授权先修复现有可确认记录，并保留修复前快照及依据。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `task-lifecycle`: 明确派生元数据、清理后绑定、历史修复及交付后归档契约。

## Impact

涉及 internal/commands/task 的完成、交付、清理、修复及归档流程，internal/taskx 的元数据与工件定位，相关测试和工作流文档。保留 Workflow Core 权威性、现有字段兼容性和 Git 操作授权边界；不新增依赖。本 change 仅规划生产代码修复，本轮执行现有记录的数据修正。
