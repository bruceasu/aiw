## Why

`cz` 主要生成和整理 Conventional Commit 摘要，通常不需要与通用 AI 工作流相同的高能力模型。当前实现会在全局 `[ai]` 配置存在时覆盖 `[cz]` 的 provider/model，使用户无法为 `cz` 单独选择更快或更低成本的模型。

## What Changes

- 调整 `cz` 的 provider/model 配置解析优先级，使 `[cz]` 优先于全局 `[ai]` 回退配置。
- 保留 CLI `--provider` 与 `--model` 的显式覆盖能力。
- 保留现有环境变量作为运行时覆盖机制，并明确其与 CLI、文件配置的优先级。
- 保留 provider 默认模型作为最终回退。
- 增加配置优先级测试与文档说明，覆盖 generic 和 provider-specific model 配置。
- 不改变 `cz` 的提交摘要生成、交互确认、提交或 provider 调用协议。

## Capabilities

### New Capabilities

- `cz-configuration-priority`: 为 `cz` 定义独立且可预测的 provider/model 配置优先级。

### Modified Capabilities

无。现有 CLI capability 的运行入口不变；本 change 增加并明确 `cz` 的配置解析契约。

## Impact

影响 `cz` 配置加载和共享 AI 配置解析之间的合并逻辑，以及相关单元测试和配置文档。无需新增依赖、迁移数据或改变命令参数。
## Clarification: global fast profile fallback

When `cz` falls back to global AI configuration, it MUST first use the
`ai.profiles.fast` profile when that profile is configured. Only when the fast
profile is not configured should it use the top-level `ai.provider` and
`ai.model` values. This keeps `cz` lightweight by default while preserving the
existing provider/model fallback.
