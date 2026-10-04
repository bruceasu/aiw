# Agent Proxy CLI client specification

## Purpose

提供单次 ai 命令调用 Go 网关 Responses 支持子集，不改变独立 aiw ask。

## Requirements

### Requirement: Explicit bounded input

客户端 MUST 将多个非选项位置参数以空格组成 Prompt，保留单个参数内容。单独的 `-` 读取完整 stdin，不与其它提示词参数混用；`--` 后参数不解析为选项。无参数时读取一行交互输入。空白/EOF MUST 不请求；非终端且未用 `-` MUST 立即报错。system @ 文件 MUST 在本地以 UTF-8 有界读取，缺失/空/无效/过大在请求前失败且不泄露内容。

交互输入时 Ctrl+E MUST 使用 EDITOR 编辑当前输入，支持多行。EDITOR 为程序及参数，支持带引号的路径，不解释 shell 表达式；Windows 使用真实 exe。编辑器继承终端，保存并正常退出后提交；空内容、启动失败、非零/信号退出、无效 UTF-8、超限和清理失败 MUST 不请求。临时文件 MUST 在请求前清理，Unix 初始文件权限 0600；Windows 使用当前用户临时目录的 ACL。GUI 编辑器等待行为由运营者配置。

### Requirement: Gateway identity and Responses mapping

AIW_AI_API_KEY MUST 必填且至少 43 字符、无空白。AIW_AI_MODEL 为可选逻辑模型，未设置或为空时 MUST 默认 gpt-6-luna；非空设置覆盖默认值。所选模型仍须由网关配置并为当前主体授权。旧 AIW_AI_PROVIDER MUST 明确报已移除，不读取上游凭证。--url MUST 接受 HTTP(S) origin 并追加 /v1/responses，拒绝 URL 凭证、路径、query/fragment；默认 loopback 与 AIW_AGENT_PROXY_PORT 保留。

-h 与 --help MUST 显示用法、选项、输入方式、最少配置、默认模型和地址，以及 Windows/Linux 设置示例；帮助 MUST 无需凭据且不发请求。Key 缺失或格式错误 MUST 提示最低要求和帮助入口，不泄露 Key。

最少配置指引 MUST 使用服务端 `principals[].keys` 与客户端 `AIW_AI_API_KEY` 相同的网关 Key 字符串。`key_hashes` 已废除，网关不计算摘要并拒绝该字段；文档 MUST 不将哈希工具列为当前认证配置步骤。旧摘要若被用作不透明 Key，两端 MUST 使用该摘要字符串，不能发送摘要计算前的原始 Key。

客户端 MUST 发送一次 Bearer 认证的 POST Responses，input 为文本，--system 为独立 instructions，--json 为 text.format.json_object；不直接访问 Provider、不自动重试、不持久保存问题。EDITOR 输入可短暂写入本机临时文件，正常或可捕获失败路径须清理；强制终止及编辑器备份由本机使用者管理。

### Requirement: Honest output

客户端 MUST 从标准 completed Response 的 assistant output_text 读取文本；--json MUST 校验为单个 JSON object，stdout 与诊断分离。--verbose MUST 仅 stderr 显示响应 ID/模型/可得 token，缺失费用为 unknown、推理摘要 unavailable，不能编造金额或展示隐藏推理。客户端请求总期限默认 660 秒，包含完整响应读取，响应有界；`--timeout-seconds` MUST 只接受 1–3660 的十进制整数秒，缺值、重复和非法值 MUST 在请求前以退出码 2 报错。此参数 MUST 不进入 Prompt 或 Responses JSON，不改变服务端执行上限。超时取消后不能恢复该次执行。

#### Scenario: Key rotation

- WHEN 服务端为同一主体配置新 Key 并重启、调用者更换 AIW_AI_API_KEY
- THEN 新请求仍关联同一主体，客户端不需要 Provider 登录材料

#### Scenario: JSON output

- WHEN --json 的结果不是 JSON object
- THEN 客户端报错，不输出伪成功的 JSON
