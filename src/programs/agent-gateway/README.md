# Agent Gateway 独立程序

## Chat Completions 文本接口

`POST /v1/chat/completions` 可供现有 `aiw say` 接入 Codex 后端。仅支持 `model`、`messages`、省略或 false 的 `stream`，以及省略或 1 的 `n`。消息内容必须为非空白字符串；开头的 system/developer 转为指令，user/assistant 保持对话顺序，至少需要一条 user 消息。工具、图片/音频、content 数组、流式及其他生成参数在执行前拒绝。

成功返回单个 `choices[0].message.content` 和 `finish_reason:stop`。已知 usage 使用 prompt_tokens/completion_tokens/total_tokens，未知为 null。该接口与 Responses 共用认证、模型授权、额度、超时、取消和 `openai_codex` 执行，不需要额外配置。

默认本地监听配置下，Say 接入示例（请替换网关 Key 和允许的逻辑模型）：

```powershell
$env:OPENAI_BASE_URL = "http://127.0.0.1:43127/v1"
$env:OPENAI_API_KEY = "你的网关Key" # df11c91346f9836ddd828ce256efd3827068b6e082ce50816ac3745016859128, fbf233cd659a483392663ee9b85c8eeebb03d12490614c9ec47ff4bc9fedb661
aiw say --model "允许的逻辑模型" "请确认明天会议时间"
```

现有配置和记录可由新版继续加载。新接口产生的 HTTP 审计记录包含 `/v1/chat/completions` route；旧二进制可能不识别该值，不能保证直接降级读取新记录。请保留状态数据，降级处理需另行安排，不应删除记录绕过校验。

## Codex backend

The Gateway uses `openai_codex` with the AIW-managed ChatGPT Codex login. It does not start a Codex App Server subprocess. See the backend configuration and login sections below.

`agent-gateway` 与 `aiw.exe` 同级安装，直接调用：

- Windows：`agent-gateway.exe`
- Linux：`agent-gateway`（需要执行权限）

在仓库根目录运行 `python build.py gateway` 可在 `bin/` 生成两个平台的程序，
并安装到 ``AIW_INSTALL_DIR``（默认 `C:\green\aiw`）。`python build.py bin` 和
`python build.py all` 也会安装 Gateway；`python build.py plugins` 不再安装 Gateway。
网关是独立 Go 模块，
构建目标必须是 `src/programs/agent-gateway`，不可使用根目录的 `cmd/aiw-req`。

```sh
agent-gateway start --config /absolute/path/gateway.json
agent-gateway stop --config /absolute/path/gateway.json
```

Windows 直接运行 `agent-gateway.exe`。启动和停止均须显式传入 `--config`，
并使用同一配置文件。安装时将配置样例写入程序同目录的 `gateway-new.json`，
保留已有 `gateway.json`；首次使用应按实际环境修改样例并另存为 `gateway.json`。
安装不再复制 `aiw-gw.py`。旧安装目录中的插件和配置不会自动删除或迁移。

New configuration uses `state_dir` for `openai_codex` or `proxy_base_url` for `openai_proxy`. Remove legacy `workspace_dir`, `codex_path`, and `codex_script` fields.

新配置使用 `principals[].keys` 保存运营者签发的明文网关 Key，请求直接匹配。
仅接受 `keys`，不支持 `key_hashes`，配置加载和认证均不计算摘要。原摘要字符串
可以直接作为 Key，客户端发送同一字符串即可。真实 Key 不应进入版本库或日志，
配置变更重启生效。

`openai_codex` persists HTTP request metadata under `state_dir/requests`; local `GET /v1/usage` reports that principal's HTTP and execution groups.
`openai_proxy` does not persist HTTP request metadata and does not provide local usage statistics.

旧配置可以通过 `--config` 显式指定；移动配置时应保留或正确调整这些绝对路径。

## 后端模式

Startup config must explicitly set `backend_mode`. An omitted value or legacy `codex` value fails with a message to set `openai_codex` for the AIW-managed ChatGPT Codex login.

代理模式将本地 `/v1` 下的 HTTP 请求和 Realtime WebSocket 透传至固定上游，`/internal/*` 保留为本地控制接口。网关用服务端上游 Key 替换入站 Authorization/api-key，并移除调用方组织、项目身份头。上游 3xx 原样返回，不自动跟随或重试。统计和计费尚未实现；代理模式不解析 usage，也不持久化请求记录。

可从 `gateway-proxy-example.json` 创建独立代理配置。为所有启用主体设置 `upstream_api_key_env`，或使用全局 `proxy_api_key_env`；环境变量必须在启动进程时已设置。不要把上游 Key 写入配置文件或日志。共享上游 Key 的主体共用该 Key 的权限。

## Direct OpenAI Codex mode

Set `backend_mode` to `openai_codex` to send Responses requests through AIW's
user-managed OpenAI Codex login. This mode does not start or require the Codex
executable. It still requires an absolute `state_dir`, model mappings, enabled
principals, Gateway client keys, allowed models, RPM/concurrency limits and a
daily limit. It does not use `workspace_dir`, `codex_path` or `codex_script`.

The first login is managed by the AIW command line. Run `aiw auth import-codex
--from <path>` when an existing Codex credential file is available, or run
`aiw auth login openai-codex` to sign in through a browser. Use `aiw auth
status` to inspect login expiry and `aiw auth logout` to remove the AIW copy.
Import reads but does not change the source file. A request refreshes an expired
access token when possible; it never starts an interactive login.

Use `gateway-codex-example.json` as a configuration template. Replace its
placeholder Gateway key and paths before starting the service. Clients continue
to use a Gateway key; the Codex OAuth credential stays in AIW's protected user
credential store. `aiw say`, `aiw-cz` and `http-openai-proxy` can use this
Gateway through their existing OpenAI-compatible URL and key settings.

Direct mode uses a local PI-compatible Codex Responses endpoint. That endpoint
is not documented as a stable public API, can change independently, and does
not provide the `openai_codex`'s local read-only sandbox. The Gateway does not
execute returned function calls; callers remain responsible for handling them.

## DeepSeek through the OpenAI proxy mode

The Gateway's existing `openai_proxy` backend can forward compatible requests
to DeepSeek without a DeepSeek-specific backend. Set `proxy_base_url` to
`https://api.deepseek.com/v1` and `proxy_api_key_env` to `DEEPSEEK_API_KEY`, then
provide that environment variable to the Gateway process. Gateway clients still
authenticate with their Gateway key; the Gateway replaces it with the
DeepSeek key upstream. See `gateway-deepseek-example.json`.
