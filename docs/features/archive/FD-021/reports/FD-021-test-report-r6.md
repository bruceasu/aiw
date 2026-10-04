# FD-021 当前 Gateway 模式真实验收测试报告 R6

<!-- aiw-data: FD-021-test-report-r6.json -->

**当前 Provider 的 aiw_gateway 模式真实请求通过。** 本轮只发出一次 HTTP POST、只触发一次关联模型执行，7 项断言全部通过。没有直接 SDK fallback、重试或其他模型请求。

## 授权与版本

用户本轮明确提出“那么现在FD-021 可以测试和验收了”，授权当前已运行本地 Gateway 的一次最小 Provider 验收。此为新一轮限定授权，不复用 R4 预算。授权：docs/features/reports/FD-021-test-authorization-r6.md 及同名 JSON，basis 为 human-approved。脚本先核对 exact command、Tester、FD revision/digest、测试摘要、Gateway 二进制摘要及全部可用 Provider 模块摘要，才读取配置中的 Key 并调用。

- External Tester：fd021-external-acceptance-20261004-c813f0；source_event / implementation_event 为 null。未伪造缺失 Tester 或实现事件。
- FD Revision 4；执行时摘要：7a8a9d4ed224b34e653e2d48d4bbb5ca1bf528f900e425579d164cc04e09af98。
- 测试摘要：76838479dfeabd55c4fc875ec26b4826938db351796c4645984d3dcfdfbb0131。
- Provider 摘要：0cafbfec7809bd71cd8a7e3a775e256b60f3374daa63ecb1e2f04a48198691bb；其余 config/errors 摘要见同名 JSON 与 raw。
- Gateway 二进制摘要：97a6753035e944876d942ec48d7ec2de029f0a93e9d7f906f49cbdc49ef378a3；OpenAI SDK 6.49.0。

## 真实结果与关联审计

Endpoint 为 http://127.0.0.1:43127/v1/responses，逻辑模型 gpt-6-luna。请求显式设置 aiw_gateway，只包含 input/model，POST、redirect:error，HTTP 200。结果文本存在且精确匹配预期标记；只记录布尔值，不记录提示或响应正文。状态 completed，Provider 的完整 usage 与安全响应映射逐字段一致，包括合法未知值为 null。

关联 request_id：resp_4bb86213f415c62ead5a4ed6a110bbd7。只读该请求的 HTTP 审计安全字段，确认 state=succeeded、http_status=200、execution_started=true。没有读取 content 或其他审计记录。request_id 使用严格字符白名单；路径须位于配置 state_dir/requests 的真实路径边界内。有限文件等待不产生额外 HTTP 请求。

Gateway 报告 input=11557、output=9、total=11566、cached input=0；service tier、cache-write、reasoning output 未知，均为 null；cost 为 null，不推算费用或实际收费。

## 完整清单、当前执行与历史证据

完整清单 40 项，逐项 data.scenarios 区分实际运行、匹配历史及未验证。**本轮只执行 7 项、通过 7 项、失败 0**，另 33 项未在本轮执行。当前实测覆盖 7/40（17.5%）。

R5 的 8 项离线 profile 场景在可用摘要范围内与当前版本匹配：Provider/config/errors 源文件、已安装 SDK 版本和保留测试脚本摘要都一致。这些可作为历史有效证据，**不是本轮重新执行**。合计有运行证据支持的场景为 15/40（37.5%），其中当前 7、历史 8；其余 25 未在当前源版本取得运行证据。

R2 24/24 和 R3 26/26 的源摘要与当前 Provider 不一致，仅保留历史参考，不计入当前版本覆盖；不把所有旧通过数相加为当前完整通过。历史摘要核对只涵盖已记录的模块、SDK 版本和测试脚本，不声称整个机器环境完全相同。

未验证的当前版本场景包括非法 JSON、官方 instructions、published summary 和摘要限制、headers/成功正文/错误正文 deadline、HTTP/连接错误脱敏与不重试、debug 日志、文本限制、稀疏/非法 usage、URL 安全和默认 endpoint、缺少 Key、legacy output、timer 清理、依赖及扩展文档、其他服务/Provider 的兼容边界。相关旧运行结果不能代替新版本验证；可由独立静态评审补充。

分支覆盖未测量。37.5% 证据覆盖及分支缺口需要 PM 明确静态证据和风险例外，不能据单个真实请求宣告所有 FD 行为均运行通过。

## 实际命令与副作用

仓库根唯一命令 `node tests/fd021_gateway_acceptance.mjs`，exit 0，约 6.20 秒。只独占写 docs/features/reports/FD-021-test-results-r6.json；没有再次执行或额外 HTTP 请求。

配置 Key 只读入内存；不输出配置、认证头、Key、提示、正文或原始错误。usage 对比前由白名单与数值规则净化，raw 仅安全摘要。Provider 原有 60 秒完整响应控制保留，另有 65 秒整体保护；本次约 6 秒完成，没有触发真实超时测试。环境、fetch 包装和 timer 最终恢复/清理。

Gateway 由父会话启动或复用，本 Tester 没有启停进程或改配置，也没有修改生产源码、FD、旧证据、依赖或 Git。没有下载、编译、最终构建、全库测试、覆盖仪器或其他 Provider 请求。Gateway 自身仍按运营配置保存审计、用量及内容，这是实际模型调用的正常服务侧写入，本 Tester 只读取匹配的 HTTP 审计安全字段。

## 建议

建议 **pass** 本轮 External 真实验收测试，交 PM 作完整范围接受与风险例外决定，并创建真实 Reviewer 交接。当前 Gateway 模式的最小文本路径已经得到 Provider 到 Gateway 到模型及审计的真实证据；JSON、reasoning、timeout 和其他路径不可由此类推。本报告不自行标记正式 Complete，不再发起模型请求。
