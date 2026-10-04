# FD-021 独立评审 R1

<!-- aiw-data: FD-021-review-r1.json -->

结论：**要求修改**。发现完整响应超时与稀疏用量解析两项回归，以及 SDK 调试日志暴露正文的问题。

- Reviewer 会话：`fd021_reviewer`，独立于实现者与 Tester。
- 评审基线：`c9b6d51cd2308e1ffbaf51755d697f6acda3ad83` 对当前工作区未提交差异。
- 范围：`program/aiw-agent/src/providers.ts`、`package.json`、`package-lock.json`、`README.md` 与 FD-021 Revision 4。
- source_event：`null`。没有本次实现报告或 Reviewer 派发回执；这是用户直接请求的评审，未 claim/emit 事件，不能作为正式 `verification-passed`。
- Test policy：External；Evidence policy：Dual。

## 发现

### R1-01 · P1：SDK timeout 不覆盖响应正文读取

位置：`program/aiw-agent/src/providers.ts:129`、`:137`。

新实现仅配置 SDK `timeout`，移除了原实现覆盖 `fetch` 和 `response.json()` 的 AbortController 定时器。安装的 `openai@6.49.0` 在 `node_modules/openai/client.mjs:528-552` 中等待 fetch 返回响应头后立即清除计时器，随后 `internal/parse.mjs:32` 才读取 JSON 正文；服务调用方也只是直接 await complete，没有额外完整请求截止保护。

可复现场景：上游立即返回 HTTP 200 和 JSON Content-Type，发送部分正文后保持连接超过 60 秒。新实现的读取不会由该 SDK timeout 在 60 秒取消；旧实现会中止并映射 `provider_timeout`。错误响应正文的 `response.text()` 同样在 SDK timer 清除后读取。此问题违反 Work Item 1.2 与保留 timeout 的验收要求。

建议：为整个 Responses 调用保留独立截止计时和传入 signal，并将该 signal 引起的中止统一映射为 `provider_timeout`；确保所有出口清理计时器。

### R1-02 · P2：缺少嵌套 usage details 时成功结果变为 502

位置：`program/aiw-agent/src/providers.ts:159-161`。

`usageRow?.input_tokens_details.cached_tokens` 只保护 usageRow；当 usage 存在而 details 缺失或为 null 时，会访问空值属性，抛 TypeError 后转成 `provider_error`。旧实现为两个 details 字段分别检查对象并使用 `{}` 兜底。

可复现场景：标准成功响应包含非空文本与 `usage: {input_tokens: 1, output_tokens: 2, total_tokens: 3}`，省略 input/output details。旧实现返回文本与 tokens、details 为 null，新实现返回 502。网关自身的 usage details 也是可选字段，不能以 SDK 静态类型保证兼容端点运行时总会提供。

建议：分别保护两个嵌套 details，再保留 safeInteger 的空值/无效值容错。

### R1-03 · P2：SDK 环境调试日志绕过正文脱敏

位置：`program/aiw-agent/src/providers.ts:129-135`。

新 SDK client 未固定 logLevel，`client.mjs:161-168` 使用 console 并读取 `OPENAI_LOG`。当宿主已有 `OPENAI_LOG=debug` 时，`:385-390` 输出请求 options（含 prompt、instructions），`internal/parse.mjs:38-43` 输出响应 body；非 JSON 错误在 `client.mjs:480-488` 输出原始错误文本。SDK 仅对已知敏感 header 做遮盖，不会删除请求/响应正文。原手写 fetch 无该环境变量日志路径；外层脱敏 ProxyError 不能撤销已输出的原始上游正文。

可复现场景：设置 `OPENAI_LOG=debug`，向本地兼容响应器发送带唯一敏感文本的 prompt，并返回带另一个标记的正文或非 JSON 错误；捕获 console 即可看到原文。

建议：显式关闭 SDK 日志，或提供满足当前脱敏契约的 logger。

## 已确认与契约边界

- 官方 SDK 的 Responses create 路径、SDK 版本与 lockfile 声明一致；`maxRetries: 0` 使本 API-key 路径不自动重试，`redirect: error` 保留重定向拒绝。
- URL 拒绝 userinfo、query、fragment，远程只允许 HTTPS，HTTP 只接受 localhost 子域或明确的 loopback IP；此为静态配置检查，未运行真实网络/DNS 验证。
- 标准 `object: response` 输出经 SDK `ResponsesParser.mjs:152-164` 聚合 message/output_text；调用方继续执行 JSON、结果大小与 reasoning summary 检查。SDK 会覆盖顶层 output_text，且比旧提取逻辑更严格；非标准仅顶层文本/缺少 output 的兼容端点仍有风险。
- `openspec/specs/agent-proxy/spec.md` 当前定义 Go Gateway，明确替代历史 TypeScript 多 Provider/ACK 服务，不能当作后者全部历史契约的当前规格。
- Go Gateway 的 `protocol.go:15-23,47-53` 拒绝未知字段；当前 TS provider 总发送 `max_output_tokens:8192`，可选 reasoning 也不属于支持子集。普通请求无法仅通过切换 base URL 接入当前 Gateway。这是**既有**限制，旧实现已有相同字段，未列为 SDK 新增回归；Gateway 实际兼容验收仍缺证据，README 不能暗示已经验证。
- ws、其类型与 TypeScript 的现有用途保留；无无关 Provider 或服务 HTTP/WebSocket 实现改动。

## 验证证据与未执行检查

Reviewer 实际只执行 Get-Content/rg 等定向读取、`git diff -- ...`、`git diff --stat -- ...`、`git rev-parse HEAD`；审阅已安装 SDK 实现，不访问网络，不安装依赖，不改 Git 状态。未执行测试、编译、构建、formatter、lint、vet 或自动评审；FD 中历史安装及 tsc 结果仅为既有记录，不能替代本次运行证据。

Tester 原始证据：`docs/features/reports/FD-021-test-results-r1.json`。实际运行记录为 22 个 case、18 passed、4 failed；其中稀疏 usage 与 headers 后 body deadline 两项失败有效，分别支持 R1-02 与 R1-01。使用真实已安装 SDK、进程内 fetch mock；将配置的 60000ms 定时器加速为 5ms，并非等待真实 60 秒。

Tester 在仓库根实际执行 `node tests/fd021_sdk_blackbox.mjs` 两次：首次因报告器 package exports 路径错误失败，修正后唯一重试退出 1 并产生上述 raw。授权来自用户本轮明确的“test and review the FD-021”；本 FD 为 External，无 Planner authorization 文件，不能伪造。主持已检查完整测试代码及副作用后准许运行。测试绑定 Revision 4 的原 FD digest `ab621e640cfaea752e0366be49af603040ba6d267c74091fda1d38277818d344` 与 raw 内 source_digests；随后 FD 仅补充 TODO/Verification 证据，验收未改变。

其余两项失败（主文本/system_prompt 参数与 debug 日志）由测试 loader 的宽泛字符串替换引起：替换 `"openai"` 同时污染生产代码中的 provider 比较，导致请求尚未到 SDK 就拒绝。因此不算产品失败，也不能作为主参数或日志脱敏的有效运行覆盖。R1-03 保持 SDK 源码静态确证，未声称有效运行复现。Tester 随后修正 loader，但因重试预算不再执行；现存 raw 属于修正前运行，不能当作修正后测试源码的执行结果。

## 剩余风险

上述回归尚未修改。主参数/system_prompt 与 SDK debug 脱敏没有有效运行覆盖；真实 OpenAI API、真实 Go Gateway 与完整 60 秒正文截止未由 Reviewer 执行验证；即使模拟测试通过，也不等价于外部 Gateway 兼容通过。FD 的既有 npm 高危依赖告警未通过新的网络审计核实。
