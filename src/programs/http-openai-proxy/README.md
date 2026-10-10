# HTTP OpenAI Proxy

`http-openai-proxy` 为不能直接连接 OpenAI API 的应用提供普通 HTTP JSON
接口，并通过 OpenAI Responses API 调用一个配置好的上游。上游可以是
OpenAI，也可以是实现了 Responses 接口子集的 OpenAI API 兼容服务，例如
AIW `agent-gateway`。

它是请求适配器，不是任意 HTTP 路径的透明转发器：入站请求使用本程序的
JSON 格式，程序将提示词转换成 Responses 请求，再返回统一的结果和用量。
它不会按请求选择 Codex、Copilot 或其他后端，也不会启动本地模型 CLI。

## 启动

需要 Node.js 22.12 或更高版本。在本目录安装依赖并编译后启动：

```powershell
npm install
npm run build
node ./http-openai-proxy.js start --port 43127
```

`npm start` 使用相同的新入口。旧 `aiw-agent-proxy.js` 仍可作为兼容启动
入口。默认只监听 `127.0.0.1`。可用 `HTTP_OPENAI_PROXY_HOST`、
`HTTP_OPENAI_PROXY_PORT` 和 `HTTP_OPENAI_PROXY_STATE_DIR` 配置监听地址、端口
和临时状态目录；旧 `AIW_AGENT_PROXY_HOST`、`AIW_AGENT_PROXY_PORT`、
`AIW_AGENT_PROXY_STATE_DIR` 作为回退设置保留。

远程 HTTP 监听没有内置认证，也不启用 TLS。远程使用时必须用防火墙或可信
网络限制访问；需要加密传输时，应放在 HTTPS 终止代理后。任何能够访问该
端口的调用方都能消耗上游 API 配额。

## 上游配置

上游通过环境变量对整个进程统一配置，不能由单个请求切换：

| 变量 | 必需 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `HTTP_OPENAI_PROXY_API_KEY` | 是 | 无 | 上游 API Key |
| `HTTP_OPENAI_PROXY_BASE_URL` | 否 | `https://api.openai.com/v1` | 上游 Responses API 根地址 |
| `HTTP_OPENAI_PROXY_API_PROFILE` | 否 | `openai` | `openai` 或 `aiw_gateway` |

旧配置 `OPENAI_API_KEY`、`OPENAI_BASE_URL` 和 `OPENAI_API_PROFILE` 仍可作为
回退值使用。新部署建议迁移到带 `HTTP_OPENAI_PROXY_` 前缀的设置。
远程上游必须使用 HTTPS；HTTP 仅允许 loopback 地址，便于连接本机
`agent-gateway`。

`aiw_gateway` profile 会省略 OpenAI 专用的 `max_output_tokens`，并在发出
请求前拒绝 reasoning effort 和 reasoning summary，因为当前 Gateway 不支持
这些参数。Profile 必须显式设置，不会根据 URL 自动推断。无效 profile 会在
发出请求前报配置错误。所有 Responses 请求关闭 SDK 自动重试，完整请求期限
为 60 秒。

## HTTP 接口

`POST /v1/requests` 接受 JSON：

```json
{
  "client_id": "editor-1",
  "request_id": "optional-id",
  "model": "gpt-6-luna",
  "output_format": "markdown",
  "prompt": "Summarize the requested change."
}
```

`provider` 字段可以省略；为兼容旧调用方，若提供则只接受 `openai`，不再
用于后端选择。`output_format` 支持 `json` 或 `markdown`。可选的
`system_prompt` 和 `include_reasoning_summary` 会映射到 OpenAI Responses
参数；`json` 输出会请求 JSON object 并校验返回内容。响应提供结果、请求 ID
和可获得的 token 用量；Responses API 未提供货币金额，因此 `cost` 为 `null`。

请求体最大 1 MiB，提示词最多 64 KiB，返回文本最多 256 KiB。调用过多时返回
429。服务写入不含提示词和结果正文的 `audit.jsonl`。WebSocket
`/v1/events`、结果暂存和 ACK 仍保留为旧客户端的兼容通道；新集成应使用
普通 HTTP 请求。

## 本地 OpenAI 检查

`scripts/check-openai.mjs` 直接请求官方 OpenAI Responses API；
`scripts/check-openai-proxy.mjs` 通过本服务发起一次真实请求。后者会调用
上游并消耗额度，只应在有意验证时手动运行。`scripts/smoke-local.mjs` 使用
模拟响应，不需要真实上游凭据。以上脚本不属于构建步骤。

## 安全和状态

`client_id` 是调用方自报的路由标签，不提供身份认证。服务默认 loopback
监听；远程监听时任何能访问端口的人都可提交请求。结果暂存目录默认位于
系统临时目录，结果在 ACK 后删除，未确认结果一小时后过期。不要将状态目录
放在同步或共享文件夹中。
