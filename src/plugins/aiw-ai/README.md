# ai 客户端

Node 22.12+，入口 `aiw-ai.mjs`（npm bin 为 ai）。调用 Go Agent Proxy 的 Responses 支持子集，不直接调用 Provider。

## 最少配置

1. 配置并启动网关：`aiw gw start`。默认读取程序同目录的 `gateway.json`。
2. 设置 `AIW_AI_API_KEY` 为网关 Key，至少 43 字符、无空白。服务端 `principals[].keys` 保存相同字符串，直接比较，不计算哈希；此 Key 不是上游 OpenAI API Key。
3. 网关必须配置并授权所选逻辑模型。客户端默认 **`gpt-6-luna`**，无需设置 `AIW_AI_MODEL`。

### 网关 Key 配置

`key_hashes` 已废除，网关拒绝包含该字段的配置，认证无需运行哈希脚本。服务端 `keys` 与客户端
`AIW_AI_API_KEY` 必须使用相同字符串。已有摘要字符串可作为不透明 Key，但此时两端
都使用该摘要字符串，不能让客户端继续发送摘要计算前的原始 Key。真实 Key 不应进入版本库或日志。

历史 `scripts/hash-gateway-key.bat` 不再用于当前网关的认证配置。
由运营者维护完整配置的 `principals[].keys`，客户端填写同一个 Key，配置修改后重启网关生效。

网关配置中需合入下列模型映射，并在对应主体的 `allowed_models` 中包含该名字：

```json
"models": { "gpt-6-luna": "gpt-6-luna" }
```

```json
"allowed_models": ["gpt-6-luna"]
```

保留其它需要的映射和权限；配置变更后重启网关。后端模型可用性由所用 Codex 环境决定。
若网关只提供 `gateway-default`，可设置 `AIW_AI_MODEL=gateway-default` 使用既有逻辑模型。

PowerShell（当前终端）：

```powershell
$env:AIW_AI_API_KEY = '<你的网关 Key>'
aiw ai "How to learn English"
# 可选，覆盖默认模型：
$env:AIW_AI_MODEL = 'gateway-default'
```

Linux shell：

```sh
export AIW_AI_API_KEY='<你的网关 Key>'
aiw ai "How to learn English"
```

## 设置与帮助

| 设置 | 是否必填 | 默认值 / 含义 |
| --- | --- | --- |
| `AIW_AI_API_KEY` | 是 | 网关认证 Key，至少 43 字符、无空白 |
| `AIW_AI_MODEL` | 否 | `gpt-6-luna`；非空设置覆盖默认值 |
| `AIW_AGENT_PROXY_PORT` | 否 | `43127`，范围 1–65535 |
| `--url` | 否 | `http://127.0.0.1:43127`，端口可由上述变量覆盖 |
| `--timeout-seconds` | 否 | 客户端请求总期限默认 660 秒；仅接受 1–3660 的十进制整数秒 |

移除旧 `AIW_AI_PROVIDER`。`--url` 仅接受 HTTP(S) origin，追加 `/v1/responses`，拒绝凭证、路径、query、fragment。远程网关须显式传入 `--url`。

```sh
aiw ai --help
aiw ai -h
aiw help ai
```

这些帮助入口无需认证或服务启动，不调用模型。缺失或不合法的 Key 会提示最少要求和帮助入口。

## 使用方式

```sh
aiw ai "Explain this text"
aiw ai --timeout-seconds 300 "Explain this text"
printf 'Summarize this text' | aiw ai -
aiw ai --url https://gateway.example --system @instructions.txt --json --verbose "Return a JSON object"
```

所有非选项参数按顺序用空格组成 Prompt，可直接运行 `aiw ai How to learn English`。
选项及其值不加入 Prompt；以 `-` 开头的文字需放在 `--` 后。
单独的 `-` 读取完整 stdin，不能与其它提示词参数混用。空白/EOF 不请求。
`--system` 单独映射 instructions，@ 文件为本机 UTF-8 文件内容，不把本机路径发送给服务端。文件缺失、空、无效 UTF-8、超限在请求前失败且不曝光内容。

### 交互输入与外部编辑器

无参数运行 `aiw ai`，输入一行后按 Enter 提交，Ctrl+C 取消。
也可按 **Ctrl+E** 用 `EDITOR` 编辑当前输入，支持多行；保存并关闭编辑器后立即提交。

```powershell
$env:EDITOR = 'notepad.exe'
aiw ai
# VS Code 使用真实 exe，并等待窗口关闭（按本机路径修改）：
$env:EDITOR = '"C:\Program Files\Microsoft VS Code\Code.exe" --wait'
```

```sh
export EDITOR='vi'
aiw ai
# Linux VS Code：export EDITOR='code --wait'
```

`EDITOR` 支持程序、参数及单/双引号包围的路径；不支持 shell 展开、管道或重定向。
Windows 使用真实 `.exe`，不使用 `.cmd`/`.ps1` 启动脚本。GUI 编辑器必须配置等待行为。
未设置 EDITOR、启动失败、异常退出、非 UTF-8 或内容超限均不请求。
临时文件在提交前清理；空内容不请求。强制结束进程可能留下临时文件，编辑器自己的备份不由客户端管理。

`--json` 请求 text.format.json_object，stdout 为单个 JSON object；非 JSON 或非 object 报错。普通模式输出 Responses assistant output_text。`--verbose` 仅 stderr 输出模型、响应 ID、可得 token，缺失费用为 unknown、推理摘要 unavailable；不展示隐藏推理。一次有界请求，客户端默认总期限 660 秒，包含完整响应读取，无自动重试，不持久保存输入。`--timeout-seconds` 只改变客户端期限，不发送到 Prompt 或 Responses JSON，也不改变网关 `timeout_seconds` 的执行上限；缺值、重复或非法值在发请求前以退出码 2 报错。网关配置修改后需重启生效。客户端超时取消请求后不能恢复该次执行。Key 不写日志，诊断与 stdout 分开。

旧 /v1/requests/result/system_prompt 及多 Provider 客户端契约已经退役；必须同时升级服务与客户端。测试和容器部署由用户另行安排，SDK/CLI 运行兼容尚未验证。
