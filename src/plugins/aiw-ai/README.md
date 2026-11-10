# aiw ai 客户端

Node 22.12+。`aiw ai` 每次发送一个请求，默认使用 AIW `[ai]` 配置的 Provider；可用 `--backend gateway` 选择 Go Agent Gateway。命令不会因一路由失败而改走另一路由。独立的 `aiw ask` 用于 AIW 使用指导，行为不变。

## 路由

| 路由 | 选择方式 | 配置 |
| --- | --- | --- |
| AIW Provider | 默认；也可用 `--backend aiw` | `aiw.toml` / `.aiw.toml` 的 `[ai]` 和标准 `AIW_LLM_*` 环境变量 |
| Go Agent Gateway | `--backend gateway` | Gateway Key、逻辑模型和本机/远程 Gateway 地址 |

AIW 路由调用 AIW 共享 Provider 层，支持其当前配置的 Provider。命令不保存请求；CLI Provider 使用共享 AI 层的只读请求选项。`--timeout-seconds` 对本次调用设定客户端期限。Provider 配置错误或调用失败会直接报错，不会回退到 Gateway。

### Gateway 设置

Gateway 仅在所选路由时需要配置并启动：

1. 启动网关：`aiw gw start`。默认读取程序同目录的 `gateway.json`。
2. 设置 `AIW_AI_API_KEY` 为网关 Key，至少 43 字符、无空白。服务端 `principals[].keys` 保存相同字符串；这不是上游 Provider API Key。
3. Gateway 的 `models` 和当前主体 `allowed_models` 必须包含所选逻辑模型。

```json
"models": { "gpt-6-luna": "gpt-6-luna" }
```

```json
"allowed_models": ["gpt-6-luna"]
```

`key_hashes` 已废除；服务端和客户端必须使用相同 Key 字符串。真实 Key 不应放入版本库或日志。Gateway 配置变更后重启生效。

Gateway 相关设置：

| 设置 | 默认值 / 含义 |
| --- | --- |
| `AIW_AI_API_KEY` | Gateway 路由必填；至少 43 字符、无空白 |
| `AIW_AI_MODEL` | Gateway 逻辑模型，默认 `gpt-6-luna` |
| `AIW_AGENT_PROXY_PORT` | 默认 `43127`，范围 1–65535 |
| `--url` | Gateway HTTP(S) origin；默认 `http://127.0.0.1:43127` |

只提供 `--url` 而省略 `--backend` 时，命令仍将其视为显式 Gateway 选择以兼容旧调用。`--backend aiw` 与 `--url` 同时使用时报参数错误。`--url` 拒绝凭证、路径、query 和 fragment；远程网关须显式传入。

## 输入

```sh
aiw ai "How to learn English"
aiw ai --backend gateway "Explain this text"
aiw ai --file report.txt "Summarize this report"
aiw ai --file legacy.txt --encoding gb18030 "Summarize this report"
aiw ai --backend gateway --url https://gateway.example --json "Return one JSON object"
```

位置参数按空格组成提示词。指定 `--file PATH` 后，文件内容追加在位置提示词之后，以一个空行分隔；没有位置提示词时直接使用文件内容。每次只能指定一个文件。文件在本机解码；仅组合后的文本发送给所选 Provider，文件路径不会传给模型。

文件默认严格按 UTF-8 解码，也可用 `--encoding LABEL` 指定 Node `TextDecoder` 支持的编码标签。文件原始大小上限为 256 KiB，组合后的提示词上限为 65536 个 Unicode 码点。无效字节、不支持的编码、不可读文件和超限输入均在请求前报错。`--encoding` 只与 `--file` 一起使用；`--file` 不能与 stdin 的 `-` 组合。`--system @FILE` 仍使用 UTF-8，不受 `--encoding` 影响。

单独的 `-` 读取完整 stdin，不与其它位置提示词组合。没有位置提示词和文件时读取一行终端输入；Ctrl+E 用 `EDITOR` 编辑当前文字并支持多行，Ctrl+C 取消。空白/EOF 不发送请求。非终端 stdin 如需读取，须显式使用 `-`。

`--system TEXT` 设置系统提示词；`--system @FILE` 在本机读取 UTF-8 文件。文件缺失、空、无效 UTF-8 或超限时不发送请求。

## 输出和选项

- `--backend aiw|gateway` 选择后端，默认 `aiw`。`--url` 单独出现时兼容旧调用并选择 Gateway。
- `--timeout-seconds N` 设定请求总期限，默认 660 秒，只接受 1–3660 的十进制整数。超时取消当前调用，不自动重试。
- `--json` 要求 JSON object 并校验结果；无效 JSON 或非 object 报错，不输出伪成功结果。
- `--verbose` 仅向 stderr 写可得的 Provider/model/响应 ID/用量信息；缺失字段标为 unknown/unavailable。
- `-h` / `--help` 不需要凭据，也不调用模型。

stdout 仅包含回答，诊断写入 stderr。Gateway 路由使用 `text.format.json_object`；AIW Provider 路由要求 JSON object 并在客户端校验。AIW Provider 如未报告 token/费用等信息，verbose 不会推算或编造。

## 交互编辑器

```powershell
$env:EDITOR = 'notepad.exe'
aiw ai
# VS Code 使用真实 exe，并等待窗口关闭：
$env:EDITOR = '"C:\Program Files\Microsoft VS Code\Code.exe" --wait'
```

```sh
export EDITOR='vi'
aiw ai
# Linux VS Code：export EDITOR='code --wait'
```

`EDITOR` 支持程序、参数及单/双引号包围的路径；不解释 shell 展开、管道或重定向。Windows 使用真实 `.exe`，不使用 `.cmd`/`.ps1` 启动脚本。GUI 编辑器须配置等待行为。编辑内容短暂写入本机临时文件，提交前清理；强制结束进程或编辑器自建备份不由客户端管理。
