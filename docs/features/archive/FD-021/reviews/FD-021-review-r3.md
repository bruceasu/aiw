# FD-021 独立复评 R3

<!-- aiw-data: FD-021-review-r3.json -->

结论：**本轮历史 output_text 兼容回归已修复，修复范围内未发现新的实质缺陷**。独立边界测试修复后 26/26 通过；真实 Gateway 兼容验收与正式生命周期仍未完成，不能标记 FD Complete。

- Reviewer：`fd021_reviewer`；Tester：`fd021-external-tester-r3-20261004-fd021_tester`，会话独立。
- 评审基线：`c9b6d51cd2308e1ffbaf51755d697f6acda3ad83` 至当前工作区差异，重点为 R3 公开 SDK POST 决策、输出映射及边界证据。
- Revision 4，External / Dual，source_event null；本次为直接评审，未 claim/emit、未伪造回执。
- 重试 FD digest：`178e04dec0b1866820fc670d8a384c1b1048832259d0d3758b2469b00fbbbaff`。
- Provider digest：`dbac61df7776afece4fd2a9e7e603f8b51e65c53df3d7a8440193454cc674aaf`。
- 测试 digest：`8ee746ef2a6c5aba48e961119fc1679b85fe88e83179208a1564cdbd667c0bb2`。

## 发现与修复复核

R3 首次测试确认 P2 兼容回归：既有成功响应 `object: response, output_text: legacy answer, output: []` 经 generated Responses.create 在 SDK 内覆写文本为为空，导致 provider_error。旧手工实现接受该形态；此要求是历史兼容性，不是新增标准 wire 格式或 Gateway 支持承诺。

当前 `program/aiw-agent/src/providers.ts:142` 改用官方 SDK **公开** `client.post<OpenAI.Responses.Response>("/responses", ...)`，body 由 `ResponseCreateParamsNonStreaming` 约束。避开 generated create 的 addOutputText 覆写，并保留先取类型为 string 的顶层 output_text、否则调用既有 extractOutput 的原顺序；空字符串仍按原逻辑报无文本。标准 output 提取、usage 与 summary 逻辑仍在。没有修改 SDK、手写 fetch 或重新实现 JSON 解码。

本地 SDK 静态链路：`client.d.ts:224` 声明公开 post；`client.mjs:348-349` 转入 methodRequest；默认安全方案在 `:395,662,668` 为 bearerAuth:true，`:269` 生成 Bearer API Key。不依赖内部 __security 配置；显式字符串 API Key 路径没有 workload identity 刷新重试。maxRetries:0 继续关闭重试，`:460-473` 仅在 retriesRemaining 非零时重试，SDK APIError 映射不变。

完整调用独立 timer/signal、abort 优先 timeout、finally 清理、nested usage 可选访问和 logLevel:off 保留；README 与 FD 明确解释 SDK 公开方法决策及 Gateway 请求字段边界。服务 HTTP/WebSocket、其他 Provider、依赖版本、Gateway 实现无本轮修改。

## 独立测试证据核对

已读取 `tests/fd021_sdk_edges.mjs`、R3 Tester Markdown/JSON 与 `FD-021-test-results-r3-retry.json`。精准 import 装载器未污染 provider 比较，源码读取仅用于转译/装载和摘要。真实已安装 SDK 6.49.0 配合进程内 fetch mock，无 socket/网络。

Tester 在仓库根执行 `node tests/fd021_sdk_edges.mjs` 两次：初测退出 1、25/26，通过修复后的唯一重试退出 0、26/26，分别约 0.4574613 秒与 0.4310404 秒。重试测试脚本仅在既有 legacy 场景增加 POST 与合成 Bearer header 断言；已通过，证明公开 post 的默认认证与文本/usage 保留。授权来自用户本轮明确要求继续测试修复评审，没有虚构 Planner 授权。

边界运行还确认空/无文本输出错误保 usage、summary 16Ki 边界、未请求摘要忽略、无效 token/identifier 归 null、合法 identifier 边界、非 JSON 正文脱敏、URL userinfo/query/fragment 拒绝、redirect:error 配置、默认 API URL、usage false/null 容错，以及成功/HTTP/JSON 错误后的 timer 清理。timer 清理用例将 60000ms 缩至 5ms，检查 pending 为 0，再等真实 15ms 观察 signal 未 abort；不声称实测真实 60 秒网络等待。

两次 raw 独立保存，没有覆盖历史。R2 24/24 属于前一个 Provider digest，**本轮没有重跑 R2，不合并为当前实现 50/50 通过**。R2 主字段、完整 body 截止和 debug 日志在当前版本只有历史运行证据加本轮静态保留追踪，不能声明当前已重新执行。场景通过率不是代码分支或全部 FD 验收覆盖率。

实现记录 `docs/features/reports/FD-021-implementation-r3.md` 与主持实际结果确认 `.\\node_modules\\.bin\\tsc.cmd --noEmit -p tsconfig.json`（program/aiw-agent）退出 0；Reviewer 未重复执行。

## 剩余风险与正式状态

当前 Go Gateway 仍严格拒绝 TS Provider 历史已有 max_output_tokens 和 reasoning；不能仅切换 base URL 获得兼容，mock 通过不能满足真实 Gateway 运行验收。该限制已明确写入 README 和 FD，不列为本轮新增回归。稳定 agent-proxy 规格已转为 Go Gateway，不把历史 TypeScript 接口当成当前规范。

真实 API/Gateway、网络慢流、服务入口、audit/store、其他 Provider、分支覆盖率与真实跨主机重定向未验证。公开 SDK POST 与 typed Responses body 在当前安装版本经编译与边界调用验证；未来 SDK升级仍需重新检查该公开方法契约。

缺少正式实现交接回执、source_event null，本报告仅修复范围内通过，不 emit verification-passed，不关闭/归档、不标记 Complete。

## Reviewer 实际操作

执行 rg、Get-Content 定向静态读取公开 SDK 声明/实现、FD、README、Provider diff、实现记录、edge 测试及其证据；执行 `git diff -- program/aiw-agent/src/providers.ts program/aiw-agent/README.md docs/features/FD-021_AIW_AGENT_OPENAI_SDK.md`。一次猜测的 edge 文件路径不存在，后按明确路径读取；Tester 报告尚未落盘时读取缺失，落盘后核对。均未归因产品缺陷。

未运行任何测试、编译、构建、formatter、lint、vet、网络或 Git 写操作；只新增本 R3 中文 Markdown 与 JSON 证据。
