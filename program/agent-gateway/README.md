# Agent Gateway 独立程序

## Chat Completions 文本接口

`POST /v1/chat/completions` 可供现有 `aiw say` 接入 Codex 后端。仅支持 `model`、`messages`、省略或 false 的 `stream`，以及省略或 1 的 `n`。消息内容必须为非空白字符串；开头的 system/developer 转为指令，user/assistant 保持对话顺序，至少需要一条 user 消息。工具、图片/音频、content 数组、流式及其他生成参数在执行前拒绝。

成功返回单个 `choices[0].message.content` 和 `finish_reason:stop`。已知 usage 使用 prompt_tokens/completion_tokens/total_tokens，未知为 null。该接口与 Responses 共用认证、模型授权、额度、超时、取消和 App Server 执行，不需要额外配置。

默认本地监听配置下，Say 接入示例（请替换网关 Key 和允许的逻辑模型）：

```powershell
$env:OPENAI_BASE_URL = "http://127.0.0.1:43127/v1"
$env:OPENAI_API_KEY = "你的网关Key"
aiw say --model "允许的逻辑模型" "请确认明天会议时间"
```

现有配置和记录可由新版继续加载。新接口产生的 HTTP 审计记录包含 `/v1/chat/completions` route；旧二进制可能不识别该值，不能保证直接降级读取新记录。请保留状态数据，降级处理需另行安排，不应删除记录绕过校验。

## Codex 后端

Gateway 使用 Codex App Server 的 stdio JSON-RPC，并按 `global_concurrency` 建立有界进程池。每个请求在一个空临时工作目录中启动新的 ephemeral thread；健康进程可复用。每个 turn 使用配置中的模型映射、只读且禁网的 sandbox，以及 `approvalPolicy=never`。

子进程继承允许列表中的 `CODEX_HOME` 和现有登录状态，不复制凭据，也不修改用户 Codex 配置。启动 turn 前会读取有效配置和 MCP server 状态；只要 MCP 状态启用或无法确认，就拒绝该请求。Gateway 显式关闭本地工具、apps、plugins、hooks、web、浏览器/电脑使用、图像生成和多 agent 功能。协议异常或执行状态不明的进程会被丢弃，Gateway 不会自动重试。

此版本支持无工具文本 SSE、`json_object` 和非流式顺序 function call。严格 function 参数只接受 FD-045 列明的 JSON Schema 子集；调用方仍负责执行 function 并在后续请求重放调用和结果。App Server 的实际 SDK/进程运行验证须单独授权；本地编译本身不验证安装版本、登录状态或运行时协议兼容性。

`agent-gateway` 与 `aiw.exe` 同级安装，直接调用：

- Windows：`agent-gateway.exe`
- Linux：`agent-gateway`（需要执行权限）

在仓库根目录运行 `build.bat gateway` 可在 `bin/` 生成两个平台的程序，
并安装到 `%INSTALL_DIR%`（默认 `C:\green\aiw`）。`build.bat bin` 和
`build.bat all` 也会安装 Gateway；`build.bat plugins` 不再安装 Gateway。
网关是独立 Go 模块，
构建目标必须是 `program/agent-gateway`，不可使用根目录的 `cmd/aiw-req`。

```sh
agent-gateway start --config /absolute/path/gateway.json
agent-gateway stop --config /absolute/path/gateway.json
```

Windows 直接运行 `agent-gateway.exe`。启动和停止均须显式传入 `--config`，
并使用同一配置文件。安装时将配置样例写入程序同目录的 `gateway-new.json`，
保留已有 `gateway.json`；首次使用应按实际环境修改样例并另存为 `gateway.json`。
安装不再复制 `aiw-gw.py`。旧安装目录中的插件和配置不会自动删除或迁移。

配置文件中的 `state_dir`、`workspace_dir`、`codex_path` 必须是当前操作系统
的绝对路径。Windows 示例：

```json
{
  "state_dir": "C:/green/aiw/gateway-state",
  "workspace_dir": "C:/green/aiw/gateway-work",
  "codex_path": "C:/green/nodejs/node.exe",
  "codex_script": "C:/green/nodejs/node_modules/@openai/codex/bin/codex.js"
}
```

这是路径字段片段，需要合入完整配置。按真实安装路径替换 Node/Codex 路径；
Windows npm 安装须使用 `node.exe` + `codex_script`，不能指定 `codex.cmd` 或
`codex.ps1`。模型映射和 Key 等示例占位值也必须替换后才能使用。

新配置使用 `principals[].keys` 保存运营者签发的明文网关 Key，请求直接匹配。
仅接受 `keys`，不支持 `key_hashes`，配置加载和认证均不计算摘要。原摘要字符串
可以直接作为 Key，客户端发送同一字符串即可。真实 Key 不应进入版本库或日志，
配置变更重启生效。

所有 HTTP 请求均记录请求 ID、网关调用主体、接口、结果和耗时；HTTP 元数据保存
于 `state_dir/requests`，与执行额度元数据分开。`GET /v1/usage` 的 `http_groups`
提供当前主体的请求及拒绝统计，原 `groups` 继续提供实际执行与 token 用量。
旧配置可以通过 `--config` 显式指定；移动配置时应保留或正确调整这些绝对路径。

## 后端模式

`backend_mode` 缺省为 `codex`，旧配置继续使用 Codex 后端。设置为 `openai_proxy` 时，进程只使用固定的 `proxy_base_url` 和环境变量中的上游 Key；该模式不需要 Codex 程序、工作目录或状态存储。两种模式使用相同的网关认证、主体 RPM 与并发限制。

代理模式将本地 `/v1` 下的 HTTP 请求和 Realtime WebSocket 透传至固定上游，`/internal/*` 保留为本地控制接口。网关用服务端上游 Key 替换入站 Authorization/api-key，并移除调用方组织、项目身份头。上游 3xx 原样返回，不自动跟随或重试。统计和计费尚未实现；代理模式不解析 usage，也不持久化请求记录。

可从 `gateway-proxy-example.json` 创建独立代理配置。为所有启用主体设置 `upstream_api_key_env`，或使用全局 `proxy_api_key_env`；环境变量必须在启动进程时已设置。不要把上游 Key 写入配置文件或日志。共享上游 Key 的主体共用该 Key 的权限。
