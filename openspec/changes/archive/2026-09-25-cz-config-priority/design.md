## Context

`cz` 当前先读取项目、AIW_ROOT 和可执行文件附近的 `cz` 配置，但随后调用共享 AI 配置解析；只要共享 `[ai]` 有 provider，就会覆盖 `cz` 已解析的 provider/model。共享解析器本身也把 `[ai]` 放在 `[cz]` 之前，因此 `[cz]` 实际上只是历史兼容回退。

`cz` 的用途是从 staged diff 提取提交摘要，模型需求通常低于 managed workflow。用户需要在不改变全局 AI 工作流配置的情况下，为 `cz` 选择独立模型。

## Goals / Non-Goals

**Goals:**

- 让 `cz` 的 provider/model 能独立于全局 `[ai]` 配置生效。
- 保持 CLI 显式选项和运行时环境覆盖的能力。
- 对 generic 与 provider-specific model 配置提供确定、可测试的优先级。
- 保持现有没有 `[cz]` 配置时的兼容行为。

**Non-Goals:**

- 不改变 `cz` 的命令名称、交互流程、commit message 格式或提交行为。
- 不修改 managed workflow 的 routing/profile 选择。
- 不重新设计所有 API key、base URL、command 等连接参数的配置体系；只在 provider/model 解析所需范围内保持一致。
- 不引入新的配置文件格式或依赖。

## Decisions

1. **Provider/model 选择优先级固定为：**

   `CLI --provider/--model` > 运行时环境覆盖 > `[cz]` provider/model > `[ai]` provider/model > provider 默认值。

   CLI 选项必须继续覆盖文件和环境解析结果。现有环境变量仍作为显式运行时覆盖，不因本 change 降级到文件配置之后。

2. **`cz` 的 provider 选择独立于全局 provider。** 当 `[cz].provider` 有效时，`cz` 使用它；只有 `[cz]` 未提供 provider 时才回退到 `[ai].provider`。provider-specific model 解析必须以最终选定的 provider 为准。

3. **Model 选择遵循同层优先。** 对最终 provider，`[cz]` 对应的 provider-specific model（如适用）优先于 `[cz].model`，然后回退到 `[ai]` 对应的 provider-specific model、`[ai].model`，最后使用 provider 默认模型。环境变量和 CLI 在此顺序之上覆盖最终结果。

4. **连接配置保持兼容。** provider、model 的选择不能导致 API key、base URL 或 CLI command 被错误地绑定到另一 provider；实现应复用现有 provider-specific fallback 和默认值逻辑，并只调整必要的合并顺序。

5. **设计准备度：FD_NOT_REQUIRED。** 这是局部配置优先级调整，目标、优先级和兼容边界已由用户确认，不涉及新的系统边界、持久化、权限或并发设计。

## Risks / Trade-offs

- [现有项目依赖 `[ai]` 覆盖 `[cz]`] → 只有明确配置 `[cz]` 时才改变行为；没有 `[cz]` 时保持全局回退，并增加回归测试。
- [provider 与 endpoint/credential 来源不一致] → 以最终 provider 重新应用对应的 provider-specific 配置和默认值，测试 provider/model 组合而非只测试单字段。
- [环境变量语义被误解] → 在配置文档和测试中明确 CLI、环境变量、`[cz]`、`[ai]` 的顺序。

## Migration Plan

无需数据迁移。现有仅使用 `[ai]` 的项目行为保持不变；需要独立 `cz` 模型的项目新增 `[cz]` provider/model 即可。回滚只需恢复配置合并逻辑和相关测试/文档。

## Open Questions

无。
## Additional Decision: global fast profile precedence

When no higher-priority `cz` or runtime override supplies a provider/model,
global configuration is resolved in this order:

1. `ai.profiles.fast` provider/model values;
2. top-level `ai.provider` and `ai.model` values;
3. the selected provider's defaults.

The fast profile is treated as an atomic global fallback selection: provider,
model, and any profile-scoped connection settings must be resolved together so
that `cz` does not combine a fast-profile model with unrelated top-level
provider settings. If `ai.profiles.fast` is absent, resolution falls back to
the top-level global values.
