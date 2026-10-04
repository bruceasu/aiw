# FD-020: Agent Gateway request observability

**Status:** Complete
**Revision:** 11
**Priority:** Medium
**Test policy:** Independent
**Evidence policy:** Dual

## Problem

当前网关只在 Responses 进入 execute 并返回后输出请求日志。认证失败、参数错误、模型拒绝、RPM/并发拒绝以及 Models/Usage 请求没有完整请求日志。调用主体已经存入执行元数据，但后端 Codex 登录帐户未识别或记录。现有 Usage 只统计实际启动的执行，不能表示所有 HTTP 请求及拒绝原因。

## Options and decision

用户已明确选择服务端保存明文 Key、直接匹配，并确认日志和统计按网关调用主体完善。后续确认不再支持 key_hashes，原摘要字符串已经作为 keys 中的 Key 使用。仅接受 principals[].keys，使用常量时间直接比较；配置加载和请求认证均不计算摘要。配置加载时检查重复 Key，拒绝身份歧义。Key 是不透明字符串，其历史来源不影响直接匹配；运营者保存和分发 Key，重启生效；本轮不新增在线签发接口。此决定替代初版的旧摘要兼容方案，历史 r1 报告保持不变。

## Solution

在 HTTP 入口统一产生请求开始及终止日志，关联 x-request-id、已认证 principal、接口、结果、错误码和毫秒耗时。认证失败使用空主体，不能从用户自报字段认定身份。未知接口/方法归入 other，模型仅记录服务端配置中的逻辑模型。日志不含 Key、Key 摘要、Authorization、提示词、响应正文或查询参数原文。

保持现有执行用量及额度口径：成功启动 Codex 才扣额度；启动前拒绝不扣额度。增加 state_dir/requests/<request-id>.json 和 requests-manifest.json，复用已有排他锁、原子文件写入，schema_version=1，长期保留脱敏 HTTP 元数据。旧存储首次升级仅新增目录及版本标记，无删除或历史回填；已标记后目录缺失拒绝启动。入口持久化 in_progress，完成时替换终态；重启将未完成记录标为 interrupted，不猜测实际终止时间或耗时。写入失败记录脱敏错误并阻止新执行。

GET /v1/usage 保留原字段，新增 principal 和 http_groups，仅查询已认证主体。HTTP 分组使用请求收到时间所属日，按日期、模型、接口、方法统计总请求、成功、拒绝、失败、超时、取消、中断、处理中、实际执行数、HTTP 状态及错误码计数、已知耗时样本数和总毫秒数。查询请求本身处于处理中；未认证记录仅运营者本地查看，无公共全局统计接口。执行 groups 继续按实际启动日汇总。

授权依据：用户选择直接保存明文 Key，并确认包含主体/接口/结果/耗时及拒绝区分的统计范围；持久化与 Usage 的附加字段用于落实该已批准范围。保留期限沿用长期元数据策略，调用主体之外的后端登录帐户不在本轮范围。

## Scope

覆盖 program/agent-gateway 的配置认证、HTTP 日志、独立 HTTP 元数据及 Usage 附加统计，并更新稳定规格和说明。保留主体隔离、模型权限、额度和执行 token 口径。不修改现有真实凭据，不生成示例可用 Key。无需依赖下载、Git 写操作或部署。

## Work items

- [x] 1.1 确认身份、Key 和统计契约【小 / 中；无依赖】。完成标准：解决上述待决项，明确兼容性、查询授权和保留期限。
- [x] 1.2 统一 HTTP 请求日志【小 / 中；依赖 1.1】。完成标准：所有接口和提前返回均有终止日志，开始与终止通过请求 ID 关联，主体来源可信且字段脱敏，记录耗时和结果。
- [x] 1.3 补充请求统计【小 / 中；依赖 1.1、1.2】。完成标准：按批准口径区分 HTTP 请求、拒绝和实际执行；主体隔离，拒绝不扣额度，现有执行用量兼容。由 1.3.1 和 1.3.2 分别实现。
- [x] 1.3.1 独立 HTTP 元数据与中断恢复【小 / 中；依赖 1.1】。完成标准：请求开始与终态原子落盘，旧目录可升级，未知版本/损坏拒绝启动，重启未完成标 interrupted，不回填已知耗时。
- [x] 1.3.2 Usage 增量 HTTP 分组【小 / 中；依赖 1.2、1.3.1】。完成标准：只返回当前主体，含拒绝及实际执行、状态/错误码、已知耗时统计，保留原 groups 与额度字段。
- [x] 1.4 更新说明与证据【小 / 低；依赖 1.2、1.3】。完成标准：日志字段、身份含义、统计口径和限制可查；中文 Markdown 报告及同名 JSON；更新 TODO 和 Verification。
- [x] 1.5 明文 Key 及旧配置兼容【小 / 中；依赖 1.1】。初版完成 keys 直接匹配、禁用、重复凭据检查及旧摘要兼容；后续认证契约由 1.6 替代，保留历史完成记录。
- [x] 1.6 移除旧摘要认证【小 / 低；依赖 1.5】。完成标准：仅接受 keys，移除 key_hashes 和 SHA-256/hex 分支；重复 Key、禁用和直接匹配继续有效，原摘要字符串可直接作为 Key；同步规格和说明，新报告不覆盖 r1。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- 同一请求的开始、终止及已有执行元数据可通过请求 ID 关联。
- 已认证请求记录稳定主体；未认证请求不冒认用户身份。
- 成功、参数拒绝、认证失败、限流、并发拒绝、后端失败和超时均可辨别。
- 请求拒绝不混入已启动执行数，不改变每日扣额规则；未知 token 不当成零。
- 不记录 Key 或其摘要，不记录提示词和响应正文。
- 统计仅访问当前主体；HTTP 请求按收到日、执行按启动日；重启未完成请求标 interrupted，未知耗时为空。
- Key 直接匹配，不做摘要转换；不接受 key_hashes 字段；原摘要字符串作为 keys 的 Key 与其他合法 Key 一样处理。轮换不改主体、统计和额度。

## TODO

- Worker 实施完成；独立 Tester 与 Reviewer 已完成，测试覆盖限制及审查证据见 Verification；本轮不归档或部署。
- 用户已将原摘要值作为 Key 配置于 keys，保留用户配置，不再要求恢复原始 Key。
- 2026-10-04：用户明确要求 test then review。使用正式 refresh-tester 恢复当前修订的测试交接，旧事件和报告保留为历史，不复用旧测试授权。
- 历史实施阶段未替换二进制、未运行测试。本轮可在 Planner 检查并记录精确授权后运行聚焦的临时隔离测试；真实凭据、外部 AI、发布、部署和下载不在范围内。
- 请求文件长期增长；轮转及高流量性能评估不在本轮范围。

## Verification

- 2026-10-04 独立测试：`reports/FD-020-test-report-r1.md/json` 及 raw-r1/r2；修正未指定状态码的测试假设后 29/29 通过，需求覆盖 40/54（74.07%），14 场景未执行，分支覆盖未测量。精确授权见 authorization-r1/r2，PM 例外及接受决定见 test-decision-r1。
- 2026-10-04 独立 Reviewer `fd020-reviewer-20261004-93d81a` 精确认领事件 `FD-020-000010-test-accepted`（revision 10）；报告 `reviews/FD-020-review-r1.md/json`。当前实现/契约和证据静态审查通过，无阻断缺陷；本次未运行测试、编译、服务、网络或 Git 写操作，未将 14 个未测场景算作通过。发布 verification-passed 后由正式回执记录状态。

- 静态依据：server.go 的 ServeHTTP/responses 提前返回路径及 execute 后日志；config.go 的 authenticate 与 childEnv；runner.go 的主体元数据；storage.go 的 Aggregate 仅包含 StartedAt 非空的执行。
- 已运行只读文件读取、rg 搜索、aiw fd --help；运行 aiw fd new 创建 FD，并 claim Planner 事件 FD-020-000002-design-requested，Session 为 fd020-planner-20261003-a7c92d。
- 本轮读取既有 FD、稳定规格、配置/HTTP/存储/执行路径和 compile.py，确认只使用标准库与既有互斥锁；保留工作区已有其它变更。
- 一次实现后只读静态检查覆盖 observation.go、server.go、config.go 及存储/执行连接点和文档契约。随后补充独立请求 manifest，静态确认已升级目录缺失不再被误认成旧存储。
- 实际运行 `python program/agent-gateway/scripts/compile.py`，退出码 0，Windows/Linux amd64 编译通过；脚本设置 GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local、CGO_ENABLED=0，输出到 os.devnull，不保留发布产物。
- 未运行服务、测试、运行时验证、最终产物构建、格式化、lint、vet、网络请求或 Git 写操作。
- 交付证据：reports/FD-020-implementation-r1.md 及同名 JSON；Worker source event 为 FD-020-000004-design-ready，Session 为 fd020-worker-20261003-bf319e。
- Revision 6：按用户明确决定移除 KeyHashes、SHA-256、hex 及摘要回退；改为原字符串重复检查。静态追踪 strictDecode 的未知字段拒绝、Key 长度校验、禁用主体和直接匹配分支，并确认用户 gateway.json 已为 keys，未读取或修改凭据值。
- 本轮再次执行一次 `python program/agent-gateway/scripts/compile.py`，退出码 0，Windows/Linux amd64 离线编译通过，无保留产物；只做一次实现后只读静态检查，未运行测试、服务、最终产物构建或网络。
- Revision 6 证据为 reports/FD-020-implementation-r2.md 及同名 JSON；r1 保留为历史。本轮未认领 Tester、未写 Tester/PM 结果、未创建虚假交接；旧待认领事件不再绑定当前 FD。

## Sources

- Issue: none

- 用户本轮要求：完善日志、跟踪和统计，并质疑 Key 摘要的必要性。
- openspec/specs/agent-proxy/spec.md
- docs/features/archive/FD-017/FD-017_GO_SHARED_AI_GATEWAY.md
- docs/features/FD-018_AGENT_GATEWAY_PLUGIN_ENTRY.md

**Completed:** 2026-10-04
