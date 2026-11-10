# Agent Gateway CLI client specification

## Purpose

提供单次 `aiw ai` 请求，可选择 AIW 共享 Provider（默认）或 Go 网关 Responses 支持子集；不改变独立 `aiw ask`。

## Requirements

### Requirement: Explicit bounded input

客户端 MUST 将多个非选项位置参数以空格组成 Prompt，保留单个参数内容。单独的 `-` 读取完整 stdin，不与其它提示词参数混用；`--` 后参数不解析为选项。可指定单个 `--file PATH` 作为请求正文文件，文件 MUST 在本机读取，默认严格按 UTF-8 解码；`--encoding LABEL` MUST 指定 Node TextDecoder 支持的编码标签，且仅可与 `--file` 一起使用。文件原始字节上限为 256 KiB；文件和位置参数同时提供时，位置参数 MUST 在前，二者以一个空行分隔；合并后上限为 65536 个 Unicode 码点。文件路径不得传给 Provider。文件输入与 stdin `-` 不可组合。无参数、无文件时读取一行交互输入。空白/EOF MUST 不请求；非终端且未用 `-` 或 `--file` MUST 立即报错。system @ 文件 MUST 在本地以 UTF-8 有界读取，缺失/空/无效/过大在请求前失败且不泄露内容。

交互输入时 Ctrl+E MUST 使用 EDITOR 编辑当前输入，支持多行。EDITOR 为程序及参数，支持带引号的路径，不解释 shell 表达式；Windows 使用真实 exe。编辑器继承终端，保存并正常退出后提交；空内容、启动失败、非零/信号退出、无效 UTF-8、超限和清理失败 MUST 不请求。临时文件 MUST 在请求前清理，Unix 初始文件权限 0600；Windows 使用当前用户临时目录的 ACL。GUI 编辑器等待行为由运营者配置。

### Requirement: Gateway identity and Responses mapping

`--backend` MUST 接受 `aiw` 或 `gateway`，默认 `aiw`。AIW 路由 MUST 使用共享 `[ai]` Provider 配置与标准 `AIW_LLM_*` 环境覆盖，通过 Go 共享 Provider 层执行一次只读生成。显式 Gateway 路由 MUST 只调用 Gateway；Gateway 配置错误或请求失败 MUST NOT 改走 AIW Provider。Gateway Key/模型/端口设置仅适用于 Gateway 路由；Gateway 路由中 AIW_AI_API_KEY MUST 必填且至少 43 字符、无空白。AIW_AI_MODEL 为可选逻辑模型，未设置或为空时 MUST 默认 gpt-6-luna；非空设置覆盖默认值。所选模型仍须由网关配置并为当前主体授权。旧 AIW_AI_PROVIDER MUST 明确报已移除，不读取上游凭证。`--url` MUST 接受 HTTP(S) origin 并追加 /v1/responses，拒绝 URL 凭证、路径、query/fragment；默认 loopback 与 AIW_AGENT_PROXY_PORT 保留。仅提供 `--url` 而不指定 backend MUST 兼容性地选择 Gateway；`--backend aiw` 与 `--url` 同用 MUST 在请求前报错。

-h 与 --help MUST 显示用法、选项、输入方式、AIW/Gateway 配置方法、Gateway 默认模型和地址，以及 Windows/Linux Gateway Key 设置示例；帮助 MUST 无需凭据且不发请求。Gateway Key 缺失或格式错误 MUST 提示最低要求和帮助入口，不泄露 Key。

最少配置指引 MUST 使用服务端 `principals[].keys` 与客户端 `AIW_AI_API_KEY` 相同的网关 Key 字符串。`key_hashes` 已废除，网关不计算摘要并拒绝该字段；文档 MUST 不将哈希工具列为当前认证配置步骤。旧摘要若被用作不透明 Key，两端 MUST 使用该摘要字符串，不能发送摘要计算前的原始 Key。

Gateway 路由 MUST 发送一次 Bearer 认证的 POST Responses，input 为文本，--system 为独立 instructions，--json 为 text.format.json_object；不直接访问 Provider、不自动重试。AIW 路由 MUST 通过有界内部 JSON stdin/stdout 协议调用共享 Provider 层，stdin MUST 限制为 1 MiB 且拒绝未知字段和尾随 JSON；单次提示词和系统提示分别最多 65536 个 Unicode 码点。Provider 响应文本 MUST 限制为 4 MiB UTF-8 字节，序列化后的响应 envelope（含末尾换行）MUST 限制为 25 MiB。AIW 路由 MUST 传递超时并使用只读 Provider 请求选项，不写 AIW Session。`--json` MUST 要求单个 JSON object 并校验，不合法或非 object 报错、不输出伪成功内容；Provider JSON 模式 MUST 明确请求一个 JSON object。EDITOR 输入可短暂写入本机临时文件，正常或可捕获失败路径须清理；强制终止及编辑器备份由本机使用者管理。

### Requirement: Honest output

Gateway 客户端 MUST 从标准 completed Response 的 assistant output_text 读取文本；Gateway 的 `--verbose` MUST 仅 stderr 显示响应 ID/模型/可得 token，缺失费用为 unknown、推理摘要 unavailable，不能编造金额或展示隐藏推理。AIW 路由 MUST 返回 Provider 最终文本；verbose 仅显示实际可得的 Provider/model 信息，缺失的 ID/用量 MUST 显示 unknown/unavailable。两路由 stdout 与诊断 MUST 分离。客户端请求总期限默认 660 秒，包含完整响应读取/Provider 调用，响应有界；`--timeout-seconds` MUST 只接受 1–3660 的十进制整数秒，缺值、重复和非法值 MUST 在请求前以退出码 2 报错。此参数 MUST 不进入 Prompt 或请求 JSON（内部桥接 timeout 字段除外），不得改变 Gateway 服务端执行上限。超时取消后不能恢复该次执行。

### Requirement: Backend selection

`aiw ai` MUST default to the AIW shared Provider route. `--backend gateway` MUST select only the Go Agent Gateway. Providing `--url` without `--backend` MUST select Gateway for compatibility; providing `--backend aiw` with `--url` MUST fail before sending a request. Route configuration and request failures MUST NOT silently switch to the other route.

#### Scenario: Default route and explicit Gateway route

- WHEN a caller runs `aiw ai` without route-specific Gateway options
- THEN AIW resolves `[ai]` and the standard provider environment variables
- WHEN a caller supplies `--backend gateway`
- THEN the client uses Gateway configuration and does not call the AIW Provider
- WHEN the chosen route is missing configuration or returns an error
- THEN the client reports that error and sends no request through the other route

### Requirement: Prompt file input

`--file` MUST read one local prompt file and MUST combine its contents after positional prompt text with a blank line when both are present. Without positional text, the file contents are the prompt. The default encoding MUST be strict UTF-8; `--encoding` MUST specify a supported TextDecoder label. Invalid bytes, unsupported labels, unreadable or oversized files MUST fail before a request. `--encoding` without `--file`, repeated `--file`, and `--file` combined with stdin `-` MUST fail before a request. The file path MUST NOT be sent to the Provider.

#### Scenario: File input with an explicit legacy encoding

- WHEN a caller provides positional instructions, `--file report.txt`, and `--encoding gb18030`
- THEN the file is decoded locally, appended after the instructions with a blank line, and only combined text is sent
- WHEN the selected encoding is unsupported or bytes cannot be decoded
- THEN the command reports an input error and sends no request

#### Scenario: Key rotation

- WHEN 服务端为同一主体配置新 Key 并重启、调用者更换 AIW_AI_API_KEY
- THEN 新请求仍关联同一主体，客户端不需要 Provider 登录材料

#### Scenario: JSON output

- WHEN --json 的结果不是 JSON object
- THEN 客户端报错，不输出伪成功的 JSON
