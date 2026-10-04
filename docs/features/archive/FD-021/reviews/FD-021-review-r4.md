# FD-021 独立真实 Gateway 评审 R4

<!-- aiw-data: FD-021-review-r4.json -->

结论：**当前 Provider 仍不兼容真实 Gateway，需先确定兼容配置设计，再修改实现**。直接官方 SDK 最小子集的成功不能算 Provider 验收通过。本轮没有发现新的 SDK 迁移回归，但既有 Gateway 边界现已由真实请求确认；不标记 Complete。

- Reviewer：`fd021_reviewer`；Tester：`fd021-external-live-tester-r4-20261004-fd021_tester`，独立会话。
- FD Revision 4 / External / Dual；source_event null，未 claim/emit，formal_handoff false。
- 测试 FD digest：`796f11312fbea0d033d668b372bc0ccf5b249d934fae63f983e70fb059fc1cce`。
- Provider digest：`dbac61df7776afece4fd2a9e7e603f8b51e65c53df3d7a8440193454cc674aaf`，与 R3 相同。
- 测试 digest：`e85cbbf182295d5770f49add1be37f05eddef63d16468e03cdb4ba199e859ace`。
- 读取证据：R4 Tester Markdown/JSON、raw、完整 `tests/fd021_gateway_live.mjs`，以及两个关联请求的 Gateway 安全审计字段。

## R4-01 · P1：Gateway 兼容验收尚未满足，需要设计决定

位置：`program/aiw-agent/src/providers.ts:146`。

Provider 最小文本请求仍携带 `max_output_tokens:8192`。真实 Gateway 请求字段为 input/max_output_tokens/model，HTTP 400；Provider 返回 provider_error / 502。当前 Gateway `protocol.go:15-23` 无此字段，`:47` 严格拒绝未知字段，`server.go:121-122` 在 `:141` execute 之前拒绝。reasoning 也不在该子集，但本轮未发送或测试 reasoning。

第二阶段使用官方 SDK 的公开 POST，仅发送 input/model；真实 HTTP 200，response status completed，文本存在并匹配目标标记。此证据证明 SDK 认证/传输与当前 Gateway **支持子集**可工作，不能证明现有 Provider 的请求契约兼容。

该问题是此前实现已有 max_output_tokens 与 Gateway 子集之间的契约缺口，并非本轮新增 SDK 回归。当前 FD 同时要求保留既有参数并使用 Gateway，范围又禁止修改 Gateway；不能靠偷偷删除参数或更改 Gateway 解除冲突。结论为 `changes-required-for-gateway-compatibility`，需人类确认设计边界。

可供选择的最小方向：显式 opt-in Gateway client profile，默认官方模式继续保留 8192 与现有 options；Gateway profile 不发送内部 max_output_tokens 默认，继续遵守 Gateway 256KiB 文本限制及客户端结果限制；reasoning 请求显式返回不支持，不能静默丢弃。字节/字符大小限制不等同 token 上限，这项变化必须写入 FD 与配置契约，经用户决定后再编码。本报告未实现该方案，也未批准变更配置契约。

## 真实证据与一次模型预算

用户明确授权一次最小真实模型请求。脚本仅允许 loopback POST /v1/responses，最多两条 HTTP；只有第一条确为 provider_error / OpenAI returned HTTP 400 且未超时才允许直接 SDK fallback，其他失败均停止。SDK retries:0、redirect:error、日志关闭，外层 65 秒截止与请求 60 秒截止存在，无重试或额外模型请求。

| 阶段 | 请求 ID | HTTP / raw 结果 | 关联 Gateway 审计 |
| --- | --- | --- | --- |
| Provider | resp_9236f447e795032e0a3ab825dfb58b0f | 400，provider_error / 502 | state rejected，http_status 400，execution_started false |
| 直接 SDK 子集 | resp_cee7aa0ae2db7a7e3ab5551cb7cbefb7 | 200，completed，text_present/text_match true | state succeeded，http_status 200，execution_started true |

Reviewer 获主持授权后只读运行配置的 state_dir，并只取上述两个 requests/<id>.json 的 request_id/state/http_status/execution_started/executed 安全字段；未读 content、未输出配置或凭据。两条记录 executed 字段不存在，输出 null；不把其当零。对本次两个关联请求可以确认仅第二条 execution_started，raw 也仅观测一次完成；不是全服务执行总数统计。

返回 usage 为 input 11557、output 7、total 11564、cached input 0，reasoning output 未知，cost null；来自 Gateway 响应而非 Reviewer 重新查询，不推算费用。报告不保存 key、Authorization、prompt 或响应正文。

## 命令与证据限制

Tester 从仓库根仅执行一次 `node tests/fd021_gateway_live.mjs`，工具报告 exit 1、约 5.2978509 秒。当前脚本的 fallback success 分支意图设置 exitCode 2，和工具返回 1 存在未解释差异；如实保留，不把它改写为退出 0，不据此否定 raw 中已记录的两阶段 HTTP 结果，也没有重跑。

R4 Tester 报告把 Provider 失败与直接 SDK 成功分开，摘要与 raw 相符。本轮只覆盖这两个顺序场景，不代表 JSON、reasoning、instructions、SSE、其他模型、限额、超时或服务入口验证，也不是分支覆盖率。R2/R3 历史证据保留，不据此宣告完整 FD 验收通过。

Reviewer 仅执行 Get-Content/rg 静态读取源码、规格、报告、raw，以及获授权的两条本地安全审计字段读取；未执行测试、网络、模型请求、编译、构建、Git 写或业务修改。只新增 R4 Markdown/JSON。

## 下一步与状态

主持已将兼容 profile 设计作为待用户选择项记录；在决定之前保留 Provider/Gateway 行为。真实最小 SDK 支持子集验证缺口已缩小，但现有 Provider 兼容验收失败、设计门禁与正式交接回执缺口仍存在。source_event null，不派发 verification-passed，不关闭、归档或标记 Complete。本轮一次模型预算已使用，不再发起请求。
