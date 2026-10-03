# FD-021 外部聚焦测试报告 r2

<!-- aiw-data: FD-021-test-report-r2.json -->

本轮聚焦测试 **通过**：24 个场景全部有效执行并通过，失败、阻断、未执行均为 0。建议交给独立 Reviewer 复核，本报告不代表正式生命周期已完成。

## 测试对象与授权

- FD：Revision 4；执行时文件 SHA-256：`24fee3ad97b7ebc1f65b34548ad9046a3857fe70803bfc54980f4b892bae554c`。
- Provider 原文件 SHA-256：`adcd853cf15b82db74caca48ad1ea76f03bb1a820eceec3a21fd1a2e88f45f35`。
- 测试脚本 SHA-256：`bb76a5285e0abfa102c65d7a935c551b254b009fed5c61f9b78d9023f43d42c9`。
- Tester 会话：`fd021-external-tester-r2-20261004-fd021_tester`。Test policy 为 External；用户明确继续“test and fix then review”，授权本轮同范围离线测试。
- source_event / implementation_event 为 null。未找到正式实现事件或回执，不伪造派发、验收或完成事件。
- 实际使用本地已安装 OpenAI SDK 6.49.0，fetch 完全替换为进程内 mock；没有网络、socket、真实 Gateway/API、依赖下载或最终构建。

## 本轮覆盖与结果

缺失和显式 null usage details 均保留已知 input/output/total tokens，details 返回 null。headers 前超时、HTTP 200 成功正文和 HTTP 500 错误正文延迟均产生 `provider_timeout / 502`，并取消上游且只尝试一次。超时用例仅把配置的 60,000ms 定时器缩短到 5ms；没有真实等待 60 秒。

`OPENAI_LOG=debug` 下，console 捕获未发现合成 prompt、instructions、成功正文或错误正文 canary；错误消息也未出现 canary。r1 测试装载器已改为仅替换 import specifier，本轮系统提示词场景与日志场景均有效通过。

其余通过项包括模型/input/instructions/effort/reasoning summary 请求字段、文本与 JSON 输出及无效 JSON 拒绝、完整 usage 元数据和完全缺失 usage、reasoning summary 存在/缺失、HTTP 401/429/500 与连接失败脱敏且不重试、loopback IPv4/IPv6/localhost HTTP 与远程 HTTPS 接受、远程 HTTP/FTP/无效 URL 拒绝、缺少 key 在发送前拒绝，以及 256Ki 字符结果边界和超界错误契约。

适用场景 24，覆盖 24/24，行为测试通过 24/24；这些数值只代表本次枚举场景，**不是代码分支覆盖率，也不是全部 FD 验收项的运行覆盖率**。没有仪器，分支覆盖率未测量；依赖保留和扩展设计文档属于静态评审范围。

## 实际命令与副作用

在仓库根执行一次 `node tests/fd021_sdk_blackbox.mjs`，退出码 0，耗时约 0.41 秒；没有重试。

脚本只写 `docs/features/reports/FD-021-test-results-r2.json`。生产 TypeScript 源文本只用于内存转译、模块装载与摘要，不用于设计断言；环境变量、fetch、定时器、console 捕获在进程内临时改动并恢复，没有可分发构建产物。r1 历史报告与原始结果保留；未修改生产代码或执行 Git 写操作。

## 未验证范围与残余风险

未运行真实 Gateway/API 兼容性、服务 HTTP/WebSocket、audit/store、其他 Provider、全库测试、分支覆盖仪器、build、lint、formatter、vet、验证脚本或网络操作。真实 60 秒 wall-clock、实际 SDK/网络慢流时序仍未实测；本轮仅验证缩短定时器后的截止与取消机制。r1 装载器语义污染属于历史测试基础设施问题，本轮通过受影响场景验证了修正，但未对任意 TypeScript 语法开展通用 loader 验证。

原始结果、FD/生产文件与脚本摘要见 `FD-021-test-results-r2.json`；建议独立 Reviewer 结合源码、差异和本报告判断修复，正式状态与缺失回执由 PM 按已有工作流处理。
