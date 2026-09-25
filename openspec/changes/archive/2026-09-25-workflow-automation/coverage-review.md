# 覆盖检查：SW / AC / AX 与 E01–E08

Task/change：`workflow-automation`；Work Item：`wi-0011`；authored checklist：`2.1`；Attempt：`attempt-1789728545423365800`；日期：2026-09-18。

本项完成需求、规格、实施记录与待验收证据的静态映射。下表覆盖 SW01–SW34、同号 AC01–AC34 和 AX01–AX05；当前已有下述局部测试证据，完整运行验收仍待补齐。实施记录中的完成标记仅表示相应实现已提交给 supervisor，不表示本表场景通过。本文不登记新的 Core Evidence、不解除 Gate，也不替代 2.2 的实际验证、3.1 的范围接受或 3.2 的无关变更检查。

## 来源与有效政策

E08 接受来源登记原两项在测试基线修复后全部 passed，见 [当前结果](verification-results/e08-registration-terminal-20260925T045038918158Z/report.md)；覆盖重启复用和历史缺快照拒绝补造，仍不证明真实 Git/receipt 封存。

E07 配置快照、缺省/禁用和请求限额三项已通过，见 [结果](verification-results/e07-preflight-terminal-20260925T044622374590Z/report.md)；局部边界证据不替代生产环境及完整验收。

E07 config：四项全部 OK；Python 3.12.13，见 [结果](verification-results/e07-config-terminal-20260925T043543064108Z/report.md)。

E07 protocol：四项全部 passed；Go 1.25.1，见 [结果](verification-results/e07-protocol-terminal-20260925T043800322332Z/report.md)。

E07 process：三项和两个子场景全部 passed；Go 1.25.1，见 [结果](verification-results/e07-process-terminal-20260925T044205271177Z/report.md)。

上述为限定范围通过证据，未完成整体验收；本轮修复中文记录编码，保留原始证据。

E04 后续三个用例及两个子场景已 passed，见 [原始证据](verification-results/e04-remaining-terminal-20260924T093831058329Z/report.md)；覆盖共享六次修复、路由快照与旧预算的局部边界。E05 新增 [三个来源/固定输入用例](e05-source-verification-plan.md)，尚未运行，不计入通过证据，也不替代持久恢复与实际宿主验收。

E07 通知状态机取证已通过：四个顶层用例及五个非网络结果子场景均 passed，见 [原始结果](verification-results/e07-terminal-20260924T093425416837Z/report.md)。仅临时 Store 和发送替身，无真实通知；提供 SW28–SW30/AX04 的局部证据，实际适配器与宿主仍待验证。

E01 输入/报告取证已通过：用户正常终端五个用例全部 passed，见 [当前输入版本与结果](verification-results/e01-terminal-20260924T092824207444Z/report.md)。提供输入变化、报告状态/身份与旧证据复用的局部覆盖，不代表完整 E01/AC 验收。

E03 授权边界取证已通过：用户批准的三个内存授权用例首跑全部 passed，七个日志解析子场景全部 passed，见 [输入版本与原始结果](verification-results/e03-terminal-20260924T091624949037Z/report.md)。范围只涉及解析、拒绝/完整替代和目标变更，提供 SW11/SW12、AC11/AC12/AX03 的局部证据，不宣称整个 E03 验收。

E02 最新有效结果：修复冻结模型选择后，用户正常终端的原三个用例全部 passed、退出 0，见 [当前输入版本与结果](verification-results/e02-terminal-20260924T074441498470Z/report.md)。提供 SW05/SW06/SW08 的局部恢复证据；下方此前失败记录保留为历史，不覆盖当前结论，也不将局部证据扩展为完整 AC/AX 通过。

E02 正常终端证据：事件尾部恢复用例已 passed，另外两项因 fixture 缺失模型选择失败；已修复测试数据，修复后尚待验证。见 [输入版本、原始结果与修复](verification-results/e02-terminal-20260924T073650196536Z/report.md)。此前工具环境路径拒绝的记录保留为历史环境结果，不再作为本次正常终端失败原因。

E02 根路径定位：用户授权继续后，默认 Temp 和 worktree 临时目录各一次相同范围运行均失败；已确认根路径 EvalSymlinks 被拒绝，见 [证据和终端取证入口](verification-results/e02-recovery-20260924T025737Z/path-diagnosis-report.md)。安全检查保持，三个用例不计入通过证据。

E02 最新定位：分步诊断后的获批运行仍为三个失败，错误位于迁移的激活工件读取边界；见 [运行与静态追踪](verification-results/e02-recovery-20260924T025737Z/migration-diagnostic-run/report.md)。路径规范化权限仅为疑点，不作为通过证据。

E02 证据增量（2026-09-24）：[持久恢复计划](e02-recovery-verification-plan.md) 的三个用例首跑失败，另获授权后的唯一诊断重跑仍失败，定位到 migrate fixture execution: Access is denied，见 [两次原始结果](verification-results/e02-recovery-20260924T025737Z/report.md)。它们不计入通过证据；迁移内部具体失败操作和根因尚未确认。

恢复回归证据增量（2026-09-24）：清单修复保留依赖、直接接管按新清单选择工作项的两个用例已通过，见 [测试报告](verification-results/recovery-regression-20260924T021434Z/report.md)。这是恢复调度边界的局部证据，不代表完整 AC/AX 或实际平台故障注入验证。

2026-09-24 证据增量：用户授权的两个 E04 预算单元用例已通过，详见 [执行结果](verification-results/e04-budget-20260924T010710Z/report.md)。它们为 SW08/SW15/SW16 提供部分证据；下表完整 AC/AX 场景仍未全部取证，不能将 JSON 往返测试等同于真实崩溃恢复，也不能把局部通过解释为全表通过。

依据：[Requirement Plan](../../../docs/requirements/REQ00001-workflow-automation/requirement-plan.md) 的“已确认政策决定”“关键业务规则与例外”，[原始软件需求](../../../docs/requirements/REQ00001-workflow-automation/software-requirements.md)，[历史验收矩阵](../../../docs/requirements/REQ00001-workflow-automation/acceptance-traceability.md)，[工程交接](../../../docs/requirements/REQ00001-workflow-automation/engineering-handoff.md)，本 change 的 [proposal](proposal.md)、[design](design.md) 和九个 capability delta。原始需求和验收矩阵保留历史正文；冲突按 Plan 已确认修订解释，不以旧条款验收当前实现。

| 适用条款 | 当前有效判据 | 不采用的解释（含历史冲突） |
| --- | --- | --- |
| SW11/SW12、AC11/AC12、AX03 | 初始测试计划由 AI 按用户政策批准或拒绝；用户调整政策。授权先可靠保存至 `.ai/<task-id>/grant.md`，至少含日期、权限、原因、是否同意、授权人类型 user/ai、申请环节，并绑定精确计划/政策/目标。范围内发现刷新执行证据；危险、未知或范围变化不能沿用旧 grant。宿主限制不能被批准豁免。 | 初始计划和危险变更一律逐次人工审批；以泛化 LOG 或旧 consumed 记录充当新授权。 |
| SW33、AC33 | AI 批准具体本地 commit、分支创建、merge 计划；禁止 push 和 Agent 直接 Git 删除。cleanup 只经 AIW、对应 grant、精确归属及来源封存检查。 | 将现有自动交付行为当作授权，或把清理例外解释为允许 Agent 直接删除。 |
| SW28/SW29、AC28/AC29、AX04 | 只记录已尝试/失败。仅第一次明确网络失败后等待至少 60 秒，最多重发一次，总发送最多两次；重启保留原 due time。未知、非网络失败、第二次任何结果均不自动重发。允许该次网络重发存在重复投递风险，不假设查询/幂等/回执。 | 最多三次及 30 秒/两分钟退避；必须有可靠幂等才允许任何重发；本地 ack 表示送达。AX04 的“无幂等不重发”仅保留在明确网络失败例外之外。 |
| SW30、AC30 | aiw-notify 提供 console 与调用 send_teams_msg.py 的 Teams 文本通道；严格受管配置、冻结目标、凭据环境引用和 TLS；兼容在插件边界处理。 | 要求受管入口继承全部旧脚本隐式默认值；以通知外发能力授权开发测试联网。 |

R1–R4 在 design 中为 FD_APPLIED；运行启用证据仍待取得。产品中的 AI 审批规则不授权本工程会话执行测试、联网通知或 Git 交付。

## 规格索引

下表简称均指本 change 的 delta，而非声明 stable spec 已同步。

| 简称 | Capability | SW / 同号 AC |
| --- | --- | --- |
| SUP | [workflow-supervision](specs/workflow-supervision/spec.md) | 01、02、05、06、07、08 |
| VER | [verification](specs/verification/spec.md) | 03、04、10、11、12、13、14、31、32 |
| ROUTE | [ai-routing](specs/ai-routing/spec.md) | 15、16、17 |
| MEM | [task-memory](specs/task-memory/spec.md) | 09、18、19 |
| KNOW | [project-knowledge](specs/project-knowledge/spec.md) | 20、21、22、23、24、26、27 |
| INPUT | [agent-session](specs/agent-session/spec.md) | 25 |
| NOTIFY | [workflow-notifications](specs/workflow-notifications/spec.md) | 28、29、30 |
| GIT | [workspace-delivery](specs/workspace-delivery/spec.md) | 33 |
| LIFE | [task-lifecycle](specs/task-lifecycle/spec.md) | 34 |

## 工程证据入口与依赖

以下文件位置来自已有实施记录及工作区文件清单，用于定位后续验证；本轮没有重新逐行审计实现，也不把实现叙述作为受控运行结果。Go 路径除特别注明外相对 `internal/workflow/`。

| 工程组 / authored 项 | 已有记录 / 实现定位 | 本项检查的衔接与剩余证据 |
| --- | --- | --- |
| E01 / 1.1 | [实施记录](e01-implementation.md)；execution_report.go、execution_inputs.go、execution_reuse.go、internal/taskx/execution_context.go | 先提供报告/来源/输入版本；受控编译、测试及下游读取必须复核同一适用版本。 |
| E02 / 1.2 | [实施记录](e02-implementation.md)；execution_protocol.go、stage_execution.go、execution_acceptance.go、execution_stop.go、managed_delivery.go | 依赖 E01，与 E03 共设接受；持久写入、在途结果、Stop、交付/cleanup 恢复仍需平台证据。 |
| E03 / 1.3 | [实施记录](e03-implementation.md)；grant_log.go、controlled_test_plan.go、tester_context.go、verification_service.go、execution/verification_host.go | 依赖 E02；独立 Tester、真实宿主约束、范围授权、失败归因与当前接受需联合取证。 |
| E04 / 1.4 | [实施记录](e04-implementation.md)；generation_budget.go、generation_routing.go、execution/generation_routing.go | 依赖 E03；新接受与请求预算一起启用，不能继续用旧三次阈值；未知旧账不补零。 |
| E05 / 1.5 | [核心记录](e05-implementation.md)、[宿主接入](e05-host-integration.md)；auxiliary_source.go、auxiliary_queue.go、auxiliary_resources.go、auxiliary_seal.go、execution/auxiliary_helper.go | 依赖 E04；固定来源、恢复账、宿主、存储与 cleanup 封存供 E06/E07/E08 共用。 |
| E06 / 1.6 | [实施记录](e06-implementation.md)；knowledge_generation.go、knowledge_review.go、knowledge_selection.go、knowledge_history.go | 依赖 E05；提取/汇总消费已提交事实，审阅/失效按版本，注入与历史处理服从共享资源账。 |
| E07 / 1.7 | [实施记录](e07-implementation.md)；notification.go、notification_facts.go、internal/notification/host.go、plugins/aiw-notify.py、plugins/send_teams_msg.py | 依赖 E05；通知使用自己的逐消息次数和 60 秒等待，不借用模型辅助恢复额度。默认通知未启用。 |
| E08 / 1.8 | [实施记录](e08-implementation.md)；verifier_snapshot.go、verifier_auxiliary.go、verifier.go | 依赖 E05；接受时封存，独立异步只读消费；失败/缺口不反转接受或交付。 |

清单依赖：E01→E02→E03→E04→E05；E06/E07/E08 各依赖 E05；全部 E 项→2.1→2.2→3.1→3.2。SW34 为全部组的生命周期约束，不另造 E09 或 Task。

## 逐项覆盖与待取证场景

每行的“判据”是后续场景分解，不是已运行用例。“证据”均待 2.2 按获授权范围登记，未指定实际测试命令或虚构 test ID。跨组责任按判据细化，可多于旧分组摘要。

| 条款 / 验收 | 规格 / 工程组 | 触发与关键判据 | 待取得的证据 |
| --- | --- | --- | --- |
| SW01 / AC01 | SUP；E01 | 两轮实现/修复保留不同身份、内容、版本及引用；unknown/empty/unverified 区分，旧报告不覆盖。 | 原始报告历史、绑定与字段拒绝记录。 |
| SW02 / AC02 | SUP；E01/E02 | 缺失或无效仅补报告一次，不改源码；再次失败及重启仍拒绝第二次补交。 | 原请求/补交关联、只读范围、恢复前后计数与拒派事实。 |
| SW03 / AC03 | VER；E01/E03 | 实现、测试、fixture、相关配置/依赖、计划、需求验收变化使旧通过失效；M→M 也检测；日志/memory 不失效。 | 前后内容清单、适用性判定、受控 compile/test 引用。 |
| SW04 / AC04 | VER；E01/E02/E03 | 旧适用结果复用；仅缺报告补交；缺授权不运行；Tester 可读未接受实现但不释放下游。 | 旧记录迁移解释、来源/授权、接受引用和下游状态。 |
| SW05 / AC05 | SUP；E02/E03 | 提前勾选/Coder completed 不接受；安全终态或恢复后才把独占写入权转给 Tester。 | 阶段请求快照、状态转换、租约/写入权历史和查询输出。 |
| SW06 / AC06 | SUP；E02/E05 | 未知先对账；前台退出/超时不释放写入权；Stop 持久化且重启不新派发。 | 在途身份、持久 Stop、重启轨迹、派发去重与恢复依据。 |
| SW07 / AC07 | SUP；E02/E03 | 排除环境/需求后仍不明只诊断一次；同项同阶段基础设施最多追加两次且必须安全无在途；不叠加专门预算。 | 分类及只读诊断输入/输出、阶段恢复账、等待/拒派理由。 |
| SW08 / AC08 | SUP；E02/E04 | 重启/换 Session/改配置不刷新预算；旧账可恢复则继续，不可解释集中列缺口，不能归零或伪记耗尽。 | 旧账来源、重建结果、持久余额及汇总待办。 |
| SW09 / AC09 | MEM；E05 | 前台关闭/Task 完成后继续已登记获授权工作，队列结束退出；关机后再启动先对账且遵守 Stop。 | 实际宿主与队列记录、退出/重启、授权及拒派证据。 |
| SW10 / AC10 | VER；E03 | 实现偏离需求仍按需求写断言；独立 Tester 实际装载完整上下文，越界写入不接受。 | 独立 Session/prompt、允许路径、用例/计划及写入范围。 |
| SW11 / AC11 | VER；E03 | AI 批准范围内新增测试/fixture 自动发现并刷新证据；初始无 grant、越界或实质改计划不执行；旧固定计划不扩大。 | 精确计划/grant/政策、两次发现清单、受控派发/拒派记录。 |
| SW12 / AC12 | VER；E03/E02 | 实际路径/链接目标、操作或政策变化重新评估；未知不放行；grant 未可靠落盘不执行；LOG 恢复用原阶段额度。 | 真实目标、规则版本、批准/拒绝日志与 Core 锚点、恢复账。 |
| SW13 / AC13 | VER；E03/E04 | 实现/测试缺陷分别回交，环境等待、需求冲突待决定；不得删弱断言换通过；修复后重编译/相关测试。 | 原失败输出、归因 handoff、修复理由及当前 compile/test 记录。 |
| SW14 / AC14 | VER；E02/E03 | 缺必需测试、waived、Agent 声明或提前勾选均不接受；N/A 需明确规则，执行型 prompt 不自动按文档豁免。 | 有效报告、当前执行引用、N/A 理由、拒绝/接受和依赖状态。 |
| SW15 / AC15 | ROUTE；E04/E08 | 路由无效用配置默认且不递归；同实际 provider/model 别名非升级；Compiler/Runner 不选模型，旧请求不改写。 | 推荐及回退依据、解析后模型快照、实际 Actor 派发引用。 |
| SW16 / AC16 | ROUTE；E04 | 同生成请求多缺陷/重复观察只计一次；两次有效失败才合法升级；每 Actor 最多两次，成功及其他 Actor 不清零。 | 原生成请求归因、独立角色账、阈值前后模型/派发与停止记录。 |
| SW17 / AC17 | ROUTE；E04/E03 | 共享第六次修复成功仍可接受；失败不派第七次；旧三次编译阈值不提前截停；专门预算不叠加。 | 六次派发与结果、第七次拒派、接受/停止原因及未虚扣升级证据。 |
| SW18 / AC18 | MEM；E05/E01 | 原始资料齐全时摘要失败可降级；必需资料缺失只阻塞相关派发；无实质变化不调用，迟到旧摘要不覆盖人工/新视图。 | Task 来源与 Session 投影、固定输入、条件发布和降级/拒派原因。 |
| SW19 / AC19 | MEM；E05/E06/E08 | 对 memory/提取/汇总/Verifier 分别覆盖终止无结果、无效输出、重存失败，共享各自固定输入的一次恢复；未知先对账，跨 Task 不刷新。 | 操作类型/来源/摘要键、原请求与恢复账、结果引用及局部 unavailable。 |
| SW20 / AC20 | KNOW；E05/E06/E07 | 接受登记提取，Task 完成先持久保存并登记通知，再异步汇总；提取失败不冒充无新增或撤销完成。 | 接受/完成事实、消费游标、通知/提取/汇总登记及有据空结果。 |
| SW21 / AC21 | KNOW；E06 | 部分提取失败仍可发布带缺口草稿；补齐产生新版本，不继承新正文旧确认，不变条目确认保留。 | 覆盖集合、各来源状态、条目/汇总版本和审阅引用。 |
| SW22 / AC22 | KNOW；E06 | 五态版本审阅；并行旧版本提交冲突不覆盖；编辑新候选，拒绝不能换 ID 绕过，废弃不要求替代。 | 人工主体/时间、条件提交结果、五态历史和重复输入指纹。 |
| SW23 / AC23 | KNOW；E06 | 来源变化即使结论相同也生成候选；读取时复核；同内容临时不可读恢复后可人工复核原版，无关时间戳不升级。 | 来源摘要、读取结果、失效关联、新旧版本及人工复核事实。 |
| SW24 / AC24 | KNOW；E05/E06 | 无负责人采用当前操作者且标明未验证身份；人工原文入候选/正式决定引用；普通知识审阅不阻塞，确认不写正式文档。 | 主体标记、原文与来源、候选/决定关联、仅建议同步的记录。 |
| SW25 / AC25 | INPUT；E01/E05/E06 | 必需正文缺失、精确版本不可得或完整输入超限则不派发；可选知识失败保留降级；历史输入不被 latest 改写。 | 实际冻结 prompt/正文、来源状态/摘要、选择跳过理由和完整输入计量。 |
| SW26 / AC26 | KNOW；E06 | 同输入选择可复现；拒绝/废弃不注入；候选/待复核为次要参考；需求优先，可安全满足时降级，否则集中裁决。 | 排序键、可信分区、注入版本、冲突依据与裁决/降级记录。 |
| SW27 / AC27 | KNOW；E05/E06/E08 | 仅回填相关历史、先复用；拆批/重启不刷新累计硬上限；必需输入不截断，空间不足暂停相关写入、不删历史。 | R3 政策/能力版本、相关性集合、单批/Task/项目资源账及超限结果。 |
| SW28 / AC28 | NOTIFY；E07/E05 | Task 完成即登记通知，不等 Git/草稿/Verifier；重要晚到结果关联更新，普通进度静默，本地 ack 非送达。 | 完成事实与 outbox 因果顺序、消息/内容版本及分维度展示。 |
| SW29 / AC29 | NOTIFY；E07 | 首次明确网络失败后至少 60 秒仅重发一次；非网络、未知、第二次任意结果不重发；恢复原次数/目标/due time，禁用不补全部历史。 | 逐消息两次上限账、错误分类、时钟边界、启停/过期/重启轨迹。 |
| SW30 / AC30 | NOTIFY；E07 | console/Teams 统一文本协议；缺显式配置不发送；目标不可改投；凭据不入 argv/日志；TLS 失败不降级或当网络重发。 | 插件协议/脚本版本、脱敏配置与调用记录、TLS/HTTP/权限分类。 |
| SW31 / AC31 | VER；E08/E05 | 接受时封存需求/验收、准确 diff 与验证证据；实际异步只读 Verifier，晚到只关联旧版本；历史补审不运行测试。 | 固定快照/基线、覆盖清单、独立请求/输出和来源保全记录。 |
| SW32 / AC32 | VER；E08 | 有据不满足 failed；无明确失败但证据不足 inconclusive；完整有据才 passed；空覆盖/调用故障不通过，负面结论不重生成求通过。 | 逐项依据/缺口、汇总结论、保存结果和开发/交付状态前后对照。 |
| SW33 / AC33 | GIT；E02/E05 | AI 批准具体本地计划；无关改动不提交；push/直接 Git 删除拒绝；未知先对账，cleanup 须 AIW、grant、祖先与来源封存。 | 计划/授权、预期与实际 Git 状态、动作意图/观察、来源封存及受管清理记录。 |
| SW34 / AC34 | LIFE；E01–E08 | 保持单 Requirement→Task/change，按 E 分组；设计未闭合不实施，字段细化不加审批，行为/权限/成本变化才请求决定。 | 既有 Task/worktree/branch、proposal/design/spec/tasks 关联、R1–R4 与决定记录。 |

## 跨组关系覆盖

这些是本地 Workflow 状态序列的验收设计，不授权集成/E2E 业务测试。均未执行。

| 场景 | 条款 / 工程组 | 序列与需同时成立的证据 |
| --- | --- | --- |
| AX01 正常闭环 | SW01–SW05、SW10–SW14、SW20、SW28、SW31；E01/E02/E03/E05/E06/E07/E08 | 固定输入→Coder 报告→受控编译→独立 Tester→AI 授权计划测试→当前证据接受→释放依赖。对齐同一接受版本的阶段/写入权/执行记录；提取和 Verifier 已登记，Task 完成通知不等辅助或 Git。 |
| AX02 恢复且不双写 | SW02、SW05–SW08、SW16；E01/E02/E03/E04；补充关联 SW09/SW19/SW33 与 E05 | 在途未知→重启对账→同请求结果→安全转交 Tester；重复观察不重复计失败，报告仅补一次，Stop 不新派发。覆盖 R1 意图/结果/消费崩溃窗口；辅助登记补回且不新增恢复额度，交付未知不重放副作用。 |
| AX03 授权与证据分离 | SW03、SW11–SW13；E01/E02/E03；R2 关联 SW08/SW14/SW33 | AI 范围批准→新增 fixture→新发现清单→旧通过失效→授权内重验。危险链接/实际目标变化先拒绝旧清单；grant 未可靠提交、后续拒绝、旧 consumed、宿主不能强制约束均不运行。记录精确计划/grant 与执行清单两个不同摘要。 |
| AX04 完成后的辅助失败 | SW19–SW21、SW28–SW32；E05/E06/E07/E08 | 开发完成后部分草稿、Verifier 故障/负面结论和通知故障各保留原账及状态，不返工；重要结果关联原事实。通知 unknown 不重发，明确网络失败可在 60 秒后一次重发；与四类辅助各一次恢复分账。证据含开发/交付状态前后对照。 |
| AX05 历史与知识版本 | SW04、SW21–SW27；E01/E05/E06；历史只读 Verifier 关联 E08 | 复用旧适用证据→相关历史达到累计限额→仅停止可选工作；来源变化生成候选且不继承确认，未变条目确认保留，历史 prompt 不改写。记录跨批/重启/Task 引用的来源账、版本和选择结果。 |

## 验证交接与缺口

2026-09-25 E07 最新：[四项离线适配器测试](verification-results/e07-adapter-terminal-20260925T042827204819Z/report.md) 全部 OK，形成 Python 受管协议/类型分类的局部证据。实际解释器 Python 3.9.13，真实配置解析被替身隔离，tomllib 运行条件仍待核验；Go 全链、真实 TLS/网络和送达未验证。

2026-09-25 E07 后续：[四项适配器离线计划](e07-adapter-verification-plan.md) 已准备，覆盖 Python 受管错误分类和协议边界，尚未执行；无真实网络或通知，Go 适配器全链仍待验证。

2026-09-25 E06 登记最新：[两项临时 Store 测试](verification-results/e06-registration-terminal-20260925T042234944157Z/report.md) 全部 passed，补充接受来源登记、重启去重及草稿持久发布证据。接受事实由 fixture 构造，完整接受链/宿主及登记中断窗口仍缺证据；不宣称完整 AC20/AC21/AX05。

2026-09-25 E06 后续：准备 [登记与持久发布两项计划](e06-registration-verification-plan.md)，以临时 Store 验证接受来源登记、重开去重与草稿版本保存，尚未运行；不替代真实接受链及后台宿主。

2026-09-25 E06 生成最新：[三项及六个子场景](verification-results/e06-generation-terminal-20260925T041741918570Z/report.md) 全部 passed，支持 SW20/SW21/SW23 的无新增、部分覆盖和版本保留局部契约。后台登记、真实 Store 发布及宿主仍待验证；此前尚未执行描述保留为历史。

2026-09-25 E06 SW20/SW21/SW23：已补 [提取/部分草稿/版本测试计划](e06-generation-verification-plan.md)，三项直接调用语义校验及发布转换，尚未运行；异步登记、真实持久发布和模型/宿主仍单列缺口。

2026-09-25 E05 最新：[共享保存恢复三项](verification-results/e05-save-recovery-terminal-20260925T041224728311Z/report.md) 全部 passed，补充 SW19/R1 的 memory 固定输入额度与落盘对账证据。仅重建持久中断状态，真实故障与其他辅助操作的保存路径仍待验证；此前“尚未执行”为准备阶段记录。

2026-09-25 E05 SW19/R1 后续：新增 [模型与保存共享恢复计划](e05-save-recovery-verification-plan.md)，三项测试覆盖持久中断状态下重存计次、恢复耗尽和已落盘输出对账，尚未执行。它不替代真实保存错误/进程崩溃和四类辅助操作完整验收。

2026-09-25 最新 E08：[修复后原范围重跑](verification-results/e08-verifier-terminal-20260925T031341466094Z/report.md) 三项及 22 个子场景全部 passed、退出 0，覆盖结构契约、冻结引用和临时 Store 负面报告发布。下方“等待重跑”保留为首跑时点历史。E01–E08 均已有局部测试证据，具体尚缺路径及推进顺序见 [剩余验证](verification-remaining.md)，不能等同于完整 AC/AX 通过。

2026-09-25 E08 首跑：[结果与 fixture 修复](verification-results/e08-verifier-terminal-20260925T015608050801Z/report.md)。三态规则十个子场景通过；另两组尚未通过合法基线解析，不能作为版本隔离/发布成功的证据。已修复空数组编码，等待原范围重跑；保留首跑失败，不把生产编译成功视作测试通过。

2026-09-25 E06 补充：[三项及两个子场景](verification-results/e06-knowledge-terminal-20260925T013754348449Z/report.md) 全部 passed，形成 SW22–SW24/SW26/SW27 的局部证据；异步提取/部分草稿及真实计量仍待验证。E08 已准备 [报告契约与状态隔离计划](e08-verifier-verification-plan.md)，对应 SW31/SW32 的局部契约，尚未执行。上述更新不代表完整 AC/AX 通过。

2026-09-25 E05 持久化补充：[三项结果](verification-results/e05-durable-terminal-20260925T013046528023Z/report.md) 均 passed，补充 SW09/SW18/SW19 与 R1/R3 的临时 Store 重开、队列拒绝、恢复和发布证据；不覆盖真实宿主及完整 AC/AX。E06 新增 [审阅/注入/历史额度计划](e06-knowledge-verification-plan.md)，对应 SW22–SW24/SW26/SW27 的局部契约，目前未执行。

2026-09-25 E05 局部取证补充：用户 confirm 授权后，来源快照、跨 sponsor 固定工作标识、四类辅助操作完整输入字节上限的三个测试及四个子场景全部通过，见 [报告](verification-results/e05-source-terminal-20260925T005720742474Z/report.md)。补充 SW18/SW19 与 R3 的局部契约证据；持久共享恢复、队列/宿主、memory 发布、Stop/迟到结果及完整 AC/AX 仍未验证。本文先前“尚未运行”描述保留为历史时点，不代表本组当前状态。

2.2 应在执行获授权后逐场景登记：SW/AC/AX、实际测试文件/检查位置、精确命令和目录、Task/Work Item/Attempt/阶段请求、源输入内容摘要、工具链/宿主、计划及政策版本、精确 grant 与本次发现清单、开始/结束时间、退出码/超时/原始输出引用、判定和未执行原因。分别记录未执行、失败、过期、不适用和通过；grant 不是运行通过，supervisor 编译也不覆盖全部 AC/AX。

优先在现有实施记录指向的入口规划确定性状态/版本/预算用例；真实进程、崩溃恢复、Windows 持久性、路径/链接隔离、模型/Teams 外发须独立明确授权与环境。此处不生成可误当已批准的测试计划，也不代填 grant。supervisor 继续负责冻结 Compile Plan 及有界修复。

%% ACTIVATION_EVIDENCE：E01–E04 新链的 schema 10 联合启用仍需平台和真实宿主证据；默认 schema 9 的历史成功不能证明新链已运行。E05–E08 还需能力/授权、完整输入计量、存储盘点及显式配置；本轮未迁移、启用或探测。

%% EXECUTION_EVIDENCE：AC01–AC34、AX01–AX05 尚未取得逐场景完整证据；E01–E08 部分路径已有实际通过记录。静态映射没有遗漏编号，不代表实现不存在缺陷。尤其保留恢复/双写、预算边界、grant 撤销与链接竞态、部分草稿版本、网络错误分类及封存 diff 的运行缺口。无法取得某场景的证据时，2.2 必须保留未执行/未知，不把本文或实现勾选当 passed。

%% HISTORICAL_NOTES：tasks.md 和 E01–E08 记录包含不同时点的“下游尚未实现”等历史描述；当前 authored 1.1–1.8 已勾选，运行启用与验收依然待证。本文只建立当前覆盖与依赖，不改写历史 Attempt、Core 状态或历史验证声明。

## TODO 与 Verification

- [x] 建立 34 对 SW/AC 的唯一主规格入口、工程责任、场景边界及待取证清单。
- [x] 建立五项 AX 跨组映射；显式替换授权、Git 和通知历史冲突解释。
- [x] 更新 tasks.md authored 2.1、TODO 和 Verification Record。
- [ ] 2.2 依授权取得并登记实际验证证据；3.1/3.2 另行完成整体范围及无关变更检查。

本轮执行本地 `Get-Content`（后续正文显式 UTF-8）、`Get-ChildItem`、`Get-Command aiw`、`rg`、PowerShell 只读 Core JSON 筛选及 `aiw patch --help`；Git 检查严格使用用户指定 safe.directory/-C 前缀的 status/diff。首次读取部分中文时默认编码显示异常，后续相关正文用 UTF-8 读取，不据异常显示改写文件。既有实施记录说明 aiw patch 内部调用原始 Git apply，与本轮 Git 只读限制不兼容，因此使用直接文件补丁回退，仅写本文与 tasks.md。

编辑后一次静态差异/正文核查结果：SW/AC 行数 34、唯一 SW 数 34、缺失 SW 为零、同号 AC 错配为零；AX 行数 5、编号 01–05；本文相对链接目标缺失为零。清单 2.1 已勾选，2.2/3.1/3.2 仍未勾选，既有依赖保留。Git 仅提示现有 LF/CRLF 转换规则；差异包含前轮未提交内容，不全部归于本轮。随后仅补记此结果并澄清政策表列名，按补丁文本核对，未重复运行检查。

没有运行编译、测试、最终制品构建、formatter、lint、vet、OpenSpec validator、网络/模型/通知调用或 Git 写入；编译结果由 supervisor 另行提供。已有源码和其他未提交变更保留，本项不声称 3.2 已验证。
