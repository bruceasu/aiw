# FD-021 外部聚焦边缘测试报告 r3

<!-- aiw-data: FD-021-test-report-r3.json -->

修复后本轮边缘测试 **通过**：26/26 个场景有效执行并通过，失败、阻断、未执行均为 0。首次运行 25/26 通过，发现一项历史兼容性回归；修复后同一命令唯一重试通过。初次与重试原始文件独立保存，不覆盖 R1/R2 历史证据。

## 对象、授权与版本

用户本轮明确要求继续测试、修复和评审；测试范围限定在既有 OpenAI Provider 边缘契约。使用独立 `tests/fd021_sdk_edges.mjs`，原 R2 `fd021_sdk_blackbox.mjs` 未修改。Tester 会话为 `fd021-external-tester-r3-20261004-fd021_tester`；External 直接用户授权，source_event / implementation_event 均为 null，未伪造正式交接回执或完成事件。

- 初测：FD Revision 4，摘要 `e9eeaca26a85843be4809fb23b39e9e300b68d5b3d5a825753c81b1fee464470`；Provider 摘要 `adcd853cf15b82db74caca48ad1ea76f03bb1a820eceec3a21fd1a2e88f45f35`；测试摘要 `d8820caeefb673594fec6f44251bc6304e399135862cd53ef867abbfef12f0ca`。
- 重试：FD Revision 4，摘要 `178e04dec0b1866820fc670d8a384c1b1048832259d0d3758b2469b00fbbbaff`；Provider 摘要 `dbac61df7776afece4fd2a9e7e603f8b51e65c53df3d7a8440193454cc674aaf`；测试摘要 `8ee746ef2a6c5aba48e961119fc1679b85fe88e83179208a1564cdbd667c0bb2`。
- 两次测试脚本摘要不同：重试前仅在既有 legacy 场景补充 POST 和合成 Bearer 认证断言，经父会话检查后执行。不能将初版和修复版摘要混用。

## 首次发现与修复后结果

首次唯一失败是历史已接受形态 `object: "response", output_text: "legacy answer", output: []`，报“OpenAI returned no text”。这项要求来自 SDK 迁移前已有的顶层文本兼容行为，不是新增标准 OpenAI wire 格式要求，也不是 Gateway 支持承诺。

Worker 修复后，重试通过同一场景的文本、usage、POST 和合成 Bearer 认证断言。Tester 不读取 Provider 实现设计断言，不修改生产代码；具体生产修复由独立 Reviewer 核对。

其余场景两轮均通过：

- 空输出与无文本输出的错误分类和 usage 保留。
- reasoning summary 精确 16Ki 字符允许、超界 `result_too_large / 502` 且保 usage；未请求时忽略超长 summary 且不返回字段。
- 负数、小数、字符串及不安全整数计数归 null；非法、过长和非字符串 identifier 归 null；128 字符合法边界保留。
- HTTP 200/500 非 JSON 正文失败脱敏、单次尝试；URL userinfo/query/fragment 在发送前拒绝。
- `redirect: error` fetch 配置与拒绝后的脱敏、无重试；默认官方 Responses URL 的本进程 mock 请求。
- 成功、HTTP 错误、JSON 解析错误三出口计时器清理及完成后不再 abort。
- usage 为 false/null 时正常文本仍返回，token 未知为 null。

## 命令、覆盖与副作用

仓库根同一聚焦命令 `node tests/fd021_sdk_edges.mjs`：

1. 首次执行 exit 1，约 0.46 秒，25 通过、1 失败，写出 `docs/features/reports/FD-021-test-results-r3-initial.json`。
2. 相关业务修复后获确认的唯一重试 exit 0，约 0.43 秒，26 通过、0 失败，写出 `docs/features/reports/FD-021-test-results-r3-retry.json`。

本轮适用、覆盖、执行场景为 26/26，最终通过率 26/26。这些只代表本轮枚举边缘场景，不是全部 FD 验收覆盖率，也不是代码分支覆盖率。未使用覆盖仪器，分支覆盖率未测量。

两轮都使用已安装 OpenAI SDK 6.49.0、内存 TypeScript 转译装载和进程内 fetch mock；无 socket、网络、下载、真实 API/Gateway、子进程或最终构建。仅写上述原始 JSON；文件使用独占创建，防止覆盖历史。环境、fetch、计时器拦截恢复，逐场景清理尚未清理的 60 秒 timer，避免残留等待。cleanup 场景把配置 60 秒缩短为 5ms，再用原始 15ms timer 观察 signal；未真实等待 60 秒。

## 未验证与残余风险

R3 修复后没有重新执行 R2 请求字段、完整 body 截止和 debug 日志场景；R2 24/24 是先前 Provider 版本的历史证据，不能转述为当前修复版已经重跑通过。真实 Gateway/API、HTTP/WebSocket、audit/store、其他 Provider、全库测试、最终 build、lint/formatter/vet 和验证脚本均未运行。本轮 URL/redirect 用例验证传入配置和拒绝分支，不模拟真实跨主机重定向。

当前 Go Gateway 支持子集与此 Provider 仍有参数边界，不能据本轮 mock 通过声明只改 baseURL 即兼容；本轮未修改 Gateway 或扩大其支持范围。正式生命周期缺失回执仍由 PM 处理；本报告仅建议独立 Reviewer 复核本轮修复和证据，不标记正式 Complete。
