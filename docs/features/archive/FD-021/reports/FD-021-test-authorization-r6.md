# FD-021 R6 真实验收运行授权

<!-- aiw-data: FD-021-test-authorization-r6.json -->

用户本轮明确要求“那么现在FD-021 可以测试和验收了”。结合已确认的真实 Gateway 场景，本轮授权一次当前 aiw_gateway Provider 的最小真实模型请求，不沿用已经耗尽的 R4 配额。精确命令为仓库根 `node tests/fd021_gateway_acceptance.mjs`，预计模型响应数秒，最长约 65 秒。External 策略无正式 Tester 事件，source_event 为空，独立 Tester 会话及实现/测试摘要绑定如下；不补造过去交接。

已完整检查源码：只发送一次 POST 至 http://127.0.0.1:43127/v1/responses，模型 gpt-6-luna，只有 input/model，显式 aiw_gateway；禁重定向，没有重试和直接 SDK 回退。运行前检查 FD、Provider/config/errors、测试及安装 Gateway 摘要。C:\green\aiw 安装配置中的 enabled 且允许该模型主体 Key 只读入内存，不输出配置、Key、提示词、正文或原始错误。仅按严格 request-id 和 realpath 边界读取一条关联 HTTP 审计，允许最多 1.5 秒文件终态等待，不增加 HTTP 请求。

副作用为一次真实后端执行及正常审计、用量和内容存储，另独占写 R6 脱敏原始 JSON；不启动/停止服务、不改配置、不下载依赖、不构建、不改权限、不做 Git 写操作。父会话已按此前启动授权运行新版 Gateway，保持运行。历史报告仅在可用源码/SDK/测试摘要匹配时作为历史证据，不能计为本轮运行。失败立即报告，不允许第二次模型请求。
