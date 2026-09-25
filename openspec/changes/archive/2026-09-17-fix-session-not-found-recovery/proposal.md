## Why

新 Task 首次启动 supervisor 时，即使工作树创建成功，也可能以 `load prepared request Session: session record not found` 退出。Session 存储返回的包装错误未被启动端旧式缺失判断识别，导致本应执行的 Session 自动创建被跳过。

## What Changes

- 启动和恢复流程统一识别 Session 明确缺失语义，兼容错误包装。
- 仅对允许自动创建的同名 Task Session 执行创建；损坏、权限错误、身份冲突和归档记录不得被当成缺失。
- 保留现有工作区和 Attempt 审计记录，允许修复后继续首次启动，避免重复创建请求或 Session。
- 在现有 Workflow 命令边界补充启动与恢复回归测试；本轮仅规划，不执行测试。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `workflow-supervision`: 明确首次请求准备的 Session 缺失处理与既有恢复入口的分类行为。

## Impact

影响监督启动、准备请求、缺失 Session Attempt 恢复及邻近测试。不改变 CLI、Session 存储格式、归档只读契约或 Git 授权规则，无新依赖。与 fix-task-metadata-lifecycle-sync 独立，先修复本启动问题可解除其执行阻塞。

## User Stories

1. 作为 Task 使用者，我希望首次启动能自动创建合法的缺失 Session，从而继续执行已有清单。
2. 作为 Task 使用者，我希望重试复用已建立的工作树，从而不必删除并重建任务。
3. 作为维护者，我希望损坏、不可读或冲突的 Session 明确报错，从而不丢失历史记录。
4. 作为维护者，我希望归档 Session 保持只读，从而不因恢复操作复活旧会话。
5. 作为操作者，我希望恢复缺失 Session 的 Attempt 时保留审计历史，从而可以解释原失败。

## Out of Scope

不修复任务元数据投影、不自动合并/清理/归档、不新增 Session 创建 CLI、不自动重启受阻 supervisor，不补造历史结果或消除其他 Gate。
