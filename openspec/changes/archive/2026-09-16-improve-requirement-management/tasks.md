## 1. 上下文与方法装载

- [x] 1.1 在需求领域实现每轮上下文构造与来源清单，装载正式工件正文、版本、确认事实和候选内容；覆盖无历史线程及新需求两种入口。
- [x] 1.2 实现有界本地来源选择、允许路径检查、必需与可选输入缺失处理，保证省略可追溯且不静默判就绪。
- [x] 1.3 实现通用发现基线与领域方法选择，实际装载允许的 Skill 内容并记录来源；不可用时明确降级或阻塞。

## 2. 专业发现与阶段推进

- [x] 2.1 定义并校验有来源的覆盖评估与候选输出，区分已明确、待确认、冲突和不适用；无效输出不得推进结论。
- [x] 2.2 将高影响缺口排序与每轮 1～3 个专业问题规则接入对话，复用已确认内容并对冲突显式回问。
- [x] 2.3 将对话编排接入需求入口，在新回答及确认 capture 后刷新上下文和阶段，移除仅凭文件存在判充分的对话路径。

## 3. 恢复、汇总与确认边界

- [x] 3.1 使用现有 Session 工件与 turn 记录保存来源和候选评估，绑定需求版本，实现旧会话重建及过期候选复核。
- [x] 3.2 实现对话就绪报告与 Requirement Plan 内容检查，区分草稿捕获、批准建议和可延后的设计问题，保留现有人工确认与直接 CLI 兼容语义。

## 4. 文档与行为验证材料

- [x] 4.1 更新 Requirement Management 及必要支持 Skill 的受维护源与对应副本，使方法加载、通用领域路由和确认规则与运行链路一致。
- [x] 4.2 按确认后的测试边界补充上下文、阶段刷新、恢复、无效模型输出和授权边界的聚焦回归用例；使用可控响应，不依赖真实模型。
- [x] 4.3 编写三个固定专业提问评审案例与判定标准，更新使用说明、TODO 和 Verification，明确机器验证与人工质量评审边界。

## 5. Requirement 自动编号

- [x] 5.1 为新 Requirement 自动分配 REQ00001-slug 格式的递增 ID，保留显式旧 ID 创建与读取，接通聊天确认后的真实 ID，覆盖并发互斥、编号不回收及计数损坏边界，并更新使用说明与稳定规格。

- [x] 5.2 实现序号文件缺失或损坏时的自动恢复：持锁扫描活动、archive、cancelled 需求目录最大编号，无编号从 1 开始；正常计数不倒退，损坏原件备份后安全重建，读取或备份失败停止；补充聚焦用例、说明及稳定规格。
- [x] 5.3 在 5.2 后将正式需求工件根目录统一改为 docs/requirements，运行状态与临时文件归入 .ai/requirements；不兼容旧路径、不提供迁移命令，当前项目现存需求直接移动且不改身份与正文，使旧候选重新复核；同步来源路径、Skill、文档和聚焦用例。

## TODO

5.2、5.3 已按顺序完成代码、测试源码与说明更新。经用户授权执行聚焦测试后发现并修复 Windows 扫描错误分类问题，同一聚焦测试两包均通过，编译通过。正式工件使用 docs/requirements，运行状态使用 .ai/requirements；不兼容旧路径，无迁移命令。受保护的项目 Skill 副本尚未同步，维护源已更新。历史记录中的“待实施”“未执行测试”描述保留为当时证据，不表示当前状态。

5.1 已在当前 develop 主工作区完成实现、规格和使用说明更新；编译检查通过，新增聚焦测试尚未执行。下面原有 1.x～4.x 记录保留为历史证据；本次清单完成不表示测试或人工质量验收通过。

本次追加 5.1，用户授权在 develop 当前主工作区实施。旧 1.x～4.x 完成记录保留；5.1 完成前不得沿用旧清单全部完成的结论。编号范围为同一本地仓库，跨独立克隆的全局编号不在范围内。

主测试边界已由用户确认，Design Readiness 为 FD_APPLIED。本轮完成 4.3，实施与评审材料清单均已完成。AIW sync 据清单派生 DONE / execution=completed / validation=not-required / delivery=pending；该状态不代表人工质量验证通过。
后续待授权：执行三个真实模型人工案例并记录结论；按需要补跑此前尚未运行的命令层/完整相关测试。不得直接将清单完成解释为质量验收、提交或归档授权。
以上条目按依赖排序；1.x → 2.x → 3.x，4.1 随方法确定同步，4.2 与对应实现配套，4.3 汇总验收。
不将本 Gate 或人工批准伪装为可由实现 Agent 自行完成的 Work Item。

1.1 已提供只读领域入口 LoadConversationContext 和带来源的 Prompt 渲染；事实、假设及未决问题保持捕获文档原文，不通过解析标题自动提升可信度。
2.3 已将上下文、方法建议及装载、覆盖校验和问题选择接入生产聊天循环。3.1 改为每轮通过 Session 证据恢复候选，不依赖聊天进程内存。

## Verification

### Sync / Archive 授权与同步记录

- 用户明确请求 sync & archive，目标为本 Task；15 项清单已完成，编号/存储聚焦测试与编译已有通过证据。本次不提交、不推送、不清理分支或 worktree。
- 比对两份增量规格与稳定规格：requirement-management 已包含自动编号、旧 ID、恢复、单一新存储路径及确认身份；requirement-discovery 保留更具体的代码基线，并补齐专业问题依据、已确认范围复用、方法/缺失来源证据和显式人工确认约束，避免因中英文标题不同重复加入整套需求。
- 通过 AIW 执行归档，保留 .ai 生命周期与 Workflow 记录；不直接改 Task 状态或移动运行数据。OpenSpec PowerShell 入口此前执行策略拒绝，本次未重试或绕过，未宣称其 CLI 校验通过。
- %% 本次归档是工件生命周期操作，不表示真实模型人工质量评审、完整回归、跨卷场景或只读项目 Skill 副本同步已完成；这些边界及已有测试证据随本清单保留。

### 5.2 扫描失败回归修复与授权测试

- 首次授权运行 `go test ./internal/requirement ./internal/commands/task -run '^TestRequirement(Numbering|Storage)' -count=1 -vet=off -v`：命令包通过，领域包仅 RecoveryStopsOnIOFailure/scan 失败，损坏计数被误写成 `1\n`。该失败是实际测试结果，不是此前用户报告的 no tests to run。
- 根因：Windows 下枚举普通文件可能返回可匹配 os.ErrNotExist 的错误，原扫描直接忽略它，错误进入空目录恢复路径。新增扫描 helper 先检查目录及父路径类型；只有路径确实缺失且现存祖先是目录才可跳过，已确认目录的枚举错误一律返回。符合既有“扫描失败不得修改计数”契约，未改变规格或公共 API。
- 保留原失败用例，补充 scan-parent、scan-archive、scan-cancelled，断言扫描失败后计数原文不变，且未创建恢复备份。未引入临时调试日志或扩大测试命令范围。
- 用户确认修复并复跑后，上述同一命令退出码 0：`aiw/internal/requirement 9.475s`，`aiw/internal/commands/task 1.992s`，全部匹配测试通过，包含新增子用例。GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local，GOCACHE 使用本项目 .ai/compile-cache/go。
- 随后离线 `python scripts/compile.py` 退出码 0，不保留最终二进制。使用 apply_patch 修改；静态检查 helper 的缺失、非目录和枚举失败分支及用例断言，不进行重复审查。未运行全包/全仓测试、vet、格式化、网络或 Git 写入，也未更新已安装 aiw 或只读 Skill 副本。

### 5.2 / 5.3 Implementation Evidence

- 5.2 在创建锁内区分计数缺失/格式损坏与 I/O 错误，扫描主工作区及当前工作区的活动、archive、cancelled 需求目录名；正常计数不倒退，无编号从 1 开始。损坏原件独占备份并 Sync，临时新计数落盘后 Rename 替换，不以删除原件作为失败重试；恢复向 stderr 输出原因、现存最大编号、预留及下一个编号和备份位置。
- 5.3 统一 Root 为 docs/requirements，编号及锁放到主仓库 .ai/requirements/sequence 和 sequence.lock；备份与恢复暂存文件留在运行目录，正式工件写入暂存放在 temporary。取消原 atomicWrite 删除目标后重试的分支；跨文件系统 Rename 或替换失败时保留原件并报错，不静默降级覆盖。
- 读取元数据按已注册工件类型返回实际路径，不重写已移动文档；上下文和快照由当前 Root 解析，既有历史恢复的来源比对会使旧路径候选过期。无旧目录/旧计数回退、无迁移入口，Session 历史存储不变。
- 更新 README、使用手册、需求工作流文档和稳定规格；按 skill-creator 的最小变更原则给 Requirement Skill 维护源补充正式/运行存储边界，未改变授权流程。项目 .agents 副本为只读，本轮未写入、未提权；该副本原本未硬编码旧工件路径，但不包含新增存储说明，后续安装同步仍待处理。
- 新增/更新 TestRequirementNumberingRecovery、RecoveryStopsOnIOFailure 以及 TestRequirementStorage 系列，覆盖损坏备份、无编号/归档恢复、溢出、I/O 失败、锁和高水位、新路径生命周期、不回退旧路径、历史元数据不改写及候选过期。CLI 用例增加实际新路径断言；均未执行。
- 静态检查新增代码与测试 patch、Read/Root/上下文/历史来源比对调用链、测试所用签名；scoped git diff --check 无空白错误（仅 CRLF 提示）。使用 apply_patch 编辑，固定路径替换采用批量机械改写；未使用 aiw patch。
- 执行离线 `python scripts/compile.py`（GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local），退出码 0，使用项目本地缓存且不保留可分发二进制。未运行测试、格式化、lint、Skill 验证脚本、网络、真实模型或 Git 写入；编译不证明测试通过，也不会更新已安装 aiw。
- %% 计数丢失且历史工件已删除时可能复用历史编号；不同 clone 不保证全局唯一，残留锁仍需人工处理。备份/替换故障的真实运行、跨卷工作树、完整回归及项目 Skill 副本同步尚未验证；本轮不宣称测试或交付验收通过。

### 5.2 / 5.3 Planning Evidence

- 后续用户明确取消旧根目录兼容和迁移命令。已同步 proposal、design、delta spec 与 5.3 清单；当前项目 requirements/recommend-routing 的 4 个文件已直接随目录移动到 docs/requirements/recommend-routing，未修改正文或元数据。移动前检查源、目标均在项目内、没有重解析点且目标不存在，未覆盖文件。旧序号文件不存在，无需移动；生产代码切换仍待 5.3，不能宣称新路径已接通。

- 用户同意按现存工件恢复编号，以及正式工件与运行文件分离；to-spec 与 managed fd-workflow 将这两个改动分别列项，沿用 improve-requirement-management 的 develop / primary / worktree=. 绑定。
- 设计第 9、10 节规定恢复及迁移契约；本轮不修改生产代码，不改稳定规格来冒充功能已实现，不移动用户文档。
- 用户提供的 `go test ./internal/requirement ./internal/commands/task -run '^TestRequirementNumbering' -count=1 -vet=off` 两包均显示 `[no tests to run]`；记录为未匹配测试，不是编号行为通过。当前工作区存在相应测试源码，但用户执行位置及源码副本差异尚未证实。
- 测试计划沿用领域创建和 CLI 创建/读取入口，增加计数恢复、无编号目录、终态编号、失败不覆盖和新路径生命周期场景；不测试已取消的旧路径兼容或迁移命令。本轮不运行测试、编译或网络操作。
- OpenSpec PowerShell 入口此前被执行策略拒绝，本轮不重试或绕过；沿用本地 spec-driven 结构。实现前仍须检查实际调用路径与迁移影响。
- %% 恢复只保证基于现存工件的编号下界；计数损坏且已删除历史工件时，无法证明历史编号永不复用。5.3 不改写 Session 历史正文；文件移到新目录后，旧路径代码在 5.3 完成前无法读取，不能把文件移动视为端到端验证通过。

### 5.1 Implementation Evidence

- 默认 `aiw requirement new <slug> [title]` 自动编号；`new --id <id> [title]` 保留精确 ID 创建。领域层原 Create 契约和旧 pending action 保持兼容。
- 主仓库 `.ai/requirement-sequence` 保存追加式高水位，排他锁串行化创建；扫描当前及主工作区的活动、归档、取消目录。先持久化编号再创建，允许跳号，拒绝损坏计数及已用编号；不自动回收旧锁。
- 聊天准备不分配编号，确认后使用真实 ID 绑定 Session、记录记忆并刷新聊天目标。创建后绑定或记录失败明确显示 ID，并提示不要重复创建。
- 新增 numbering_test.go、requirement_creation_test.go，补充自动递增、旧 ID、终态目录、计数损坏、锁竞争、位数增长及聊天确认/capture 用例；适配既有确认调用的返回值。测试源码已编写，尚未运行。
- 已静态检查实现差异、确认调用链及新文件，scoped git diff --check 无空白错误（仅 CRLF 提示）；增量规格 4 项 requirement 和稳定规格 9 项 requirement 的 MUST/SHALL 与 Scenario 结构检查通过。这不替代 OpenSpec CLI 校验或运行测试。
- 最后代码修改后执行离线 `python scripts/compile.py`，设置 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local，退出码 0。此前一次编译调用未捕获完成状态，不作为通过证据。
- 未运行测试、格式化、lint、真实模型或网络调用，未执行 Git 写入操作。OpenSpec PowerShell 入口被执行策略拒绝，未绕过或重试。
- %% 独立 clone 不保证全局编号唯一；计数丢失只能从现存目录恢复下界，已删除需求的历史编号无法推断；遗留锁需人工处理。Requirement 与 Session 多文件写入不是事务，报错后应先核对实际状态，不能盲目重试。

### 5.1 Planning Evidence

- Task / change 同为 improve-requirement-management；task.toml 绑定 develop、primary、worktree=.，本次不创建分支或工作树。
- to-spec / managed fd-workflow 明确分配时机、旧 ID 兼容及串行化规则，详见 design 第 8 节；计划复用创建/确认入口及现有临时目录测试设施。
- OpenSpec PowerShell 入口被执行策略拒绝；未换入口、提权或联网，改用仓库 openspecgen 的 spec-driven 结构。CLI 校验未运行，不能宣称其通过。
- %% 本项聚焦测试尚未运行；写测试与测试边界确认不等于执行授权。

### 4.3 实施 Evidence

- 按 implement Skill 完成文档交付，使用 apply_patch 编辑；未修改生产代码或 Skill 副本。
- 新增 docs/usage/requirement-discovery-review.md，提供模糊通知、已确认范围、两工件冲突三份固定输入、夹具及准备步骤。准备需真实宿主确认，不允许伪造 Session 确认记录。
- 每个案例给出可观察通过条件和失败示例；统一 NOT_RUN/BLOCKED/FAIL/PASS，评分不使用 prompt 全文或固定措辞匹配，不允许挑选最好一次输出、平均分掩盖越权或冲突问题。
- 提供固定模型/代码/来源版本、首轮输出留存、Session turn 及记录文件定位、逐条件判定模板，区分输入缺失、模型忽略、契约拒绝与语义质量失败。
- 更新使用手册的通用路由、片段确认、草稿/就绪/批准边界、恢复证据及排查方法；README 链接人工评审并移除默认金融路由说明。
- 三个案例状态均为 NOT_RUN。真实模型、网络、费用及准备写入需另行授权；本项完成指材料已编写，不是人工评审已通过。
- `aiw task workflow sync improve-requirement-management` 已将全部清单映射为 completed，并派生 Task DONE；未创建人工评审通过 Evidence，delivery 仍 pending，未提交或归档。
- 静态检查新评审全文、使用说明及 README 链接和状态边界；scoped git diff --check 通过（仅 CRLF 提示）。执行离线 `python scripts/compile.py`（GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local），退出码 0。文档项未重跑测试、未运行真实模型或网络，也未声称编译能验证人工案例质量。
- %% 全部清单项完成不消除未验证事项：既有命令层测试尚未全部运行、真实 provider/完整终端交互及安装可读性未验证；本轮不提交、归档或宣称 Task 已完成验收。

### 4.2 实施 Evidence

- 使用 implement，仅补充测试，不修改生产逻辑。主边界仍为 RunConversationTurn，可控回调提供模型输出；命令确认边界使用真实临时 Requirement/Session 存储。
- 新增 internal/requirement/conversation_regression_test.go：必需工件缺失/摘要变化/输入超预算在模型前阻断；两阶段取消不重试且保留原文；模型响应期间 revision 变化拒绝旧评估；已确认范围不重复提问而新冲突携带双方依据重新打开；旧批准不被新讨论撤销；实际方法文件变化使恢复候选失效且历史正文保留。
- 新增 internal/commands/task/requirement_regression_test.go：检查点展示后 pending 内容变化、Requirement revision 变化，以及未展示的事实确认均拒绝；断言拒绝后没有额外 revision、capture 或确认记录写入。
- 与已有用例共同形成覆盖：context/coverage/questions 覆盖路径与预算、无效输出及问题选择；conversation/readiness/history 覆盖 capture 刷新、只读生命周期、就绪与恢复；命令层 checkpoint/readiness 用例覆盖批准前复核及直接 CLI 兼容。此覆盖清单不代表所有既有用例已在本轮运行。
- 本项明确为补充测试，按仓库测试任务授权例外运行一次聚焦命令：`go test ./internal/requirement ./internal/commands/task -run '^TestRequirementRegression' -count=1 -vet=off`。设置 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local，GOCACHE 使用已有工作区缓存；结果为领域包 0.876s、命令包 0.478s，均通过。未扩大测试范围、未运行 vet、真实模型或网络。
- %% 受保护 Skill 副本在当前沙箱 git status 中显示不可见；本轮未将其解释为实际删除，也未修改或重新同步副本。新用例使用临时方法夹具，不验证实际安装可读性。真实 provider/完整终端交互、写盘故障及专业提问质量不由这些用例证明。
- 静态检查后将历史方法断言从数组位置改为来源路径，避免绑定实现顺序；随后同一聚焦命令复跑一次，最终领域包 0.431s、命令包 0.423s，均通过。检查新测试全文及 scoped git diff --check 通过（仅 CRLF 提示）；离线 `python scripts/compile.py` 退出码 0。未运行全包或全仓测试、格式化、真实模型及网络。

### 4.1 实施 Evidence

- 使用 implement 与 skill-creator：收紧主入口与支持方法边界，保持原 invocation 元数据，不扩展个人目录或无关 Skill。
- 更新 skills/ 下 requirement-management、五个 finance 发现方法及 domain-modeling 共七个维护源。主入口改为通用基线、证据驱动金融路由和实际项目级方法加载，移除运行时不支持的自动 grill-with-docs/FD 路由。
- 支持方法明确 host JSON 契约优先于 Markdown 草稿模板；只使用本轮已加载资料，不自动读取链接或执行 sibling Skill。domain-modeling 增加只读 Requirement 模式，不自动写 CONTEXT.md/ADR。
- Synthesis Plan 模板补充目标、非目标、规则、验收实例及剩余决定；engineering-options 与主入口统一延后需人工确认、业务缺口不可豁免。补充版本恢复、候选不可信、草稿与批准区别及两次批准复核说明。
- 按名称 aiw skills sync 返回旧摘要/already_installed；修正为显式 ./skills/<名称> 后七个项目副本均 installed，保留托管登记，不修改个人目录。受保护副本同步经过权限批准；源编辑使用 apply_patch。
- 用户提供 3.2 领域测试通过：`go test ./internal/requirement -run TestReadiness -count=1` → `ok aiw/internal/requirement 0.055s`，不覆盖命令层测试及本项提示词行为。
- 同步后沙箱读取副本被拒；经批准的只读检查确认七份 SKILL.md 的 SHA256 与源完全一致，git diff --check 通过（仅 CRLF 提示）。执行离线 `python scripts/compile.py`（GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local），退出码 0；编译不证明提示词行为正确。
- %% Skill 的 quick_validate、真实模型评测、测试及人工提问质量评审未运行；正文一致不等于模型能正确使用，行为验证和人工案例仍归 4.2/4.3。

### 3.2 实施 Evidence

- 用户确认：设计问题只有经人类明确确认才可延后；复用 Plan 原文片段的 capture --facts-json 检查点，不新增可由模型设置的 confirmed 标记。
- 新增 requirement/readiness.go：从原始覆盖输出重验后，分别报告已确认事实、业务阻塞、Plan 缺项、待确认延后及已确认延后。不允许设计延后豁免 needs_input/conflict；not_applicable 也需要人工来源确认才能建议批准。
- CoverageAssessment 增加可选 plan_review；旧候选可继续讨论，但缺少 Plan 检查不能建议批准。九类 Plan 内容要求正文引用、摘要和说明；事实、目标、范围、规则、验收实例还需对应已确认片段。标题、显式占位符或文件存在均不足以就绪。
- 对话每轮生成报告并随现有 Session 证据保存；不增加模型调用次数。命令层展示内容依据；聊天 APPROVED 检查点绑定 pending action、最新讨论及确认记录摘要，确认前重新读取来源并计算就绪，不信任保存的 Ready 布尔值。
- 不完整 Plan capture、DEFERRED/REJECTED 和直接 approve CLI 保持原语义；报告不会撤销旧批准或自行批准。
- 新增 TestReadiness* 及 TestRequirementReadiness*，覆盖一轮对话、未确认/确认延后、业务阻塞、Plan 缺失或无效、草稿保存、批准前来源变化和直接 CLI 兼容。测试已编写但未执行。
- 用户报告 3.1 测试 `go test ./internal/requirement -run TestConversationHistory -count=1` 通过：`ok aiw/internal/requirement 0.047s`；不覆盖本项。
- 使用 implement Skill，按上级编辑约束使用 apply_patch 而非 aiw patch。静态检查新增文件、领域到检查点调用链及测试材料；git diff --check 通过（仅 CRLF 提示）。未运行测试、格式化、真实模型或网络调用。
- 本代理执行 `python scripts/compile.py`，设置 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local 禁止下载及工具链切换，退出码 0；使用现有工作区缓存脚本，未改动脚本，不保留最终二进制。编译不验证测试文件或业务行为。
- %% 引用存在和人工片段确认不证明模型的语义映射完整；是否遗漏设计问题、验收是否充分及业务/工程分类仍需人类审阅，4.3 固定案例评审未完成。检查点沿用单聊天执行假设，不提供跨存储事务或恶意 provider 写入隔离。

### 3.1 实施 Evidence

- 用户已确认持久化边界；新增 conversation_history.go，运行证据有格式版本及 Requirement 绑定，恢复时重读来源、比较 revision/来源/方法并重验候选；已确认事实仍由宿主当前记录提供，不信任历史候选自我确认。
- 命令适配层逐轮写入 Session artifact，关联实际完成的两个模型 turn；调用前安装 running 最新记录，失败、损坏、未知版本或缺失最新记录均不回退旧成功候选。历史记录保留，不新增数据库，不改变 Requirement 状态。
- 保存方法选择与覆盖评估两份上下文快照、候选原文、诊断及确认片段输入；Session 原有 prompt/output 保留实际调用内容。恢复不会直接推进阶段，正常续聊也通过相同恢复校验。
- 新增 TestConversationHistoryRecovery 及 TestRequirementConversationHistoryLatest，覆盖恢复输入、重新评估、版本/方法变化、无效候选、旧会话、未完成及缺失最新记录；未执行。
- 用户报告上一项 `go test ./internal/requirement -run TestConversationTurn -count=1` 通过：`ok aiw/internal/requirement 0.053s`；不覆盖本项新增代码，也不覆盖命令层检查点测试。
- 使用 apply_patch 而非 aiw patch；本轮静态检查变更类型及调用链，不运行测试、格式化、网络或真实模型。此前代理编译缓存权限失败没有环境恢复证据，本轮未重试。
- %% 保存为 Session 多文件操作而非事务，不支持新增的同 Session 并发执行场景。记录中可能包含需求正文，沿用 Session 的本地访问权限；不自动删除历史或从自由文本 memory 提升确认事实。

1.1～4.3 的实现与材料交付均完成，人工质量评审仍未执行。本次不创建 Attempt、不领取写入租约；Core 派生 DONE 与人工质量验收、Git 交付分别记录。
规划交付检查：Task ID 与 change ID 一致；proposal/design/specs/tasks 完整；OpenSpec 结构校验；Workflow Core 映射与本清单一致。
规划检查结果以本次命令结果和交付报告为准，不能被解读为实现测试通过。

后续实现需覆盖：
- 无模型历史、新需求及旧会话恢复的输入完整性。
- 必需来源缺失、超预算、路径越界、方法不可读及无效模型输出。
- 确认后阶段刷新、已确认问题不重复、冲突不被静默解决。
- 保存草稿不等于批准；候选评估不绕过确认、不改写历史状态。
- 三个固定案例的人类提问质量评审。

本代理的 1.1 聚焦测试曾因 go-build 缓存 Access is denied 在准备阶段失败；之后用户提供本地通过结果（见下文）。本轮新增代码未运行测试、真实模型或网络调用。

### 1.1 实施 Evidence

- 新增 internal/requirement/context.go：重读 Requirement 身份及 revision，按确定顺序加载登记工件和可选决策日志，保存内容、路径、用途及实际/登记摘要。
- 工件缺失或摘要变化时返回可诊断快照及错误，拒绝把不完整快照渲染为可用输入；捕获记录不等于文档内每条断言已确认。
- 候选内容单独绑定 Requirement/revision，不匹配时省略并说明；新会话不创建 Requirement，不依赖后端历史。
- 新增 context_test.go：新会话、无历史恢复、来源损坏或缺失、候选失效、归档来源和非法输入的用例；已编写但未执行。
- 静态检查范围：新增代码、上述用例、文档 Gate 与清单一致性。未运行编译、测试、格式化或模型评测；编译限制见 TODO。

### 1.1 用户提供的运行 Evidence

用户在本地执行 `go test ./internal/requirement -run TestConversationContext -count=1`，输出 `ok aiw/internal/requirement 0.079s`。
这是用户报告的 1.1 版本聚焦测试结果，不是本代理运行结果，也不覆盖随后新增的 1.2 代码。

### 1.2 实施 Evidence

- 新增 context_sources.go：默认最多 64 KiB 原始 UTF-8 输入预算，调用者可调低；计入用户输入、元数据、必需工件、决策日志、候选和背景内容。JSON 包装及转义不是该原始字节预算的计量对象。
- 显式背景及指定模块的 README.md/CONTEXT.md 合计最多 32 个候选，每个引用最多 1024 字节；稳定排序、去重，不递归扫描。
- 必需工件超限或不可读返回错误并禁止 Prompt；可选文档保留 omitted/unavailable/rejected 状态及原因，不冒充已读取。
- 使用 os.Root 限定项目范围；元数据同样受限读取并复用 readMeta 解码；读取前拒绝路径穿越、绝对路径、Windows ADS、符号链接、常见凭据路径和非普通文件。背景仅接受 md/txt/rst。
- 新增 context_sources_test.go，覆盖预算优先级、必需超限、来源选择与去重、路径拒绝、符号链接和缺失背景；符号链接测试在宿主机不允许创建链接时会明确跳过。
- 本轮仅静态检查，未运行测试、格式化或编译；既有代理缓存权限阻塞没有恢复证据，不重复失败命令。用户可在本地运行相同聚焦测试命令验证新增用例。
- 该策略不扫描文档内容是否含秘密，也不承诺隔离恶意进程并发替换文件或硬链接；调用者只应显式选择适合进入模型上下文的项目资料。

### 1.2 用户提供的运行 Evidence

用户报告 `go test ./internal/requirement -run TestConversationContext -count=1` 通过：`ok aiw/internal/requirement 0.122s`。这是 1.2 版本的用户本地结果，不覆盖随后新增的 1.3。

### 1.3 实施 Evidence

- 新增 context_methods.go：内置通用发现基线；按有来源的领域/缺口建议选择一个金融主方法，可补充 domain-modeling。非金融或未知领域不强制金融方法。
- 建议必须引用当前输入（user-input）或已加载 Requirement 来源路径，且摘要与原文片段匹配；无效或过期建议明确降级到通用基线。引用校验不证明模型的语义判断正确。
- 仅加载固定 .agents/skills/<允许标识>/SKILL.md，记录实际正文、路径、摘要及状态；不搜索个人目录、不递归执行链接，不自动授权 Skill 中的写入动作。
- 基线、建议及所选方法纳入统一预算，优先于候选和可选背景。选定方法缺失、为空或超预算时明确阻塞可用 Prompt。
- 新增 context_methods_test.go：通用路由、方法正文与指纹、领域降级、非法建议、预算优先级、不可用方法及快照隔离用例；未运行。
- 本轮使用 apply_patch 编辑；未使用 aiw patch。仅进行一次静态校验，不运行测试、格式化、真实模型或网络调用；未重试既有缓存权限失败的编译命令。
- 静态检查追踪了方法选择、预算和 Prompt 拒绝路径，并发现原 16 字节可选输入用例需加上新增必需基线的预算，已相应调整。git diff --check 无错误（仅 CRLF 提示）；未跟踪的新文件通过直接读取检查，不把该 diff 命令视为覆盖全部新文件。
- %% 生产对话集成仍待 2.3，Skill 源及安装副本的路由说明统一仍待 4.1；当前仅加载主 SKILL.md，不声称已经加载其链接材料。

### 1.3 用户提供的运行 Evidence

用户报告 `go test ./internal/requirement -run TestConversationContext -count=1` 通过：`ok aiw/internal/requirement 0.135s`。这是用户本地的 1.3 结果，不覆盖 2.1。

### Task 恢复记录

用户确认 .ai 被误删，授权恢复并继续在 develop 主工作区实现。现有 new 命令拒绝已存在的规格目录，repair/bind 依赖已有元数据；因此按先前实际读取的原值恢复本 Task 的 task.toml，再执行 `aiw task workflow sync improve-requirement-management`，重建清单映射。
%% 此次只是本 Task 管理记录的重建，不是整个 .ai 的历史恢复。旧 Session 内容、Attempt、Evidence、租约及其他 Task 数据未找回，不伪造它们；session 字段仅恢复原关联 ID。

### 2.1 实施 Evidence

- 新增 coverage.go：版本化候选评估、11 个完整且不重复的维度、四种状态及来源引用；ParseCoverage 保留原始输出和拒绝诊断，任何错误均不返回 Assessment。
- 校验单个 JSON 对象、未知字段、64 KiB 输出上限、Requirement ID/revision、来源读取状态、摘要和原文片段；方法文本不能作为业务来源。
- resolved 必须有可信调用方提供的已确认片段，不能仅凭 capture 或模型自称确认；当前回答及候选笔记可作为待确认依据，但不能自我升级。conflict 至少提供两个不同片段，not_applicable 必须说明理由。
- 新增 coverage_test.go，覆盖四状态、无效结构/状态/来源、过期绑定、冲突、确认边界、原文保留；未执行。
- 使用 apply_patch 直接编辑；未使用 aiw patch。静态检查类型、校验拒绝路径和测试材料；不运行测试、格式化、模型或网络调用，既有编译缓存权限失败未重试。
- %% 结构和引用有效不等于业务语义成立或需求已就绪；实际确认片段供给、聊天接线和持久化仍需在 2.3/3.1 中完成。2.1 不调用任何生命周期写入。

### 2.1 用户提供的运行 Evidence

用户报告 `go test ./internal/requirement -run TestCoverage -count=1` 通过：`ok aiw/internal/requirement 0.033s`。不覆盖随后新增的 2.2。

### 2.2 实施 Evidence

- 新增 questions.go：SelectDiscoveryQuestions 先调用 ParseCoverage 重新验证当前来源，再以冲突优先、影响类别及稳定维度顺序选择最多三个问题；相同问题按大小写及空白规范化去重，未选缺口仍保留在完整 Coverage 中。
- resolved/not_applicable 保留为 Settled 上下文，不再产生问题；出现有效冲突时重新提问。全部已明确时允许零个问题，但不输出批准结论。
- CoverageItem 增加可选 impact_kind 与 options；备选方案必须有不同标签、取舍和有效来源引用。未提供影响类别时使用维度默认排序，兼容 2.1 候选。
- Context.Prompt 接入 Easy English 提问规则；问题、影响说明、双方冲突依据和选项原样保留。新增 questions_test.go，覆盖排序、数量、去重、已明确复用、冲突重问及非法选项；未执行。
- 使用 apply_patch 编辑，未调用 aiw patch；静态检查新增选择器、覆盖校验调用链与用例。未运行测试、格式化、模型或网络调用；既有编译缓存权限阻塞未重试。
- %% 生产 CLI 仍待 2.3 接入；跨轮已确认结论的恢复属于 3.1。当前仅保证相同文本去重和已标记 resolved 内容不再询问，语义重复及问题专业性需人工案例评审，不声称静态排序能证明。

### 2.2 用户提供的运行 Evidence

用户报告 `go test ./internal/requirement -run TestDiscoveryQuestions -count=1` 通过：`ok aiw/internal/requirement 0.029s`，不覆盖 2.3。

### 2.3 实施 Evidence

- 新增 requirement/conversation.go，领域拥有两步有界模型编排及基于覆盖的阶段判定；命令层 requirement_discovery.go 适配 Session 并渲染问题和确认检查点。每次回答及确认后重读上下文，不再复用初始 phase 或以文件存在判断充分。
- 用户已同意提前实现最小确认事实记录：capture 支持可选 --facts-json 原文片段；宿主展示后接受确认，检查 pending 内容、Requirement revision、草稿摘要，保存实际片段而非确认整篇文档。旧草稿 capture 仍不确认事实。
- 确认片段绑定当前 Requirement/revision/来源摘要，未修改的片段可在受控 capture 后延续；历史无记录或外部变化时不凭猜测恢复。完整候选和 turn 恢复仍待 3.1，不把本项视为 3.1 完成。
- Session 保存两次实际 prompt/output；无效覆盖停止处理并清除 pending action；原始响应不伪装成有效评估。确认动作仍由宿主现有分支执行，直接 approve CLI 未变。
- 新增 conversation_test.go 的可控模型用例（实际方法加载、缺失方法阻塞、capture 后阶段刷新、旧版本确认失效），以及 requirement_discovery_test.go 的片段检查点/摘要变化/确认后可用用例；均未运行。
- 使用 apply_patch 编辑，未使用 aiw patch。仅进行一次静态检查；未运行测试、格式化、网络或真实模型；此前编译缓存权限问题无代理环境修复证据，未重试。
- %% Session 仍使用既有可调用工具的 provider；提示词不是写权限沙箱，本项不声称防御恶意 provider 直接修改 Session。capture 与 Session 确认记录并非跨存储事务：若后者失败会明确报错，不能盲目重复 capture。专业性、完整就绪门槛和旧 Session 恢复分别待 4.3、3.2、3.1。
