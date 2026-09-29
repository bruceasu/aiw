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

`cz` 需要 Node.js 22.12.0 或更高版本。运行 `build.bat cz` 安装构建依赖、
编译 TypeScript，并在 `plugins/aiw-cz/release/` 生成当前平台的发布样本。
`build.bat plugins` 只将该样本安装到 `plugins/aiw-cz`，保留安装目录已有的
`cz.toml` / `.cz.toml`，不会复制 TypeScript 源码和 `node_modules`。
Linux 构建需在 Linux 主机运行 `npm install` 和 `npm run build:release`，
再将完整的 `release/` 内容安装到 AIW 的 `plugins/aiw-cz/`。

发布样本的 `release-manifest.json` 列出实际文件、来源平台和用途，并记录
开发目录的体积基线。样本包含 Copilot 和 Codex 对应平台的运行资源、Koffi
原生组件、语言文件和打包后的 JavaScript；Node.js 仍由运行环境提供。
`aiw cz` 继续运行 `aiw-cz.js` 插件入口。打包后无法找到必需的 provider
二进制时，该 provider 会报告缺失资源并按原有顺序尝试下一种方式。
