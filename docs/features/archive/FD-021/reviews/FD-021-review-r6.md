# FD-021 正式独立验收评审 R6

<!-- aiw-data: FD-021-review-r6.json -->

结论：**验证通过，未发现实质问题**。当前 aiw_gateway Provider 最小真实请求已成功，R4 的“仅直接 SDK 成功、Provider 未通过”缺口已由 R6 当前 Provider 证据解决。通过依据为真实 R6、匹配 R5 历史、当前静态证据与 PM 明确例外，不代表全量运行覆盖。

- Reviewer：`fd021-reviewer-r6-20261004-independent`，区别于本次 External Tester `fd021-external-acceptance-20261004-c813f0` 和实现/PM 会话。
- source_event：`FD-021-000005-review-requested`，来源 Revision 5、摘要 `2b0a54ae4a03c89aa73d5f26f6b8d9a8e43a3ecd08c1cc3069ff3ac017e9ea59`；先检查 pending 再 claim。
- Test policy 保持 External。PM authored 接受后转 Pending Verification，再创建当前真实 request-review；未补造旧 Worker/Tester 事件。历史 R1–R5 的直接评审结论仍保留，不能改写为过去正式派发。
- 差异基线 `c9b6d51cd2308e1ffbaf51755d697f6acda3ad83`；当前四项 aiw-agent 差异为既有实现。本轮生产源码未改变，重点为 R6 新真实验收及当前证据绑定；FD-025 shutdown 不纳入实现范围。
- 发现：无。

## 当前真实验收核对

读取 `FD-021-test-report-r6.md/json`、`FD-021-test-results-r6.json`、完整 `tests/fd021_gateway_acceptance.mjs` 及 R6 版本绑定授权。用户本轮明确要求测试和验收，是新增一次最小真实 Provider 模型预算；没有复用耗尽的 R4 预算。

脚本在配置 Key 读取与网络前核对 exact command、Tester、FD revision/digest、测试摘要、Provider/config/errors 和安装 Gateway 摘要。网络包装硬限制一次 loopback POST /v1/responses、input/model、gpt-6-luna、redirect:error，无回退。配置 Key 仅内存；只输出安全布尔值/usage/ID。关联 HTTP 审计只读一个严格 ID，realpath 边界限制于 state_dir/requests，未读 content。最后恢复环境/fetch、清理 timer；raw 独占创建，历史不覆盖。

Tester 实际一次运行 `node tests/fd021_gateway_acceptance.mjs` 退出 0，6.2033522 秒；一个 HTTP POST 200，7/7 断言通过，text_present/text_match true、completed、usage 与安全 wire 映射一致。请求 ID `resp_4bb86213f415c62ead5a4ed6a110bbd7` 的关联审计 succeeded、HTTP200、execution_started=true。与 R4 不同，本次使用生产 complete 和 aiw_gateway profile，不是直接 SDK fallback。

返回 token input=11557、output=9、total=11566、cached=0；其他未知细项 null，cost null，不推算费用。7 个断言来自同一个请求的不同可观察结果，不是7次模型调用。

## 历史和当前版本绑定

Reviewer 用 Get-FileHash 独立确认当前 Provider/config/errors、R5 profile 测试、R6 acceptance 测试和安装 Gateway exe 摘要，与 R6 raw/授权一致；SDK package 6.49.0 与 lockfile一致。

- 当前 Provider：`0cafbfec7809bd71cd8a7e3a775e256b60f3374daa63ecb1e2f04a48198691bb`。
- R6 测试：`76838479dfeabd55c4fc875ec26b4826938db351796c4645984d3dcfdfbb0131`。
- 安装 Gateway：`97a6753035e944876d942ec48d7ec2de029f0a93e9d7f906f49cbdc49ef378a3`。
- R5 profile 测试：`4317e0dec4f06ba9653e2fdcfd58d9a5be7d74e8b450c8cff2d2dc933442c3b0`。

R5 的8项在可用源文件/SDK版本/测试摘要范围内匹配，可用作当前实现历史证据；不是本轮重新执行，也不保证整个机器环境完全相同。R2/R3 的 Provider 摘要不匹配，旧24/26通过仅历史参考，不计当前覆盖。保留旧 Worker R3记录和 R5独立评审的范围、编译与接口决策，不重跑。

## 当前静态验收依据

| 契约 | 当前检查 |
| --- | --- |
| 官方 SDK Responses | providers 的公开 typed POST /responses 与 ResponseCreateParamsNonStreaming 类型约束，保 SDK认证、传输、JSON解码、APIError；未使用手写 fetch。已在同一SDK版本的此前静态评审确认公开POST默认Bearer。 |
| 兼容配置 | openai 缺省保8192及reasoning；Gateway显式省略固定上限，reasoning在发送前400，unknown/empty503；不按URL自动选模式。README与FD批准范围一致。 |
| 安全URL/错误 | 远程HTTPS、loopback HTTP，拒绝userinfo/query/fragment和非法协议；redirect:error；APIError仅HTTP状态，其他错误固定分类，不输出上游body或Key。 |
| deadline/重试/日志 | maxRetries:0、logLevel:off；独立60秒controller覆盖SDK POST完整await与正文；abort优先timeout，finally清timer。SDK自身headers计时不替代本地全文截止。 |
| 输出/usage/summary | 顶层output_text优先、标准output fallback；无文本错误保usage；tokens与identifier净化、nested details可选访问；外层JSON与MAX_RESULT_CHARS，summary16Ki检查保留。 |
| 依赖和服务边界 | manifest/lockfile保ws、@types/ws和TypeScript现有用途，SDK6.49.0；HTTP/WebSocket、Codex/Copilot及存储审计接口无本轮修改。README说明扩展接缝与当前无知识库。 |

以上未运行行为保持静态证据，不升格为运行通过；默认官方端点没有在本轮真实请求。

## 覆盖例外及剩余风险

完整清单40项：本轮执行7、通过7、失败0，当前实测7/40（17.5%）；匹配R5历史8项后运行证据支持15/40（37.5%）。本轮未执行33项，其中8项有匹配历史，另25项无当前版本运行证据；分支未测量。

`FD-021-external-acceptance-r6.md/json` 明确接受低于70%及branch未测例外，要求独立静态核实，不免除行为要求。尊重该当前决策，不把历史/静态混入本轮执行数，也不重新要求已明确豁免的更广测试。

真实错误/超时/JSON/instructions等路径、其他服务入口/Provider、默认OpenAI实网、SDK未来升级均不能由这一次成功外推；R6只证明当前Gateway最小文本链路。历史npm告警未作新网络核实。本轮无新增生产代码，无需重复编译或扩大验证。

## 操作与正式交接

Reviewer 实际执行 Get-Content/rg 定向读取、Get-FileHash、scoped git diff --stat、git rev-parse HEAD，以及 claim 当前精确事件；没有测试、网络、模型请求、编译、构建、Git写、归档或部署。写本报告/JSON并更新FD 1.15、TODO、Verification；基于实际证据 emit verification-passed，历史缺失事件不补造。
