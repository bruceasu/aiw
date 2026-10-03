# FD-021 真实 Gateway 最小请求报告 r4

<!-- aiw-data: FD-021-test-report-r4.json -->

**官方 SDK 到真实 Gateway 的最小文本请求通过；现有 Provider 请求不兼容。** 本轮只观测到一次实际模型完成，未重试。正式 FD 完成与完整 Provider/Gateway 兼容性不能据此宣告通过。

## 授权与测试对象

用户已启动真实 Gateway，并明确批准“允许一次最小真实模型请求”。父会话检查了请求限制、凭据保护和 Gateway 的前置校验：HTTP 400 参数拒绝发生在 Runner 执行之前，因此仅该明确分支允许随后发送一个最小 SDK 请求；其他失败、超时、认证失败或限额拒绝都停止。

- Endpoint：`http://127.0.0.1:43127/v1/responses`；逻辑模型 `gpt-6-luna`；OpenAI SDK 6.49.0。
- FD Revision 4；执行时摘要 `796f11312fbea0d033d668b372bc0ccf5b249d934fae63f983e70fb059fc1cce`。
- Provider 摘要 `dbac61df7776afece4fd2a9e7e603f8b51e65c53df3d7a8440193454cc674aaf`。
- 测试摘要 `e85cbbf182295d5770f49add1be37f05eddef63d16468e03cdb4ba199e859ace`。
- Tester 会话：`fd021-external-live-tester-r4-20261004-fd021_tester`。External 直接用户授权；source_event / implementation_event 均为 null，未伪造正式派发或验收事件。

脚本只在运行时把已启用 team-a 的第一个配置 Key 读入内存，没有打印或落盘凭据、Authorization、请求正文、响应正文、原始错误消息或配置内容。原始结果只保存安全摘要。

## 实际请求结果

1. **Provider：失败。** POST 请求字段名为 `input, max_output_tokens, model`，Gateway 返回 HTTP 400；Provider 将其映射为 `provider_error / 502`。这是预先静态确认的 Runner 前拒绝分支，没有因此消耗一次实际模型执行。请求 ID：`resp_9236f447e795032e0a3ab825dfb58b0f`。
2. **直接官方 SDK 最小子集：通过。** 唯一许可 fallback 使用 POST，字段名仅 `input, model`，返回 HTTP 200 和 `completed`。文本存在，且精确匹配所要求的标记；报告只记录匹配布尔值，不保存文本。请求 ID：`resp_cee7aa0ae2db7a7e3ab5551cb7cbefb7`。

第二次请求由 SDK 传输发送，没有模拟 fetch。返回 token 用量：input 11557、output 7、total 11564、cached input 0；reasoning output 未提供，记录 null；cost 为 null。用量来自 Gateway 的响应，未据此推算费用。

两条请求的字段差异与先前静态记录的 Gateway 支持子集边界一致。最小 SDK 请求成功不等于 Provider 仅改 baseURL 即可兼容；本轮没有修改 Provider、Gateway 或其配置，也没有新增支持承诺。

## 命令与预算

在仓库根仅执行一次 `node tests/fd021_gateway_live.mjs`，工具报告退出码 1，耗时约 5.30 秒。非零退出体现 Provider 路径未通过；直接 SDK 子集成功单独记录。没有任何再次请求或命令重试。

脚本末尾对“Provider 失败、直接 SDK 成功”分支设置预期非零退出码 2，但执行工具实际返回 1；本报告按已返回执行结果记录 1。该差异未进一步探查，不为核对退出码追加任何模型请求或命令。

网络目标硬限制为 `127.0.0.1:43127/v1/responses` 的 POST，最多两次 HTTP 请求，最多一次实际模型执行；无 GET、其他主机或路径、自动重试。SDK 日志关闭，重定向拒绝，正文读取受 60 秒 abort timer 保护，整体安全截止为 65 秒。此请求约 5 秒完成，没有测试真实超时分支。

唯一写入文件为独占创建的 `docs/features/reports/FD-021-test-results-r4.json`；没有修改配置、生产源码、已启动的服务、依赖或构建产物，没有启停进程或 Git 写操作。环境变量、fetch 包装和 timer 在 finally 中恢复/清理。

## 覆盖与残余风险

本轮执行两个顺序场景：Provider 兼容性失败、直接 SDK 最小子集通过；没有代码覆盖仪器，不把这两项视作完整 FD 验收覆盖率或分支覆盖率。真实模型完成计数为 1，实际 POST 计数为 2，第一次为前置拒绝。

未验证 JSON、reasoning、system instructions、SSE、其他模型、请求限额边界、真实超时、服务 HTTP/WebSocket/audit/store、其他 Provider 或更广范围兼容性。R2/R3 离线证据仍属于各自版本和范围。完整 Provider/Gateway 兼容性仍受参数子集限制；正式生命周期仍存在回执缺口。本轮不再发起任何模型请求。
