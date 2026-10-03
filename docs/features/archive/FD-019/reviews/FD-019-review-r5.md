# FD-019 独立静态审查（修订 5）

<!-- aiw-data: FD-019-review-r5.json -->

结论：请求修改，存在 1 项阻塞发现；不能通过 Verification 或归档为 Complete。

Reviewer：`codex-reviewer-FD019-20261004-r5`。来源事件：
`FD-019-000005-review-requested`，领取前收据为 pending，已由本 Reviewer 正式领取。
审查提交：`c9b6d51cd2308e1ffbaf51755d697f6acda3ad83`；差异基线：该提交的父提交。
该提交同时包含其他 FD 内容，本报告只评估 FD-019 的客户端、编辑器、说明、哈希 BAT 与相关规格。
无历史 Worker 报告，依据用户授权的状态恢复记录和真实源码/提交审查，不补造 Worker 完成事件。

## R1：最少配置指导使用网关已拒绝的 key_hashes（高优先级）

`plugins/aiw-ai/aiw-ai.mjs:20` 指导服务端保存 `principals.key_hashes`；
`plugins/aiw-ai/README.md:8`、`:21` 要求保存 SHA-256 哈希并让客户端发送原始 Key。
当前 `openspec/specs/agent-proxy/spec.md` 的 Authenticated OpenAI text subset 要求
`principals[].keys` 保存实际 Bearer 字符串，直接比较、不计算摘要，且明确拒绝任何 `key_hashes` 字段。
当前网关源码 `program/agent-gateway/config.go:25` 定义 keys，`:85` 要求 keys 非空，`:106` 直接常量时间比较请求 Key 与配置字符串，与该契约一致。
因此依帮助/README 设置的配置会被当前网关拒绝加载；仅把字段更名为 `keys`，但服务端保存哈希、
客户端发送原始 Key，也会因两个字符串不同而认证失败。

这违反工作项 1.2 的最少配置指引、1.8 的帮助/README/稳定 spec 对齐，以及 FD Acceptance 的现有请求契约兼容要求。
Worker 应统一指导服务端 `keys` 与客户端 `AIW_AI_API_KEY` 使用相同字符串，并说明旧摘要只有作为
双方一致的不透明 Key 才可使用；哈希 BAT 不得再被描述为当前网关配置的必要步骤。
保留八项既有编号和历史勾选，TODO/Verification 记录 1.2、1.8 的当前阻塞，由正式交接请求 Worker 后续修复；本次不修改认证实现、帮助或 README，不改变当前网关认证契约。

## 工作项与静态证据

| 工作项 | 结论 | 证据与限制 |
| --- | --- | --- |
| 1.1 | 静态支持 | DEFAULT_MODEL 与 config 的非空环境变量覆盖；缺 Key 错误不包含凭据并指向帮助。 |
| 1.2 | 请求修改 | -h/--help 共用 HELP，main 在读取 stdin/config/fetch 前返回；最少认证配置有 R1。help 路由源码向插件传 -h。 |
| 1.3 | 模型部分静态支持 | README、客户端 spec 与源码均默认 gpt-6-luna，并要求 models/allowed_models 映射；认证文档问题并入 R1。 |
| 1.4 | 脚本静态支持 | Read-Host -AsSecureString、UTF8.GetBytes、SHA-256 与小写十六进制；无文件配置写入。用途已过时见 R1；未执行 BAT。 |
| 1.5 | 静态支持 | parseArgs 分离选项和值、空格拼接、多提示词含 - 时拒绝混用；-- 后按输入处理。 |
| 1.6 | 静态支持 | commandWords 解析引号；spawn shell:false；有界读取、fatal UTF-8 解码与 finally rm，清理失败阻止提交。 |
| 1.7 | 静态支持，运行未验 | 取 rl.line、close/off keypress/pause 后编辑，成功结果返回 main；readline SIGINT 关闭取消。真实终端、编辑器与信号行为未执行。 |
| 1.8 | 请求修改 | 参数和编辑器文档静态一致，认证指引未对齐当前 gateway spec；本机同步仅有历史记录，本轮未核实安装副本。 |

请求仍为一次 Bearer POST /v1/responses，input/instructions/text.format 分离；stdout 输出结果、stderr 输出诊断，保留 URL 约束与超时。
本轮未更改实现；其他未提交改动均保留。

## 命令和证据边界

实际运行：定向 `Get-Content -Encoding UTF8` / `rg` / `rg --files` 读取规则、FD、事件、源码、规格；
`git status --short`、`git log -4 --oneline -- plugins/aiw-ai scripts/hash-gateway-key.bat`、
`git rev-parse HEAD`、`git show --stat c9b6d51 -- plugins/aiw-ai scripts/hash-gateway-key.bat openspec/specs/agent-proxy-client/spec.md`、
`git diff c9b6d51^ c9b6d51 -- plugins/aiw-ai/aiw-ai.mjs plugins/aiw-ai/editor.mjs plugins/aiw-ai/README.md scripts/hash-gateway-key.bat`；
`aiw fd claim --help`、`aiw fd emit --help`；
`aiw fd claim FD-019 FD-019-000005-review-requested --session codex-reviewer-FD019-20261004-r5` 成功。
初次读取猜测的 receipts 路径不存在，随后改读真实 events 路径；默认编码显示乱码的中文文件改为 UTF-8 读取。
编辑后执行一次静态读取批次：定向 git diff FD、读取报告 JSON/Markdown、rg 网关 config。
rg 的 `program/agent-gateway/*.go` 参数在 Windows 下无效，显式 config.go 参数仍返回 keys 和直接比较证据；未重试该命令。

未执行测试、编译、CLI 客户端、网关、外部编辑器、哈希 BAT、网络/模型请求、构建、依赖下载、Git 写操作。
Test policy 为 External，测试未运行不作为本次独立审查失败原因；R1 是当前行为要求不满足。
FD 中既有 node --check 成功、本机文件同步、认证核对均为历史记录，不视为本轮结果。
修复后实际跨平台终端接管、编辑器等待、Ctrl+C、认证及模型可用性仍由外部验证；强制终止和编辑器备份可能留下临时 Prompt，现有 FD 已记录风险。

