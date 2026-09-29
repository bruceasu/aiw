# CZ 配置

`aiw cz` 使用独立的 `[cz]` 配置，不读取 `[ai]`、`[ai.profiles.fast]` 或 `AIW_LLM_*`。
配置文件按以下顺序加载，后面的值覆盖前面的值：插件目录的 `cz.toml`、
`AIW_ROOT` 的 `aiw.toml`、当前 Git 项目根目录的 `aiw.toml`。

```toml
[cz]
llm = false
candidates = 3
# provider = "copilot" # 可选；指定后只尝试这一项

[cz.copilot]
model = "your-copilot-model"

[cz.codex]
model = "your-codex-model"

[cz.openai]
model = "your-openai-model"
# api_key = "..." # 推荐使用 CZ_OPENAI_API_KEY 或 OPENAI_API_KEY
# base_url = "https://api.openai.com/v1"
```

`--llm` 按 Copilot SDK → Codex SDK → OpenAI SDK 的顺序尝试。各方式使用各自的
`model`、认证与端点配置，前一项失败或没有有效候选时尝试下一项；全部失败后提示
原因并进入手工向导。`--no-llm` 直接进入手工向导。`--provider` 只尝试指定方式；
`--model` 覆盖所选方式的模型。CLI 参数优先于 `CZ_LLM_PROVIDER`、
`CZ_<PROVIDER>_MODEL`、`CZ_<PROVIDER>_API_KEY`、`CZ_<PROVIDER>_BASE_URL`，
这些环境变量优先于配置文件。OpenAI 还可使用 `OPENAI_API_KEY`。

Copilot SDK 可使用已登录的 GitHub 凭据，也可在 `[cz.copilot].api_key` 中设置
GitHub 用户令牌。Codex SDK 可使用已登录的 Codex 凭据，也可在
`[cz.codex].api_key` 中设置 API key。OpenAI SDK 需要模型和 API key。

`cz` 需要 Node.js 22.12.0 或更高版本。运行 `build.bat cz` 安装插件依赖并把
TypeScript 编译为 JavaScript。`aiw cz` 直接运行 `aiw-cz.js` 插件入口。
