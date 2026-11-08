# CZ 配置

`aiw cz` 使用独立的 `[cz]` 配置，不读取 `[ai]`、`[ai.profiles.fast]` 或 `AIW_LLM_*`。
配置文件按以下顺序加载，后面的值覆盖前面的值：插件目录的 `cz.toml`、
`AIW_ROOT` 的 `aiw.toml`、当前工作目录的 `aiw.toml`。每处若主文件不存在，
则读取对应的 `.cz.toml` 或 `.aiw.toml`；同一处的两个文件不会合并。

`aiw` 启动时会把与 `aiw` 可执行文件放在同一目录的 `.env` 当作环境变量默认值加载，
已有的进程环境变量优先。它不会自动读取当前项目目录的 `.env`；通过 `aiw cz` 启动的
Python 插件会继承 `aiw` 已加载的环境变量。若直接运行插件脚本，则需先在进程环境中
设置变量。建议把 API key 放在 provider 或其 SDK 已约定的通用环境变量中，而不是写入
TOML；本机使用 `.env` 时，请放在 `aiw` 可执行文件旁，并避免提交该文件。Codex CLI 和
Copilot CLI 使用本机登录状态；模型、地址等非敏感设置优先写在 `[cz.<provider>]`
配置段。仅在需要 cz 独立覆盖时使用 `CZ_*` 环境变量。

例如，使用 OpenAI 或 DeepSeek 时，密钥分别使用已有的 `OPENAI_API_KEY` 或
`DEEPSEEK_API_KEY`。使用 DeepSeek 时，也可用已有的 `DEEPSEEK_MODEL` 和
`DEEPSEEK_BASE_URL` 设置模型与地址：

```dotenv
# OpenAI API（如使用）
# OPENAI_API_KEY=your-api-key
# DeepSeek API（如使用）
DEEPSEEK_API_KEY=your-api-key
DEEPSEEK_MODEL=deepseek-flash
# 可选：仅在使用兼容代理或自定义端点时设置
# DEEPSEEK_BASE_URL=https://api.deepseek.com
```

provider 可在配置文件的 `[cz]` 段中设为 `provider = "deepseek"`，或在调用时使用
`--provider deepseek`。`CZ_<PROVIDER>_*` 是 cz 专用覆盖项，仅当需要让 cz 使用
不同于其他工具的凭据、模型或地址时才需要；对 OpenAI 和 DeepSeek，专用变量优先于
各自已有的通用环境变量。自动选择时也可通过 `CZ_LLM_PROVIDER=deepseek` 指定 provider。

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
# api_key = "..." # 不建议把密钥写入配置；优先使用 OPENAI_API_KEY
# base_url = "https://api.openai.com/v1"

[cz.deepseek]
model = "deepseek-flash"
# api_key = "..." # 不建议把密钥写入配置；优先使用 DEEPSEEK_API_KEY
# base_url = "https://api.deepseek.com"
```

`--llm` 按 Codex CLI → Copilot CLI → OpenAI Responses HTTP 的顺序尝试；若配置了
DeepSeek key，则在 OpenAI 之后尝试 DeepSeek Responses API。各方式使用各自的
`model`、认证与端点配置，前一项失败或没有有效候选时尝试下一项；全部失败后提示
原因并进入手工向导。`--no-llm` 直接进入手工向导。`--provider` 只尝试指定方式；
`--model` 覆盖所选方式的模型。CLI 参数优先于 `CZ_LLM_PROVIDER`、
`CZ_<PROVIDER>_MODEL`、`CZ_<PROVIDER>_API_KEY`、`CZ_<PROVIDER>_BASE_URL`，
这些环境变量优先于配置文件。OpenAI 还可使用 `OPENAI_API_KEY`。
DeepSeek 默认使用 `https://api.deepseek.com` 和 `deepseek-flash`；可通过
`DEEPSEEK_API_KEY`、`DEEPSEEK_BASE_URL`、`DEEPSEEK_MODEL` 设置全局回退，或用
`CZ_DEEPSEEK_*` 覆盖 cz 专用配置。DeepSeek Responses 为无状态请求，响应 ID 不用于续接对话。

Codex CLI 从 `PATH` 查找，也可通过 `CZ_CODEX_COMMAND` 指定命令。
Copilot CLI 仅在设置 `CZ_COPILOT_COMMAND` 后使用；未设置时直接跳过。
两个 CLI 使用本机登录状态，并要求支持当前插件所需的非交互参数。
OpenAI HTTP 需要模型和 API key，可通过 `[cz.openai]`、`CZ_OPENAI_API_KEY`
或 `OPENAI_API_KEY` 配置。
DeepSeek HTTP 可通过 `[cz.deepseek]`、`CZ_DEEPSEEK_API_KEY` 或
`DEEPSEEK_API_KEY` 配置；其 Responses 接口及参数兼容性以
[DeepSeek 官方文档](https://api-docs.deepseek.com/guides/responses_api/)为准。

`cz` 需要目标机器预装 Python，默认只使用标准库。Windows 上运行
`python build.py build cz` 会在 `dist/plugins/<version>/` 生成版本化 ZIP，包含 Python 入口、模块、
`requirements.txt` 和语言文件；`python build.py install cz` 将已发布包安装到 AIW
插件目录，并保留已有的 `cz.toml` / `.cz.toml`。
`aiw cz` 运行 `aiw-cz.py`。CLI 缺失、失败或返回无效候选时，插件按上述顺序
尝试下一种方式；所有方式失败后进入 TUI。安装和交互细节见
[Python runtime](cz-python-runtime.md)。
