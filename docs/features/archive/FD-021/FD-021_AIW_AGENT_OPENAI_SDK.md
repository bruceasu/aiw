# FD-021: AIW Agent OpenAI SDK integration

**Status:** Complete
**Revision:** 6
**Priority:** Medium  
**Test policy:** External  
**Evidence policy:** Dual

## Problem

`program/aiw-agent` currently builds OpenAI Responses requests with `fetch` and
decodes the response JSON by hand. This duplicates SDK protocol handling and
makes it harder to use the existing OpenAI-compatible Gateway endpoint.

## Options and decision

- Keep the hand-written HTTP client: no new dependency, but retain custom
  request/response handling.
- Use the OpenAI Node API SDK: adopt the official Responses client while
  retaining the current service API, provider selection, result contract,
  timeout, usage audit, and no-retry behavior.
- Use the OpenAI Agents SDK: adds agent orchestration outside this service's
  current single-response contract and requires broader Gateway support for
  tools and agent flows.

Decision: use the OpenAI Node API SDK for this change. Plan Gateway support for
the OpenAI Agents SDK in a separate FD. Do not add an Agent SDK runtime to this
service in this FD.

## Solution

Use `openai`'s Responses API client in the OpenAI provider. Preserve current
model, input, instructions, JSON output, reasoning effort/summary, timeout,
usage metadata, result limits, and sanitized errors. Disable automatic retries
so one service request remains one upstream attempt. Permit HTTP only for
loopback `OPENAI_BASE_URL` values so the local Gateway simulator can be used;
require HTTPS for other hosts. Remove dependencies only when static usage
review proves they are no longer needed.

R3 compatibility decision: use the SDK's public typed POST method for
`/responses` rather than its generated `responses.create` convenience helper.
The helper overwrites a compatible endpoint's supplied `output_text`; the
public SDK transport preserves the pre-existing text preference and standard
output extraction without manual HTTP transport or JSON decoding. Keep the
request body checked against `ResponseCreateParamsNonStreaming` and retain
SDK authentication, timeout, zero retries, error handling and logging controls.

## Scope

Only `program/aiw-agent` OpenAI provider configuration, dependency manifest and
lockfile, documentation, and this FD. Do not change the service HTTP/WebSocket
contract, Gateway implementation, Copilot/Codex providers, or run live API
requests. Existing user changes in the workspace must be preserved.

2026-10-04：用户批准新增显式客户端 Gateway 模式，默认 OpenAI 行为保持不变。友好关闭与停止脚本是新增 Gateway 生命周期工作，另以编号 FD 记录；不混入本 FD 的 API 支持子集。

## Work items

- [x] 1.1 Inspect current OpenAI provider and dependency use. Acceptance:
  identify SDK replacement seam and confirm whether current dependencies are
  still used.
- [x] 1.2 Replace manual OpenAI Responses transport with the OpenAI Node API
  SDK. Acceptance: preserve request options, output/usage mapping, configured
  timeout, sanitized failures, and disable retries.
- [x] 1.3 Update dependency manifest, lockfile, and README for SDK use and
  loopback simulator configuration. Acceptance: no unused direct dependencies
  remain; HTTP is limited to loopback and remote endpoints require HTTPS.
- [x] 1.4 Review changed source and run the permitted compile-only check.
- [x] 1.5 Document the extension workflow and knowledge-base design checkpoints.
  Acceptance: identify request, provider, storage and audit seams; state that
  the current service has no knowledge-base implementation.
- [x] 1.6 修复 R1 三项发现。规模：小；难度：中；依赖：1.2。完成条件：完整响应读取受原有截止时间保护，缺失或 null 的嵌套 usage 字段容错，显式关闭 SDK 日志，保持既有错误分类与零重试。
- [x] 1.7 独立聚焦离线重测。规模：小；难度：中；依赖：1.6。完成条件：保留 R1 历史证据，验证修复和上轮阻断场景，保存 R2 中文报告及 JSON。证据：`docs/features/reports/FD-021-test-report-r2.md`，24/24 通过。
- [x] 1.8 独立复评。规模：小；难度：中；依赖：1.7。完成条件：核对实际差异、测试装载器和运行证据，记录是否仍有阻断问题；不伪造缺失的生命周期事件。证据：`docs/features/reviews/FD-021-review-r2.md`，三项修复确认，未发现新的实质缺陷。
- [x] 1.9 补充 Provider 异常响应与边界测试。规模：小；难度：中；依赖：1.8。完成条件：独立验证空输出、摘要限制、无效用量、URL 组件、错误正文与计时器清理；修复发现的既有契约回归，保存 R3 证据，不扩展 Gateway 支持子集。证据：`docs/features/reports/FD-021-test-report-r3.md`，初测 25/26，唯一修复重试 26/26 通过。
- [x] 1.10 独立复核 R3 差异与证据。规模：小；难度：中；依赖：1.9。完成条件：确认新测试可信度和修复范围，明确 SDK 与 Gateway 的兼容性边界，不把模拟通过标为正式 Complete。证据：`docs/features/reviews/FD-021-review-r3.md`，顶层文本兼容回归修复确认，未发现新的实质缺陷。
- [x] 1.11 验证用户已启动的真实 Gateway。规模：小；难度：中；依赖：1.10。完成条件：仅执行一条聚焦命令，最多触发一次最小真实模型执行，区分当前 Provider 请求与官方 SDK 支持子集的兼容性，保存 R4 脱敏证据。授权：用户于 2026-10-04 明确同意“允许一次最小真实模型请求”。不修改 Gateway 配置或支持范围，不输出凭据、响应正文或原始错误。证据：`docs/features/reports/FD-021-test-report-r4.md`；Provider HTTP 400，直接 SDK 最小请求 HTTP 200，不能将 Provider 兼容性标为通过。

- [x] 1.12 新增用户批准的显式客户端 Gateway 模式。规模：小；难度：低；依赖：1.11。完成条件：默认 `openai` 保留原有参数，`aiw_gateway` 省略固定 token 上限，调用前明确拒绝 reasoning，非法模式报配置错误，保留 SDK 安全控制。
- [x] 1.13 独立验证配置模式并复评。规模：小；难度：中；依赖：1.12。完成条件：聚焦离线验证请求字段、错误和默认兼容；无产物编译；中文 R5 双文件证据，不重复已用完授权的模型请求。证据：`docs/features/reports/FD-021-test-report-r5.md`，8/8；`docs/features/reviews/FD-021-review-r5.md`，批准范围内通过。
- [x] 1.14 当前 Gateway 模式真实验收。规模：小；难度：低；依赖：1.13。完成条件：按用户本轮测试验收请求，在 FD-025 已安装 Gateway 上只发一次最小 Provider 请求，明确设置 aiw_gateway，无重试或直接 SDK 回退；验证成功文本、状态、用量、请求字段及关联执行审计，保存 R6 脱敏双证据。证据：`docs/features/reports/FD-021-test-report-r6.md`，单次请求 HTTP 200、7/7 断言通过，关联审计 succeeded/execution_started=true。
- [x] 1.15 外部测试接受与正式独立评审。规模：小；难度：低；依赖：1.14。完成条件：PM 核对当前源码摘要、R6 与历史证据后记录 External 接受决策，再以当前 FD 发起正式 request-review；独立 Reviewer 验证后记录真实结果，不补造过去缺失的实现或 Tester 事件。证据：`docs/features/reviews/FD-021-review-r6.md`，正式独立评审无实质发现、验证通过。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- OpenAI provider calls Responses through the official Node API SDK.
- 默认 `openai` 配置保留现有 input/output、instructions、JSON、reasoning summary、usage、timeout 和结果大小契约；用户批准的 `aiw_gateway` 模式对不支持的 reasoning 选项在调用前明确报错，保留其余契约。
- SDK automatic retries are disabled; errors do not expose credentials or raw
  upstream bodies.
- 通过 `OPENAI_API_PROFILE=aiw_gateway` 和 `OPENAI_BASE_URL` 显式选择当前 Gateway；非 loopback HTTP 仍被拒绝。
- `ws`, its type declarations, and TypeScript remain while their current source
  and build uses remain.

## TODO

- 2026-10-04 用户在 FD-025 完成后要求“那么现在FD-021 可以测试和验收了”。本轮按真实 Gateway 兼容性测试请求新增一次最小模型执行预算，只验证当前 aiw_gateway Provider，不重试、不回退、不扩大接口支持范围。继续 External 策略，由外部 Tester 保存直接运行证据，PM 记录接受后将 authored 状态转入 Pending Verification 并通过 request-review 创建真实 Reviewer 交接；旧实现/Tester 回执缺失作为历史保留，不能补造。

- R4 是直接 SDK 的历史成功；R6 当前 `aiw_gateway` Provider 已真实通过，不能再以 R4 直接 SDK 代替 Provider 验收。本 FD 不扩展 Gateway API 支持子集。
- R1 三项发现已按用户后续授权修复，独立重测与复评完成；历史结论保留于 `docs/features/reviews/FD-021-review-r1.md`，本轮结论见 R2 报告。
- %% VERIFICATION_GAP: R6 当前模式已真实通过；本轮只执行7/40场景，匹配R5历史8项后运行证据覆盖15/40（37.5%），分支未测量，其他25项以独立静态证据及明确风险例外处理；不描述为全量运行通过。
- 真实 Gateway 验证已获一次最小模型执行授权，地址为 `http://127.0.0.1:43127/v1`，测试模型选配置允许的 `gpt-6-luna`；Key 仅从运行进程指定配置读入内存。先验证当前 Provider；只有明确 HTTP 400 参数前置拒绝时才补一次直接 SDK 最小子集请求，保证最多一次模型执行。该真实验证是用户对 Scope 中“no live API requests”的本轮限定例外，不授权其他请求、网络探测、重试或接口变更。
- 过去的 External 测试缺少正式实现交接回执，直接实现记录保留 R2/R3。本轮使用上述明确的 External 接受和新 Reviewer 交接，不把旧直接报告称作正式事件。
- External PM 接受已记录 `reports/FD-021-external-acceptance-r6.md`；独立 Reviewer 已 claim 当前真实 `FD-021-000005-review-requested`，R6 正式评审验证通过，无实质发现。历史实现/Tester 事件缺失保留，不补造；正式结果由 verification-passed 事件记录。
- R3 补充边界测试发现 SDK helper 覆写顶层 `output_text` 的历史兼容回归，已使用 SDK 公开 typed POST 修复；README 已明确当前 Go Gateway 既有请求字段限制。
- 用户已明确批准 `OPENAI_API_PROFILE=aiw_gateway` 配置模式；默认 `openai` 保留既有参数。Gateway 模式省略固定 `max_output_tokens`，依赖现有结果上限；非空 `effort` 或 `include_reasoning_summary=true` 在调用前返回 `unsupported_option`。该批准不等于追加真实模型请求授权。

## Verification

- R6 正式独立 Reviewer 会话 `fd021-reviewer-r6-20261004-independent`，source_event `FD-021-000005-review-requested`；报告 `reviews/FD-021-review-r6.md` 与同名 JSON 验证通过。独立核对当前源/SDK/测试/安装二进制摘要、单真实 Provider HTTP200 和关联 succeeded/execution_started 审计，以及 PM External 接受和覆盖例外；未追加测试、模型、网络、构建或 Git 写操作。
- 本轮7项通过、R5匹配历史8项可用；本轮实测17.5%，合计运行证据覆盖37.5%，25项无当前运行证据，分支未测量。当前安全/兼容性静态追踪支持要求，不将未运行或历史计作本轮运行。

- 实际在仓库根执行 `node tests/fd021_gateway_acceptance.mjs` 一次，exit 0，约6.20秒；单真实Provider请求、7/7断言通过，profile=aiw_gateway、字段input/model、HTTP200/completed、文本匹配及usage正确，匹配HTTP审计succeeded/execution_started=true。报告、raw、授权均为R6独占文件。Gateway按已有用户启动授权运行并保持监听；未改生产代码、配置、依赖或Git。
- R6 核对历史摘要：R5当前Provider/config/errors、SDK和测试摘要匹配，历史8项保留可用；R2/R3不匹配，不能合并为当前版本测试通过。覆盖口径和未运行风险详见R6 External PM接受决策，正式独立Review由新request-review交接。

- R6 计划：仓库根 `node tests/fd021_gateway_acceptance.mjs` 单次真实 Provider 调用；用户本轮真实 Gateway 测试验收请求为新授权，运行前保存版本/源码/测试摘要绑定记录。只读安装配置取允许模型的 Key，脱敏输出，不使用旧已耗尽 R4 授权，不运行旧脚本或覆盖历史。Gateway 启动沿用用户之前对 scripts/start-gateway.bat 的授权。

- Static source review confirmed `ws` is used by the WebSocket service and
  smoke script, `@types/ws` supplies service typing, and TypeScript is used by
  the build script.
- `npm install openai --save --ignore-scripts` selected `openai@7.27.0` but was
  rejected because its optional `ws` peer requires `^8.21.0` while the service
  pins `8.18.3`.
- `npm view openai@6 version peerDependencies --json` confirmed the latest v6
  peer range `ws ^8.18.0`; `npm view openai@6.49.0 peerDependenciesMeta --json`
  confirmed its optional peer dependencies.
- `npm install "openai@^6.49.0" --save --ignore-scripts` succeeded and resolved
  `openai@6.49.0`; npm reported one high severity vulnerability during install.
- Initial implementation: no tests, live API requests, or Gateway compatibility checks run; later offline runs are recorded below.
- `\.\node_modules\.bin\tsc.cmd --noEmit -p tsconfig.json` exited 0; no output
  or compiled artifacts were produced.
- 2026-10-04：按用户明确要求执行离线测试与独立评审。测试报告：`docs/features/reports/FD-021-test-report-r1.md`；原始结果：`docs/features/reports/FD-021-test-results-r1.json`；独立评审：`docs/features/reviews/FD-021-review-r1.md`。人类报告均配同名 JSON。
- 从仓库根目录执行 `node tests/fd021_sdk_blackbox.mjs` 两次：首次因报告器读取 SDK 版本文件的包导出限制失败；修正该路径后唯一重试退出 1。重试的 22 个用例记录为 18 通过、4 失败，其中 2 个失败属于测试装载器缺陷；有效业务结果为 18 通过、2 失败，另 2 个场景未验证。未将测试器失败归因于业务实现。
- 测试使用本进程模拟 fetch 与本地已安装 SDK，无网络、下载、真实 API 或 Gateway 请求。超时用例将配置的 60 秒定时器缩短为 5 毫秒，只验证截止机制，不声称实测等待 60 秒。
- 评审结论为需要修改：完整响应正文超时和稀疏 usage 回归有离线证据；SDK 调试日志正文泄露有静态证据。测试装载器已修正但未进行第三次执行；保留既有状态和历史验证记录。
- 2026-10-04 后续修复：用户明确要求继续测试、修复、复评；恢复覆盖完整 Responses 调用的 AbortController 定时器和 signal，统一取消为 `provider_timeout`，在 finally 清理计时器；嵌套 usage 使用可选访问，SDK `logLevel` 固定为 `off`。
- 本轮在 `program/aiw-agent` 执行 `.\node_modules\.bin\tsc.cmd --noEmit -p tsconfig.json`，退出 0，无构建产物。聚焦测试与独立复评证据另存 R2，不覆盖 R1。
- 独立 Tester 从仓库根单次执行 `node tests/fd021_sdk_blackbox.mjs`，退出 0，约 0.41 秒，24/24 场景通过，无失败或装载器阻断；报告 `docs/features/reports/FD-021-test-report-r2.md` 和原始证据 `docs/features/reports/FD-021-test-results-r2.json`。覆盖原请求参数、完整 usage 映射、缺失/null details、成功和错误正文截止取消、调试日志脱敏以及既有 JSON/URL/结果限制。未测量代码分支覆盖率。
- 独立 Reviewer 在 `docs/features/reviews/FD-021-review-r2.md` 确认 R1 三项均已解决，未发现新的实质缺陷；核对实际源文件、修正后的测试装载器、运行记录及摘要。结论只适用于本轮修复范围，不表示真实 Gateway 兼容性或正式生命周期验收通过。
- R2 测试绑定执行时 FD SHA-256 `24fee3ad97b7ebc1f65b34548ad9046a3857fe70803bfc54980f4b892bae554c`，本节和 Work Item 完成标记随后仅补充证据；业务实现与验收未再改变。实现记录：`docs/features/reports/FD-021-implementation-r2.md`。未运行网络、下载、真实 API/Gateway、全库测试或最终构建，无 Git 写操作。
- R3 在仓库根运行 `node tests/fd021_sdk_edges.mjs`：初测退出 1，约 0.46 秒，25/26 通过；修复历史顶层文本兼容后，同命令唯一重试退出 0，约 0.43 秒，26/26 通过。初测和重试原始证据分别保存于 `docs/features/reports/FD-021-test-results-r3-initial.json`、`FD-021-test-results-r3-retry.json`，不覆盖历史记录。
- R3 修复后在 `program/aiw-agent` 执行 `.\node_modules\.bin\tsc.cmd --noEmit -p tsconfig.json`，退出 0，无构建产物。独立复评 `docs/features/reviews/FD-021-review-r3.md` 确认修复并未发现新的实质缺陷；测试报告及实现记录分别为 `docs/features/reports/FD-021-test-report-r3.md`、`FD-021-implementation-r3.md`，均配同名 JSON。
- R3 重试绑定 FD SHA-256 `178e04dec0b1866820fc670d8a384c1b1048832259d0d3758b2469b00fbbbaff` 与 providers SHA-256 `dbac61df7776afece4fd2a9e7e603f8b51e65c53df3d7a8440193454cc674aaf`。后续只补充本节与完成标记。R2 原用例没有在 R3 实现版本重跑，不能合称当前实现 50/50 通过；其请求字段、正文截止及调试日志场景为历史运行证据，当前保留行为由独立静态追踪补充。
- R4 在仓库根只执行一次 `node tests/fd021_gateway_live.mjs`，约 5.30 秒：当前 Provider 请求字段为 input/max_output_tokens/model，真实 HTTP 400、未启动模型；按限定条件补充直接 SDK input/model 请求，真实 HTTP 200、status completed、预期回复匹配。总计两个 HTTP POST，最多一次模型执行，无自动重试。原始脱敏记录 `docs/features/reports/FD-021-test-results-r4.json`；运行命令退出状态及细节以 R4 测试报告为准。
- R4 Key 仅从运行进程指定配置读取至内存；不记录认证头、Key、提示词、响应正文或原始错误。Gateway 自身仍按运行配置写入审计、用量和内容存储。没有启动、停止、重新配置 Gateway，未改业务实现、未构建或下载。
- R4 独立评审：`docs/features/reviews/FD-021-review-r4.md` 及同名 JSON，结论为 Gateway 兼容性需要修改，不能以直接 SDK 请求成功替代 Provider 验收。Reviewer 仅只读两条关联请求审计的安全字段：首条 rejected/HTTP 400/execution_started=false，第二条 succeeded/HTTP 200/execution_started=true；确认本次两请求中一次执行启动，不推断整个服务总执行数。
- R4 工具实际报告命令退出 1，而脚本对于“Provider 失败、直接 SDK 成功”分支意图退出 2；差异未进一步调查，不声称命令退出 0，不追加请求或重试。HTTP 结果、模型完成与关联审计证据各自保存。R4 当时配置方案尚未实施；随后用户批准并在 R5 实施。
- R5 已实现显式配置，执行 `.\node_modules\.bin\tsc.cmd --noEmit -p tsconfig.json` 退出 0；独立 Tester 单次 `node tests/fd021_gateway_profile.mjs` 退出 0，8/8 通过，零网络与模型执行，证据为 `docs/features/reports/FD-021-test-report-r5.md` 和同名 JSON。保留历史 R1–R4 证据，不合并不同实现版本的通过数。

## Sources

- User request to adopt an OpenAI SDK and remove unused dependencies.
- `program/aiw-agent/src/providers.ts`
- `program/aiw-agent/src/service.ts`
- `program/aiw-agent/package.json`
- `openspec/specs/agent-proxy/spec.md`

**Completed:** 2026-10-04
**Disposition reason:** done
