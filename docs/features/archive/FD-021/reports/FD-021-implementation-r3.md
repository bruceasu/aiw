# FD-021 边界兼容修复 R3

<!-- aiw-data: FD-021-implementation-r3.json -->

独立 Tester 补充 26 个边界场景，首次聚焦运行 25 通过、1 失败。失败是历史已接受的响应：`object: "response"`、`output: []`、顶层 `output_text` 非空；生成的 SDK Responses helper 覆写该文本，导致请求错误。

修复使用官方 SDK 公开 `post<OpenAI.Responses.Response>("/responses", ...)`，请求体以 `ResponseCreateParamsNonStreaming` 静态检查。继续由 SDK 完成 Bearer 认证、传输、JSON 解码和错误处理，保留完整响应截止时间、零重试、禁用日志以及旧版文本优先顺序；不使用手写 fetch 或手动 JSON 解码，不改变外部服务接口。Reviewer 已静态确认当前 SDK 的公开 post 默认使用 Bearer 认证。

README 和 FD 同步解释选用 SDK 公开方法的原因，以及当前 Go Gateway 拒绝已有请求字段的兼容性边界。不修改 Gateway 支持子集、不隐式删除请求参数。

实际编译命令：在 `program/aiw-agent` 执行 `.\node_modules\.bin\tsc.cmd --noEmit -p tsconfig.json`，退出 0，无构建产物。边界测试与独立复评结果以 R3 报告为准。

结果：独立 Tester 从仓库根执行 `node tests/fd021_sdk_edges.mjs`，首次退出 1，25/26 通过；上述修复后同命令唯一重试退出 0，26/26 通过。公开 SDK POST 默认 Bearer 认证与历史文本形态均有模拟运行证据。独立 R3 评审确认修复且无新的实质缺陷。R2 原用例未在本轮实现版本重跑，不汇总为当前 50/50 通过。

未运行真实 API/Gateway、网络、下载、全库测试、最终构建、lint 或 formatter，无 Git 写操作。External 策略下缺少正式生命周期回执，source_event 为 null，不伪造事件，不标记 Complete。
