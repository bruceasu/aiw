# `cz` AI 配置

`cz` 会按以下顺序选择 provider 和 model，优先级从高到低：

1. 命令行参数 `--provider` / `--model`。
2. 运行时环境变量：`AIW_LLM_PROVIDER`、`AIW_LLM_MODEL`，以及所选 provider 对应的 model 环境变量（例如 `OPENAI_MODEL`、`GEMINI_MODEL`）。
3. 项目配置中的 `[cz]`：provider-specific model 优先于通用 `model`；缺少 `[cz].provider` 时使用 `[ai].provider`。`[cz]` 中明确设置的连接和命令值继续用于 `cz`。
4. `[ai.profiles.fast]`：当 `[cz]` 和 `AIW_LLM_PROVIDER` 没有指定 provider/model 时，完整的 fast profile 优先于顶层 `[ai]`。
5. 顶层 `[ai]` provider/model，以及所选 provider 的默认值。

选择 provider 后，只使用该 provider 对应的 model、endpoint、credential 和 command 设置；fast profile 的连接设置与其 provider/model 一起解析。没有配置完整的 fast profile 时，`cz` 回退到顶层 `[ai]`，再使用 provider 默认值。

要让 `cz` 默认使用轻量模型，可配置 `ai.profiles.fast`：

```toml
[ai]
provider = "openai"
model = "your-general-model"

[ai.profiles.fast]
provider = "openai"
model = "your-lightweight-model"
```

上述配置会让没有更高优先级覆盖的 `cz` 使用 `your-lightweight-model`。也可以只为 `cz` 指定 provider 或 model：

```toml
[cz]
provider = "openai"
model = "your-lightweight-model"
```

`cz` 的 CLI 参数仍可临时覆盖这些文件设置，例如 `cz --provider openai --model your-model`。
