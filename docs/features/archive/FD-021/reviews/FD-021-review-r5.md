# FD-021 独立配置模式复评 R5

<!-- aiw-data: FD-021-review-r5.json -->

结论：**用户批准的显式 Gateway profile 已正确实现，本轮范围内未发现新的实质缺陷，离线配置模式测试 8/8 通过**。R4 的设计门禁已由用户决定解决；新 Provider profile 未重复真实模型验证，不能宣称真实端到端通过或正式 FD Complete。

- Reviewer：`fd021_reviewer`；Tester：`fd021-external-profile-tester-r5-20261004-fd021_tester`，独立会话。
- Revision 4 / External / Dual，source_event null，formal_handoff false；未派发正式验证通过事件。
- 测试 FD digest：`2c08ae8653f87a30394e431a3dea21ce7c984bb62d9325ac34019b5b39c2d46f`。
- Provider digest：`0cafbfec7809bd71cd8a7e3a775e256b60f3374daa63ecb1e2f04a48198691bb`。
- 测试 digest：`4317e0dec4f06ba9653e2fdcfd58d9a5be7d74e8b450c8cff2d2dc933442c3b0`。

## 实现与设计复核

`program/aiw-agent/src/providers.ts:114-119` 使用显式 OPENAI_API_PROFILE 配置：未设置默认为 openai，未知或空字符串返回 provider_not_configured / 503；aiw_gateway 对 effort 或 summary=true 在构造 SDK 与 fetch 前返回 unsupported_option / 400，不静默丢弃。

`:153` 仅在 openai 模式发送 max_output_tokens:8192，默认模式 reasoning 参数仍保留。aiw_gateway 的文本请求仅 model/input，instructions 与 JSON 继续可用；没有按 URL 自动识别模式。R4 未满足的字段子集冲突通过用户批准的 opt-in 配置解决，不改变 Gateway 实现或服务 HTTP/WebSocket 请求契约。

SDK 公开 typed POST、默认 Bearer 认证、maxRetries:0、logLevel:off、redirect:error、远程 HTTPS / loopback HTTP 校验、独立完整响应 deadline 及 finally timer 清理保留。输出 text 优先顺序、usage details 容错、summary 与错误脱敏映射保留。

客户端 `:23` MAX_RESULT_CHARS 检查仍适用于所有模式，JSON 检查 `:24-26` 仍保留。Gateway 仍有服务端文字大小限制，但字符/字节限制不是 8192 token 上限；README 明确用户选择 Gateway 模式改变 token 控制语义，不作等价保证。

已复核 README 与更新后的 FD Acceptance/TODO：默认官方参数契约、Gateway 显式不支持 reasoning、profile 与 base URL 配置、真实新模式未验证均清楚；R4 “尚未实施”已标记为历史状态，当前批准门禁不再误记为待决策。

## 测试证据

已读取完整 `tests/fd021_gateway_profile.mjs`、Tester Markdown/JSON 和 raw `docs/features/reports/FD-021-test-results-r5.json`。精准 import 装载器保持 provider 比较；source 仅用于内存转译/装载与 digest。真实安装 SDK 6.49.0，进程内 mock fetch，无网络、socket 或模型执行；环境/fetch 最后恢复，结果文件独占创建。

Tester 在仓库根单次运行 `node tests/fd021_gateway_profile.mjs`，退出 0，耗时 0.3423121 秒；8/8 通过，无重试。默认/显式 openai 均保 token 上限与 reasoning，Gateway 文本与 JSON/instructions 仅发支持字段、标准 output/usage 正常，reasoning 与未知/空模式四个拒绝案例均 0 次 fetch。报告、raw、身份与摘要一致。

父会话实际执行 `.\\node_modules\\.bin\\tsc.cmd --noEmit -p tsconfig.json`（program/aiw-agent），退出 0；Reviewer 未重复执行编译或测试。用户明确批准本次配置模式及持续测试评审，未扩大真实模型预算。

## 范围与剩余风险

R5 仅验证这 8 个配置模式案例。R2/R3 未在当前 Provider 版本重跑，历史测试通过数不合并为当前实现的完整运行覆盖；正文截止、日志禁用和边界安全的保留由当前静态追踪补充，不冒称运行通过。分支覆盖率未测量。

R4 仅直接 SDK 最小子集真实成功，默认 Provider 当时 HTTP 400；R5 新 aiw_gateway Provider 未再调用真实 Gateway，已用完的一次模型预算未增加。当前无新增实质代码缺陷，但真实新模式端到端结果仍无直接运行证据。

缺少正式生命周期回执，source_event null，不 emit verification-passed、不归档、不标记正式 Complete。本轮批准的实现、离线测试与独立复评工作已完成；新增 shutdown 工作属于另一个编号 FD，不纳入本报告或 FD-021 变更。

## Reviewer 实际操作

只执行 git diff、rg、Get-Content 定向静态读取当前代码/README/FD、测试和证据；没有测试、网络、模型调用、编译、构建、formatter、lint、vet、Git 写或业务修改。只新增本 R5 中文 Markdown 与同名 JSON。
