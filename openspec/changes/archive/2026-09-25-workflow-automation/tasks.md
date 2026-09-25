# 自动化开发工作流

## Goal

实现可恢复、有证据和有界预算的 Coder—编译—Tester—测试—接受闭环，以及 Task memory、知识、通知与独立 Verifier。

## Scope

来源：[Requirement Plan](../../../docs/requirements/REQ00001-workflow-automation/requirement-plan.md)、[验收追踪](../../../docs/requirements/REQ00001-workflow-automation/acceptance-traceability.md)、[工程交接](../../../docs/requirements/REQ00001-workflow-automation/engineering-handoff.md)。使用一个 Task/change，按 E01–E08 分组；来源旧规则与 Plan 已确认修订冲突时，以后者为准。

## 1. Confirmed Workflow

以下 E01–E08 保留已映射的工作项编号，并以本 change 的 proposal、design 和九个 capability delta specs 为实现依据。各组实施前须闭合 design 中适用的工程项；这不是新的需求审批或用户确认环节。

- [x] 1.1 E01 执行事实与基础上下文：保存每轮报告、绑定身份、验证输入与来源版本，处理失效和旧 Task 复用；追踪 SW01–SW04/SW25、AC01–AC04/AC25。
- [x] 1.2 E02 阶段、恢复与写入权：建立阶段转换、串行写入权、未知结果对账、Stop 和可恢复预算；与 E03 共设接受边界，追踪 SW05–SW08/SW14/SW33、AC05–AC08/AC14/AC33；Git 本地交付由 AI 审批，仅允许 commit、分支创建和本地 merge，禁止 push 和直接 Git 删除，cleanup 仅经 AIW。 <!-- aiw:depends-on=1.1 -->
- [x] 1.3 E03 独立 Tester 与测试闭环：以需求构造测试，按 AI 批准的计划执行、判定风险并回交修复，凭当前证据接受；授权记录使用 .ai/<task-id>/grant.md 及已确认字段，追踪 SW10–SW14、AC10–AC14、AX01/AX03。 <!-- aiw:depends-on=1.2 -->
- [x] 1.4 E04 模型路由与预算：按生成请求去重归因，实现每档两次有效失败、每 Actor 最多两次升级和共享六次修复上限，保留跨重启预算；追踪 SW08/SW15–SW17、AC08/AC15–AC17。 <!-- aiw:depends-on=1.3 -->
- [x] 1.5 E05 辅助宿主与 Task memory：持久登记有界后台工作，保存固定输入及一次恢复账、Task 来源和 Session 投影，处理 Stop、重启和迟到结果；追踪 SW09/SW18/SW19/SW24/SW25/SW27、AC09/AC18/AC19/AC24/AC25/AC27。已补齐受控 Worker、独立宿主、生产接线与资源维护入口，见 [E05 宿主接入](e05-host-integration.md)；编译由 supervisor 负责，启用与运行验收仍待证据。 <!-- aiw:depends-on=1.4 -->
- [x] 1.6 E06 知识生成、审阅与注入：实现异步提取、部分覆盖、版本化五态审阅、读取时复核、确定性选择及历史处理限额；追踪 SW20–SW27、AC20–AC27、AX05。实现证据见 [E06 实施记录](e06-implementation.md)；编译由 supervisor 负责，运行验收仍未执行。 <!-- aiw:depends-on=1.5 -->
- [x] 1.7 E07 文本通知：通过 aiw-notify 支持 console 与调用 send_teams_msg.py 的 Microsoft Teams 方式；仅记录已尝试/失败，明确网络失败后等待 1 分钟（60 秒）后最多重发一次，其他情况不自动重发；追踪经政策修订的 SW28–SW30、AC28–AC30、AX04。实现与验证缺口见 [E07 实施记录](e07-implementation.md)；编译由 supervisor 负责，运行验收未执行。 <!-- aiw:depends-on=1.5 -->
- [x] 1.8 E08 独立 Verifier：固定需求、diff 和验证证据版本，异步只读审查并记录 failed/inconclusive/passed，不阻塞接受、交付或 Task 完成；追踪 SW19/SW27/SW31/SW32、AC19/AC27/AC31/AC32。实现与验证缺口见 [E08 实施记录](e08-implementation.md)；编译由 supervisor 负责，运行验收未执行。 <!-- aiw:depends-on=1.5 -->

## 2. Verification

- [x] 2.1 细化并检查 SW01–SW34、AC01–AC34、AX01–AX05 与 E01–E08 的覆盖，采用 Plan 已确认的授权与通知修订。见 [覆盖检查](coverage-review.md)；仅完成静态映射，全部运行场景仍待证据。 <!-- aiw:depends-on=1.1,1.2,1.3,1.4,1.5,1.6,1.7,1.8 -->
- [x] 2.2 Record existing execution evidence and failed history: [index](closeout-evidence.md). User approved A: remaining runtime acceptance deferred, not passed; no product grant fabricated. See [decision](closeout-decision.md). <!-- aiw:depends-on=2.1 -->

## 3. Verification

- [x] 3.1 Confirm the change meets the approved scope. 用户 A：开发交付范围，剩余运行验收延期；见 [收尾检查](closeout-scope-review.md)。 <!-- aiw:depends-on=1.1,1.2,1.3,1.4,1.5,1.6,1.7,1.8,2.1,2.2 -->
- [x] 3.2 Confirm there are no unrelated changes. 已核对共同基线、工作区 diff 与未跟踪范围；develop 后续提交不算本任务删除，见 [收尾检查](closeout-scope-review.md)。 <!-- aiw:depends-on=3.1 -->

## TODO

- [x] 收尾范围与变更归属核对完成：按用户 A 接受开发交付，运行验收延期；不自动合并、归档或清理。

- [x] 收尾 A：2.2 已通过 AIW reopen/complete 完成，原授权 Gate 已 waived；延期不记为测试通过。
- [ ] 最终范围归属：核对分支共同基线及历史差异，见 [收尾检查](closeout-scope-review.md)，完成后处理 3.1/3.2。

- [x] 收尾决定：用户明确跳过 E08 真实接受封存集成验证，记为未验证。
- [ ] 按 [收尾范围提案](closeout-decision.md) 确定剩余运行验收是否延期；不再扩展可选测试。

- [x] E08 接受来源登记：修复测试基线后原两项全部 passed，见 [结果](verification-results/e08-registration-terminal-20260925T045038918158Z/report.md)；保留失败历史，不代替真实接受封存。

- [x] E07 配置快照与请求边界：用户终端三项全部 passed，见 [结果](verification-results/e07-preflight-terminal-20260925T044622374590Z/report.md)；输入及日志哈希一致。

- [x] E07 config：四项全部 OK；Python 3.12.13，见 [结果](verification-results/e07-config-terminal-20260925T043543064108Z/report.md)。
- [x] E07 protocol：四项全部 passed；Go 1.25.1，见 [结果](verification-results/e07-protocol-terminal-20260925T043800322332Z/report.md)。
- [x] E07 process：三项和两个子场景全部 passed；Go 1.25.1，见 [结果](verification-results/e07-process-terminal-20260925T044205271177Z/report.md)。



- [ ] E07 真实配置取证：已准备 [四项配置计划](e07-config-verification-plan.md) 与 run-e07-config-terminal.py；用户提供 D:\green\python3.12，已确认 python.exe 存在，待执行取证。本轮离线生产 Go 编译退出 0，不代表 Python 测试通过；不修改默认 PATH。

- [x] E07 适配器离线取证：用户正常终端四项 OK、退出 0，见 [结果](verification-results/e07-adapter-terminal-20260925T042827204819Z/report.md)；网络/发送子进程均为替身，真实配置解析及 Go 全链仍待验证。
- [ ] E07 配置运行条件：本组使用 Python 3.9.13，project_config 被替身替代，尚未验证 tomllib 解析器条件；后续先核验配置入口，再推进 Go 适配器协议，不自动安装依赖。

- [x] E06 登记与持久发布：用户正常终端两项全部 passed、退出 0，见 [结果](verification-results/e06-registration-terminal-20260925T042234944157Z/report.md)，覆盖临时 Store 登记、重启去重及草稿持久版本；完整接受链和真实宿主仍待验证。

- [x] E06 提取/部分草稿/版本：用户正常终端三项及六个子场景全部 passed、退出 0，见 [结果](verification-results/e06-generation-terminal-20260925T041741918570Z/report.md)；后台登记、真实 Store 发布及宿主仍待验证。

- [x] E05 共享保存恢复：用户正常终端三项全部 passed、退出 0，见 [结果](verification-results/e05-save-recovery-terminal-20260925T041224728311Z/report.md)。覆盖构造的持久中断边界下共用额度及落盘对账，不证明实际磁盘故障。

- [x] E05 持久化取证：用户正常终端三项全部 passed、退出 0，见 [结果](verification-results/e05-durable-terminal-20260925T013046528023Z/report.md)。覆盖重启/共享恢复、队列游标、迟到摘要/Stop 的局部行为，真实宿主和完整验收仍待证据。
- [x] E06 聚焦取证：用户正常终端三项及两个子场景全部 passed、退出 0，见 [结果](verification-results/e06-knowledge-terminal-20260925T013754348449Z/report.md)。异步提取/部分草稿、真实计量和完整边界仍待验证。
- [x] E08 聚焦取证：fixture 修复后原三项及 22 个子场景全部 passed、退出 0，见 [当前结果](verification-results/e08-verifier-terminal-20260925T031341466094Z/report.md)。保留 [失败与修复记录](verification-results/e08-verifier-terminal-20260925T015608050801Z/report.md)，不宣称完整 E08 验收。

- [x] E05 来源/固定输入取证：用户 confirm 后执行 run-e05-source-terminal.py 一次，三个契约测试及四个子场景全部通过，见 [结果](verification-results/e05-source-terminal-20260925T005720742474Z/report.md)。不宣称持久恢复或后台宿主已验证。
- [ ] E05 后续验证：局部队列/恢复/发布及 memory 共享保存额度已取证；真实 I/O 故障/进程崩溃、其他辅助操作业务保存失败、赞助 Task Stop、工件损坏、真实宿主与全部资源边界仍待验证。

- [x] E04 剩余取证：用户正常终端的三个用例及两个子场景全部 passed，见 [结果](verification-results/e04-remaining-terminal-20260924T093831058329Z/report.md)。保留各轮输入版本，未重复此前两项。

- [x] E07 通知状态机取证：用户正常终端四项及五个子场景全部 passed，退出 0，见 [结果](verification-results/e07-terminal-20260924T093425416837Z/report.md)。未发送真实通知，不改变 E05 生产依赖。

- [x] E01 输入/报告取证：用户正常终端运行五项全部 passed，退出 0，见 [结果](verification-results/e01-terminal-20260924T092824207444Z/report.md)。本组无修复或重跑，完整 E01/AC 验收边界仍保留。

- [x] E03 授权边界取证：用户批准后原三个用例首跑全部通过，七个日志解析子场景也通过，退出 0；见 [证据与边界](verification-results/e03-terminal-20260924T091624949037Z/report.md)。归属 wi-0012 / 2.2，不提前派发整体验收。

- [x] 修复 E02 测试数据：正常终端已通过尾部恢复用例；另外两项因冻结输入遗漏 AISelection 被正确拒绝，现已补齐并与 StageRequest.Model 共用同一选择，见 [终端结果与修复](verification-results/e02-terminal-20260924T073650196536Z/report.md)。
- [x] 修复后验证：用户在正常终端重跑原三项，全部 passed，退出 0；已保存 [当前版本结果](verification-results/e02-terminal-20260924T074441498470Z/report.md)，冻结模型选择的 fixture 修复得到验证。

- [x] 正常终端对照取证：用户执行脚本后未复现 EvalSymlinks 拒绝，已取得一项通过与两项 fixture 失败；工具环境限制仍未改变，后续聚焦运行继续使用正常终端，见 [实际结果](verification-results/e02-terminal-20260924T073650196536Z/report.md)。

- [x] 迁移错误定位：为 MigrateDurableExecution 的十四处下层错误补充步骤上下文，保留原错误链与协议；离线编译通过，见 [诊断实现](e02-migration-diagnostics.md)。运行根因仍未确认。

- [x] 持久恢复聚焦取证：三个 E02 用例在正常终端全部通过，见 [结果与边界](verification-results/e02-terminal-20260924T074441498470Z/report.md)；工具环境权限失败和此前 fixture 失败保留为历史记录，完整验收仍未完成。

- [x] 恢复回归验证：用户批准两个指定用例；修复已有测试源码的两处编译错误后，同命令唯一重跑均通过，见 [执行报告](verification-results/recovery-regression-20260924T021434Z/report.md)。完整 2.2 及 AC/AX 缺口仍保留。

- [x] 首轮验证（2026-09-24）：用户批准两个 E04 预算用例；实际首跑均 passed，证据见 [聚焦验证结果](verification-results/e04-budget-20260924T010710Z/report.md)。此项替代下方“首轮提案待批准”这一历史状态，不改变其他验证范围。
- [ ] wi-0012 剩余验证：按 [剩余验证顺序](verification-remaining.md) 补齐完整 AC/AX 的计划、授权和实际证据；E01–E08 局部结果不能关闭 2.2/3.1 或解除整个验证 Gate。

- [x] wi-0012 首轮验证提案已获批准并执行，两个 E04 用例通过，见 [2026-09-24 聚焦验证计划](verification-plan-20260924.md)；完整验收仍未完成。

- [ ] wi-0012 / authored 2.2：E01–E08 已有按各组限定范围执行的局部证据；剩余验证须取得对应范围的授权并保存计划、输入及实际结果。产品 AI 授权规则不自动扩展工程会话权限；不以局部通过解除整个 Gate 或推进 3.1/3.2。
- [x] 补齐恢复路径修复：直接 turn/takeover 选择新的已映射工作项前同步清单，保留当前 Attempt 的复用路径；新增两个依赖恢复回归用例，见 [修复与验证计划](recovery-regression-plan-20260924.md)。运行验证状态以该计划及实际结果为准。
- [x] wi-0011 / authored 2.1：补齐 34 对 SW/AC、五项 AX、九个 capability 与 E01–E08 的覆盖表及待取证判据；明确 AI grant、受管本地 Git 和通知 60 秒一次网络重发的修订。见 [覆盖检查](coverage-review.md)。
- [ ] 覆盖检查后续：2.2 依授权登记当前版本的实际命令、计划/grant、执行结果与缺口；3.1/3.2 保持未完成。编译由 supervisor 负责，全部 AC/AX 未执行。

- [x] E08 / wi-0010：接受快照、准确 diff 基线、覆盖清单、异步只读 Worker 和三态报告已接入 E05；更新 authored 1.8，详见 [E08 实施记录](e08-implementation.md)。
- [ ] E08 Verification：supervisor 编译；AC19/AC27/AC31/AC32 的运行验收、真实模型调用和联合启用证据仍待补齐。

- [x] E07 / wi-0009：完成文本通知、逐消息网络重试账、严格 v2 协议、Teams 受管入口和 E05 宿主接线，详见 [E07 实施记录](e07-implementation.md)。
- [ ] E07 完整 Verification：已有通知状态机四项通过证据；实际适配器、TLS/HTTP 分类、宿主与 Teams 运行证据仍缺，不将替身结果标记为完整 AC28–AC30 / AX04 验收通过。

- [x] E06 / wi-0008：接入异步提取/汇总、五态版本审阅、失效复核、人工原文候选、Coder/Tester 确定性注入及有界历史补录；复用 E05 来源/恢复/资源账。详见 [E06 实施记录](e06-implementation.md)，不改变 Task/Gate 或默认 schema。
- [ ] E06 启用与验证：supervisor 编译；接收模型完整输入计量、辅助授权和资源盘点具备后，另行授权 AC20–AC27 / AX05 的聚焦验证。

- [x] 修复 RepairWorkflowChecklist 遗漏 DependsOn 的转换，避免一次性 repair 清空持久依赖；历史错绑 Attempt 保留，不计作 E05 或整体验收完成。源码修复仅在本 worktree，已安装 AIW 未更新。

- [x] E05 / wi-0007 已补齐 Worker、helper 入口、生产注册与受管盘点实现，见 [本轮接入记录](e05-host-integration.md)。正式 1.5 标记实现完成；下游工作项与运行验收保持未完成，不修改 Core/Gate。

- [x] E05 / wi-0007 本轮核心组件：来源事实、消费游标、固定输入恢复账、项目资源预约、有界宿主循环、Task memory 条件发布与 Session 来源投影、cleanup 来源封存接缝。此项仅记录部分进度，不代表正式 1.5 完成。
- [x] E05 / wi-0007 接入：受控 HTTP Worker 使用本地能力证据、完整输入上界与输出限制；独立 helper 重建配置并对账原请求日志；生产 Store/Session 接线及资源政策/盘点/存储结算入口已实现。具体模型能力未知时 waiting，网络结果不明时保留 unknown 及预约，未创建实际启用配置。详见 [E05 宿主接入](e05-host-integration.md)。

- [x] E04 / wi-0006 实现请求去重预算、实际模型档位升级与共享六次修复限制，补齐 E02/E03 服务组装和旧账 unknown 保留；详见 [E04 实施记录](e04-implementation.md)。默认 schema 9 不变，联合启用与 AC/AX 验证仍待证据。

- [ ] 整体范围验收（清单 3.1 / wi-0003）：等待 E05–E08 实现、E01–E04 联合启用及当前输入版本的验收证据，再确认整个 change 满足批准范围。E03 的规范映射为 1.3 / wi-0005；保留历史 Attempt 归属，依赖以 aiw:depends-on 同步到 Core；临时全局暂停 Gate 仅在依赖落账后解除，3.1 保持未完成且不能越过依赖派发。

- [x] E03 实现 grant 服务、固定/范围测试计划、独立 Tester/受控 Runner、诊断回交和当前证据接受检查；更新 1.3。详见 [E03 实施记录](e03-implementation.md)。编译交由 supervisor，联合启用与 AC/AX 验证仍未完成。

- [x] E02 本轮 Core、受管执行与交付接缝已实现；下方 E02–E04 联合启用事项仍未完成，详见 [E02 实施记录](e02-implementation.md)。

- [x] 根据已批准需求替换 proposal/design/spec 通用示例；九个 capability delta specs 覆盖 SW01–SW34 及对应 AC01–AC34，并落实 Plan 已确认的授权、Git 和通知政策。
- [x] 闭合 [design.md](design.md) 的 R1 设计：用户选择 A，固定本机磁盘、现有 Store、平台持久替换、系统锁/遗留锁迁移、崩溃恢复和旧记录兼容；实际实现及验证仍由对应工作项完成。
- [ ] R1 联合启用：E01 工件/输入、E02 持久提交/恢复、E04 去重预算及 E05 宿主接入已实现；启用前仍须取得平台、模型能力、资源盘点与实际验证证据。
- [x] E01 初步接入精确 Task/Work Item/Attempt/Session/turn 结果校验及 Task 原始报告归档；同身份不同内容拒绝覆盖，重复观察复用同一报告。
- [x] E01 实现来源正文/输入版本清单、结构化报告与一次只读补交、相关内容变化失效及旧结果适用性；代码与接缝见 [E01 实施记录](e01-implementation.md)。遵守 R1 一致启用边界：schema 9 不启用新接受/预算链，E02–E04 完成迁移与接受后共同启用；AC 场景仍未执行。
- [x] 闭合 R2 设计：见 [R2 工程决策](r2-authorization-design.md)，固定 grant 编码/解析、计划绑定、执行前检查、旧授权兼容及 AIW cleanup 接缝；不表示相关代码或运行验证完成。
- [ ] E01–E04 接缝已实现：E02 持久提交/恢复、E03 授权/Runner 与 E04 请求预算可组装；完成平台、真实宿主和联合接受验证前不切换 schema 10，cleanup 另等来源封存证据。
- [x] 闭合 R3 设计：见 [资源政策](r3-resource-design.md)，固定 token/字节、历史/Task 累计、队列/宿主和存储上限；模型能力未知时停止相关辅助调用，不假设默认 Profile 的容量。
- [x] 闭合 R4 设计：见 [通知适配](r4-notification-design.md)，固定显式配置、版本化协议、环境凭据、TLS、错误分类、60 秒一次网络重发及旧回执兼容。
- [ ] E05 实现 R3 资源/来源账与有界宿主；E06/E08 复用该账，不另起累计额度；E07 接通 v2 适配及受管 Teams 入口。具体能力和外发配置未满足时保留局部等待。

## Verification Record

2026-09-25 收尾：用户明确跳过 E08 真实接受封存集成验证；未标为通过。已检查 AIW diagnose，历史授权 Gate 仍开放；其他缺口的延期边界见 [收尾提案](closeout-decision.md)。

E08 登记修复后原两项全部通过，347 项输入与原始证据哈希一致，见 [当前结果](verification-results/e08-registration-terminal-20260925T045038918158Z/report.md)。本轮仅登记结果，未运行测试或编译，保留整体验证 Gate。

E08 登记首跑一项通过、一项测试断言失败：主动修改标题后错误比较旧基线。已改为登记前后完整状态比较，生产逻辑未改；原两项待重跑，见 [失败与修复](verification-results/e08-registration-terminal-20260925T044930011542Z/report.md)。

本轮继续 wi-0012：新增 E08 登记测试；接受事实和快照为临时 Store fixture，不宣称真实接受封存已验证。

E07 配置快照与请求边界三项全部通过，Go 1.25.1；347 项输入及证据哈希已核对，见 [结果](verification-results/e07-preflight-terminal-20260925T044622374590Z/report.md)。本轮仅登记文档，未重跑测试或编译，保留整体验证 Gate。

本轮修复 E07 中文报告编码并新增配置快照、禁用和请求限额测试；保留原始证据文件，不重跑此前用例。

E07 config：四项全部 OK；Python 3.12.13，见 [结果](verification-results/e07-config-terminal-20260925T043543064108Z/report.md)。

E07 protocol：四项全部 passed；Go 1.25.1，见 [结果](verification-results/e07-protocol-terminal-20260925T043800322332Z/report.md)。

E07 process：三项和两个子场景全部 passed；Go 1.25.1，见 [结果](verification-results/e07-process-terminal-20260925T044205271177Z/report.md)。

上述为限定范围通过证据，未完成整体验收；本轮修复中文记录编码，保留原始证据。






2026-09-25 E07 配置续作：核对实施记录明确 Python 3.11+、Go dispatcher 使用 PATH python；当前命令定位只有 C:\Python39\python.exe 和 WindowsApps 别名，所查注册项未发现其他版本。新增四项实际临时 aiw.toml 配置测试及带版本前置检查的 runner，无生产逻辑修改。用户已被询问合格解释器路径；未执行本组，不猜测其他位置或自动安装。沿用直接补丁回退，无 Git 写入。

2026-09-25 E07 离线结果：用户 e07-adapter-terminal-20260925T042827204819Z 四项 unittest 全部 OK、退出 0、输入期间未变。正常测试日志在 stderr，stdout 为空。新增报告并更新局部 TODO，明确 Python 3.9.13 下真实 tomllib 配置解析尚未覆盖；不将替身传输视为真实发送。仅更新文档与核对证据，不重复测试/编译、调用网络或操作 Git，2.2/3.1/3.2 与 Gate 保留。

2026-09-25 E07 适配器续作：按 R4 受管协议新增四项 Python 测试，覆盖错误类型分类、TLS/重定向策略、冻结目标/协议及凭据传递。仅导入 managed 入口并使用传输/进程替身，无生产代码修改；离线 python scripts/compile.py 退出 0，仅覆盖生产 Go，不覆盖 Python 测试。不执行旧 CLI、真实网络/通知或 Git 写入。本组未运行，沿用直接补丁回退；实际 TLS/服务与 Go 全链保留缺口。

2026-09-25 E06 登记结果：用户 e06-registration-terminal-20260925T042234944157Z 两项全部 passed、退出 0、执行期间输入未变。登记报告并更新局部 TODO/剩余缺口，不将 fixture 接受事实和临时 Store 扩展为完整接受链或真实模型证据。本轮仅文档和原始证据核对，无代码修改、重复测试/编译或 Git 写入；2.2/3.1/3.2 与 Gate 保留。

2026-09-25 E06 登记续作：新增两项 Windows 临时 Store 测试，构造接受来源后调用真实登记/观察/发布入口，验证重启去重及部分草稿持久版本。接受链和真实宿主明确在范围外，无生产逻辑修改；离线 python scripts/compile.py 退出 0，不覆盖新增测试源码，尚未执行测试。沿用直接补丁回退，无 Git 写入。

2026-09-25 E06 生成结果：用户 e06-generation-terminal-20260925T041741918570Z 三项及六个子场景全部 passed、退出 0，执行期间输入未变化。登记结果报告、更新局部 TODO 和剩余缺口；不将内存发布转换扩展为异步登记或真实 Store/宿主证据。本轮仅文档及证据核对，未修改代码、重复测试/编译或操作 Git，2.2/3.1/3.2 与 Gate 保留。

2026-09-25 E06 生成续作：仍选 wi-0012 / 2.2，按 SW20/SW21/SW23 新增三个内存状态测试，覆盖无新增契约、完整部分覆盖、草稿不可变及生成版本不继承确认；直接调用既有校验和发布转换，不改生产逻辑。离线 python scripts/compile.py 退出 0，不覆盖新增测试源码。后台登记及真实 Store/宿主路径保留缺口。本组测试尚未运行；沿用直接补丁回退，没有 Git 写入。

2026-09-25 E05 共享保存恢复结果：用户 e05-save-recovery-terminal-20260925T041224728311Z 三项全部 passed、退出 0、输入期间未变。新增结果报告，更新局部 TODO 与剩余缺口，保留构造中断状态和真实故障的区别。本轮仅登记文档及核对证据，不重复运行测试或编译，不修改生产代码或 Gate；2.2/3.1/3.2 保持未完成。

2026-09-25 E05 共享恢复续作：仍选 wi-0012 / 2.2。核对 PublishAuxiliaryOutput、内容寻址保存及 Store 提交顺序后，新增三项 Windows 临时 Store 测试；通过受控 fixture 持久重建发布预约/输出落盘的中断边界，再调用真实发布入口。未修改生产代码或增设故障钩子；离线 python scripts/compile.py 退出 0，不覆盖测试源码。真实 I/O 失败与进程崩溃保持待证，未运行本组测试。沿用 aiw patch 内部 Git apply 不可用时的直接补丁回退，无 Git 写入。

2026-09-25 E08 重跑结果：用户 e08-verifier-terminal-20260925T031341466094Z 退出 0，三项及 22 个子场景全部 passed；输入期间未变，原始计划/清单/输出哈希已核对。将 E08 局部 TODO 标完成并保留首次失败与 fixture 修复历史；补充剩余验证排序，纠正“仅两个 E04 用例”这一过时当前描述。仅更新文档，无新增代码或测试运行，不重复编译，不改 Gate/Attempt/lease，不执行 Git 写入。2.2/3.1/3.2 仍未完成。

2026-09-25 E08 首跑诊断：用户结果 e08-verifier-terminal-20260925T015608050801Z 退出 1。结论规则组十个子场景通过；固定报告及两个负面发布子场景均报 null is not a contract value。核对 strictJSON 和 Verifier 数组契约，定位 verifierTestRow 未初始化空切片。仅初始化 Evidence/Gaps 为 []，不改生产解析或断言；修复后离线 python scripts/compile.py 退出 0，不覆盖测试文件。失败原始证据保留，等待正常终端原三项一次重跑，不扩大授权或清除 Gate。

2026-09-25 E06 结果与 E08 续作：用户返回 e06-knowledge-terminal-20260925T013754348449Z，三项及两个历史额度子场景全部 passed、退出 0、输入未变化。保留冻结计划/输出并新增结果报告。继续同一 wi-0012，按 E08 固定输入及 report-only 规格新增三项 Windows 测试和单次取证入口；离线 python scripts/compile.py 退出 0，仅覆盖生产代码。未修改生产逻辑、当前项目账本或 Gate，E08 本组尚未执行；沿用直接补丁回退，无 Git 写入。

2026-09-25 E05 持久化结果与 E06 续作：用户返回 e05-durable-terminal-20260925T013046528023Z，三项测试全部通过、退出 0、输入期间未变。保留原始计划与输出，新增结果报告。继续同一 wi-0012，按既有 E06 规格新增三项 Windows 临时 Store 测试及取证入口，无生产逻辑修改；离线 python scripts/compile.py 退出 0，不覆盖测试源码。E06 本组尚未执行，授权不从 E05 扩展。完整 AC/AX 与 E08 等缺口保留；沿用直接补丁回退，未执行 Git 写入。

2026-09-25 E05 持久化续作：仍选 wi-0012 / 2.2。依据现有规格在 Windows 测试文件中新增三项真实临时 Store 用例，授权/能力/盘点/容量明确使用测试替身，观察工件内容由回调复核。准备冻结输入及输出的单次脚本；离线 python scripts/compile.py 退出 0，仅编译生产代码，新增测试尚未执行。未改生产代码或实际项目资源账，未扩大既有测试授权。模型与保存恢复共享、崩溃窗口、真实宿主及其余硬限额保留未验证；沿用直接补丁回退，没有 Git 写入。

2026-09-25 E05 聚焦结果：用户 confirm 授权后在既有 worktree 执行 run-e05-source-terminal.py 一次，证据 e05-source-terminal-20260925T005720742474Z，退出 0，三项及四个子场景全部 passed。Go 1.25.1 windows/amd64；离线、-vet=off，执行期间输入未变。保留冻结计划、输入清单与原始输出；授权来自本会话，不伪造产品 Runner grant。本轮仅登记文档，未改生产代码、扩大测试、运行真实宿主/模型/网络或 Git 写入；2.2 与 Gate 保留待完整证据。

2026-09-24 E04 后续结果与 E05 续作：用户 e04-remaining-terminal-20260924T093831058329Z 原三项及两个子场景通过，退出 0。按既有 E05 规格新增来源实质变化、跨 sponsor 标识和超限全文三个测试，无生产代码修改。离线 compile.py 退出 0，不覆盖测试源码；准备单次取证脚本。未执行 E05 测试、网络、真实宿主或 Git 写入；通过记录与未执行范围分别保留。

E07 结果登记后，静态核对 E04 尚未执行的三个已有用例及其内存/JSON/路由替身边界，准备精确计划和单次终端取证脚本；没有执行本组或扩大到全包，没有修改生产代码或重复编译。

2026-09-24 E07 通知结果：用户提供 e07-terminal-20260924T093425416837Z，四个用例及五个子场景均 passed，退出 0，输入未变。核对原始输出、计划和输入摘要；本组无源码修复、重跑或真实通知调用。同步局部清单并保留完整 Gate，见 [E07 证据](verification-results/e07-terminal-20260924T093425416837Z/report.md)。

2026-09-24 E01 输入/报告结果：用户提供 e01-terminal-20260924T092824207444Z，原五项全部 passed，退出 0，运行期间输入未变。保存结果报告并核对输入、计划和输出摘要，未重复测试、编译或修改生产代码，不解除整体 Gate。详见 [E01 证据](verification-results/e01-terminal-20260924T092824207444Z/report.md)。

继续准备 E07：静态核对 notification_test.go 的临时 Store、替身发送器、冻结目标、重试/未知结果断言和 R4 规则，准备固定四项的终端取证入口。仅更新验证工件，没有执行本组测试、通知程序、网络、生产迁移或编译；结果待另行执行后登记。

2026-09-24 继续 wi-0012：E03 授权用例通过后，选择 E01 输入/报告的五个已有用例，核对内容发现、字段状态、turn/内容绑定、旧证据与工具链适用性断言。准备固定范围正常终端取证脚本及计划，避免在已知路径受限的工具环境重复试跑。仅变更 worktree 验证工件；未运行新测试、编译、网络、Git 写入或迁移真实 Task，Gate 保留。见 [E01 聚焦计划](e01-input-verification-plan.md)。

2026-09-24 E03 授权聚焦验证：用户 confirm 后执行 run-e03-terminal.py 一次，固定三个用例全部 passed，七个日志解析子场景均 passed，退出 0。工具环境本次未触发路径权限失败；保存批准计划快照、输入清单、环境/时间/退出码、JSON 输出及会话授权来源。无源码修复、重跑、额外编译或范围扩展；不生成产品 grant、不解除 Gate。详见 [E03 执行报告](verification-results/e03-terminal-20260924T091624949037Z/report.md)。

2026-09-24 转入 E03 授权验证准备：继续使用既有 Task/worktree，核对 wi-0012 Gate 与 3.1/3.2 依赖，选取 grant_log_test.go 的三个已有授权用例；静态核对日志解析、拒绝/替代与目标变更断言。新增精确计划及取证脚本，尚未运行本组测试；不重开 E03 实现、不新建 Attempt、不解除 Gate、不改生产授权行为。见 [E03 聚焦计划](e03-grant-verification-plan.md)。

2026-09-24 E02 修复后终端结果：用户提供 e02-terminal-20260924T074441498470Z，实际退出 0；原三个用例均 passed，stdout/stderr 与输入摘要按原记录核对，执行期间输入未变。更新局部 TODO、覆盖记录与报告，不重跑测试或编译；未更新安装版、解除 Gate 或完成整项验收。详见 [通过证据](verification-results/e02-terminal-20260924T074441498470Z/report.md)。

2026-09-24 E02 正常终端结果：已核对用户提供目录的原始记录，原三项中尾部恢复 passed，另两项因 frozen Actor input differs 失败。静态定位 fixture 未设置 AISelection，而请求 Model 非空；仅在 worktree 补齐共享选择，保留全部断言与生产校验。离线 compile.py 退出 0，不覆盖修改后测试源码；尚未重跑。已保存 [结果与修复依据](verification-results/e02-terminal-20260924T073650196536Z/report.md)，未解除 Gate 或接受整项。

2026-09-24 E02 根路径诊断：用户授权继续处理，增加根/目标/正文错误上下文，离线 compile.py 通过。原三项测试首跑与仅调整子进程 TMP/TEMP 到 worktree 后的一次重跑均退出 1，定位到根路径 EvalSymlinks 的 Access is denied；两次源码摘要一致且运行期间输入无变化。保留全部证据，未变更 ACL、提权、联网或迁移真实 Task。已准备未执行的正常终端取证脚本，详见 [根路径诊断记录](verification-results/e02-recovery-20260924T025737Z/path-diagnosis-report.md)，Gate 与完整验收缺口保留。

2026-09-24 E02 迁移内部诊断运行：用户 confirm 批准同命令一次，三个用例均在 read activation artifact 返回 Access is denied，退出 1；输入未变化，原始输出另存 migration-diagnostic-run。静态追踪 ReadExecutionArtifact/confinedFile 和本机 Go 的 EvalSymlinks/toNorm/normBase，目录查询权限为疑点而非已证实根因。没有再次测试、源码改动、重复编译、改环境、提权、网络或真实 Task 迁移；保留 Gate 和未完成清单，见 [详细证据](verification-results/e02-recovery-20260924T025737Z/migration-diagnostic-run/report.md)。

2026-09-24 E02 迁移诊断实现：用户确认后，仅在 worktree 的 MigrateDurableExecution 增加十四处分步错误包装，保留 %w 错误链、返回状态及原有调用/提交顺序。离线 python scripts/compile.py 退出 0；未运行第三次测试、真实 Task 迁移、网络、Git 写入或最终制品构建。旧失败结果保留；新的诊断信息尚无运行证据，详见 [诊断实现与下一步](e02-migration-diagnostics.md)。

2026-09-24 E02 诊断重跑：用户明确批准权限失败后的同环境同命令重跑一次，三个用例均失败，退出 1；新增错误上下文将失败缩小到 MigrateDurableExecution，之前的 fixture 创建、锁准备/获取和激活工件保存已成功。运行期间输入未改变，原始结果另存 rerun-1，未覆盖首跑；未执行第三次测试、提权、网络或真实 Task 迁移。本轮仅补录文档，未改生产代码，无重复编译；保持 Gate 与未完成项。

2026-09-24 E02 聚焦验证：获用户 confirm, continue 授权后执行计划中同一命令一次，退出 1，三个用例均在 fixture 初始化阶段 Access is denied，输入摘要未变化。保留授权、计划快照、命令/环境、时间及原始输出；补充两个 fixture 的九处错误上下文，未改变测试断言。离线 python scripts/compile.py 退出 0（不覆盖测试源码）；按权限失败规则未重跑、未提权或换环境。根因未确认，Gate 和 2.2/3.1/3.2 保留；详细结果见 [E02 首跑记录](verification-results/e02-recovery-20260924T025737Z/report.md)。

2026-09-24 wi-0012 续作：核对 Task 元数据、worktree HEAD、设计就绪度、现有 Gate 和三个 Windows 持久恢复测试。补齐精确命令、判据、临时 Store 范围和取证要求，修正当前 2.2 的过时“全部未执行”描述；历史记录保留。仅修改 worktree 验证工件，没有新增生产实现或执行测试/编译/网络/Git 写操作。E01–E08 的勾选仍仅为实现记录，2.2/3.1/3.2 保持未完成；当前授权 Gate 保留。

2026-09-24 恢复回归实际执行：用户再次确认后运行 recovery-regression-plan-20260924.md 的唯一聚焦命令。首跑 exit 1、无用例执行，发现 checklist_test.go 的四变量 range 及 archive_test.go 对 session.Store 调用 SetDelivery 的编译错误；最小修复为结构体字段访问及正确的 workflow Store，保留原断言。仅重跑一次，两个指定恢复用例均 passed、exit 0，stderr 为空，输入运行期间无变化。失败与成功原始输出、精确 argv、环境、工具链、输入摘要及用户授权保存在 verification-results/recovery-regression-20260924T021434Z/。未执行两个包的其他用例、完整 AC/AX、网络、Git 或最终制品构建；没有重新运行无关 E04 用例。安装版 AIW 尚未更新，现有 2.2 Gate 不因两项局部通过自动解除。

2026-09-24 自动修复续作：用户要求自动修正问题并继续实现。按 implement 流程修复直接接管未先同步 authored 依赖的调用路径，并新增清单 repair 保留依赖/身份/一次性语义及新 Attempt 正确绑定的回归用例。使用直接补丁工具，因为 aiw patch 依赖 Git apply 而本轮未授权 Git 写入。修复不改变既有已准备或运行中的 Attempt，不手改真实 Core/租约，不重新运行已通过且未受本次改动影响的 E04 两个测试。清理了 TODO 中过期的“未批准/全部禁止测试”说明。新的两个恢复用例属于不同测试范围，运行需要明确授权；编译与结果随后登记。

本次修复的 compile-only：禁用 Go 依赖及工具链下载，`python scripts/compile.py` 退出 0，临时制品已由脚本清理；静态检查确认调用顺序、依赖传递和清单 12 个唯一编号。新的两个回归测试尚未运行，不能以编译主程序成功代替测试源码编译或测试通过。代码修复仅在 worktree，未更新安装版 AIW、启动 Supervisor 或解除 wi-0012 的剩余授权 Gate。可审阅的精确后续命令见 recovery-regression-plan-20260924.md。

2026-09-24：用户“confirm, continue”仅批准 verification-plan-20260924.md 的两个 E04 用例。按原精确 go test 参数执行一次，关闭依赖/工具链下载、关闭 vet，使用 worktree 本地缓存；退出 0，两个明确测试均 passed，stderr 为空，输入摘要运行前后一致。保存原批准计划、会话授权、模块/源码/规格输入清单、工具链、实际命令/环境/时间和 JSON 输出。报告见 verification-results/e04-budget-20260924T010710Z/report.md。未扩大范围或重跑，未执行完整 AC/AX、最终制品构建、网络或 Git 操作；没有运行产品 Runner 或伪造 grant。保留 wi-0012 Gate 及正式 2.2 未完成；Core 仅登记本轮有限用户授权，command Evidence 不绕过其授权 Gate 校验。

2026-09-18 wi-0012 / attempt-1789728912870964500：仅完成验证缺口登记，2.2 保持未完成。静态核对 handoff、Task metadata、Core Work Item 映射、proposal、design readiness、verification delta/stable spec 和 coverage-review；当前路径及分支匹配既有 isolated Task。仅编辑本文件的 authored 2.2 说明、TODO 和本记录，保留清单依赖及其他工作区改动，不修改 Core、Gate 或 Task 状态。aiw patch 帮助已读取；该入口使用 Git apply，与本轮只读 Git 边界不符，采用直接文件补丁。

| 证据范围 | 已观察事实 | 适用性与缺口 |
| --- | --- | --- |
| 历史 wi-0011 supervisor 编译摘要 | 主仓库 `.ai/workflow-automation/compile-diagnostics/wi-0011-2026-09-18T10-55-11.4573509Z.json` 保存 adapter.kind=`plan`、name=`request-compile-plan`、command 为空、exit_code=`0`、diagnostics=`null`、recorded_at=`2026-09-18T10:55:11.4573509Z`。 | 仅转录已有摘要，不声称本轮执行编译。该文件未提供具体 argv、工作目录、完整输入摘要、冻结计划正文或 grant 引用；不能由退出码推定 AC/AX 通过或当前版本适用。 |
| 本轮 wi-0012 编译 | 本 Agent 未运行；按当前 handoff 由 supervisor 负责冻结 Compile Plan 及有界修复。 | 当前 Attempt 的编译结果尚未提供，不能代填退出码或通过状态。 |
| AC01–AC34、AX01–AX05 | coverage-review 逐项判据及历史实施记录均保留未执行；本轮未运行任何场景。 | 状态为未执行，不是 passed、failed、waived 或 N/A。实际命令/目录、阶段请求、输入版本、工具链/宿主、计划/政策/grant、本次发现清单、时间、退出码和输出引用均待受管执行取证。 |

实际命令仅为本地 Get-Content、Get-ChildItem、rg、PowerShell JSON 只读筛选、aiw patch --help，以及指定 safe.directory/-C 前缀的 git status --short --branch；编辑后仅做一次指定前缀的 git diff -- openspec/changes/workflow-automation/tasks.md 静态核对。首次中文读取显示异常后改用显式 UTF-8；历史编译摘要首次误定位到 artifacts，按实际 compile-diagnostics 路径纠正读取一次。没有运行测试、编译、最终制品构建、格式化、lint、vet、验证脚本、网络、模型、通知或 Git 写操作。

%% WI0012_AUTHORIZATION_PENDING：本轮明确禁止测试，产品的 AI 计划审批不构成工程运行授权。2.2 所需实际 AC/AX 证据不能靠文档编辑生成；由 supervisor 安排获授权的受管取证并提供精确引用后继续登记。本轮不申请扩大权限、不伪造 grant、不解除 Gate、不勾选 2.2，也不确认整体范围或无关改动检查已完成。

2026-09-18 wi-0011 / attempt-1789728545423365800：完成 authored 2.1 的静态覆盖检查；新增 [覆盖检查](coverage-review.md)，逐项关联 SW01–SW34 / AC01–AC34、AX01–AX05、九个 delta spec、E01–E08 实施记录和待取证场景。明确历史验收矩阵中的人工逐计划审批、三次通知/幂等前提按 Plan 修订解释；不修改历史来源或稳定规格。更新 2.1、TODO 与本记录，保留 2.2/3.1/3.2 未完成及 Core/Gate/Task/worktree。执行本地读取、rg、aiw patch 帮助与指定前缀只读 Git，直接文件补丁回退及一次编辑后静态核查；未运行编译、测试、构建、格式化、网络、模型、通知或 Git 写操作。supervisor 编译与全部 AC/AX 运行证据仍待提供。

2026-09-18 E08 / wi-0010 / attempt-1789727915907036500：完成接受版本封存、按固定来源派发独立 Verifier、覆盖/依据/缺口校验和报告保存，更新 authored 1.8、TODO 与实施记录。一次静态差异核查覆盖接受、资源预约、辅助来源/宿主/Worker 和发布路径；编译明确交由 supervisor，未运行测试、构建、格式化、网络、模型或 Git 写操作。保留真实 Task/worktree、Core/Gate 和其他未完成清单；1.8 勾选仅表示实现完成，详见 [E08 验证记录](e08-implementation.md)。

%% E08_ACTIVATION_PENDING：辅助能力、授权、存储盘点和 schema 10 联合启用仍须实际证据；超限/缺少固定来源显示 unavailable，不产生空 passed，不撤销接受或阻断交付。所有对应 AC 场景未执行。

2026-09-18 E06 / wi-0008 / attempt-1789725422735098500：完成实现并更新 authored 1.6、TODO 与实施记录。一次静态检查核对 Core 映射、CLI 差异、来源/队列/审阅/注入接缝；发现并修正 schema 10 条件提交与 Tester 可选来源降级问题，修正后仅按补丁文本核对。仅执行本地读取、帮助和指定前缀的只读 Git；没有运行编译、测试、构建、格式化、网络、宿主、模型或 Git 写操作。编译交给 supervisor；所有相关 AC/AX 仍未执行，详见 [E06 验证记录](e06-implementation.md)。

%% E06_ACTIVATION_PENDING：本地能力记录必须另外证明接收方完整输入计量；缺失时跳过可选知识。历史和资源缺口不自动补零/扩额、不撤销开发接受。本轮不解除 Gate、不切换 schema，也不宣称 E07/E08 或 Task 整体完成。

2026-09-18 E05 / wi-0007 / attempt-1789723715632420700：完成受控 Worker、独立 helper、生产注册、资源政策与盘点/结算入口、普通 Session/Tester 来源投影和冻结输入原始工件复核。更新 authored 1.5 与 TODO；没有修改 Task/Core/Gate、迁移 schema 或执行配置。静态证据与限制见 [E05 宿主接入](e05-host-integration.md)。实际命令仅本地读取、rg/文件定位和用户指定前缀的只读 Git；直接补丁工具写入源文件及文档。编译明确交由当前 supervisor；未执行任何测试、制品构建、格式化、lint/vet、网络、模型、通知、后台宿主或资源维护命令。所有 E05 AC/AX 场景仍未运行。

2026-09-18 恢复链路缺陷：turn 9 返回归属冲突，Session 已 active，旧 Attempt 已结束且无写租约。静态追踪确认 workflow_artifacts.go 的普通 sync 传递 DependsOn，而 RepairWorkflowChecklist 未传递；Core repair 将其解释为空依赖，且保留旧 PlanFingerprint，导致之后同指纹 sync 不重建依赖。此次补齐 repair 字段传递；当前真实运行记录通过受管 plan 强制按同一清单重建依赖，保留 ID 与历史，不手改 JSON。编译及恢复命令结果另记，未授权测试或最终制品构建。

实际验证及恢复更正：`python scripts/compile.py` 退出 0，GOPROXY/GOSUMDB 关闭、GOTOOLCHAIN=local，临时产物由脚本清理。一次编辑后静态检查确认两处转换均传递 DependsOn，同时发现安装版 `plan` 也走同指纹短路，并未如预期重建依赖。因此将 3.1 已有的传递前置条件显式写为 E01–E08、2.1、2.2 的直接依赖，语义不变，刷新清单指纹；随后 sync 退出 0。非执行 run 正确准备 E05 / wi-0007 / attempt-1789723715632420700，确认调度不再选取 3.1；repair 退出 0，为该原请求补齐编译计划。此前一次性 checklist repair 已登记完成，后续 repair 不再重复清单导入。未从工具环境派发模型请求，Session 保持可运行状态；由用户终端的 Supervisor 消费此已准备请求。源代码修复尚未安装到 C:/green/aiw/aiw.exe；不能将当前状态修复说成已安装版本升级。未运行测试、最终制品构建、格式化、网络或 Git 写操作。

2026-09-18 E05 / wi-0007 / attempt-1789720672532536600：**INCOMPLETE（dependency）**。已加入 Core 来源/队列/资源/恢复/发布组件、宿主循环和来源封存校验，但真实受控 Worker、独立 helper 入口及生产组装尚缺，因此正式 1.5 保持未勾选。未修改当前 Task/Core/Gate/租约或默认 schema 9，未操作 Git 写入。执行一次静态命令批次，读取受影响接缝并检查已有差异；指定 safe.directory 前缀的 `git diff --check` 未报告空白错误，仅有 LF/CRLF 提示。该命令不覆盖未跟踪新文件，且后续局部修正与本实施记录未再运行检查；新文件按补丁文本静态检查。编译由 supervisor 负责，本执行者未运行编译、测试、制品构建、formatter、lint、vet、验证脚本或网络操作。所有 E05 AC/AX 均未执行。证据和继续路径见 [E05 实施记录](e05-implementation.md)。

%% E05_ACTIVATION_PENDING：受控 Worker 与独立 helper 的生产接入已补齐；真实本地能力证据、显式授权配置、凭据及历史盘点仍须在启用时满足，Profile 别名不能替代证明。编译由 supervisor 执行；不解除 Gate，不把静态实现或 schema 10 未启用视为运行验收通过。

2026-09-18 E04 / wi-0006 / attempt-1789718088692918200：实现请求预算账、固定模型路由、生成归因与 E02/E03 服务组装；1.4 勾选仅为实现完成。保留现有 Task/worktree/Core/Gates，不启用 schema 10、不改写历史 Attempt。实际验证与未执行缺口见 [E04 实施记录](e04-implementation.md)；所有 AC/AX 仍未执行。

E04 最终源码 Verification：禁用网络/工具链下载后执行 `python scripts/compile.py`，退出 0；进行一次静态批次，核对预算/路由源码、阶段接缝和清单，`git diff --check -- openspec/changes/workflow-automation/tasks.md` 未报告空白错误，仅提示既有 LF/CRLF 转换。未执行测试、最终制品构建、formatter、lint、vet 或运行验收。新增测试代码不包含在该 compile-only 检查中。

2026-09-18 wi-0003 / attempt-1789698959531607000 范围核对：**BLOCKED（dependency）**。本轮 handoff 指定 wi-0003；只读读取 Core 确认其 checklist.item 为 3.1，标题为 “Confirm the change meets the approved scope.”，不能按编号猜作 E03 或 E04。proposal 要求 E01–E08，当前 authored 1.4–1.8、2.1/2.2 未完成；design 要求 E01–E04 一致启用，既有 E03 记录仍列出生产注册、预算/迁移及运行验证缺口。因此 3.1 保持未勾选，不以局部实现或编译代替整体验收。

%% WORK_ITEM_MAPPING：2026-09-18 对账确认 Core 映射为 wi-0003 → 3.1（整体验收）、wi-0005 → 1.3（E03）、wi-0006 → 1.4（E04）。历史 E03 执行确实登记在 wi-0003 的 attempt-1789697370140599800 下，不能据此把 wi-0003 重映射为 E03，也不能把该 Attempt 的证据追溯迁移到 wi-0005。1.3 已勾选且 wi-0005 completed 仅是当前清单/Core 状态，不补造 E03 的正确归属执行或验收证据。历史报告原文保留，并附本次更正；后续受管调度须按清单语义和依赖选择 E04，不能根据 wi 编号猜测 E 序号。

2026-09-18 清单恢复 Verification：静态读取 worktree 清单与主工作区 Core，发现 TODO 的复选框再次以 3.1 开头，与正式验收项重号。将 TODO 改为描述性前缀，保留全部 12 个正式清单编号、完成标记和 Core 身份；历史 E03 记录追加映射更正，不改原 Attempt。此修复仅消除清单解析阻塞，不表示 E04–E08、联合启用或 AC/AX 已完成。后续通过 AIW sync 对账，仅在解析成功后处理重复编号 Gate；整体验收依赖 Gate 保留。未修改代码或手写 Core 状态，未运行编译、测试、构建、网络或 Git 操作。

本轮实际恢复结果：一次编辑后静态检查得到 12 个正式编号、0 个重复；`aiw task workflow sync workflow-automation` 与 `aiw task workflow gate workflow-automation checklist-reconciliation-duplicate-checklist-number resolved` 均退出 0。解析 Gate 已解除，唯一开放 Gate 为 supervised-dependency-wi-0003，Workflow 仍 BLOCKED / execution blocked；E04 / wi-0006 为 ready。已读取 workflow 帮助和 diagnose；当前帮助未提供显式按 Work Item 选择 run 的参数或依赖编辑入口。没有启动 supervisor、重开整体验收项或操作过期租约。下一步应处理受管调度的顺序/依赖表达，使 E04–E08 和验证先于整体验收；仅解除解析 Gate 不等于自动执行已恢复。两次源码定位使用了不正确的路径，未产生源码分析结论；后续以 rg --files 定位到 internal/workflow 和 internal/taskx，仅取得符号位置，不据此宣称调度根因已证实。

本轮验证仅为静态范围与依赖核对：读取 handoff、Task/Core、proposal、design、tasks、相关 capability delta 和 E03 记录；执行本地 Get-Content/Get-ChildItem/Get-Command、rg、指定 safe.directory 前缀的 git status，以及 aiw patch --help。一次默认编码的 JSON 读取失败后改用 UTF-8 成功，未修改运行记录。仅直接编辑本文件 TODO/Verification，以保持本轮 Git 只读约束；编译留给 supervisor，未运行测试、构建、formatter、lint、vet、validator、网络或 Git 写操作。尚缺剩余实现、联合启用和当前证据，建议 supervisor 先核对工作项映射并调度未完成依赖，再重新派发 3.1。

2026-09-18 E03 / wi-0003 Verification：完成授权日志/尾部锚点、不可变执行清单、独立 Session Tester、受控测试执行与输出保存、按请求恢复、一次只读归因及接受候选服务；Tester 后重新编译，Coder 不得改弱测试。新增 grant 聚焦测试代码但未运行。静态核对字段、引用、Core 条件提交与宿主调用路径；编译由 supervisor 运行冻结 Compile Plan，本执行者未运行编译、测试、最终制品构建、formatter、lint、vet、validator 或网络命令。保留真实 Task/worktree、Core 状态和 Gate；1.3 勾选仅表示本工作项实现完成。证据位置、实际命令及启用缺口见 E03 实施记录。

%% E03_ACTIVATION_PENDING：默认 schema 9 不变。E03 提供 VerificationService.Connect、JournaledVerificationHost 和受管派发/归因入口；生产注册须同时连接 E04 预算/迁移、原 Coder/Compiler 宿主、真实隔离边界与决策/断言审阅权威，并取得平台及授权/接受验证证据。缺失提供方返回 host-unavailable，不启用无约束替代执行。所有 AC/AX 仍未执行。

本次补齐的是需求推广工件，不是功能实现。SW/AC 的覆盖及来源优先级可以静态核对；AC01–AC34、AX01–AX05 均未执行，以上编号实现与验收工作项保持未完成。AX04 按已批准通知政策解释：无查询/回执/可靠幂等仍允许明确网络失败后等待 60 秒重发一次，未知结果及其他失败不自动重发。

2026-09-18：用户选择方案 A，R1 设计闭合且 E01 具备设计输入；受管同步、设计证据及 Gate 处理以 AIW 实际结果为准。已检查安装版本帮助与诊断，诊断仅显示原 R1 Gate。没有将设计闭合计作功能完成或运行验收，R2–R4 保持未闭合。

恢复命令已成功：sync、记录 `r1-design-local-disk-20260918` 静态证据、resolve 原 Gate、reopen wi-0001。E01 已 ready，执行已 queued；validation=passed 仅为设计静态证据。E01 本轮代码仅覆盖精确结果身份和逐轮原始报告保存，1.1 未勾选，不将阶段性进展视为完整验收。

代码静态核对覆盖请求生成的预期 turn、Session 输出命名、Task 报告保存和后续 outcome 引用。已执行 `python scripts/compile.py`，退出码 0；设置 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local，未下载依赖，脚本清理临时可执行文件。未运行测试、formatter、linter、OpenSpec validator 或最终制品构建；编译通过不代表恢复场景已验证。

## Notes

2026-09-18 E02 / wi-0002 Verification：完成本轮 Core 与受管执行接缝，更新 1.2 和 TODO；保留 E03–E08、全部 AC/AX 和 Task 总体验收未完成。新增 Windows 聚焦测试代码但未运行。编译由 supervisor 执行冻结 Compile Plan，本执行者未运行编译、测试、最终制品构建、formatter、lint、vet、validator、网络或 Git 写操作。静态证据与实际命令见 [E02 实施记录](e02-implementation.md)。未修改真实运行状态、锁协议、Task 元数据或 Core Gate。

%% E02_ACTIVATION_PENDING：schema 9 仍为默认；新增 schema 10 路径只有受管维护、E01–E04 服务和精确平台/授权/接受证据齐备后才能迁移。当前没有注册生产 ExecutionServices/DeliveryServices，不能启动新链。E03 接授权/结果/接受与宿主核验，E04 接迁移/预算，E05 接完整来源封存。实现勾选不代表当前代码编译成功或 AC05–AC08/AC14/AC33 已执行。

2026-09-18 E01 本轮 Verification：已更新 1.1 及对应 TODO；新增聚焦测试代码，未运行。验证由 supervisor 执行冻结 Compile Plan，本执行者未运行编译、测试、格式化、lint、vet、最终制品构建或网络命令。本轮静态检查范围为工件写入、来源加载、只读补交、内容适用性及其调用方，不将历史编译结果作为本轮结果。详细命令与启用缺口见实施记录。

%% ACTIVATION_PENDING：E01 新输入/报告接缝在 schema 10 边界后，未切换当前 schema 9；E02 仍须实现持久提交、锁协议与迁移，E02/E03 产生 Core 接受引用，E04 接入请求预算后才能一致启用。1.1 的实现勾选不代表 AC01–AC04/AC25 运行通过或整个 Task 已完成。

%% PROMOTION：保留初次创建时已映射的验证项 3.1/3.2 及原文。编号整理产生的 checklist-identifier-deleted-wi-0003/wi-0004 仅在恢复映射后按 resolved 处理，不豁免功能验收。
%% IMPLEMENTATION_PENDING：R1–R4 设计已闭合；E02–E08 实现、2.1/2.2 和 3.1/3.2 仍未完成。按依赖实施并取得当前输入版本的实际证据，不以设计静态通过替代功能通过。

2026-09-18 R2 Verification：以当前 Store、VerificationPlan、FocusedTestAuthorization、ExecuteFocusedTest 和 localMergeDelivery/resumeMergedDeliveryCleanup 为静态依据，补齐严格授权日志、计划/执行清单、真实目标与宿主复核、旧 consumed 兼容、E02/E03/E04 接受/预算责任及清理恢复场景。proposal 范围未变，稳定 specs 未改；既有编号与 E02 未完成状态保留。本轮仅设计修改，未运行编译、测试、validator、网络或 Git 操作。

2026-09-18 R2 对账：AIW diagnose 确认唯一开放 Gate 为 supervised-dependency-wi-0002。读取 Core 时 E01 Work Item 与 Attempt 已 completed，E02 Attempt 已 failed 且无写入租约；无需手工结束 E01 或释放租约。E01 当前版本的实际编译/AC 证据仍须单独核对，不将历史 validation=passed 解释为本次通过。设计完成后的 sync、静态证据及 Gate 恢复以实际命令结果为准。

2026-09-18 R2 恢复结果：`aiw task workflow sync workflow-automation`、登记 `r2-authorization-design-20260918`（wi-0002 / static-review / passed）、`gate workflow-automation supervised-dependency-wi-0002 resolved`、`reopen workflow-automation wi-0002 <R2 design closure reason>` 均退出 0。E01 authored/core 均 completed；E02 authored=open、core=ready；Workflow 为 DRAFT、execution=queued、delivery=pending，无开放 Gate。validation=passed 包含设计静态证据，不表示功能测试通过。未创建新 Attempt、启动 supervise 或执行 E02 代码。本轮结构检查确认必需工件存在、Task/change ID 一致及 12 个编号映射保留；OpenSpec CLI 曾被执行策略阻止，本轮未重试或绕过。
%% SOURCE_PRECEDENCE：初始测试与 Git 计划由 AI 授权；grant 路径和字段遵循 Plan；禁止 push 与直接 Git 删除，cleanup 仅经 AIW；通知仅明确网络失败时等待 60 秒后重发一次。

2026-09-18 R3/R4 Verification：用户授权按建议继续补齐剩余设计。静态核对 Profile 字段、默认 routing-plan、需求工件尺寸、aiw-notify/Go 调用方和 Teams 脚本的 TLS、参数解析及错误输出；未读取环境凭据或 .env。新增资源/通知附件及四个 capability 的场景，既有编号和范围保持。实际仅执行本地读取、rg、文件尺寸统计与补丁编辑；不运行编译（本轮无代码实现）、测试、validator、网络或 Git 操作。OpenSpec CLI 先前被执行策略阻止，未重试或绕过。AIW 同步与设计证据结果另记，不将旧 validation=passed 当作 AC/AX 已执行。

2026-09-18 R3/R4 同步结果：一次静态结构检查确认九个相关工件存在、Task ID 正确、12 个编号映射保留且当时无开放 Gate。`aiw task workflow sync workflow-automation` 及四次 `evidence ... static-review passed` 均退出 0：r3-resource-e05-20260918 → wi-0007，r3-resource-e06-20260918 → wi-0008，r3-resource-e08-20260918 → wi-0010，r4-notification-e07-20260918 → wi-0009。同步返回 E02 core=running、Workflow IN_PROGRESS/execution=running/delivery=pending；本轮未启动执行器、创建 Attempt 或操作租约。该状态是现有受管流程的新观察，不是 R3/R4 功能完成。设计参数和协议尚待实现，实际模型能力、Teams 配置及 AC/AX 验证缺口按附件保留。

2026-09-18 调度恢复 TODO / Verification：已静态定位到 SelectReadyMappedWorkItem 按 Core 保存顺序选取 ready 项，NextRunnerOutcome 先返回任一开放 Gate。原清单没有声明依赖，历史较早创建的 3.1/3.2 因此会先于 E04 被选择。使用现有解析器支持的 aiw:depends-on 注释补齐实现与验证依赖：E01→E02→E03→E04→E05，E06/E07/E08 依赖 E05，2.1 依赖全部 E 项，2.2→3.1→3.2 依次执行。保留原 12 个编号、ID、完成标记和历史执行归属；未修改调度源码或稳定规格。先经 AIW sync 落账并检查依赖，再将临时 supervised-dependency-wi-0003 Gate 转为持久依赖约束后解除、受管 reopen 3.1；这不接受 3.1，也不豁免验收。用户已要求 fix it and continue，后续采用单步 run，不启动包含自动 Git 交付的 supervise start。实际恢复结果另记。

调度恢复实际结果：sync 退出 0；一次编辑后静态检查确认全部依赖已持久保存，3.1→wi-0012、3.2→wi-0003，E04→wi-0005，且当时无写租约或已准备请求。随后 gate resolved、reopen wi-0003（附依赖转移原因）及 run 预备均退出 0；Core 正确选中 E04 / wi-0006，创建 attempt-1789718088692918200，3.1/3.2 仍未完成。执行 `aiw task workflow run workflow-automation --execute` 退出 1，停在工作区协调预检：`read Task worktree branch: read current branch: exit status 128`。没有证据表明该 E04 请求已实际派发；不能把预备后的 core=running 当成 Agent 已执行。当前命令未返回 Git stderr，历史所有权失败仅作可能原因；不重试等价命令、不修改信任配置或绕过权限。保留已准备 Attempt 供后续受管恢复，不手改 Core/租约。修复仅为清单依赖与记录，无源码修改，因此未编译；未运行测试、最终构建或网络命令，未执行 Git 写入。`run --help` 被 CLI 解释为 Task ID 并报路径不存在；使用已读取的 workflow 总帮助确定正式 run 语法。
