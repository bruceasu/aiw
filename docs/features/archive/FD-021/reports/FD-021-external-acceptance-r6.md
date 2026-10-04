# FD-021 External 测试接受决策 R6

<!-- aiw-data: FD-021-external-acceptance-r6.json -->

接受 R6 当前 aiw_gateway Provider 最小真实调用验收，交正式独立 Reviewer。核对测试报告、原始结果、版本绑定授权：单次 HTTP POST 200，7 项断言通过，关联审计 succeeded/execution_started=true；无回退或重试。当前 Provider/config/errors、SDK 6.49.0 及保留测试摘要与 R5 一致，R5 的 8 项离线场景可作匹配历史证据，不计成本轮执行。R2/R3 摘要不一致，仅保留历史，不能合计旧通过数。

完整 40 场景中本轮执行 7 项（17.5%）；加匹配历史 R5 的 8 项后有运行证据支持 15 项（37.5%），25 项无当前版本运行证据；本轮未执行 33 项，分支未测量。明确批准低于 70% 及分支未测量的验证例外，剩余要求由独立静态评审核实。该例外不免除行为要求，也不把静态证据或未运行项标为运行通过。静态依据为 SDK typed POST、零重试、关闭日志、URL 限制、完整正文 AbortController deadline/finally 清理、nullable usage/结果限制和服务边界保留；Reviewer 必须复核，实质问题仍退回。

External 生命周期处理：旧 FD 无实现和 Tester 交接事件。本次不写假回执、不把直接报告称为正式 Tester 事件。PM 基于真实外部结果记录接受，将 Markdown authored 状态从 Pending Test 转入 Pending Verification，再调用 request-review 创建首个真实 Reviewer 交接。旧缺失交接的历史保持可见；尚未宣告 Complete。Work Item 1.14 有实际证据，1.15 的独立评审部分待 Reviewer 完成。

此次只新增测试、报告和验收文档，没有改生产代码。沿用用户已有启动授权运行新版 Gateway（当前 loopback 43127），保持运行；没有模型调用之外的外部请求、下载、最终构建、Git 或归档。服务 HTTP/WebSocket、其他 Provider、真实错误/超时和 JSON 路径没有在本轮运行；R6 成功不能推广到这些路径。
