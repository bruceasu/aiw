# FD-021 客户端 API 配置模式测试报告 r5

<!-- aiw-data: FD-021-test-report-r5.json -->

本轮聚焦离线测试 **8/8 通过**，失败、阻断、未执行均为 0。默认官方模式保持原有请求选项，显式 Gateway 模式只发送支持字段，并在发送前拒绝不支持的 reasoning 选项。

## 对象与授权

用户明确同意客户端 Gateway profile，并持续授权测试、修复和评审。本轮只处理 FD-021，使用 tests/fd021_gateway_profile.mjs，不修改生产实现、旧测试或旧证据。

- FD Revision 4；执行时摘要：2c08ae8653f87a30394e431a3dea21ce7c984bb62d9325ac34019b5b39c2d46f。
- Provider 摘要：0cafbfec7809bd71cd8a7e3a775e256b60f3374daa63ecb1e2f04a48198691bb。
- 测试摘要：4317e0dec4f06ba9653e2fdcfd58d9a5be7d74e8b450c8cff2d2dc933442c3b0。
- Tester 会话：fd021-external-profile-tester-r5-20261004-fd021_tester。External 直接用户授权；source_event / implementation_event 为 null，未伪造正式派发、验收或完成事件。

## 场景与结果

1. 不设置 OPENAI_API_PROFILE：保留 max_output_tokens:8192、reasoning effort 和 summary 请求选项。
2. 显式 openai：同样保留上述选项。
3. aiw_gateway 最小文本：仅发送 input/model，省略固定 token 上限与 reasoning；标准 output 与 usage 正常映射，缺失 cached details 为 null。
4. aiw_gateway instructions 与 JSON：仅发送 input/instructions/model/text；instructions、JSON 格式和结果保持。
5. aiw_gateway effort：unsupported_option / 400，0 次 fetch。
6. aiw_gateway summary=true：unsupported_option / 400，0 次 fetch。
7. 未知 profile：provider_not_configured / 503，0 次 fetch。
8. 显式空 profile：同样配置错误，0 次 fetch。

前四场景各一次进程内 mock fetch，同时验证 Responses 端点、POST 和 redirect:error。后四场景没有发出请求。全部采用已安装 OpenAI SDK 6.49.0 和内存 TypeScript 转译装载，无网络、socket、真实 API/Gateway 或模型执行。

## 实际命令与覆盖

仓库根仅执行一次 `node tests/fd021_gateway_profile.mjs`，exit 0，耗时约 0.34 秒，无重试。独占写入 docs/features/reports/FD-021-test-results-r5.json，环境变量与 fetch 最后恢复；无配置、服务、生产代码、依赖、构建或 Git 写操作。

本轮适用、覆盖、执行、通过场景均为 8/8。此统计只描述 profile 枚举场景，不是全部 FD 验收覆盖率或代码覆盖率；分支覆盖率未测量。生产编译与静态安全控制复核由父会话和独立 Reviewer 分别记录，不冒充本 Tester 的运行结果。

## 未验证与残余风险

已用完的真实模型预算没有增加。本轮新 Gateway profile 没有再执行真实 Gateway/API 请求；R4 直接 SDK 最小子集成功是历史证据，不能转述为当前 Provider 模式已实测成功。

未重跑 R2/R3 整套用例；其完整正文 deadline、日志脱敏及边缘错误场景属于各自实现版本的历史运行证据，本轮仅对 profile 请求和错误分支运行验证。未运行全库测试、真实网络、服务 HTTP/WebSocket/audit/store、其他 Provider、build、lint/formatter/vet、覆盖仪器或验证脚本。

Gateway 模式省略固定 max_output_tokens，远端输出预算由 Gateway 管理；现有客户端结果大小约束的当前静态保留需独立 Reviewer 复核。本报告建议通过本轮配置模式测试并交给独立 Reviewer；正式生命周期回执缺口仍由 PM 处理，不宣告 Complete。后续 shutdown 工作属于另一个 FD，不纳入本报告。
