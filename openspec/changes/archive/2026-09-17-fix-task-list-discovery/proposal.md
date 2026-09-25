# 修正 Task 列表目录识别

## Why

`aiw list` 当前枚举 `.ai/` 的所有子目录，读取 Task 元数据失败后仍将目录以 UNKNOWN 输出，并拼出对应的 OpenSpec 路径。用户因此看到 compile-cache、issue、locks、requirements、sessions、tasks 和 tmp 等内部目录被误报为 Task。

列表应展示有 Task 身份的记录，同时让真实 Task 的元数据错误可见。此问题独立于需求工件生成 Change。

## What Changes

- 根据受支持的 Task 元数据文件发现 Task，忽略无元数据的普通运行目录，不维护目录名黑名单。
- 区分不存在、不可读及无效元数据；不存在则跳过，真实错误显示明确诊断。
- 保留规范路径和旧路径/文件名兼容、同 ID 去重、稳定排序及现有 Workflow 状态来源。
- 增加从 list 命令入口验证最终输出的离线回归用例。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `task-lifecycle`: 明确 Task 列表发现、兼容与诊断行为。

## Impact

主要影响 Task 列表及其元数据发现逻辑，不改变 Task 创建、执行、批准或交付状态机。正常 Task 行维持现有列格式；错误目录不再以 UNKNOWN 正常行展示。

## User Stories

1. 作为用户，我希望缓存、锁和会话目录不出现在 Task 列表中。
2. 作为维护者，我希望未来新增的内部目录也无需加入排除表。
3. 作为用户，我希望真实 Task 的损坏或读取错误被明确报告。
4. 作为旧版本用户，我希望旧位置的 Task 能继续被识别，重复身份只列一次。
5. 作为操作人员，我希望一个错误记录不妨碍查看其他有效 Task。

## Out of Scope

不清理 `.ai` 目录、不迁移历史记录、不改 Requirement 列表、不改生成器、不重新设计元数据解析器，也不改变 Git 或工作树。
