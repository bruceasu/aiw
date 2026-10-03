# FD-021 独立复评 R2

<!-- aiw-data: FD-021-review-r2.json -->

结论：**R1 三项问题均已修复；本次修复范围内未发现新的实质缺陷**。聚焦离线测试 24/24 通过。此结论不是正式生命周期验证通过，FD 仍不能据此标记 Complete。

- Reviewer：`fd021_reviewer`，独立于实现者和 Tester。
- 基线：`c9b6d51cd2308e1ffbaf51755d697f6acda3ad83` 至当前未提交实现差异；复评重点为 R1 后修复与测试装载器。
- FD：Revision 4，External / Dual；source_event：`null`；没有正式实现交接回执，不 claim/emit 事件。
- 测试 FD digest：`24fee3ad97b7ebc1f65b34548ad9046a3857fe70803bfc54980f4b892bae554c`。
- 测试 providers digest：`adcd853cf15b82db74caca48ad1ea76f03bb1a820eceec3a21fd1a2e88f45f35`。
- 测试源码 digest：`bb76a5285e0abfa102c65d7a935c551b254b009fed5c61f9b78d9023f43d42c9`。

## R1 发现的复核

| 发现 | 当前实现与静态证据 | 独立运行证据 | 结论 |
| --- | --- | --- | --- |
| R1-01 完整响应超时 | `program/aiw-agent/src/providers.ts:138-151` 恢复独立 AbortController 和全程 timer，signal 传入 SDK create。`:173` 优先按该 signal 中止映射 timeout。`:178-179` finally 覆盖成功、解析异常、业务错误和 SDK 错误出口并清理 timer。SDK fetch 清除自身 timer 不再清除本地 timer。 | headers 前取消及 HTTP 200/500 headers 后正文延迟均通过，错误为 provider_timeout、502，调用一次。 | 已修复 |
| R1-02 稀疏 usage | `:163-165` 对 input/output details 使用嵌套可选访问，保留 safeInteger 容错。 | details 缺失、显式 null、整个 usage null 均通过；已知 tokens 保留、未知细项为 null。 | 已修复 |
| R1-03 调试日志正文泄露 | `:134` 显式 `logLevel: off` 优先于 SDK OPENAI_LOG 环境值。 | OPENAI_LOG=debug 下成功正文与 HTTP 500 错误 canary 测试通过，prompt、instructions、响应和错误正文未出现在捕获日志或错误 message。 | 已修复 |

保留现有 model/input/instructions、JSON 输出、reasoning、usage 映射、结果限制、SDK maxRetries:0 与 redirect:error。修复不涉及其他 Provider、服务接口或 Gateway 实现。

## 测试可信度与边界

读取 `tests/fd021_sdk_blackbox.mjs` 全文：装载器仅替换 import specifier，不再宽泛替换 `"openai"` 并污染 provider 比较；断言来自输入/输出行为，source 读取只用于 TypeScript 转译、装载及 digest。真实安装的 OpenAI SDK 通过进程内 fetch mock 调用，未使用网络或 socket。

测试在 finally 恢复 fetch、setTimeout、环境变量，日志 case 单独恢复 console。写入仅为 R2 raw 结果。取消 case 将配置的 60000ms timer 加速为 5ms，读取延迟为 30ms；因此证明截止机制与错误分类，不声称实测真实 60 秒网络超时。

Tester 实际从仓库根执行一次 `node tests/fd021_sdk_blackbox.mjs`，退出 0；raw `docs/features/reports/FD-021-test-results-r2.json` 记录 24 passed、0 failed。授权来自用户本轮继续测试、修复和复评的明确要求；External 不需要伪造 Planner 授权。R1 运行失败证据保留，不能被 R2 通过记录覆盖。

已核对 `docs/features/reports/FD-021-test-report-r2.md` 与同名 JSON：Tester 身份为 `fd021-external-tester-r2-20261004-fd021_tester`，与本 Reviewer 分离；报告命令、24 个场景结果、原始结果、SDK 版本和三个摘要相符。报告正确区分场景通过率、全 FD 验收运行覆盖率及未测分支覆盖率，耗时记录为 0.4110084 秒。没有把模拟兼容器当作真实 Gateway。

实现记录 `docs/features/reports/FD-021-implementation-r2.md` 报告 `.\node_modules\.bin\tsc.cmd --noEmit -p tsconfig.json`（工作目录 program/aiw-agent）退出 0；Reviewer 未重复编译或测试。

## 验收与生命周期限制

- 当前 Go Gateway 严格拒绝 `max_output_tokens` 与 reasoning，而 TS provider 仍发送历史已有字段；这不是本 SDK 修复新增回归。本次模拟测试没有证明真实 Gateway 可仅切换 base URL 使用，相关验收缺口继续保留。
- 标准 SDK output_text 聚合有运行证据；非标准仅顶层文本或缺失标准 output 的兼容端点、真实 DNS/重定向响应均未实测。
- 未测分支不能由 24/24 或其他场景通过推定通过；没有分支覆盖率数据，不将案例通过率称为代码覆盖率。
- `openspec/specs/agent-proxy/spec.md` 已定义 Go Gateway、替代历史 TS 接口；不将该规格全部要求套用于 TS，也不修改 Gateway 支持子集。
- source_event null、External、真实 Gateway 验证缺口仍存在；本报告为直接独立复评，不能 emit 正式 verification-passed、关闭、归档或自动标记 Complete。

## Reviewer 实际命令与未执行检查

Reviewer 执行 `git diff -- program/aiw-agent/src/providers.ts` 与 Get-Content 定向读取当前 FD、实现记录、测试源码、SDK 既有静态证据及 R2 raw；初次读取尚未落盘的 Tester 报告返回缺失，不是产品错误。未运行测试、编译、构建、formatter、lint、vet、网络、依赖下载或 Git 写操作；只新增本 R2 Markdown/JSON 证据。
