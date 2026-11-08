# `aiw cz` 配置

`aiw cz` 默认使用全局 `[ai]` provider 配置和对应的标准环境变量。需要让 CZ
使用不同 provider、模型或地址时，再设置 CZ 专用值。优先级从高到低为：CLI
参数、`CZ_*` 环境变量、`[cz]` / `[cz.<provider>]` 配置、共享 `[ai]` 配置和标准
provider 环境变量，最后是 CZ 原有的自动回退。CZ 不继承 `[ai.profiles.*]`。

`aiw` 启动时会从可执行文件同目录加载 `.env`，已有进程环境优先；它不会自动
加载当前项目目录的 `.env`。直接运行 Python 插件脚本时，需自行将环境变量传入。
建议把密钥放在 `.env` 或进程环境中，不要写入 TOML。

例如，在 `aiw.toml` 中选择 DeepSeek：

```toml
[ai]
provider = "deepseek"
```

并在 `.env` 中设置标准 provider 环境变量：

```dotenv
DEEPSEEK_API_KEY=your-api-key
DEEPSEEK_MODEL=deepseek-flash
# 可选：代理或自定义兼容端点
# DEEPSEEK_BASE_URL=https://api.deepseek.com
```

CZ 和其他支持共享 provider 的命令会使用相同配置。OpenAI 对应使用
`OPENAI_API_KEY`、`OPENAI_MODEL` 和 `OPENAI_BASE_URL`。

如需只覆盖 CZ，可设置 `[cz]` 与 `[cz.<provider>]`：

```toml
[cz]
llm = true
# provider = "deepseek" # 覆盖共享 provider
candidates = 3

[cz.deepseek]
model = "deepseek-flash" # 仅覆盖 CZ 使用的模型
# base_url = "https://api.deepseek.com"
# api_key = "" # 建议仍使用 DEEPSEEK_API_KEY
```

CZ 也支持 `CZ_LLM_PROVIDER` 和 `CZ_<PROVIDER>_MODEL/API_KEY/BASE_URL`。例如，只有
CZ 要使用另一模型时，可设置 `CZ_DEEPSEEK_MODEL`；通常不需要额外定义这组变量。
命令行 `--provider` 与 `--model` 优先级最高。`--provider` 只尝试所选 provider；
未指定 provider 时，如果也没有共享默认，CZ 依次尝试 Codex CLI、Copilot CLI、
OpenAI Responses，并在 DeepSeek 凭据可用时尝试 DeepSeek，失败后进入手动向导。

Git Python 命令 `aic`、`air` 和 `aib` 也使用共享 `[ai]` 默认；显式 CZ 设置可覆盖它们。
`aiw git guide` 支持 Codex CLI、OpenAI Responses 与 DeepSeek Responses。OpenAI 和
DeepSeek 的 Git guide 适配器需要安装 `src/plugins/aiw-git/requirements.txt` 中的
Python SDK 依赖。

Python runtime 与插件安装说明见 [Python runtime](cz-python-runtime.md)。
