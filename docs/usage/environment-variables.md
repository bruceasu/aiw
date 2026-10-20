# 环境变量运维参考

核查日期：2026-10-19。本文以当前仓库源码为准，覆盖主 CLI、`src/internal`、`src/cmd`、`src/plugins`、`src/programs`、根 `build.py` 与 `scripts`。所有默认值均为程序实际回退；配置示例中的值不等于默认值。未读取实际 `.env`、用户配置或运行环境中的密钥。

第三方依赖、压缩前端资产及 vendored `git-filter-repo` 的全部环境接口不在本文范围内；测试夹具中临时定义的变量也不作为部署配置。下文单独列出仓库脚本明确注入的测试环境。外部 CLI、语言运行时和 SDK 自身支持的其他变量，请按其版本单独管理。

## 1. 配置加载与空值

本项目存在多条独立配置路径，不能用一个统一优先级解释所有命令。

| 入口 | 文件加载顺序与实际优先级 | 源码 |
| --- | --- | --- |
| 主 `aiw` 启动 | 只把可执行文件同目录 `.env` 作为缺省环境导入；已有进程变量优先，包括已设置的空字符串。不自动加载当前目录 `.env`。文件不存在则跳过。 | [入口](../../src/cmd/aiw/main.go#L32)、[加载器](../../src/internal/util/env/startup.go#L10) |
| Go 全局 AI 配置 | TOML 按“向上查找到的项目根 → `AIW_ROOT` → 可执行文件目录”依次合并；每处 `aiw.toml` 后合并 `.aiw.toml`，后文件覆盖同键。配置解析后应用 provider 环境，再应用 `AIW_LLM_*`。Session 值与本次调用显式覆盖还能覆盖 provider/model。 | [文件顺序](../../src/internal/ai/config.go#L214)、[环境覆盖](../../src/internal/ai/config.go#L301)、[Session 覆盖](../../src/internal/ai/provider.go#L168) |
| Go `RunLLM` 路径 | 非空调用配置 > 非空进程环境 > 当前工作目录 `.env` > `AIW_ROOT/.env` > 可执行文件目录 `.env` > 内置默认。独立解析三个文件，不把它们全部写入系统环境。经主 CLI 启动时，可执行目录 `.env` 已导入的值属于进程环境，因而可先于当前目录文件生效。 | [解析](../../src/internal/ai/llm.go#L265)、[文件读取](../../src/internal/ai/llm.go#L287) |
| Python `aiw cz` | 插件目录首个 `cz.toml`/`.cz.toml` → `AIW_ROOT` 首个 `aiw.toml`/`.aiw.toml` → 当前目录首个同名文件，后者覆盖前者；再应用 `CZ_*` 环境。它不自行加载 `.env`。 | [加载](../../src/plugins/aiw-cz/cz_config.py#L172) |
| 根 `build.py` | 读取仓库根 `.env`，使用 `setdefault`；已有进程变量（包括空字符串）优先。不执行文件内容，不做变量插值。 | [加载](../../build.py#L47) |
| 独立旧 Teams 脚本 | 脚本目录 `.env` → `send_message.env` → `--env` 文件；每个文件直接覆盖进程环境，随后通常为 CLI > 环境 > 默认。该规则与主 CLI 相反；managed 模式采用单独协议。 | [加载器](../../src/plugins/send_teams_msg.py#L63)、[普通模式](../../src/plugins/send_teams_msg.py#L245) |

主 CLI 的 Go `.env` 加载器支持 `export`、引号、`${NAME}` 与 `${NAME:-fallback}`；单引号保持字面值，转义可阻止插值。`${NAME:-fallback}` 在值为空时使用 fallback；循环引用报错。旧 Teams 加载器另用 `expandvars/expanduser`，规则不同。[Go 解析器](../../src/internal/util/env/loader.go#L75)、[插值](../../src/internal/util/env/loader.go#L153)、[Teams 解析器](../../src/plugins/send_teams_msg.py#L63)

“未设置”和“设置为空”不总相同：Go 全局 AI 环境覆盖会 trim 并忽略空白；JavaScript `||` 会把空字符串当作缺省；Python `os.environ.get(key, default)` 只在变量不存在时取默认，空字符串通常保留。各表中的特殊情况优先。

## 2. 安装、运行时与工作流

| 变量 | 含义 | 未设置时的默认或回退、优先级 | 源码 |
| --- | --- | --- | --- |
| `AIW_PYTHON` | 插件 Python 解释器 | 非空环境 > 用户配置 `[runtime].python` > 程序目录配置 > 程序目录捆绑 Python > PATH 中 `python`、`python3`。显式路径须为现存绝对文件路径，非法时直接报错。 | [配置](../../src/internal/plugin/runtime_config.go#L20)、[解析](../../src/internal/plugin/exec.go#L222) |
| `AIW_NODE` | 插件 Node 解释器；也被根打包工具读取 | 插件：非空环境 > 用户配置 `[runtime].node` > 程序配置 > `<exe>/node/node[.exe]` > `node`；显式配置须为绝对文件路径。根 `build.py`：环境 > PATH 中 `node`，不套用插件配置规则。 | [插件](../../src/internal/plugin/runtime_config.go#L61)、[打包工具](../../build.py#L79) |
| `AIW_INSTALL_DIR` | 包安装目标目录 | `--dest` 等显式目标优先；缺省为环境值（支持 `~`），再回退 Windows `C:\green\aiw`、其他系统 `~/apps/aiw`。安装 hook 内会被覆盖为实际目标。 | [缺省目标](../../src/plugins/aiw-pkg/manager.py#L20)、[hook](../../src/plugins/aiw-pkg/manager.py#L283) |
| `AIW_SKILLS_SOURCE_ROOT` | Skill 源目录 | `pkg`：非空环境 > 源码/安装树 `skills`；assist：环境值若非空则只使用该根，否则按 `<workspace>/src/skills` → `<exe>/skills` → `<workspace>/.agents/skills` 查找。 | [pkg](../../src/plugins/aiw-pkg/skill_management.py#L55)、[assist](../../src/internal/commands/assist/context.go#L112) |
| `AIW_REQ_BINARY` | Python req 包装器的实际二进制 | 插件同目录 `aiw-req.exe`（Windows）或 `aiw-req`（Linux）；非空环境覆盖，文件缺失则失败。 | [包装器](../../src/plugins/aiw-req/aiw-req.py#L20) |
| `AIW_PACKER_BINARY` | packer 包装器的实际二进制 | 插件同目录 `aiw-packer[.exe]`；非空环境覆盖。 | [包装器](../../src/plugins/aiw-packer/aiw-packer.py#L12) |
| `AIW_FD_ROLE_RUNNER` | FD 角色派发程序文件 | 空/未设置则保留 pending handoff，打印人工 claim 提示，不自动启动角色。非空按路径解析，必须是文件；以 role、FD 路径、事件路径为参数运行。 | [派发](../../src/plugins/aiw-fd.py#L1106) |
| `AIW_ISSUE_SESSION` | Issue 对话动作准备所绑定的活动 Session | 无默认；prepare 缺少有效 ID 则报错。对话执行时临时设为当前 Session，完成后恢复原值。 | [使用](../../src/cmd/aiw-req/issue-chat.go#L36)、[临时设置](../../src/cmd/aiw-req/issue-chat.go#L112) |
| `AIW_REQUIREMENT_SESSION` | 兼容旧 Requirement 的新 Issue 对话绑定 | 无默认；非空时创建后绑定该 Session，未设置则不做此绑定。 | [创建](../../src/cmd/aiw-req/issue-create.go#L19) |
| `AIW_SHELL` | setup-project 补全目标 Shell | 支持 `powershell/bash/zsh/fish`；不匹配或缺省时看 `SHELL`，再在 Windows 检查 `PSModulePath`；无法识别则无自动目标。 | [检测](../../src/plugins/aiw-setup-project.py#L86) |
| `CODEX_HOME` | cxs 读取的 Codex 数据目录 | 未设置为 `~/.codex`；空字符串被 `Path` 解释为当前目录。gateway 也允许向 Codex 子进程透传此变量。 | [cxs](../../src/plugins/aiw-cxs.py#L42)、[gateway](../../src/programs/agent-gateway/config.go#L159) |
| `AIW_GIT_GUIDE_MODEL` | git guide 使用的 Codex 模型 | 未设置为 `gpt-6-luna`；空字符串保留。 | [模型](../../src/plugins/aiw-git/git-guide.py#L44) |
| `GITHUB_TOKEN` | GitHub 插件访问令牌 | 无内置默认；未设置返回空凭据并由调用路径处理。 | [读取](../../src/plugins/aiw-github/aiw-github.py#L48) |

## 3. Go AI 配置

下列为 Go 全局配置及 `RunLLM` 使用的 provider 环境；密钥没有通用内置默认。全局配置忽略空白环境覆盖。`RunLLM` 的解析优先级见第 1 节；它自动探测时依次尝试可用 Codex、Copilot、已配置 OpenAI、已配置 Gemini，最后 Ollama。[环境映射与默认](../../src/internal/ai/config.go#L373)、[旧路径默认](../../src/internal/ai/llm.go#L133)、[自动探测](../../src/internal/ai/llm.go#L190)

| 变量 | 含义 | 未设置且没有更高优先配置时的默认 | 优先级/限制 |
| --- | --- | --- | --- |
| `AIW_LLM_PROVIDER` | Go 全局 provider | TOML 值，否则空，由调用路径决定选择/探测 | 高于 TOML；Session/本次显式 provider 可继续覆盖 |
| `AIW_LLM_MODEL` | Go 全局模型 | TOML/provider 环境值，否则对应 provider 默认 | 高于 provider 环境；Session/本次显式模型可继续覆盖 |
| `AIW_LLM_BASE_URL` | Go 全局 API 地址 | TOML/provider 环境值，否则对应地址默认 | 高于 provider 环境 |
| `AIW_LLM_API_KEY` | Go 全局 API 密钥 | TOML/provider 环境值，否则空 | 高于 provider 环境，无内置通用密钥 |
| `OPENAI_MODEL` | OpenAI 模型 | `gpt-4o-mini` | provider 环境高于 TOML；旧 RunLLM 则低于调用配置 |
| `OPENAI_BASE_URL` | OpenAI API 根地址 | `https://api.openai.com/v1` | 同上；say 与独立 proxy 另见后文 |
| `OPENAI_API_KEY` | OpenAI 凭据 | 空 | 同上；其他入口也会读取，不能假定通用于所有 AI 命令 |
| `GEMINI_MODEL` | Gemini 模型 | `gemini-1.5-flash` | 同上 |
| `GEMINI_BASE_URL` | Gemini API 根地址 | `https://generativelanguage.googleapis.com` | 同上 |
| `GEMINI_API_KEY` | Gemini 凭据 | 空 | 同上 |
| `OLLAMA_MODEL` | Ollama 模型 | `llama3` | 同上 |
| `OLLAMA_BASE_URL` | Ollama 地址 | `http://localhost:11434` | 同上 |
| `OLLAMA_API_KEY` | Ollama 兼容接口凭据 | 空 | 全局配置和 `ConfigFromEnv` 读取；旧 RunLLM 不按此变量解析密钥 |
| `LLAMACPP_MODEL` | llama.cpp 模型 | `local-model` | 同上 |
| `LLAMACPP_BASE_URL` | llama.cpp 地址 | `http://localhost:8080/v1` | 同上 |
| `LLAMACPP_API_KEY` | llama.cpp 兼容接口凭据 | 空 | 全局配置和 `ConfigFromEnv` 读取；旧 RunLLM 不按此变量解析密钥 |
| `AIW_CZ_DEBUG` | 旧 Go LLM 配置来源调试 | 关闭 | `1/true/yes/on`（忽略大小写、首尾空白）开启；调用配置 `DebugSource` 也可开启；日志对密钥做掩码。[实现](../../src/internal/ai/llm.go#L323) |

全局解析有一个顺序细节：先按 TOML 中选定的 provider 读取该 provider 环境，再应用 `AIW_LLM_PROVIDER`。只切换 `AIW_LLM_PROVIDER` 不会重新读取新 provider 的专属环境组；跨 provider 切换时应同时明确所需模型、地址、密钥，或通过对应配置入口选择 provider。[实现](../../src/internal/ai/config.go#L301)

`ConfigFromEnv(name, model)` 是更窄的辅助函数，只读取对应 provider 的 key/base，不读取上述模型环境或 `AIW_LLM_*`；模型由参数提供。旧 `RunLLM` 对 `llamacpp` 别名与 `llama.cpp` 的环境映射也并非完全对称，建议使用规范名 `llama.cpp`。[辅助函数](../../src/internal/ai/provider.go#L207)、[旧映射](../../src/internal/ai/llm.go#L133)

`aiw say` 单独读取 `OPENAI_API_KEY` 和 `OPENAI_BASE_URL`，不读取 `AIW_LLM_*`。默认完整地址为 `https://api.openai.com/v1/chat/completions`；自定义 base 无路径则补 `/v1`，再补 `/chat/completions`。只接受 HTTPS 或 loopback HTTP，不合法的地址使配置无效。[实现](../../src/internal/say/openai.go#L19)

## 4. Python cz 的独立环境

这一组不等同于 Go 的全局 AI 配置。provider 集合为 `codex`、`copilot`、`openai`；`CZ_<PROVIDER>_*` 是按这个集合动态拼接的真实变量名。[配置加载](../../src/plugins/aiw-cz/cz_config.py#L172)、[CLI 解析](../../src/plugins/aiw-cz/cz_providers.py#L27)、[OpenAI](../../src/plugins/aiw-cz/cz_openai.py#L46)

| 变量 | 含义 | 未设置的默认或回退 | 优先级/空值 |
| --- | --- | --- | --- |
| `CZ_LLM_PROVIDER` | cz provider | 合并后的 `[cz].provider/llm_provider`，否则空 | 环境覆盖配置；空字符串也覆盖 |
| `CZ_CODEX_MODEL`、`CZ_COPILOT_MODEL`、`CZ_OPENAI_MODEL` | provider 模型 | 对应 TOML `[cz.<provider>].model`，否则空 | 环境覆盖配置，包括空字符串；OpenAI 最终无模型则报错 |
| `CZ_CODEX_API_KEY`、`CZ_COPILOT_API_KEY`、`CZ_OPENAI_API_KEY` | 对应配置槽的凭据 | 对应 TOML 值，否则空 | 环境覆盖配置；Codex/Copilot CLI 路径不把这些配置槽直接转换成 CLI 参数 |
| `CZ_CODEX_BASE_URL`、`CZ_COPILOT_BASE_URL`、`CZ_OPENAI_BASE_URL` | 对应配置槽的地址 | 对应 TOML 值，否则空；OpenAI 最终回退 `https://api.openai.com/v1` | 环境覆盖配置；CLI 路径不直接使用其 base_url 配置槽 |
| `OPENAI_API_KEY` | Python cz OpenAI 的后备密钥 | 空，缺少密钥则不可用 | 已解析 `CZ_OPENAI_API_KEY`/TOML 非空值优先，再取此变量；Python cz 不用 `OPENAI_MODEL` 和 `OPENAI_BASE_URL` 作同类后备 |
| `CZ_CODEX_COMMAND` | Codex 命令及参数 | PATH 中 `codex` | 非空环境优先，按 shell 单词规则拆分 |
| `CZ_COPILOT_COMMAND` | Copilot 命令及参数 | 无；必须显式配置 | 未配置不自动使用 PATH 中 copilot |
| `CZ_CODEX_ARGS`、`CZ_COPILOT_ARGS` | 旧参数覆盖接口 | 未设置/空白则忽略 | **非空值直接拒绝**，不能覆盖只读 CLI 契约 |
| `CZ_LANGUAGE` | 交互语言 | 配置的 `[i18n].default_language` → 系统语言 → `en` | 显式请求语言 > 非空 `CZ_LANGUAGE` > 文件 > 系统；只支持已有的 `en/zh/ja`，其他回退 `en` |

Python cz 探测/执行 CLI 时自动注入 `CI=1`、`GIT_TERMINAL_PROMPT=0`，实际生成另加 `AIW_CZ_READ_ONLY=1`；它们是子进程标记，不是关闭安全约束的用户开关。[实现](../../src/plugins/aiw-cz/cz_providers.py#L44)

## 5. `aiw ai`、HTTP OpenAI Proxy 与 Gateway

`aiw ai` 是访问 Agent Gateway 的 Node 客户端，与 Go 全局 AI 配置分离。[配置](../../src/plugins/aiw-ai/aiw-ai.mjs#L201)

| 变量 | 含义 | 未设置的默认或回退 | 优先级/限制 |
| --- | --- | --- | --- |
| `AIW_AI_MODEL` | 网关逻辑模型名 | `gpt-6-luna` | 非空环境覆盖默认；需在服务端映射并授权 |
| `AIW_AI_API_KEY` | 客户端网关 Key | 无，必填 | 至少 43 字符且不能含空白；不回退到 `OPENAI_API_KEY` |
| `AIW_AI_PROVIDER` | 已移除的旧 provider 接口 | 无 | **设置为非空即报错**，应使用网关逻辑模型 |
| `AIW_AGENT_PROXY_PORT` | 客户端默认服务端口 | `43127` | 本次显式 URL 优先；默认目标为 `http://127.0.0.1:<port>/v1/responses`；端口须为 1–65535 整数 |
| `EDITOR` | `aiw ai` 编辑提示词所用程序 | 无 | 启用编辑流程时要求非空；不回退 `GIT_EDITOR`/`VISUAL`。[实现](../../src/plugins/aiw-ai/editor.mjs#L6) |

独立 `src/programs/http-openai-proxy` 使用以下变量。JavaScript `||` 意味着空字符串会继续回退。[监听配置](../../src/programs/http-openai-proxy/src/config.ts#L5)、[provider 配置](../../src/programs/http-openai-proxy/src/providers.ts#L23)

| 变量 | 含义 | 未设置的默认或回退 | 优先级/限制 |
| --- | --- | --- | --- |
| `HTTP_OPENAI_PROXY_HOST` | HTTP 监听 IPv4 | `AIW_AGENT_PROXY_HOST` → `127.0.0.1` | 规范名优先；必须 IPv4 |
| `AIW_AGENT_PROXY_HOST` | 上述监听地址的兼容变量 | `127.0.0.1` | 仅 proxy 读取；`aiw ai` 默认客户端固定 loopback，不读取此 host |
| `HTTP_OPENAI_PROXY_PORT` | HTTP 监听端口 | `AIW_AGENT_PROXY_PORT` → `43127` | 规范名优先；1–65535 整数 |
| `HTTP_OPENAI_PROXY_STATE_DIR` | 结果与审计状态目录 | `AIW_AGENT_PROXY_STATE_DIR` → OS 临时目录下 `http-openai-proxy` | 规范名优先；结果在 `results/`，审计在 `audit.jsonl` |
| `AIW_AGENT_PROXY_STATE_DIR` | 状态目录兼容变量 | 同上 | 低于规范名 |
| `HTTP_OPENAI_PROXY_API_KEY` | 上游 API Key | `OPENAI_API_KEY` → 无 | 缺少密钥返回 provider 未配置 |
| `HTTP_OPENAI_PROXY_API_PROFILE` | 上游 API 模式 | `OPENAI_API_PROFILE` → `openai` | 仅 `openai`、`aiw_gateway`；后者不支持 reasoning 选项 |
| `OPENAI_API_PROFILE` | 上游模式后备变量 | `openai` | 低于规范名；不是 Go 全局配置变量 |
| `HTTP_OPENAI_PROXY_BASE_URL` | 上游 API 根地址 | `OPENAI_BASE_URL` → `https://api.openai.com/v1` | 仅 HTTPS 或 loopback HTTP；禁用 URL 凭据、query、fragment |

Gateway 的上游密钥变量名由配置决定，没有固定前缀：`principals[].upstream_api_key_env` 优先，未指定时取全局 `proxy_api_key_env`。它们的**值是变量名**，程序再读取该名称的进程环境。启用的主体在 `openai_proxy` 模式下必须拿到非空、无空白的密钥，否则配置失败；无密钥默认，也不自动假定 `OPENAI_API_KEY`。[实现](../../src/programs/agent-gateway/config.go#L114)

Gateway Codex 子进程只透传白名单：`PATH`、`HOME`、`USERPROFILE`、`SYSTEMROOT`、`WINDIR`、`APPDATA`、`LOCALAPPDATA`、`TEMP`、`TMP`、`CODEX_HOME`、`HTTPS_PROXY`、`HTTP_PROXY`、`NO_PROXY`。未设置则不注入默认；其他宿主变量不会由这条路径透传。[白名单](../../src/programs/agent-gateway/config.go#L159)

## 6. 通知与旧 Teams 脚本

受管通知的 `[notifications].credential_env` 也是**动态变量名**，没有固定变量名或默认密钥。teams 渠道读取其值判断 ready；未设置/空则未就绪。受管发送只透传 `SYSTEMROOT/WINDIR/PATH/TEMP/TMP/LANG/LC_ALL` 和这个指定凭据；不靠旧 `SEND_MESSAGE_*` 配置发送。[配置与就绪](../../src/plugins/aiw-notify/aiw-notify.py#L50)、[环境隔离](../../src/plugins/aiw-notify/aiw-notify.py#L127)、[managed 凭据读取](../../src/plugins/send_teams_msg.py#L369)

下表只适用于旧 `send_teams_msg.py` 普通模式；相关非空 CLI 参数优先于环境。环境存在但为空时通常保留空值。[实现](../../src/plugins/send_teams_msg.py#L260)

| 变量 | 含义 | 未设置的实际默认或回退 | 优先级/例外 |
| --- | --- | --- | --- |
| `SEND_MESSAGE_HOST` | 消息服务 host:port | `10.2.3.23:3001` | 显式完整 URL 存在时不用于构造地址 |
| `SEND_MESSAGE_PROFILE` | 服务 profile | `default` | 同上 |
| `SEND_MESSAGE_CHANNEL` | 服务 channel | `Default` | 同上 |
| `SEND_MESSAGE_PASSWORD` | 服务 ePassword 凭据 | 源码 `DEFAULT_PASSWORD` 含敏感字面默认，本文不复制其内容 | CLI > 环境 > 源码默认；运维应显式管理凭据，不把该字面值当作公共示例。[常量位置](../../src/plugins/send_teams_msg.py#L49) |
| `SEND_MESSAGE_URL` | 完整服务 URL | 空，随后按 host/profile/channel/mode 构造 | `--url` > 环境；完整 URL 优先于分段地址 |
| `SEND_MESSAGE_MODE` | `text/card/card-json` 模式 | `text` | `--mode` > 由显式 URL 检测出的模式 > 此变量 > 默认 |

## 7. 文件转换与编译器插件

| 变量 | 含义 | 未设置的默认或回退、优先级 | 源码 |
| --- | --- | --- | --- |
| `AIW_MD_CONVERTER` | Markdown 转换二进制 | 非空环境指定值优先；否则插件同目录 → `src/programs/md-converter/bin` → PATH 中 `md-converter`。显式值不可用则失败，不继续 fallback。 | [查找](../../src/plugins/aiw-file-convert/common.py#L31) |
| `AIW_CONVERT_PANDOC` | Pandoc 可执行程序 | PATH 中 `pandoc`；显式环境优先，不可用则失败 | [查找](../../src/plugins/aiw-file-convert/common.py#L31) |
| `AIW_CONVERT_ANTIWORD` | antiword 可执行程序 | PATH 中 `antiword`；显式环境优先，不可用则失败 | [查找](../../src/plugins/aiw-file-convert/common.py#L31) |
| `AIW_CONVERT_BROWSER` | md-converter 无头浏览器 | 显式值优先，否则自动找 Edge/Chrome/Chromium；Windows 还搜索标准安装目录。找不到则报错。 | [浏览器发现](../../src/programs/md-converter/browser.go#L20) |
| `ANTIWORDHOME` | 外部 antiword 映射文件目录 | AIW 不设置默认，透传给外部工具；Windows 依赖指导要求提供映射文件 | [依赖指导](../../src/plugins/aiw-file-convert/common.py#L17) |
| `BCC_ROOT` | Borland BCC 根目录 | 非空环境 → PATH 中 `bcc32.exe` 的上两级目录 → `C:\green\Borland\BCC55` | [检测](../../src/plugins/aiw-bcc.py#L89) |
| `VC_HOME`、`VC_ROOT`、`VSINSTALLDIR` | MSVC 根目录候选 | 按所列顺序取首个非空值 → PATH 中 `cl.exe` 的目录 → `D:\green\VCompiler` | [检测](../../src/plugins/aiw-vc.py#L38) |
| `TCC_HOME`、`TCC_ROOT`、`TCC_DIR` | Tiny C Compiler 根目录候选 | 按所列顺序取首个非空值 → PATH 中 `tcc.exe/tcc` 的目录 → `c:\green\tcc` | [检测](../../src/plugins/aiw-tcc.py#L36) |
| `JAVA_HOME` | exec-java 子进程 JDK | 未提供 java_home 配置时继承；提供后以解析的绝对目录覆盖，并把其 `bin` 加到 PATH 前部 | [子进程环境](../../src/plugins/aiw-exec-java.py#L181) |
| `MAVEN_OPTS` | exec-java Maven JVM 参数 | 原值（缺省空）；指定 file_encoding 时前置 `-Dfile.encoding=<encoding>`，若已有同一选项则不重复 | [参数注入](../../src/plugins/aiw-exec-java.py#L195) |

## 8. 构建、打包与脚本环境

| 变量 | 含义 | 未设置的默认或回退、优先级 | 源码 |
| --- | --- | --- | --- |
| `AIW_GO` | 根 `build.py` 的 Go 工具 | PATH 中 `go`；变量存在但为空不会自动恢复默认。`scripts/compile.py` 直接找 `go`，不读取此覆盖。 | [工具查找](../../build.py#L79)、[compile](../../scripts/compile.py#L22) |
| `AIW_NODE` | 根打包工具的 Node 工具 | PATH 中 `node`；与插件解释器规则不同，见第 2 节 | [调用](../../build.py#L164) |
| `AIW_VERSION` | 包版本 | 根 `build.py`：显式 `--version` > trim 后的环境值 > 带 UTC 时间的 snapshot 版本；编译槽固定为 snapshot。独立 package_definitions 入口：环境只作为 `--version` 默认，没有时要求提供该参数。 | [版本](../../build.py#L419)、[独立入口](../../scripts/package_definitions.py#L168) |
| `AIW_PACKAGE_REPO` | 发布到本地包仓库的路径 | 无默认；publish 必填本地路径，相对路径按仓库根解析；不接受网络地址 | [发布目标](../../build.py#L452) |
| `AIW_GOCACHE` | 根 compile 脚本专用 Go 缓存覆盖 | `<repo>/.ai/compile-cache/go`；解析后作为子进程 `GOCACHE`，覆盖继承值 | [缓存](../../scripts/compile.py#L13) |

根 `.env.example` 中 `AIW_PACKAGE_REPO=d:/playground/aiw-packages` 是本机路径示例，**不是程序默认**；其中注释的 `AIW_GO/AIW_NODE/AIW_INSTALL_DIR/AIW_VERSION` 也是配置示例。[模板](../../.env.example)

以下为脚本主动覆盖或注入的值，不能通过父进程同名变量改变它们；未覆盖的工具环境通常继承父进程。

| 场景 | 变量与实际注入值 | 源码 |
| --- | --- | --- |
| 根 build.py Go 编译 | `GOPROXY=off`、`GOSUMDB=off`、`GOTOOLCHAIN=local`、`GOFLAGS=""`、`GOWORK=off`、`CGO_ENABLED=0`、`GOOS=<目标系统>`、`GOARCH=amd64` | [设置](../../build.py#L151) |
| gateway compile | `GOPROXY=off`、`GOSUMDB=off`、`GOTOOLCHAIN=local`、`CGO_ENABLED=0`；分别 `GOOS=windows/linux`、`GOARCH=amd64` | [脚本](../../src/programs/agent-gateway/scripts/compile.py#L8) |
| md-converter compile | 上述离线与 CGO 值，加 `GOCACHE=<repo>/.ai/compile-cache/md-converter`；不主动覆盖 GOOS/GOARCH | [脚本](../../src/programs/md-converter/scripts/compile.py#L10) |
| packer compile | 上述离线与 CGO 值，加 `GOFLAGS=""`、`GOWORK=off`、`GOCACHE=<repo>/.ai/compile-cache/packer` | [脚本](../../src/programs/aiw-packer/scripts/compile.py#L12) |
| packer package | `GOPROXY=off`、`GOSUMDB=off`、`GOTOOLCHAIN=local`、`CGO_ENABLED=0`、`GOFLAGS=""`、`GOWORK=off` | [脚本](../../src/programs/aiw-packer/scripts/package.py#L13) |
| 根 scripts/test.py，仅测试 | `GOCACHE=<repo>/.ai/tmp/go-tests/cache`；`GOTMPDIR/TMP/TEMP/TMPDIR=<repo>/.ai/tmp/go-tests/tmp`；`GOPROXY/GOSUMDB=off` | [脚本](../../scripts/test.py#L30) |
| FD smoke，仅测试 | `AIW_FD_ROLE_RUNNER=<临时 runner 路径>` | [脚本](../../scripts/fd_smoke.py#L63) |
| proxy 本地 smoke，仅测试 | 临时 `AIW_AGENT_PROXY_STATE_DIR`、虚拟 `OPENAI_API_KEY` | [脚本](../../src/programs/http-openai-proxy/scripts/smoke-local.mjs#L10) |
| proxy 联网检查脚本，仅手动验证 | 必须有 `OPENAI_API_KEY`；`OPENAI_MODEL` 缺省为 `gpt-6-luna`；gateway 检查另设临时 `AIW_AGENT_PROXY_STATE_DIR` | [直连检查](../../src/programs/http-openai-proxy/scripts/check-openai.mjs#L1)、[gateway 检查](../../src/programs/http-openai-proxy/scripts/check-openai-gateway.mjs#L8) |

## 9. AIW 自动注入与剔除

普通插件继承完整父进程环境，再以 AIW 元数据覆盖同名键。直接运行插件脚本时没有这些自动值，不应把它们理解成独立程序的内置默认。[注入](../../src/internal/plugin/environment.go#L9)、[合并](../../src/internal/plugin/exec.go#L89)

| 变量 | 自动值与含义 | 未能生成时 |
| --- | --- | --- |
| `AIW_PLUGIN_NAME` | 本次插件名称 | 调用参数决定 |
| `AIW_PLUGIN_PATH` | 本次插件文件路径 | 调用参数决定 |
| `AIW_CMDLINE` | 本次命令行描述 | 调用参数决定 |
| `AIW_HOME` | 非空 `HOME`，否则非空 `USERPROFILE` | 两者缺少则不覆盖原环境 |
| `AIW_ROOT` | 当前 AIW 可执行文件目录；同时用于 AI 配置、包仓库配置和 Issue/FD 辅助程序查找 | 取可执行路径失败则不覆盖；独立运行时不应假定它已设置 |
| `AIW_WORKSPACE` | 当前工作目录 | 获取失败则不覆盖 |
| `AIW_PKG_STAGE` | 安装 hook 的 stage/data 目录 | 仅 hook 注入，无用户配置默认 |
| `AIW_PKG_ACTION` | 安装 hook 的当前 action | 仅 hook 注入，无用户配置默认 |

hook 同时把 `AIW_INSTALL_DIR` 覆盖为实际安装根目录。[hook 实现](../../src/plugins/aiw-pkg/manager.py#L283)

`aiw git merge-to` 的隔离 Git 调用会**删除**继承的 `GIT_DIR`、`GIT_COMMON_DIR`、`GIT_WORK_TREE`、`GIT_INDEX_FILE`、`GIT_OBJECT_DIRECTORY`、`GIT_ALTERNATE_OBJECT_DIRECTORIES`、`GIT_NAMESPACE`、`GIT_PREFIX`，以免调用者的仓库/索引路由劫持隔离工作区。它们无 AIW 默认；其他 Git 设置继续继承。[实现](../../src/plugins/aiw-git/git-merge-to.py#L55)

## 10. 操作系统与通用工具变量

| 变量 | 项目用途 | 未设置的实际行为/优先级 | 源码 |
| --- | --- | --- | --- |
| `PATH` | 插件、解释器、外部工具发现 | 继承操作系统值，无 AIW 通用默认；BCC 会前置自身工具目录，exec-java 可前置 JDK bin | [发现](../../src/internal/plugin/discover.go#L145)、[BCC](../../src/plugins/aiw-bcc.py#L142)、[Java](../../src/plugins/aiw-exec-java.py#L189) |
| `PATHEXT` | Windows 插件可执行扩展补充 | 未设置无补充，仍检查内置支持的扩展 | [发现](../../src/internal/plugin/discover.go#L300) |
| `HOME`、`USERPROFILE` | 用户目录、AIW_HOME、运行时配置路径 | AIW_HOME 优先 HOME；Go/Python 其余用户目录解析遵循各自平台 API，无统一硬编码目录 | [注入](../../src/internal/plugin/environment.go#L20)、[用户配置](../../src/internal/plugin/runtime_config.go#L118) |
| `APPDATA` | Windows say profile 根、Go 用户配置目录 | say 未设置则报错；其他 API 路径按平台规则处理 | [say](../../src/internal/say/profile.go#L42)、[运行时配置](../../src/internal/plugin/runtime_config.go#L118) |
| `XDG_CONFIG_HOME` | Unix say profile / Go 用户配置根 | say 缺省为 `~/.config`；运行时配置还兼容 `~/.config/aiw/aiw.toml` | [say](../../src/internal/say/profile.go#L49)、[运行时](../../src/internal/plugin/runtime_config.go#L118) |
| `GIT_EDITOR`、`VISUAL`、`EDITOR` | Go 交互编辑器 | 显式 editor 配置 > 按所列顺序首个非空环境 > Git core.editor > Windows `notepad` / 其他 `vim`；Node ai 的 EDITOR 要求见第 5 节 | [解析](../../src/internal/ui/ui.go#L72) |
| `NO_COLOR` | 禁用颜色 | Go UI 在非空时禁用；Python FD 则只要变量存在就禁用，空值也有效 | [Go](../../src/internal/ui/terminal.go#L64)、[FD](../../src/plugins/aiw-fd.py#L234) |
| `TERM` | 终端能力判断 | `dumb` 禁用颜色/ANSI；缺省依据平台及 TTY 路径判断，无 AIW 固定终端名 | [Go](../../src/internal/ui/terminal_ansi_other.go#L11)、[FD](../../src/plugins/aiw-fd.py#L234) |
| `TERM_PROGRAM`、`WT_SESSION`、`ANSICON`、`ConEmuANSI` | FD 终端 ANSI/超链接能力检测 | 无默认；识别终端名、Windows Terminal/ANSICON 存在值或 `ConEmuANSI=on` | [检测](../../src/plugins/aiw-fd.py#L237) |
| `LC_ALL`、`LANG` | Python cz 系统语言 | 非空 LC_ALL > LANG > Windows 用户语言或 Python locale；不支持的语言回退 en | [locale](../../src/plugins/aiw-cz/cz_config.py#L117) |
| `SHELL`、`PSModulePath` | setup-project Shell 判断与 PowerShell profile 目录选择 | SHELL 支持 bash/zsh/fish；Windows 非空 PSModulePath 可推断 PowerShell，含 windowspowershell 时选对应目录 | [检测](../../src/plugins/aiw-setup-project.py#L86) |
| `WSL_INTEROP`、`WSL_DISTRO_NAME` | say 剪贴板判断 WSL | 任一非空进入 WSL 剪贴板候选路径；都空则不使用该判断 | [实现](../../src/internal/say/clipboard_unix.go#L64) |
| `WAYLAND_DISPLAY`、`DISPLAY` | say 图形窗口/剪贴板可用性 | 非空 WAYLAND_DISPLAY 优先 Wayland 候选，DISPLAY 选 X11；都空时 GUI 路径报不可用 | [剪贴板](../../src/internal/say/clipboard_unix.go#L92)、[窗口](../../src/internal/say/dialog_platform_unix.go#L13) |
| `ProgramFiles`、`ProgramFiles(x86)`、`LocalAppData` | Windows 浏览器安装目录搜索 | 无 AIW 默认；缺少时继续其他浏览器候选 | [发现](../../src/programs/md-converter/browser.go#L24) |
| `SYSTEMROOT`、`WINDIR`、`LOCALAPPDATA`、`TEMP`、`TMP` | 子进程运行基础目录/临时目录 | 保留白名单中的原值，无 AIW 默认；测试脚本覆盖值见第 8 节 | [通知](../../src/plugins/aiw-notify/aiw-notify.py#L127)、[gateway](../../src/programs/agent-gateway/config.go#L159) |
| `HTTP_PROXY`、`HTTPS_PROXY`、`NO_PROXY` | gateway Codex 子进程代理环境 | 白名单透传原值；代理解释由外部工具负责，无 AIW 默认。md-converter 自身浏览器 loopback 连接明确不用环境代理 | [透传](../../src/programs/agent-gateway/config.go#L159)、[loopback](../../src/programs/md-converter/browser.go#L53) |

## 11. 核查记录与维护

本次仅用 `rg` 索引环境读取/注入点，用 PowerShell `Get-Content` 阅读相关源码、`.env.example` 与技能契约，并据此编写文档。未运行项目程序、测试、编译、最终产物构建、formatter、linter、验证脚本、网络请求或 Git 写入；本文的行为描述是静态证据，不是运行结果。

新增环境读取时，应同步更新变量名、含义、空值规则、默认值、入口优先级与源码链接。动态 `credential_env`、`proxy_api_key_env` 和 `upstream_api_key_env` 由部署配置选名，无法静态枚举每个部署实际使用的名称；应在部署自己的配置清单中登记名称，密钥值放入其凭据管理系统。
