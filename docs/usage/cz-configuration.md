# CZ 配置

`aiw cz` 使用独立的 `[cz]` 配置，不读取 `[ai]`、`[ai.profiles.fast]` 或 `AIW_LLM_*`。
配置文件按以下顺序加载，后面的值覆盖前面的值：插件目录的 `cz.toml`、
`AIW_ROOT` 的 `aiw.toml`、当前工作目录的 `aiw.toml`。每处若主文件不存在，
则读取对应的 `.cz.toml` 或 `.aiw.toml`；同一处的两个文件不会合并。

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

`--llm` 按 Codex CLI → Copilot CLI → OpenAI Responses HTTP 的顺序尝试。各方式使用各自的
`model`、认证与端点配置，前一项失败或没有有效候选时尝试下一项；全部失败后提示
原因并进入手工向导。`--no-llm` 直接进入手工向导。`--provider` 只尝试指定方式；
`--model` 覆盖所选方式的模型。CLI 参数优先于 `CZ_LLM_PROVIDER`、
`CZ_<PROVIDER>_MODEL`、`CZ_<PROVIDER>_API_KEY`、`CZ_<PROVIDER>_BASE_URL`，
这些环境变量优先于配置文件。OpenAI 还可使用 `OPENAI_API_KEY`。

Codex CLI 从 `PATH` 查找，也可通过 `CZ_CODEX_COMMAND` 指定命令。
Copilot CLI 仅在设置 `CZ_COPILOT_COMMAND` 后使用；未设置时直接跳过。
两个 CLI 使用本机登录状态，并要求支持当前插件所需的非交互参数。
OpenAI HTTP 需要模型和 API key，可通过 `[cz.openai]`、`CZ_OPENAI_API_KEY`
或 `OPENAI_API_KEY` 配置。

`cz` 需要目标机器预装 Python，默认只使用标准库。Windows 上运行
`python build.py cz` 会在 `dist/plugins/aiw-cz/release/` 准备 Python 入口、模块、
`requirements.txt` 和语言文件；`python build.py plugins` 将这份内容安装到 AIW
插件目录，并保留已有的 `cz.toml` / `.cz.toml`。
`aiw cz` 运行 `aiw-cz.py`。CLI 缺失、失败或返回无效候选时，插件按上述顺序
尝试下一种方式；所有方式失败后进入 TUI。安装和交互细节见
[Python runtime](cz-python-runtime.md)。
