# FD-021 外部聚焦测试报告 r1

<!-- aiw-data: FD-021-test-report-r1.json -->

本次建议：**changes-requested**。原始运行记录为 22 次尝试、18 个通过、4 个失败；复核后有效执行为 20 个、通过 18 个、业务失败 2 个，另有 2 个被测试装载器阻断。

- FD：Revision 4；测试时 SHA-256：`ab621e640cfaea752e0366be49af603040ba6d267c74091fda1d38277818d344`。
- Tester 会话：`fd021-external-tester-20261004-fd021_tester`。Test policy 为 External。本次来自用户明确要求“test and review the FD-021”，不是正式事件派发。
- source_event / implementation_event 均缺失，记录为 null；未伪造事件、回执或实现报告。
- 真实已安装 OpenAI SDK 6.49.0；所有 fetch 为本进程 mock Response，无网络、socket、Gateway 启动、依赖下载或最终构建。
- Provider 源文本仅用于 TypeScript 内存转译、模块装载和摘要，没有读取实现设计断言；断言来自 FD、README、公共 types/errors/config 和既有公开错误契约。

## 失败与限制

1. **测试装载器阻断，不是业务发现**：初版装载器替换所有 `"openai"` 字面量，连生产代码 provider 比较也被污染。带系统提示词和 debug canary 两个场景因而报“system_prompt is supported only by OpenAI”。模型、instructions、effort、summary 请求字段、完整 usage 映射及 debug 日志 canary 未有效执行；不能据此声称系统提示词存在业务回归或日志泄露已复现。装载器已改为只替换 import specifier，未再执行第三次。原始 JSON 保留。
2. **nullable usage details 契约回归**：供应 input/output/total tokens，但省略 input_tokens_details/output_tokens_details 时，调用报“OpenAI request failed”，未返回应保留的已知 token 与 null details。
3. **完整响应 deadline 回归**：mock 先返回 headers，再于 30ms 后供应完整 body；60,000ms SDK timer 加速到 5ms，调用仍成功，违反完整响应读取应受截止时间约束的预期。测试未真实等待 60 秒，不能声明真实 60 秒 wall-clock 已验证。

## 已通过与覆盖

已通过：JSON 请求与有效/无效 JSON、usage 完全缺失、reasoning summary 存在/缺失、HTTP 401/429/500 与连接失败的脱敏及单次尝试、headers 前超时的 provider_timeout/502 分类与取消、localhost/IPv4/IPv6 loopback HTTP 及远程 HTTPS 接受、远程 HTTP/FTP/无效 URL 拒绝、缺失 API key、256Ki 字符结果边界及超界拒绝。

适用场景 22；有效进入目标行为的场景 20/22（90.9%），其余 2 被测试装载器阻断。有效执行行为测试 20；通过 18/20（90%）、业务失败 2、阻断 2；原始执行器计数 22/18/4 保留作历史证据。没有代码分支仪器，分支覆盖率未测量。场景覆盖与通过率都不是代码覆盖率。初版转译语义污染是测试基础设施风险；父会话通过静态复核明确上述两个受影响场景。请求选项及完整 usage 字段、debug 日志泄露未覆盖；真实 Gateway/API、HTTP/WebSocket service、audit/store 和其他 Provider 不在本次运行范围。

## 实际命令与副作用

在仓库根运行 `node tests/fd021_sdk_blackbox.mjs` 两次：

- 首次约 2.85 秒，exit 1。报告器读取 `openai/package.json` 遇 package exports 错误，结果未落盘；不推断首次业务通过数。
- 仅修正报告器到安装目录的 package.json 只读路径，按仓库允许的一次路径错误修正重试，同一命令约 0.43 秒、exit 1，写出原始结果 JSON。

脚本只写 `docs/features/reports/FD-021-test-results-r1.json`；在进程内临时替换环境变量、fetch、定时器和 canary console 捕获，最终恢复。生产模块内存装载，没有最终可分发产物。新增测试和本报告位于仓库指定目录；未做 Git 写操作。

未运行全库测试、真实 API/Gateway、网络、下载、build、lint、formatter、vet、验证脚本或代码覆盖仪器。测试脚本仅为本次聚焦用例，未改生产代码。修复后应另行授权一条聚焦重测命令，保留本次历史证据。
