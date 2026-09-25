## 1. 来源与候选契约

- [x] 1.1 定义生成请求、候选、覆盖报告和状态记录；实现批准来源及目标摘要快照，拒绝缺失、冲突和过期上下文。
- [x] 1.2 将共享生成器改为消费业务候选，支持多 capability 和来源映射，补齐必需 spec 与路径约束检查。

## 2. 生成与交接

- [x] 2.1 增加显式 artifact_generation Profile 候选配置与独立 provider 解析，实现有界调用、取消、失败诊断和去重预算。
- [x] 2.2 生成无模型交接文件及可见路径，实现 prepare-spec --candidate 与 --regenerate，统一模型和 Agent 候选验证入口。
- [x] 2.3 实现必要来源覆盖检查及质量报告，拒绝通用模板、错误引用、遗漏规则与跑题候选。

## 3. 写入与推广闭环

- [x] 3.1 实现生成内容归属记录、已知旧模板识别、人工内容保护、目标预检及部分写入恢复，处理预创建 tasks.md 的跳过问题。
- [x] 3.2 将 promotion / prepare-spec 接入生成状态和清单同步，仅在接受后完成推广；处理无模型路由建议与历史 SPEC_DRAFTED 复核。

## 4. 离线回归与文档

- [x] 4.1 增加两种非 AIW 业务需求的编排 fixture 和负例，覆盖规则、禁止事项、异常、验收、多能力、确认修订及未确认输入。
- [x] 4.2 增加离线 provider/文件系统故障用例，覆盖跨模型 fallback、无配置交接、候选过期、路径越界、人工修改、部分写入、同步失败与恢复去重。
- [x] 4.3 更新 requirement-management 与 to-spec 的仓库源及安装副本、配置示例和 CLI 文档，说明已批准范围的候选接续与真实未完成状态。

## TODO

wi-0008 / 4.1 已新增退款政策与库存分配两种非 AIW 业务需求的离线编排 fixture。两者均经批准 Requirement、显式 OpenSpec Targets、prepare-spec 候选接续与来源覆盖质量检查；退款 fixture 覆盖确认修订并排除旧的日退款上限，库存 fixture 覆盖多 capability、禁止分配隔离库存、库存不足例外与审计验收。另有未确认聊天输入负例，要求在创建生成请求时拒绝。4.2–4.3 仍未完成；本项不执行真实推广，不变更 Gate、Attempt、lease 或 Task 状态。

wi-0009 / 4.2 已补齐离线故障回归：既有生成、交接和候选结构用例覆盖跨模型 fallback、无配置交接、过期候选和路径越界；新增保护式应用用例模拟检查点文件系统失败后的部分写入恢复、人工改写冲突保留，以及清单同步失败后以同一应用清单幂等恢复并仅在同步成功后 accepted。未运行测试或编译；不执行真实推广，不变更 Gate、Attempt、lease 或 Task 状态。4.3 仍未完成。

wi-0010 / 4.3 已完成：requirement-management 与 to-spec 的工作树源 Skill 已说明已批准范围的候选接续、`--regenerate` 的显式用户意图，以及 awaiting-agent、validating 与 accepted 的真实完成边界；Requirement 使用手册补充显式 artifact_generation profile、`prepare-spec` 的候选/再生成 CLI 和失败恢复说明；工作流文档将 `SPEC_DRAFTED` 限定为 accepted 后。用户已确认外部安装副本同步完成；当前账户 ACL 不允许 apply_patch 写入该目录，故该外部操作按用户确认记录，不以本 Agent 的文件哈希重写为前提。Gate 已解除并重开本项；未运行真实 promotion、模型、测试或编译。

wi-0007 / 3.2 已将 promotion 与 prepare-spec 统一到同一生成请求锁内的候选接受路径：保护式应用成功后同步清单，且仅二者均成功才持久化 accepted；同步失败保留 applying 和写入摘要以便恢复。promotion 仅在 accepted 后调用 CompletePromotion；路由建议不可用只输出诊断并继续确定性完成。历史 SPEC_DRAFTED 若无匹配 accepted 记录会停止并要求复核，不补造生成证据。4.1–4.3 仍未完成；本项不执行真实推广，不变更 Gate、Attempt、lease 或 Task 状态。

wi-0006 / 3.1 已实现保护式应用和恢复边界；3.2 仍须在同一请求锁与持久化适配器下接入应用、清单同步和 accepted，4.1–4.3 仍须补充离线回归与文档。当前不执行真实推广，不变更 Gate、Attempt、lease 或 Task 状态。

wi-0005 / 2.3 已实现：模型、Agent 与已保存候选恢复共用保守的来源原句覆盖检查，报告独立区分结构、覆盖、工程就绪和实现验收。仅本项新增完成标记；3.1/3.2 及 4.x 保持未完成，候选仍最多进入 validating，不写正式工件或完成推广。下一步由 supervisor 执行冻结 Compile Plan，再决定接受此 Work Item；未变更 Gate、Attempt、lease 或 Task 状态。

%% 2.3 限制：来源逐行核对要求保留原句，可能拒绝同义改写或包含规划元数据的 Plan；词项相关性不是语义证明，人工审阅仍须排查矛盾、隐藏的额外规则与工程判断。未运行新增离线用例；两类业务的完整编排 fixture 仍归 4.1，保护式写入与接受闭环仍归 3.1/3.2。

wi-0004 / 2.2 设计决定（用户于 2026-09-17 确认）：采用批准 Requirement Plan 中的显式 OpenSpec Targets 作为唯一的 capability 预声明来源。PrepareGeneration 在 `input.json` 冻结基础文件和每个声明 capability 的路径、`new`/`modified` 意图及已有摘要或 `absent` 基线；ValidateGenerationContext 在接续时重验清单、摘要和 `absent` 状态。候选不得新增、删除、重命名或省略清单目标，且不得从候选反向补签基线；无有效 Targets、意图/现状冲突或第三方创建预期新增文件时停止并报告。无需完整目录快照；无关目录变化不扩大写入范围。

本次实现 wi-0004 / 2.2，保留已完成的 1.1、1.2、2.1，其余编号项保持未完成。prepare-spec 已接入生成/交接及候选接续，返回 awaiting-agent 或 validating 的真实未完成诊断；不写正式工件、不宣称 accepted。2.3 负责语义覆盖报告，3.1 负责保护式写入，3.2 负责 promotion 共享接入、接受与清单同步。旧 promotion 模板调用暂留，不代表新候选被接受；修复完成前保持关联 issue 开放，不重跑真实 Requirement 推广。

## Verification

wi-0010 验证证据：静态核对 `requirement-management` 和 `to-spec` 工作树源 Skill、`docs/usage/aiw-requirement.md`、`docs/requirement-management-workflow.md` 及 `prepare-spec` 参数解析，确认文档配置名为 `[ai.artifact_generation].profiles`，CLI 为 `<id>`、`--candidate <path>` 或 `--regenerate`，且 awaiting-agent、validating 不是 accepted。用户已确认外部安装副本由其手动同步；先前 Session 因当前账户对该目录 ACL 无写权限而受阻，Gate 以此确认解除。已执行限定路径 `git diff --check`，无空白错误，仅有 LF/CRLF 提示。未运行测试、编译、模型调用、真实 promotion、Git 写入或交付操作；本项仅涉及文档与 Skill 文本。

wi-0009 验证证据：`TestGenerationFailureBudgetSurvivesResume` 与 `TestModelQualityFailureFallsBackAndPersistsReport` 覆盖独立 provider/model fallback；`TestHandoffResumeCandidateAndExplicitRegeneration` 覆盖无配置交接和旧候选拒绝；`TestRenderBusinessCandidateRejectsInvalidStructure` 覆盖越界路径；新增 `TestProtectedApplicationRejectsHumanEditAfterInterruptedWrite`、`TestProtectedApplicationRecoversPartialWriteWithoutRecreatingArtifacts`、`TestAcceptanceRecoversSyncFailureWithoutDuplicateApplication` 覆盖人工内容保护、部分写入恢复及同步失败恢复去重。均为离线替身与临时目录用例，按本 Work Item 限制未运行；编译由 supervisor 负责。

wi-0008 实现证据：`TestBusinessRequirementOrchestrationFixtures` 使用退款政策和库存分配两份完整业务 Plan，各自声明多 capability 目标并经 `PrepareCandidate` 的无模型交接与 Agent 候选接续。退款候选保留日限额、禁止超额退款、部分退款例外与成功退款审计验收；其可信确认以替代证据淘汰旧日限额。库存候选保留预留、隔离库存禁止项、库存不足时的 backorder 例外和分配审计验收。未确认聊天事实在请求准备阶段须被拒绝。静态审阅应覆盖 fixture 目标与候选路径的一一对应、确认引用的摘要绑定、来源映射、场景和编号任务映射；未运行测试、编译、格式化、lint/vet、网络、最终构建、OpenSpec CLI 或真实推广，编译由 supervisor 负责。

wi-0007 实现证据：`PrepareCandidate` 在其 generation.lock 持有期间调用接受适配器；`AcceptCandidate` 复用 `ApplyCandidate` 的预检、归属和原子写入，再同步受管清单，只有同步成功才将记录写为 accepted。同步错误保留 applying 及已写摘要，下一次 prepare-spec 在同一锁内恢复同步，不重写已匹配内容。promotion 与 prepare-spec 共享该路径；只有返回 accepted 才调用 CompletePromotion。路由建议使用原有确定性默认计划；意外不可用时输出诊断而不阻断已接受工件。历史 SPEC_DRAFTED 先核对活跃请求、身份、摘要与 accepted 状态，缺少证据时只报告人工复核。静态审阅覆盖候选状态转换、写入/同步顺序、恢复分支、生命周期调用点与历史保护；未运行测试、编译、格式化、lint/vet、网络、最终构建、OpenSpec CLI 或真实推广，编译由 supervisor 负责。

wi-0006 实现证据：新增受保护应用边界 `openspecgen.ApplyCandidate` 与 GenerationRecord 应用清单。它要求调用方提供新鲜度检查和持久化回调，先完整预检冻结目标与路径，再持久化每个目标的原摘要、候选摘要、归属和进度；逐文件临时文件同步后原子替换并立即检查摘要、保存进度。恢复仅接受原摘要或本请求已写摘要，第三方编辑会保留文件并报告冲突。仅允许不存在目标、同候选内容，或完整匹配旧 promotion 通用 `tasks.md` 模板的替换；不接受近似模板或未证明归属的既有内容。应用结束仍为 applying，未接受请求、未同步清单、未调用推广或修改 Workflow Core，留待 3.2。

%% 3.1 边界：旧 promotion 的 proposal/design/spec 模板含运行时业务内容，当前无法在不扩大权限的前提下构造其完整字节级识别；因此保护式应用只认定固定的完整通用 tasks.md，其他既有正文一律保留并报告冲突。3.2 必须在同一锁和持久化适配器下接入 ApplyCandidate、清单同步及最终 accepted，不能绕过该边界。

wi-0005 实现证据：candidateQuality 从批准 Plan 与 ActiveFacts 构建必要来源条目，排除已确认替代片段；逐项核对具体要求正文/场景，拒绝遗漏禁止事项/异常、无效或未确认引用、通用占位、跑题正文、歧义标题、未映射场景和任务。过滤注释、代码块与引用块，避免用隐藏内容冒充覆盖。result.json 保存 VerifiedCoverage、Issues、检查方法和两个 not-assessed 字段；模型 fallback 每次保留独立报告，Agent 拒绝保留报告并清除旧候选。补充未执行的正例、来源缺失/错误映射/模板/跑题/修订/工程延期及模型 fallback、Agent 持久化用例，更新已有 fixture。

wi-0005 静态检查结果：唯一一次编辑后命令退出码 0；授权 Git diff 成功，仅有 LF/CRLF 提示，合并输出部分截断，未重跑。已查看质量实现、模型检查点与 Agent 保存路径、清单和设计 diff。静态推演发现英语复数导致任务关联误拒绝，已局部修正词项匹配，补充中文相邻字匹配及数值符号保留；对应新增用例未执行，这些末尾修正未追加验证命令，仍由 supervisor 编译。

wi-0005 验证边界：实际执行定向 Get-Content/Get-ChildItem/rg、Get-Command、aiw patch --help，以及授权前缀的 git status --short、git branch --show-current；均未触发 Git 写入。首次读取编码与输出截断后按 UTF-8 补读必要片段。使用直接 apply_patch：aiw patch 依赖 Git apply，沿用本 Task 仅授权 Git 只读的交接限制。完成后执行一次限定路径的静态 diff/文件读取，检查质量入口、报告保存和清单一致性；结果见本轮结构化交付。未运行测试、编译、formatter、lint/vet、验证脚本、网络、最终构建或真实推广；编译由 supervisor 负责。

wi-0004 实现证据：PrepareGeneration 从批准 Plan 的 OpenSpec Targets 派生完整目标清单，记录基础/新增/修改意图、已有摘要与 absent 基线；同时冻结声明 capability 的稳定 spec 现状。ValidateGenerationFreshness 在模型调用及接续前重新建立快照；ValidateGenerationTargets 拒绝遗漏、重复与未声明目标，并被 RenderCandidate 共用。新增 openspecgen.PrepareCandidate 持久化 input.json、Easy English request.md、candidate.json、result.json 及活动请求指针，串行锁与原子文件替换保证同一 Task 的请求预算不会被两个接续进程同时消费。无模型进入 awaiting-agent，恢复复用预算，--regenerate 保存旧审计并建立新 ID；--candidate 与模型候选共用目标、结构及新鲜度验证。prepare-spec 输出实际路径和未完成状态，候选最多到 validating，不运行 OpenSpec CLI 或推广完成步骤。

新增未运行离线用例覆盖目标缺失/重复/越界、目标与稳定能力抢先创建、无关目录变化、modified 稳定规格前置条件、交接文件、预算复用、Agent 接续、旧候选拒绝、新请求保留旧审计和 CLI 参数。同步更新既有候选 fixture 以显式声明目标。静态审阅范围为请求来源/摘要、清单精确匹配、生成检查点、CLI 参数和状态输出。编译与接受交由 supervisor；未运行测试、编译、格式化、lint/vet、网络、最终构建或真实推广，未修改 Workflow Core/Gate 或执行 Git 写入。使用直接补丁延续既有交接方式；aiw patch 的 Git apply 路径不适用于本轮限定的 Git 只读授权。

%% 2.2 边界：CLI 尚无可信确认片段适配器，默认只使用批准 Plan 与注册来源，不从候选或 input.json 恢复确认权限。语义覆盖、人工内容合并、正式写入/同步和 accepted 仍待 2.3/3.1/3.2。进程异常退出可能留下 generation.lock；后续接续明确报告锁路径，不自动抢占。请求初始化中断留下未激活目录时保留审计，下一次可建立新请求；只有 active.json 指向的请求可接受候选。

wi-0004 命令证据：实际执行 Get-Content/Get-ChildItem/rg 的定向读取、用户授权前缀的 git status --short --branch 和一次编辑后的 git diff --（限定 CLI、设计和清单文件）。Git 均成功，仅提示 LF/CRLF；编辑后同批末尾 rg 将 requirement* 作为路径传给 Windows，报告路径语法错误（os error 123），整批退出码 1，未重跑，不能表述为整批检查通过。静态读取发现 modified fixture 需要显式回到 DECIDED 才能重新批准，已局部修正；同时限定审计读取预算、清空被拒绝提交的旧候选报告。未追加验证命令；这些最后修正仍需 supervisor 编译。

wi-0004 设计证据：已读取交接、implement 技能、管理规则、设计、清单、生成规格，以及 prepare-spec、PrepareGeneration、ValidateGenerationContext、RenderCandidate 和 GenerateCandidate 调用边界。设计现已将可信预声明固定为批准 Plan 的 OpenSpec Targets，并定义 Target Manifest、`new`/`modified` 校验、`absent` 基线及接续重验；无需全目录快照。授权前缀的 git status --short --branch 成功，分支为 feature/requirement-artifact-generation，已有实现改动保留。未运行测试、编译、格式化、网络或最终构建，编译由 supervisor 负责。首次文本读取编码不匹配后改为 UTF-8；文件搜索出现不存在的 cmd/context_reader.go 路径及 PowerShell 通配参数错误，不构成运行验证证据。

wi-0003 实现证据：internal/ai/artifact_generation.go 独立解析有序 Profile 引用（支持空数组和多行数组），无显式 provider/auto 直接交接，不进入通用探测链。按 provider 隔离 model、endpoint、credentials、command；模型/地址默认值进入审计，凭据排除在序列化和稳定去重身份之外。相同解析配置去重，最多 8 项；禁止含凭据、query 或 fragment 的 endpoint。internal/openspecgen/generation.go 复用 Provider.Generate，调用使用只读请求和 120 秒取消边界；生成记录冻结预算并在调用前通过必需 Save 回调持久化 started，恢复不补充或重试已消费项。记录 provider/超时/候选 JSON 与结构失败诊断并脱敏；CheckContext 和非候选质量错误阻断 fallback，耗尽进入 awaiting-agent；有效候选保存为 validating，不宣称 accepted。新增离线用例覆盖配置隔离、去重、配置解析、凭据不进入审计、环境 provider 切换、恢复不重试、检查点失败、来源失败及不配合取消的 provider；均未运行。

%% 2.1 的 Save/CheckContext 是后续共享编排必须提供的持久化和新鲜度适配边界，调用方须保证同一请求串行持有写权限；本项不接入 CLI、不生成交接文件、不验证语义覆盖、不写正式工件。2.2/3.2 必须接入这些回调；2.3 补齐 Validate 的覆盖检查，只有明确 CandidateRejection 可因结果质量切换模型。忽略取消的 provider 可能继续占用后台资源，但调用边界会返回，单请求最多 8 次且不重试；原生 provider 使用 context 取消。全局 CLI 配置缺少可审计 model 时记录不可用并交接。编译及运行证据仍由 supervisor 提供。

本轮使用直接文件补丁：aiw patch 内部使用未带本轮授权 safe.directory 前缀的 git apply，因此未执行该路径。只执行文件读取/搜索及授权范围内的 Git 只读核验；未执行测试、编译、格式化、网络、OpenSpec CLI 验证或最终构建，未修改 Workflow Core/Gate 或执行 Git 写操作。唯一一次编辑后静态检查包含 diff、新文件全文及 CLI 参数路径，退出码 0；Git 仅提示 LF/CRLF 转换。由该检查发现的 Codex 模型参数遗漏已限定在 artifact-generation 阶段修复并增加未运行的参数用例；权限错误判断使用 errors.Is 识别包装错误。未追加验证命令，最终编译由 supervisor 完成。

wi-0002 实现证据：internal/openspecgen 的 Render 增加请求/候选入口，RenderCandidate 保留完整业务正文及多 capability 来源映射；校验候选身份、来源标识、要求/场景层级及任务编号引用。Validate 要求三个主工件和至少一个 delta spec，拒绝非法路径、大小写重复目标、Windows 设备名。WriteMissing 写前检查全部目标及祖先，拒绝符号链接和非普通文件，并以排他创建保留已有内容。新增离线用例覆盖多能力正文与映射保留、缺失 spec、路径负例、错误引用及写前检查；未运行。静态审阅范围为生成器、候选契约及既有调用点，不修改 CLI、Workflow Core 或 Gate。

%% 1.2 只提供结构与引用位置证据；来源是否有权定义业务规则、必要来源完整覆盖及正文是否支持规则由 2.3 验证。候选接受前仍须调用 1.1 的上下文新鲜度校验。旧模板迁移、并发目录替换期间的写入约束、归属和部分写入恢复由 3.1 完成，不能据本项宣称安全恢复或 accepted。

wi-0001 实现证据：internal/requirement/generation.go 定义请求、候选、覆盖报告及独立生成状态；复用有界、拒绝符号链接的 contextReader 读取注册来源、稳定规格及目标基线。审批新增 source_digest，绑定批准时的注册来源集合；快照拒绝缺少 Plan、缺失/不符摘要、Task 不匹配、未确认引用、同级冲突、无依据替代及替代环。候选接续重新读取上下文，核对请求身份、输入摘要及目标基线。generation_test.go 增加离线单元用例但未运行。未改 Workflow Core 状态或 Gate。

验证边界：本轮仅静态检查；按 supervisor 交接要求不运行测试、编译、格式化、OpenSpec CLI 验证、网络或最终构建。编译及接受由 supervisor 执行冻结 Compile Plan 后决定。现有 Git 工作区和分支核验使用用户授权的 command-local safe.directory 前缀，未修改 Git 配置或执行 Git 写入。

%% 旧批准记录没有 source_digest 时拒绝作为新生成的批准证据，不自动补签；历史恢复策略和 CLI 诊断由后续 3.2 接入。确认片段和替代证据只允许由可信确认记录适配器提供，候选不能自行提交确认。语义覆盖验证属于 2.3；本项的来源校验不证明业务语义完整。快照总读取预算沿用 64 KiB，超限明确失败，不截断。

规划证据：已静态读取问题记录、现行 Skill、推广/生成器/模型 fallback 调用链和 requirement-management 稳定规格；已通过 AIW automatic backend 创建同名生命周期与 change。目标为主工作区 develop，无实现 worktree。

测试设计：主边界为 promotion / prepare-spec 共享编排，使用离线固定输入与 provider 替身。运行测试、编译、真实模型和网络不属于本次规格生成。

已执行 `aiw task workflow sync requirement-artifact-generation`，成功建立 10 项清单映射，均为 authored=open、core=ready；Workflow 为 DRAFT / queued，未启动 Attempt。Task ID 与 change 目录名均为 requirement-artifact-generation，必需文件已生成。OpenSpec PowerShell 入口受本机执行策略阻止，未绕过执行策略；CLI instructions / validate 未完成，不宣称 CLI 验证通过。

%% 验证限制：正式实现前补做可用环境中的 OpenSpec 结构验证；人工复核语义覆盖检查是否足以拒绝刻意跑题但结构合法的候选。
