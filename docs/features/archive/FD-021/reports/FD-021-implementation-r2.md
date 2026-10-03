# FD-021 修复记录 R2

<!-- aiw-data: FD-021-implementation-r2.json -->

用户于 2026-10-04 明确要求继续测试、修复、复评。本次只修复 R1 三项发现，不改变 SDK 依赖、服务接口、其他 Provider 或 Gateway。

- 恢复覆盖整个 Responses 调用的 60 秒 AbortController 定时器，传入 SDK 的请求 signal；正文读取期间取消也映射为 `provider_timeout`，所有出口清理定时器。
- input/output usage details 使用嵌套可选访问，缺失或 null 时保留已知 token，并将未知细项映射为 null。
- SDK 日志固定为 `off`，避免宿主 `OPENAI_LOG=debug` 输出请求或响应正文。
- 独立 Tester 修正后的装载器仅替换 import specifier，使用真实已安装 SDK 与进程内 fetch 模拟；重测和独立复评保存 R2 证据。

后续验证：独立 Tester 单次执行 `node tests/fd021_sdk_blackbox.mjs`，退出 0，24/24 场景通过，详见 `FD-021-test-report-r2.md`。独立复评确认三项修复且无新的实质缺陷，详见 `../reviews/FD-021-review-r2.md`。未将离线通过描述为真实 Gateway 或正式生命周期通过。

静态依据：R1 评审、既有请求/错误契约、本地 SDK 的取消传递与正文解析路径。实际执行 `.\node_modules\.bin\tsc.cmd --noEmit -p tsconfig.json`（工作目录 `program/aiw-agent`），退出 0，无构建产物。

未运行全库测试、真实 API/Gateway、网络请求、下载、最终构建、lint 或 formatter；没有 Git 写操作。当前 External 策略且缺少正式实现交接回执，`source_event` 为 null。本记录不是伪造的 Worker 完成事件，FD 不自动关闭或归档。真实 Gateway 支持子集仍存在既有兼容性限制。
