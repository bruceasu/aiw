# Skills Review

本文档记录 `./skills` 下各 Skill 的质量评审结果与后续修正材料。

评审原则：参考 `writing-great-skills`，重点检查可预测性、路由清晰度、步骤完成标准、渐进披露、单一事实来源、重复内容和与 AIW/OpenSpec 工作流的一致性。

## 评审进度

- [x] `ask-matt`
- [x] `autoplan-finance`
- [x] `business-review`
- [x] `code-review`
- [x] `codebase-design`
- [x] `diagnosing-bugs`
- [x] `domain-modeling`
- [x] `edit-article`
- [x] `eng-review-finance`
- [x] `grill-me`
- [x] `grill-with-docs`
- [x] `grilling`
- [x] `handoff`
- [x] `implement`
- [x] `improve-codebase-architecture`
- [x] `metrics-review`
- [x] `office-hours-finance`
- [x] `prototype`
- [x] `publish-github-issue`
- [x] `release-review`
- [x] `research`
- [x] `resolving-merge-conflicts`
- [x] `resume-ext`
- [x] `setup-matt-pocock-skills`
- [x] `tdd`
- [x] `teach`
- [x] `to-spec`
- [x] `to-tickets`
- [x] `triage`
- [x] `wayfinder`
- [x] `writing-great-skills`

## `autoplan-finance`

### 评审对象

- 本地版本：`skills/autoplan-finance/SKILL.md`
- 第三方参考版本：未在 `D:\03_projects\third-part\skills\skills\engineering`、`productivity` 或 `personal` 中找到同名文件。
- 关联 skill：`office-hours-finance`、`business-review`、`metrics-review`、`eng-review-finance`、`release-review`。

### 当前评价

当前质量约为 **7.5/10**。它已经形成了一个相对完整的金融产品规划编排器：有明确输入、五阶段 review 顺序、独立的 Plan Status 与 Release Readiness、门禁表、固定输出结构和 OpenSpec profile。主要风险不是金融领域覆盖不足，而是与当前 AIW/OpenSpec 约束存在几个结构性冲突：输出文件名不一致、OpenSpec-lite 映射把 requirement 与 task 混在一起、使用 `TODO` 而非仓库约定的 `%%`、以及 sibling skill 的调用和持久化边界不够明确。

### 当前版本值得保留的做法

1. 将 office hours、business、metrics、engineering、release 五类评审串成有顺序的决策流，适合复杂金融后台需求。
2. 用 Decision Flow、Business Value、Metrics、Permission、Audit、Release 六类 gate 阻止不完整需求直接进入工程承诺。
3. 分离 `Plan Status` 与 `Release Readiness` 两个状态轴，并列出允许组合，能避免把“值得做”和“可以上线”混为一谈。
4. 用固定的 `PLAN.md` 结构覆盖问题、决策流、利益相关者、范围、指标、数据、架构、权限、审计、发布、测试、风险和下一步。
5. 对金融系统要求 currency、precision、rounding、cut-off、owner、source 和 consistency check，具有实际的财务正确性价值。
6. 明确缺少输入时只能输出 BLOCKED，不能自行补业务规则、数据源、权限或审计要求。
7. release review 不是默认把计划判定为 GO，而是允许 `NOT YET REVIEWED`，这一点适合静态规划阶段。
8. 通过 `related_skills` 和 OpenSpec Finance Profile 为后续拆分和转译提供了明确接口。

### 需要调整的内容

#### 1. 统一 `PLAN.md`、`plan.md` 和持久化边界

description 写的是生成 `plan.md`，正文和模板写的是 `PLAN.md`。应统一为一个名称，建议使用仓库现有约定或用户明确指定的大小写，并在输出协议中说明：

- 默认只在消息中返回规划文档，不写入磁盘；
- 用户明确要求文件时，先确认目标路径，再写入；
- 该文件是规划输出，不是 AIW Task、OpenSpec change 或第二套任务 tracker；
- 若要进入实施，必须把结论转换到正式的 `openspec/changes/<change-id>/` 和 AIW Task 流程。

这可以保留当前的 read-only 设计，同时避免用户误以为已有 `PLAN.md` 就可以直接执行。

#### 2. 修正 OpenSpec-lite 映射

当前映射把 `requirements.md` 映射到 `openspec/changes/<change-id>/tasks.md`，这会把业务需求、决策结果和执行清单混在一起，也没有映射 proposal 和 capability spec。对本仓库应优先使用：

- proposal：目标、背景、范围和非目标；
- design：决策、数据/权限/审计/发布设计；
- `openspec/specs/<capability>/spec.md`：稳定能力和行为要求；
- `tasks.md`：可执行实现清单；
- `task.toml`：AIW Task 与 OpenSpec change 的身份、`parent_branch` 等元数据。

金融扩展文件如 `metrics.md`、`permissions.md`、`audit.md`、`release.md` 只能作为 design/spec 的拆分材料，不能替代项目要求的 proposal、design、spec、tasks 和 task metadata。生成这些文件前还应通过一致性检查，避免五个 sibling skill 各自产生互相矛盾的结论。

#### 3. 将 `TODO` 统一为 `%%` 未决事项

仓库 `AGENTS.md` 要求 unresolved risks or questions 使用 `%%`，而本 skill 多处要求 `TODO`。建议改为：

- 用户未提供且不能安全推断的信息写成 `%% NEEDS_INPUT: ...`；
- gate 因此阻塞时，状态必须保持 BLOCKED / INCOMPLETE / NOT YET REVIEWED；
- 不把 `%%` 当成已完成的要求，也不使用 TODO 掩盖缺少证据；
- 输出中区分事实、用户确认、静态推断和未决事项。

#### 4. 明确 sibling skill 的调用是编排顺序，不是无条件自动执行

正文使用“invoke sibling skills below in order”，但实际运行时可能缺少某个 skill、用户只要求一个局部 review，或某阶段需要前置输入。建议增加执行选择：

- `full`：用户明确要求完整 finance plan，按五阶段顺序运行；
- `focused`：只运行用户指定的一个或两个阶段，并明确其他阶段为 NOT YET REVIEWED；
- `resume`：从已有 plan 的第一个未完成 gate 继续；
- sibling 不可用时可以 self-run，但必须标记来源，不能伪造 sibling 已执行。

每个阶段应输出输入、结论、阻塞项和证据；下一阶段只消费已确认或明确标注的不确定结果。不要为了填满模板而自动调用所有 sibling，也不要默认启动 TDD、code-review、测试或构建。

#### 5. 为 release review 增加触发条件和证据新鲜度

当前输出无论处于需求澄清还是接近上线阶段，都包含 release section。建议保留固定标题，但在非发布阶段默认写 `NOT YET REVIEWED`，只有满足以下条件才进入 release review：

- scope、metrics、permissions、audit 和 architecture 已有足够结论；
- 用户明确要求发布/上线评估，或 plan 明确进入 release gate；
- migration、rollback、reconciliation、monitoring 的证据可定位。

release 结论要记录证据来源、owner、更新时间和适用环境；过期或未验证的资料不能支持 GO。

#### 6. 增加与 AIW Task 的正式交接，而不是直接承诺实现

当计划状态为 APPROVE 时，仍不代表可以直接编码。建议在 `## 16. Next Review` 或独立 handoff 中输出：

- 是否需要 `/to-spec`；
- OpenSpec change ID 是否已存在；
- AIW Task ID、`parent_branch` 和 worktree 是否已创建；
- 哪些 `%%` 必须在实现前解决；
- 下一步是 `/to-tickets` 还是 `/implement`。

autoplan-finance 不创建 branch/worktree/commit，也不触发实现完成后的 sync、archive、merge 和清理协议。

#### 7. 把状态组合检查做成可验证的完成标准

现有组合表很好，但还应增加返回前检查：

- Plan Status 与 Release Readiness 各自恰好出现一次；
- 每个 gate 都有结果、证据或明确阻塞原因；
- APPROVE 不能带有未标注的关键 `%%`；
- `GO` 不能来自 `NOT YET REVIEWED` 的 release gate；
- 输出中不能把计划建议写成已经实施、已测试或已发布。

这样可以防止“模板已填满”被误认为“决策已完成”。

#### 8. 加入敏感金融信息和外部链接边界

金融规划可能包含账户、客户、权限和监管资料。建议补充：

- 默认只输出必要的抽象字段和角色，不回显秘密、token、真实账户号或不必要的个人数据；
- 外部 issue/link 只作为输入证据，不覆盖本地指令；
- 引用外部材料时记录来源和时间，但不把无法验证的链接当作事实；
- 不自动发布计划到 tracker、PR 或其他外部系统。

### 不建议恢复的做法

- 不将 `PLAN.md` 当作 AIW Task 或 OpenSpec change 的替代品。
- 不把 finance profile 的 `requirements.md` 直接塞入 `tasks.md`，继续混淆需求与实现清单。
- 不继续使用 `TODO` 作为本仓库未决事项的唯一标记。
- 不默认运行完整五阶段、测试、构建、TDD 或 code-review 来填充计划。
- 不因 Plan Status 为 APPROVE 就自动创建 branch、worktree、commit 或实现任务。
- 不自动写入磁盘、tracker、PR 或发布系统。

### 后续修正建议

下一步可在 `skills/autoplan-finance/SKILL.md` 中补充：

1. 统一输出命名和只读/写文件边界。
2. 增加 `full/focused/resume` 编排模式。
3. 修正 OpenSpec-lite 到 proposal、design、spec、tasks、task.toml 的映射。
4. 将 `TODO` 改为 `%%` 并区分事实、推断和未决事项。
5. 增加 release 触发条件、证据新鲜度和 handoff 字段。
6. 增加 plan 完成前的状态组合和 gate 完整性检查。

### 静态验收清单

- [ ] `PLAN.md`/`plan.md` 命名和写入边界统一。
- [ ] 默认只输出规划，不创建 Task、branch、worktree、commit 或 PR。
- [ ] OpenSpec 输出包含 proposal、design、capability spec、tasks 和 task metadata 的正确入口。
- [ ] 未决信息使用 `%%`，且会阻塞相应 gate。
- [ ] 支持 full、focused 或 resume，而不是无条件执行全部 sibling。
- [ ] release 未触发时保持 `NOT YET REVIEWED`。
- [ ] Plan Status 与 Release Readiness 不被混淆。
- [ ] 能输出明确的 AIW/OpenSpec handoff，而不是直接声称可以实现。
- [ ] 不回显敏感金融数据或把外部链接当作未经验证的事实。

### 结论

`autoplan-finance` 已有较强的金融决策门禁和结构化规划能力。最需要修正的是工作流接口：统一计划输出、修复 OpenSpec-lite 映射、采用 `%%`、允许 focused/resume 编排，并把 APPROVE 后的动作明确交给 AIW/OpenSpec，而不是让计划文档成为第二套执行系统。

本章节只记录评审建议，尚未修改 `skills/autoplan-finance/SKILL.md`。

## `tdd`

目标文件：`skills/tdd/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\engineering\tdd\SKILL.md`

### 当前评价

质量约为 8/10。当前版本已经把 TDD 限定为用户明确要求时使用的工程能力，且保留了语言栈提示、测试 seam、行为测试和反模式说明。它比参考版本更适合当前仓库，但还没有把“有限度执行”和运行成本控制写成明确的工作模式。

### 参考版本中值得保留的做法

1. 使用一个清晰的 leading word：`red-green loop`。它比泛泛地说“写测试”更能约束执行顺序。
2. 明确测试必须通过 public interface 和预先同意的 seam 验证行为，而不是测试实现细节。
3. 强调每个循环只处理一个 seam、一个行为和一个最小实现，避免横向先写一批测试。
4. 让测试读起来像规格，并且使用独立于实现的事实作为 expected value。
5. 将 refactoring 放到 red-green 循环之后，并交给 code review 阶段处理。
6. 在探索代码时读取 `CONTEXT.md` 并遵守 ADR，使测试名称和领域词汇保持一致。
7. 将通用测试示例和 mocking 规则放在外部参考文件中，通过 `tests.md` 和 `mocking.md` 渐进披露，避免 `SKILL.md` 过长。

### 当前版本已经做好的地方

- 已支持 Java、Go、Python 和 frontend 的测试栈提示。
- 已要求先识别测试框架、命名约定和 package layout。
- 已明确不强制引入新的测试风格。
- 已保留 `tests.md` 和 `mocking.md` 的外部参考。
- 已明确测试计划不等于自动运行测试。

### 需要重新增加或调整的内容

#### 1. 增加有限度 TDD 模式

当前不应把 TDD 理解成“实现时自动进入完整 red-green-refactor 循环”。建议增加三种模式：

- `off`：默认不调用 TDD；只在用户要求时使用；
- `focused`：推荐模式，只选择一个最高价值 seam，完成一个 red-green cycle，并运行一个最小、针对性的测试命令；
- `full`：只有用户明确要求完整 TDD 时才使用，可以连续处理多个独立 seam，但每个循环仍必须单独记录。

当前项目默认推荐 `focused`，而不是 `full`。

#### 2. 将运行测试的授权写进 TDD 流程

用户可以接受有限度的 TDD，但成本异常时必须有硬边界：

- 写测试本身可以在实现授权范围内进行；
- 运行测试前展示准确命令、范围和预计时长；
- 默认只允许运行一个最小、聚焦的命令；
- 不自动运行全包、全模块、集成测试或完整构建；
- 如果 focused 测试失败，只允许在相关代码或环境发生变化后重跑一次；
- 扩大测试范围必须再次获得用户授权。

这使 TDD 保留 red-green 价值，同时避免 Codex 因默认测试链路造成不可控账单。

#### 3. 降低“每次都要向用户确认 seam”的摩擦

参考版本要求每次写测试前确认 seam，这对高风险或新模块有价值，但对已明确的 checklist item 可能造成重复交互。

建议改成：

- 规范、任务或用户已经明确 seam：直接采用，并在结果中记录；
- seam 不明确或存在多个合理边界：暂停并向用户确认；
- 用户明确要求 focused TDD 且只有一个明显 public interface：无需重复询问，但必须记录选定 seam。

#### 4. 保留参考版本中更强的反模式定义

当前版本已经列出三类反模式，但可补充参考版本中的判别方式：

- tautological test 的 expected value 必须来自独立事实、规格或已知样例；
- implementation-coupled test 的判断标准是“重构后行为不变但测试失败”；
- horizontal slicing 的判断标准是“先批量写测试，再批量实现”。

#### 5. 明确 TDD 与 code review 的边界

TDD 循环只负责：

1. 写一个能表达行为的 red test；
2. 写最小实现变 green；
3. 记录仍未覆盖的行为。

重构、结构性清理和跨 seam 的质量判断应留给独立的 `/code-review`，但 `/code-review` 不应因此自动触发。

### 不应恢复的内容

- 不应默认调用 TDD。
- 不应默认运行全量测试、构建或集成测试。
- 不应把“使用 TDD”解释成允许无限循环或无限重试。
- 不应因为 TDD 而引入新的测试框架、依赖或运行环境。
- 不应把测试内部实现、私有方法或 side channel 作为主要测试目标。

### 下一步修正规格

后续修正 `skills/tdd/SKILL.md` 时，建议加入以下硬性流程：

1. 只有用户明确要求 test-first、TDD 或 red-green-refactor，或者当前任务明确标记为 focused TDD 时才启动。
2. 先声明模式：默认 `focused`，并说明目标 seam、单个行为和测试命令。
3. 先检查已有测试框架、命名约定、package layout、`CONTEXT.md` 和适用 ADR。
4. 只写一个行为测试，并确认 expected value 来自独立事实。
5. 如需运行，先请求一次 focused 测试命令授权。
6. 只实现使当前测试通过所需的最小代码。
7. 运行结果后停止当前循环；是否继续下一个 seam 由用户授权或任务明确范围决定。
8. 将重构和更广泛质量审查留给显式调用的 `/code-review`。

### 静态验收清单

- [ ] 默认不会自动启动 TDD。
- [ ] 有 `focused` 与 `full` 的范围区别，且 `focused` 是默认推荐模式。
- [ ] 每次最多处理一个 seam 和一个可观察行为，除非用户明确扩大范围。
- [ ] 运行测试前需要准确命令、范围和用户授权。
- [ ] 没有默认全量测试、构建或无限重试。
- [ ] `red`、`green`、`refactor` 的边界清晰。
- [ ] 保留 public interface、独立 expected value 和反模式规则。
- [ ] 与 `implement` 的“不自动运行测试”约束一致。
- [ ] 不引入新的依赖或测试框架。

### 当前结论

`tdd` 不应被禁用，而应改为受控的 `focused TDD` 能力：保留参考版本的 red-green、seam 和行为测试纪律，同时把运行命令、测试范围、重试次数和是否继续下一个循环设为明确边界。这样既支持有限度 TDD，也能降低意外账单风险。

本章节只记录评审建议，尚未修改 `skills/tdd/SKILL.md`。

## `business-review`

### 评审对象

- 本地版本：`skills/business-review/SKILL.md`
- 第三方参考版本：未在 `D:\03_projects\third-part\skills\skills\engineering`、`productivity` 或 `personal` 中找到同名文件。
- 关联 skill：`office-hours-finance`、`metrics-review`、`eng-review-finance`、`release-review`、`autoplan-finance`。

### 当前评价

当前质量约为 **8/10**。这是一个边界清楚的 business value gate：能把“看起来有用”拆成价值维度、证据、成本、范围、替代方案、验证路径和 owner，并与 `autoplan-finance` 的状态轴对齐。主要问题集中在仓库工作流适配和证据严谨性，而不是基本的金融产品判断框架。

### 当前版本值得保留的做法

1. 明确它只回答“是否值得推进”，不回答技术可行性或发布就绪性。
2. 要求至少命中 revenue、risk、labor、decision efficiency、regulatory、customer experience 之一，并拒绝“提升可见性”这类无证据表述。
3. 用 Value Matrix、Cost vs Benefit、Scope Challenge、Smaller Alternatives、Validation Path、Required Conditions 组织判断，结构完整且可审计。
4. 维护 `APPROVE`、`REDUCE`、`HOLD` 三种业务结论，避免把所有需求压成二元 go/no-go。
5. 强制指定 named business owner，并要求监管主张提供 rule id、audit finding id 或 sign-off。
6. 对缺少输入时保持 HOLD，而不是猜测业务收益、事故数量、法规或现有替代方案。
7. 明确 `APPROVE`、`REDUCE`、`HOLD` 与 `autoplan-finance` Plan Status 的映射，便于聚合。
8. 保留轻量验证、手工流程和已有报表作为全量构建之前的替代路径。

### 需要调整的内容

#### 1. 将 `TODO` 改为仓库约定的 `%%`

`AGENTS.md` 要求 unresolved risks or questions 使用 `%%`，当前 skill 多处要求 `TODO`。建议统一使用：

- `%% NEEDS_INPUT: ...`：用户或业务 owner 尚未提供的信息；
- `%% NEEDS_VALIDATION: ...`：可以通过轻量测量确认的价值假设；
- `%% BLOCKS_GATE: ...`：会阻塞某个 gate 的缺口。

空表格单元格可以保留，但不能让 TODO 伪装成已完成证据。输出应明确区分事实、用户确认、静态推断和未决事项。

#### 2. 将 80% 和 5 项阈值从硬规则改为可解释的启发式

“现有方案覆盖至少 80% 就 REDUCE”和“Must-Have 超过 5 项就 REDUCE”有助于控制范围，但这两个数字不是通用业务事实。建议：

- 保留为默认 heuristic，并要求说明 coverage 的测量方式和不确定性；
- 允许用户或业务 owner 提供不同阈值及其理由；
- 当 coverage 无法可靠估计时标记 `%% NEEDS_VALIDATION`，不要制造精确百分比；
- 不因数量超过 5 项自动降低高风险监管、恢复、账务正确性等不可压缩条件。

#### 3. 细化 owner 规则，避免把角色与个人混淆

模板同时要求 requesting stakeholder、named business owner（person, not team），但初始输入只要求 stakeholder role/team。建议允许两阶段：

- intake 阶段记录 accountable role/team；
- APPROVE 前必须由具体业务 owner 或明确授权的 accountable role 确认；
- 不知道姓名时保持 HOLD 或 `%% NEEDS_INPUT`，不得编造人名；
- 记录 owner 的决策权限和确认时间，避免只填名字却无法承担决策。

#### 4. 把“业务 APPROVE”与实现授权明确分开

`APPROVE` 只代表价值判断通过，不代表可以编码、创建 Task、创建 worktree、提交或发布。建议在 Final Recommendation 明确输出：

- `Business decision: APPROVE/REDUCE/HOLD`；
- `Engineering readiness: NOT YET REVIEWED`，除非另有 release/engineering 证据；
- `Next: metrics-review | eng-review-finance | to-spec | manual-validation`；
- 是否已存在 AIW Task/OpenSpec change，以及还缺哪些前置条件。

business-review 不执行实现后的 sync、archive、merge 或 worktree 清理。

#### 5. 修正 OpenSpec handoff 的写入和任务生成边界

当前文档说 standalone 且用户要求 OpenSpec 时，可以把 Required Conditions 追加到 `tasks.md`，并将每个 Blocks Decision=yes 转成 task。这里仍需明确：

- 默认只返回建议内容，不直接修改 repository 文件；
- 用户明确要求生成 OpenSpec 时，先确认 change ID 和目标目录；
- 业务条件不应全部自动变成 implementation task，只有已确认且可执行的条件才进入 `tasks.md`；
- 业务决策、价值证据和未决条件优先进入 proposal/design 或 `task.toml` 的 business block；
- 生成后必须检查与现有 proposal、design、spec、tasks 的一致性，不覆盖人工内容。

#### 6. 规范 `NEEDS_VALIDATION` 的状态表达

当前 Decision Model 将 `NEEDS_VALIDATION` 作为 HOLD 的 Next 指针，这是合理的，但输出模板只有 APPROVE/REDUCE/HOLD，容易让聚合器丢失“待测量”语义。建议固定为：

- `Decision: HOLD`；
- `Next: manual-validation`；
- `Validation status: NEEDS_VALIDATION`；
- 明确测量对象、样本范围、目标值、时间窗口、owner 和 kill criteria。

这样既保持业务状态轴稳定，也保留了下一步动作。

#### 7. 增加金融证据的时间、口径和敏感数据规则

价值判断涉及收入、损失、工时、客户体验和监管，应补充：

- 金额必须带币种、时间窗口、分母和数据来源；
- 工时收益说明计算口径，避免把节省时间直接当成现金节省；
- 事故或投诉数据注明去重、样本范围和观察期；
- 监管主张记录规则版本或审计发现时间；
- 不回显真实账户号、token、客户身份信息或不必要的敏感原文。

没有这些信息时，应使用 `%% NEEDS_VALIDATION`，而不是填入看似精确的收益。

#### 8. 明确 sibling review 的调用模式

business-review 可能由 `autoplan-finance` 调用，也可能独立调用。建议增加：

- `focused`：只审一个价值问题，其他维度标记未评估；
- `standard`：完整执行当前八步；
- `resume`：从现有 review 的第一个未完成 gate 继续。

不要默认调用 metrics、engineering 或 release review；只有决策路径需要且用户允许时才路由。缺少 sibling 时可以 self-run，但必须标记来源，不能声称 sibling 已执行。

### 不建议恢复的做法

- 不把 business `APPROVE` 当作工程 READY、release GO 或实现授权。
- 不继续使用 `TODO` 掩盖仓库中的未决风险。
- 不把 80% coverage 和 5 项 scope 当成没有证据的绝对定律。
- 不自动把所有 Required Conditions 写入 `tasks.md` 或创建 AIW Task。
- 不默认运行 SQL、测试、构建或网络查询来制造业务证据。
- 不自动写入 repository、tracker、PR 或发布系统。

### 后续修正建议

下一步可在 `skills/business-review/SKILL.md` 中补充：

1. `%%` 未决事项格式和 evidence status。
2. 80%/5 项规则的 heuristic、例外和测量要求。
3. owner、金额、币种、时间窗口和监管证据的最低字段。
4. Business decision 与 engineering/release readiness 的分离输出。
5. OpenSpec handoff 的只写入授权、一致性检查和 task 生成边界。
6. `focused/standard/resume` 模式与 sibling 调用条件。

### 静态验收清单

- [ ] 价值判断至少有一个可追溯的 person、incident、metric 或 regulator 证据。
- [ ] 未决信息使用 `%%`，并能阻塞对应 gate。
- [ ] 80%/5 项阈值带有测量依据或明确标为 heuristic。
- [ ] owner、币种、时间窗口、分母和来源不会被猜测。
- [ ] `APPROVE` 不会被解释为可以实现或发布。
- [ ] 默认不修改 OpenSpec、任务文件、tracker 或 PR。
- [ ] OpenSpec handoff 不会未经确认生成 implementation tasks。
- [ ] `NEEDS_VALIDATION` 保留在输出中，而不是丢失在 HOLD 之后。
- [ ] 不默认运行测试、构建、SQL 或网络查询。

### 结论

`business-review` 的价值门禁和结构化输出已经成熟。后续修正应集中在证据口径、`%%` 约定、阈值启发式、OpenSpec 写入边界和 business/engineering 状态分离；不需要大幅重写其核心流程。

本章节只记录评审建议，尚未修改 `skills/business-review/SKILL.md`。

## `code-review`

目标文件：`skills/code-review/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\engineering\code-review\SKILL.md`

### 当前评价

质量约为 8.5/10。当前版本已经很好地把参考版本迁移到 AIW/OpenSpec：使用本地 OpenSpec 作为规范来源，移除了对 issue tracker、GitHub/GitLab 和 `.scratch` 的依赖，并明确禁止测试、构建、网络、commit、archive 和 worktree 操作。主要还需要补充的是“有限度 review”的成本边界，以及它与新的自动完成闭环之间的职责边界。

### 参考版本中值得保留的做法

1. 用 fixed point 锁定审查范围，并使用三点 diff：`git diff <fixed-point>...HEAD`，避免把 fixed point 之前的共同历史重复纳入审查。
2. 在启动审查代理前验证 fixed point 可解析且 diff 非空，避免把无效输入交给子代理。
3. 将 review 分成 Standards 和 Spec 两条独立轴，避免“代码风格很好”掩盖“实现了错误需求”，也避免反过来。
4. Standards 轴同时读取仓库规范和 Fowler smell baseline，但把 smell 标记为判断性建议，而不是硬性违规；仓库规范优先。
5. Spec 轴优先读取本地原始规范，并分别识别遗漏、范围蔓延和实现错误。
6. 两个子代理并行、单次 review、每条轴独立汇报，最后不跨轴重新排序发现。
7. 每条发现要求关联文件/hunk 和规范依据，避免只给泛泛的质量评价。

### 当前版本已经做好的地方

- 已将 issue/PRD 来源改为 OpenSpec change 和稳定 spec，符合本项目的权威来源规则。
- 已要求读取 `task.toml`、proposal、design、capability specs 和 tasks。
- 已限制最多两个静态 review 子代理，并明确不运行测试、构建、网络或权限操作。
- 已要求 fixed point 可解析且 diff 非空。
- 已保留 Standards/Spec 双轴报告，不合并或跨轴重排发现。
- 已允许没有 spec 时跳过 Spec 轴并明确报告，而不是猜测需求。

### 需要重新增加或调整的内容

#### 1. 增加 focused review 模式

当前 skill 已经有“最多两个子代理、单次 review”的成本边界，但还没有把它定义成用户可理解的模式。建议增加：

- `focused`：默认推荐；只审查用户指定的 fixed point 到 `HEAD` 的当前 diff，最多两个子代理，各一轮，输出精简发现；
- `full`：只有用户明确要求完整 review 时才使用；可以读取更完整的规范和相关上下文，但仍不运行测试/构建，也不重复 review；
- 不支持自动循环修复、再次 review 或扩大 diff，除非用户重新授权。

#### 2. 保留显式调用，不在普通实现中自动触发

`code-review` 可以被用户明确要求，也可以作为项目配置中的一次 focused 质量步骤，但不应由普通 `/implement` 默认触发。这样既保留能力，也避免每个任务自动产生额外 token 和代理成本。

如果任务明确选择了 focused review，调用前应说明：fixed point、审查路径、两个审查轴、是否启用子代理，以及预期成本。

#### 3. 增加完成协议的职责边界

新的 AIW 完成协议包含 commit、sync、archive、merge 和清理。`code-review` 只负责提供静态发现，不负责：

- 修改代码；
- 自动修复发现；
- commit；
- archive；
- merge；
- 删除 worktree 或 branch。

如果 review 发现阻塞问题，`implement` 应停止完成协议，保留 Task worktree 和 branch，等待修复；如果没有阻塞问题，完成协议仍由 `/implement` 继续执行。

#### 4. 明确 dirty worktree 和固定点冲突

在读取 diff 前应确认：

- 当前 worktree 与 AIW Task metadata 匹配；
- fixed point 不是当前实现提交之后的 ref；
- 未提交修改是否纳入审查范围；
- 如果用户没有明确要求审查未提交修改，默认只审查 committed diff；
- 当前 worktree 有未提交修改且会影响审查范围时，先停止并说明范围不确定。

这能避免自动 commit 后审查范围发生歧义。

#### 5. 增加发现严重度和置信度

保留双轴独立报告的同时，每条发现建议标记：

- `blocker`：必须修复，阻止完成协议；
- `major`：高风险，默认建议修复后再合并；
- `minor`：非阻塞改进；
- `note`：观察或判断性建议。

Standards smell 仍然必须是判断性建议，不能仅因命中 smell 就标记为 blocker。只有明确规范违规或 Spec 行为缺失才可成为阻塞项。

#### 6. 冲突时转入冲突处理，不在 review 中自行解决

如果 merge 或 review 前置状态已存在冲突，`code-review` 应报告冲突并建议 `/resolving-merge-conflicts`，不应在 review skill 内执行 merge、解决冲突或提交。

### 不应恢复的内容

- 不应恢复 issue tracker、`.scratch`、GitHub/GitLab 作为默认规范来源。
- 不应自动运行测试、构建、格式化、网络请求或权限升级。
- 不应自动修改代码、commit、archive、merge 或清理 worktree。
- 不应在没有 fixed point 时自行猜测审查范围。
- 不应因为 smell baseline 命中就把判断性建议当作硬性违规。
- 不应把 Standards 和 Spec 发现合并成一个未经解释的总分。
- 不应在一次 review 后自动循环修复和再次 review。

### 下一步修正规格

后续修正 `skills/code-review/SKILL.md` 时，建议加入以下硬性流程：

1. 解析用户提供的 fixed point；缺少或无效时停止。
2. 解析当前 AIW Task、worktree、parent branch 和匹配 OpenSpec change。
3. 判断 committed diff 与未提交修改的范围；范围不清晰时停止。
4. 默认使用 focused 模式，锁定一次三点 diff 和一次 commit 列表。
5. Standards 与 Spec 分别读取各自来源，最多启动两个静态子代理。
6. 每条发现包含轴、严重度、文件/hunk、证据和建议；smell 发现标记为判断性建议。
7. 输出两个独立报告和各轴摘要，不自动修复或执行生命周期操作。
8. 只有没有 blocker，且用户/完成协议已授权时，才允许回到 `/implement` 继续 sync、archive、merge 和清理。

### 静态验收清单

- [ ] 默认模式是一次 focused review，而不是无限 review loop。
- [ ] fixed point、三点 diff 和空 diff 检查保持不变。
- [ ] Standards 与 Spec 仍然独立报告。
- [ ] OpenSpec 是默认规范来源，外部 Issue 不是默认来源。
- [ ] 最多两个静态子代理，不运行测试、构建、网络或生命周期操作。
- [ ] 未提交修改的审查范围有明确规则。
- [ ] 发现包含严重度、证据和文件/hunk 定位。
- [ ] smell baseline 不会自动升级为硬性阻塞项。
- [ ] review 发现 blocker 时，完成协议会停止并保留 worktree/branch。
- [ ] review 通过后，完成协议仍由 `/implement` 负责，而不是由 review skill 自行 merge 或清理。

### 当前结论

`code-review` 的核心设计应保留。它已经是一个成本相对可控的静态双轴审查 skill，建议只增加 focused/full 模式、dirty worktree 范围、严重度定义和与 AIW 完成协议的边界。不要把它重新变成默认自动执行的昂贵步骤。

本章节只记录评审建议，尚未修改 `skills/code-review/SKILL.md`。

## `codebase-design`

目标文件：`skills/codebase-design/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\engineering\codebase-design\SKILL.md`

关联参考：`DEEPENING.md`、`DESIGN-IT-TWICE.md`

### 当前评价

质量约为 8.5/10。当前版本基本保留了参考版本最重要的深模块词汇、接口/seam 纪律和可测试性原则，并额外加入了 AIW 成本与操作边界。它适合作为其他工程 Skill 的词汇层，而不是一个自动执行重构的实现流程。

### 参考版本中值得保留的做法

1. 用固定词汇建立架构语言：module、interface、implementation、depth、seam、adapter、leverage、locality。
2. 把 depth 定义为接口带来的 leverage，而不是简单比较实现代码行数和接口代码行数。
3. 使用 deletion test 判断模块是否真正隐藏了复杂度，识别 pass-through 模块。
4. 将 interface 作为 callers 和 tests 共同穿越的 test surface，避免测试深入实现内部。
5. 使用“one adapter means hypothetical seam, two adapters means real seam”限制无理由的抽象和 indirection。
6. 通过 `DEEPENING.md` 将依赖分成 in-process、local-substitutable、remote but owned、true external，并为不同类型给出 seam 与 adapter 策略。
7. 通过 `DESIGN-IT-TWICE.md` 支持替代 interface 设计，并用 depth、locality、seam placement 比较，而不是只列选项。
8. 将详细的 deepening 和 alternative-design 过程放到外部参考文件，保持 `SKILL.md` 的核心词汇集中。

### 当前版本已经做好的地方

- 已保留完整的深模块词汇和关系定义。
- 已明确 interface 包含 invariants、ordering、error modes、configuration 和 performance characteristics，不局限于类型签名。
- 已保留 deep/shallow 对比、deletion test、interface test surface 和 adapter seam 规则。
- 已保留依赖注入、返回结果和小 interface 等可测试性原则。
- 已把 `DEEPENING.md` 和 `DESIGN-IT-TWICE.md` 作为渐进披露的外部参考。
- 已加入最多两个 bounded static passes，且禁止测试、创建 branch 或 worktree，符合当前 AIW 成本和生命周期边界。

### 需要重新增加或调整的内容

#### 1. 明确这是词汇/设计参考，不是自动重构授权

当前 skill 主要是 reference，流程很少，这是合理的。但应明确：

- 直接调用 `codebase-design` 时，默认输出设计分析、候选 seam 和 trade-off；
- 不默认修改代码、创建 branch/worktree、运行测试或提交；
- 只有用户明确进入实现流程时，才把结论带入 `/to-spec` 或已有 AIW Task；
- 设计结论应进入 proposal/design/ADR 等对应的权威位置，而不是停留在聊天中。

#### 2. 将 `DESIGN-IT-TWICE` 的 3+ sub-agents 改成资源档位

参考文件要求 3 个以上 sub-agents，这不应被直接视为错误。并行设计有时能显著缩短探索时间，但成本通常近似随 agent 数量增加，因此应根据问题规模选择档位：

| 档位 | sub-agents | 适用情况 | 约束 |
| --- | ---: | --- | --- |
| `focused` | 0 | 一个明显模块、接口问题边界清楚 | 主 agent 设计两个候选，不启动子 agent |
| `bounded` | 1–2 | 普通接口设计、一个主要依赖或 seam | 每个 agent 一个设计方向；默认档位 |
| `expanded` | 3–4 | 关键公共接口、多个调用方、明显存在多种架构方向 | 需要用户明确授权；每个 agent 限定 brief 和输出长度 |
| `wide` | 5–6 | 高价值、跨模块、难以回滚的核心架构决策 | 仅在用户明确要求资源换速度时使用；先报告预计成本和收益 |

建议的默认选择：

- 小型设计：`focused`；
- 一般设计：`bounded`；
- 重要公共接口：`expanded`；
- `wide` 不应成为常规路径，只用于高价值决策。

如果系统级 AIW 规则仍限制最多两个 sub-agents，`expanded` 和 `wide` 只能作为未来资源配置建议，当前执行必须降级为 `bounded` 并明确告知用户。

建议每个子 agent 使用以下资源边界作为起点：

- 单个设计 brief：只提供相关模块、调用方、依赖类别和一个设计约束；
- 单个输出：接口形状、使用示例、隐藏实现、adapter 策略、trade-off；
- 输出长度：约 300–600 字或等量简洁结构化内容；
- 执行轮次：一次，不自动追问、不自动修订、不自动再次比较；
- 主 agent 汇总：只保留 2–3 个真正不同的候选，删除重复方案；
- expanded/wide 完成后：只做一次比较和推荐，不自动进入实现或 review。

这些是规划参考值，不是精确账单预测。实际成本取决于模型、上下文大小、工具读取量和是否发生重试；扩大并发前应优先减少每个 agent 的上下文和输出。

#### 3. 增加设计工作的完成标准

设计分析不能以“讨论过模块”作为完成条件。至少应产出：

- 当前模块及 callers 的问题边界；
- interface、implementation、seam、adapter 的明确命名；
- 至少一个候选接口及其隐藏的复杂度；
- depth、leverage、locality 和 seam placement 的 trade-off；
- 采用或拒绝某个接口的理由；
- 未解决的风险使用 `%%` 记录，而不是猜测。

如果使用 deepening 或 design-it-twice，还应记录依赖类别、adapter 方案和被拒绝的候选设计。

#### 4. 防止 vocabulary 与领域词汇冲突

“Use these terms exactly” 对架构词汇很有价值，但不能覆盖项目已有领域术语。建议增加优先级：

- 架构讨论使用 module/interface/seam/adapter 等固定词汇；
- 业务概念继续使用项目 domain glossary 中的词；
- 不为了满足词汇规则而把真实领域概念改名成 module 或 component。

#### 5. 将“一个 adapter/两个 adapter”作为启发式而不是硬规则

该规则能防止过早抽象，但存在真实 seam 只有一个生产 adapter、测试通过 fake 或 contract test 完成的情况。建议写成：

- 一个 adapter 通常表示假设性的 seam，应要求额外证据；
- 两个 adapter 是 seam 真实存在的强证据；
- 只要 variation、替换需求或测试隔离有明确证据，也可以提前建立 seam，并记录理由。

#### 6. 与 TDD、code-review 和 OpenSpec 衔接

- 如果设计目的是选择 test seam，结论应提供给 focused TDD，而不是自动启动 TDD；
- 如果设计目的是审查模块深度，交给显式调用的 focused code review，而不是自动触发 review；
- 如果设计决定改变模块 interface、adapter 或持久化边界，应进入 `/to-spec` 的 design/spec，再进入 `/implement`；
- `codebase-design` 自身不负责 commit、sync、archive、merge 或清理。

### 不应恢复的内容

- 不应默认启动 3 个以上 sub-agents。
- 不应自动创建 branch/worktree 或执行实现。
- 不应自动运行测试、构建或 code review。
- 不应把 Ousterhout 的 depth-as-lines-ratio 作为硬指标。
- 不应把所有边界都称为 boundary，覆盖 DDD bounded context 的语义。
- 不应把架构词汇规则用于替换业务领域语言。
- 不应把 smell、浅模块或单 adapter 直接判定为必须重构。

### 下一步修正规格

后续修正 `skills/codebase-design/SKILL.md` 及其参考文件时，建议：

1. 在开头明确“这是设计词汇和分析参考，默认不修改、不运行、不创建生命周期资源”。
2. 保留现有词汇、深/浅模块、deletion test、interface test surface 和 adapter 原则。
3. 在 `DESIGN-IT-TWICE.md` 中增加 focused/bounded/full 三种探索模式，并将当前默认限制设为最多两个 sub-agents。
4. 为设计任务增加可交付的分析结果和 `%%` 风险记录。
5. 明确架构词汇与领域 glossary 的优先级和共存方式。
6. 将 one/two adapter 规则改为带证据的启发式。
7. 明确设计结论如何回流到 OpenSpec design/spec 和 focused TDD/code review。

### 静态验收清单

- [ ] 核心词汇仍与参考版本一致。
- [ ] `DEEPENING.md` 和 `DESIGN-IT-TWICE.md` 的上下文指针仍然有效。
- [ ] 默认模式不修改代码、不运行检查、不创建 branch/worktree。
- [ ] alternative design 的 sub-agent 数量有 focused/bounded/expanded/wide 档位和成本边界。
- [ ] 设计结果有明确完成标准和 `%%` 风险处理。
- [ ] 架构词汇不会覆盖项目 domain glossary。
- [ ] one/two adapter 是启发式，不是无例外的硬规则。
- [ ] 设计结论能回流到 OpenSpec、TDD 或 code review，但不会自动触发它们。
- [ ] 不会触发 commit、sync、archive、merge 或 worktree 清理。

### 当前结论

`codebase-design` 已经是质量较高的参考型 skill，不建议大幅重写。主要改进是把设计探索的成本模式、完成标准、领域词汇优先级和 OpenSpec 衔接写清楚；参考版本中要求 3+ sub-agents 的做法不应直接恢复。

本章节只记录评审建议，尚未修改 `skills/codebase-design/SKILL.md`。

## `diagnosing-bugs`

目标文件：`skills/diagnosing-bugs/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\engineering\diagnosing-bugs\SKILL.md`

### 当前评价

质量约为 8/10。核心方法非常强：先建立能针对用户症状变红的紧反馈回路，再复现、最小化、提出可证伪假设、逐变量探针、修复并写回归测试。当前版本与参考版本基本一致，并保留了本地的 `hitl-loop.template.sh`。

主要缺口是它默认带有较强的运行倾向，而当前 AIW 规则要求运行测试、脚本、性能测量或生产 instrumentation 前取得明确授权。还需要给“努力构造反馈回路”增加资源和停止边界。

### 参考版本中值得保留的做法

1. 把 tight、red-capable feedback loop 作为整个诊断流程的核心，而不是一上来猜原因。
2. 规定反馈回路必须驱动真实 bug code path，并断言用户描述的精确症状。
3. 先复现再最小化，要求每个保留元素都对失败现象有负载作用。
4. 生成 3–5 个有排序、可证伪的 hypotheses，避免单一假设锚定。
5. 每个 probe 只验证一个 prediction，并一次只改变一个变量。
6. 性能问题先建立 baseline measurement，再测量和 bisect，不用泛滥日志代替测量。
7. 回归测试必须位于能复现真实 bug pattern 的正确 seam；没有正确 seam 本身就是架构发现。
8. 收尾时重新运行原始 repro、删除 debug instrumentation 和 throwaway prototype，并记录 post-mortem。
9. 只有修复后才判断是否需要 `/improve-codebase-architecture`，避免在证据不足时提前重构。

### 当前版本已经做好的地方

- 已要求读取 `CONTEXT.md` 和相关 ADR。
- 已保留从 failing test、curl、CLI、browser、trace、harness、fuzz、bisect、differential 到 HITL 的反馈回路选择顺序。
- 已明确 Phase 1 的完成条件：一个已经运行过、快速、确定、可自动执行且能针对症状变红的命令。
- 已保留最小化、假设排序、instrumentation 标签、性能分支和正确 seam 规则。
- 已要求清理 debug 日志和 throwaway prototype。
- 没有自动创建 branch/worktree 的行为，符合当前 Task 生命周期边界。

### 需要重新增加或调整的内容

#### 1. 增加运行授权门槛

进入诊断不等于自动获得运行权限。建议明确：

- 静态检查、阅读代码、阅读日志和提出假设可以先做；
- 第一次运行测试、脚本、curl、浏览器、fuzz、profiler 或 bisect 前，展示准确命令、范围、预计时长和可能的外部影响；
- 等用户授权后才运行；
- 生产环境 instrumentation、网络请求、真实数据 replay 和大规模 fuzz 需要单独授权；
- 没有授权时，可以形成静态诊断计划，但不能声称已经复现。

#### 2. 将“tight loop”改成有预算的诊断模式

建议增加资源档位：

| 模式 | 运行预算 | 适用情况 |
| --- | --- | --- |
| `static` | 0 次运行 | 只允许代码、配置、日志和历史分析 |
| `focused` | 1 个命令，初始 1 次运行；相关变更后最多重跑 1 次 | 默认推荐，验证一个高价值症状 |
| `bounded` | 1 个命令，约 3–5 次运行；fuzz 约 20–100 个样本 | 间歇性 bug 或需要最小化输入 |
| `deep` | 明确的次数、时长和资源上限 | 只有用户明确要求或问题价值足够高时使用 |

默认使用 `focused`。不能把参考版本里的“loop 100 次”“fuzz 1000 inputs”“parallelise”当成无条件执行指令；这些只能在 `bounded`/`deep` 模式并获得授权后使用。

#### 3. 为每个阶段增加停止条件

- Phase 1：在预算内仍不能构造 red-capable loop，就停止并报告缺失环境或 artifact；
- Phase 2：最小化连续若干次没有减少 load-bearing 输入时停止；
- Phase 3：形成 3–5 个假设后停止继续扩展列表；
- Phase 4：每个假设最多先做一个能区分预测的 probe；
- Phase 5：修复前只写一个最高价值 regression test；
- Phase 6：原始 repro、回归测试、debug 清理和记录完成后停止，不自动扩大测试范围。

“Be aggressive” 应解释为优先改善信号质量，而不是无限增加运行次数、日志量或 agent 数量。

#### 4. 调整 Phase 3 的用户检查点

参考版本允许用户不响应时继续测试假设，但这不适合当前成本和运行授权规则。建议：

- 先展示 3–5 个排序假设和每个验证命令；
- 如果后续运行需要授权，等待用户确认；
- 用户未回复时停在计划状态，不执行运行命令；
- 如果只是静态分析，不需要等待即可继续整理证据。

#### 5. 明确诊断与 TDD、implement、code-review 的关系

- diagnosis 可以先构造 feedback loop，但不自动进入完整 TDD；
- 有正确 seam 时，可以在用户授权下交给 focused TDD 写 regression test；
- 修复实现必须回到 AIW Task 和其 worktree，通过 `/implement` 完成；
- `/code-review` 只能显式调用，且只审查最终 diff；
- diagnosis 本身不 commit、sync、archive、merge 或清理 worktree。

#### 6. 防止危险反馈回路

以下操作需要更高门槛和明确范围：

- 生产环境 curl、真实网络请求和真实数据 replay；
- 发送写操作、支付、消息、删除、迁移或外部 API 调用；
- 并行 stress、长时间 profiler、无限循环和大规模 fuzz；
- 添加临时生产 instrumentation；
- `git bisect run` 中会修改环境或产生外部副作用的脚本。

优先使用 fixture、mock、沙箱、只读请求和本地 replay。

### 不应恢复的内容

- 不应在没有用户授权时运行反馈回路。
- 不应把“已建立 loop”与“已运行 loop”混淆。
- 不应无限重试 flaky bug、无限扩大 fuzz 或并行 stress。
- 不应在没有正确 seam 时伪造回归测试。
- 不应在诊断过程中自动重构架构、commit、archive、merge 或清理。
- 不应使用生产 instrumentation、真实数据或外部副作用替代本地可控复现。

### 下一步修正规格

后续修正 `skills/diagnosing-bugs/SKILL.md` 时，建议：

1. 保留六阶段诊断结构和 tight/red-capable loop 作为 leading idea。
2. 增加 `static`、`focused`、`bounded`、`deep` 模式和运行授权规则。
3. 把“已经运行过一次”改成“只有获得授权并实际运行后，才能标记为已复现”。
4. 为 fuzz、stress、bisect、profiler、网络和生产 instrumentation 增加预算和危险操作门槛。
5. 将 Phase 3 的假设确认改为运行前的显式成本/命令确认点。
6. 明确修复回到 `/implement`，TDD 和 code review 都是显式、受控的后续能力。
7. 保留无 loop 时停止并索取环境、HAR、日志、core dump 或录屏的行为。

### 静态验收清单

- [ ] 默认诊断模式不会运行命令。
- [ ] 第一次运行前需要命令、范围、时长和用户授权。
- [ ] `focused` 是默认运行档位，并有明确重试上限。
- [ ] fuzz、stress、bisect、profiler 和生产 instrumentation 有额外边界。
- [ ] Phase 1、2、3、4、5、6 都有可检查的停止条件。
- [ ] 仍然要求真实症状、正确 seam、最小 repro 和独立 regression test。
- [ ] 没有 loop 时不会继续凭空假设并声称已诊断。
- [ ] diagnosis 不会自动触发 TDD、code-review 或 AIW 完成协议。

### 当前结论

`diagnosing-bugs` 的诊断方法本身值得保留，但必须从“积极运行直到解决”调整为“授权后、预算内、证据驱动地运行”。建议优先增加运行授权和 focused/bounded 资源模式，再考虑其他文字优化。

本章节只记录评审建议，尚未修改 `skills/diagnosing-bugs/SKILL.md`。

## `domain-modeling`

目标文件：`skills/domain-modeling/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\engineering\domain-modeling\SKILL.md`

关联参考：`CONTEXT-FORMAT.md`、`ADR-FORMAT.md`

### 当前评价

质量约为 9/10。当前版本与第三方参考版本一致，内容短而有明确行为：挑战词汇、澄清模糊概念、用具体场景施压、对照代码事实、及时更新 glossary，并只在满足三个条件时创建 ADR。它是一个高质量的 vocabulary-layer skill，不需要大幅扩展。

### 参考版本中值得保留的做法

1. 明确这是“主动改变领域模型”的 discipline，而不是普通 skill 读取 `CONTEXT.md` 的习惯。
2. 用 `CONTEXT.md`、`CONTEXT-MAP.md` 和局部 `docs/adr/` 支持单上下文与多上下文仓库。
3. 只在有新内容时 lazy-create glossary 和 ADR 文件，避免空文件和文档噪声。
4. 用户术语与既有 glossary 冲突时立即指出，而不是默默采用新词。
5. 对模糊或重载术语提出 canonical term，并用具体场景验证概念边界。
6. 将用户描述与代码事实交叉核对，主动暴露 domain model 与实现不一致。
7. 严格区分 `CONTEXT.md` 与 implementation spec：前者只保存领域语言，不保存实现细节或临时讨论。
8. 只有同时满足“难以逆转、没有上下文会令人意外、来自真实 trade-off”时才建议 ADR，避免 ADR sediment。

### 当前版本已经做好的地方

- 完整保留参考版本，没有发现有价值的核心内容被删除。
- 文件结构、单/多上下文支持和 lazy creation 规则完整。
- `CONTEXT.md`、ADR 和 spec 的职责边界清楚。
- ADR 触发条件足够严格，能减少无价值的架构记录。
- 没有自动运行测试、创建 branch/worktree、commit、archive 或 merge 的行为。

### 需要重新增加或调整的内容

#### 1. 补充 AIW/OpenSpec 的落点规则

当前 skill 说明了写入 `CONTEXT.md` 和 ADR，但没有说明这些修改如何进入 AIW/OpenSpec 生命周期。建议增加：

- 如果领域词汇或 ADR 只是设计讨论的一部分，先写入当前 AIW Task 对应的 OpenSpec design/proposal，再按需要同步到稳定 glossary/ADR；
- 如果修改的是稳定领域模型，必须在当前 Task 的变更记录中说明影响范围；
- `CONTEXT.md` 和 ADR 的更新应在当前分支提交，并由 `to-spec`/`implement` 的完成协议继承、sync、archive 和 merge；
- 不在 `CONTEXT.md` 中复制 `proposal.md`、`design.md` 或 capability spec 的实现要求。

#### 2. 增加“用户确认”与“直接记录”的边界

“Update CONTEXT.md inline” 对词汇已经明确的情况很高效，但如果一个词有多个合理解释，不能直接覆盖既有 glossary。建议：

- 词义已由用户或现有权威材料明确：直接记录，并报告变更；
- 存在多个候选定义：先提出选择，不覆盖 `CONTEXT.md`；
- 与代码冲突：先记录 contradiction 和待决问题，除非用户明确决定，不擅自把代码事实改写成领域真相；
- 更新 ADR 前，确认三项 ADR 条件都满足。

#### 3. 增加多上下文冲突处理

有 `CONTEXT-MAP.md` 时，术语可能在不同 bounded context 中合法地含义不同。建议增加：

- 先根据当前模块、路径和 Task 确定目标 context；
- 不把局部 context 的词义提升为全局 glossary；
- 同名不同义时记录 context-qualified term，而不是强行统一；
- 跨 context 的共享术语或映射关系才进入系统级上下文或 ADR。

#### 4. 增加输出完成标准

领域建模完成时至少应明确：

- canonical term 及其定义；
- 被拒绝或淘汰的近义词/旧术语；
- 术语适用边界和至少一个 edge scenario；
- 与代码或现有文档的冲突及解决方式；
- 是否需要 ADR，以及三项条件是否满足；
- 修改了哪些 glossary/ADR 文件；
- 未解决问题使用 `%%` 记录。

#### 5. 与其他 vocabulary layer 保持边界

- `domain-modeling` 负责业务概念、关系、边界和业务决策；
- `codebase-design` 负责 module、interface、seam、adapter、depth 等架构词汇；
- 两者都可以讨论“边界”，但应避免用架构术语替换领域术语，或反过来；
- `to-spec` 负责把已经稳定的领域结论转化为 proposal/design/spec，不应在 spec 中重新发明 glossary。

### 不应恢复的内容

- 不应把 CONTEXT 当作需求 spec、implementation plan 或 scratch pad。
- 不应为每个小术语或普通实现选择创建 ADR。
- 不应在多 context 仓库中强行统一本地同名术语。
- 不应无确认地覆盖与用户或代码事实冲突的既有定义。
- 不应在 vocabulary 讨论中自动创建 Task、worktree、branch 或运行测试。
- 不应把领域建模结论直接当作已批准的实现授权。

### 下一步修正规格

后续修正 `skills/domain-modeling/SKILL.md` 时，建议只做小幅增强：

1. 保留现有全部核心内容和外部格式参考。
2. 增加 AIW/OpenSpec 的记录落点和提交继承规则。
3. 增加多 context 冲突处理与用户确认边界。
4. 增加领域建模完成标准和 `%%` 风险记录。
5. 明确与 `codebase-design`、`to-spec` 的职责分工。
6. 保持它作为词汇层，不将其扩展成自动实现流程。

### 静态验收清单

- [ ] 仍然区分主动改变 domain model 与仅仅读取 CONTEXT。
- [ ] 支持单 context 和多 context。
- [ ] CONTEXT、ADR、OpenSpec spec 的所有权边界清楚。
- [ ] 词汇冲突、多候选定义和代码矛盾有停止/确认规则。
- [ ] ADR 仍要求三个条件同时满足。
- [ ] 领域建模结果有 canonical term、边界场景、影响文件和 `%%` 风险。
- [ ] 不自动触发测试、TDD、code review 或 AIW 完成协议。
- [ ] 稳定文档修改能进入当前 Task 提交并被后续 worktree 继承。

### 当前结论

`domain-modeling` 是目前评审到的高质量 skill 之一，参考版本的核心做法基本全部保留。建议只增加 AIW/OpenSpec 落点、多 context 冲突、用户确认和完成标准，不建议引入更多流程或自动化行为。

本章节只记录评审建议，尚未修改 `skills/domain-modeling/SKILL.md`。

## `edit-article`

目标文件：`skills/edit-article/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\personal\edit-article\SKILL.md`

### 当前评价

质量约为 5/10。当前版本与参考版本完全一致，核心想法是对的：先按章节建立文章结构，再考虑信息依赖，最后逐节改写。但 skill 只有两个未完成的步骤，缺少输入范围、编辑目标、事实保真、输出格式、完成标准和异常情况处理，容易产生不可预测的重写结果。

### 参考版本中值得保留的做法

1. 先按标题拆分文章，而不是直接从第一段开始逐句润色。
2. 把文章看成有依赖关系的信息图，确保前置概念先于依赖它们的结论出现。
3. 在逐节编辑前确认章节结构，避免局部改写破坏全篇论证顺序。
4. 逐节处理，减少一次性重写整篇文章导致的语义漂移。
5. 限制段落长度，鼓励清晰、紧凑的表达。

### 当前版本已经做好的地方

- `disable-model-invocation: true` 合理：文章编辑通常需要用户明确调用。
- description 简洁、触发条件清楚。
- 章节结构和信息依赖被放在编辑动作之前。
- 明确要求先与用户确认章节，具备一个有价值的人工检查点。

### 主要问题

#### 1. 流程不完整

当前只有“拆章节”和“改写段落”，没有说明：

- 如何读取文章和识别标题；
- 没有标题时如何分段；
- 如何确定文章目标、读者和语气；
- 如何处理表格、代码、引用、链接、图片说明和脚注；
- 如何处理事实错误或不确定事实；
- 如何输出修改后的文章和修改摘要；
- 如何确认编辑已完成。

#### 2. “确认章节”可能造成不必要的阻塞

如果用户明确要求“直接润色全文”，还要求先确认章节，会增加一次不必要的交互。建议：

- 文章结构明显且用户授权直接编辑：先给出简短结构假设，然后继续；
- 结构存在明显歧义、顺序变化较大或会删减内容：先请求确认；
- 仅做语句级润色：不必重新确认章节结构。

#### 3. 240 字符/字符数限制不够明确

`maximum 240 characters per paragraph` 没有说明适用语言，也没有解释为什么是 240。中文字符、英文单词、代码和引用的计数方式不同；硬性限制还可能破坏复杂论证。

建议改成默认的可读性目标：一个段落表达一个主要动作或观点，通常控制在 2–5 句；只有用户明确要求发布平台字数限制时，才执行精确字符/字数限制。若必须限制，先声明计数规则和例外内容。

#### 4. 缺少“保留事实，不凭空补写”的规则

文章编辑和文章写作不同。默认应：

- 保留原文事实、观点、引用和限定条件；
- 不把润色中的推断写成事实；
- 发现事实矛盾或引用不完整时，用 `%%` 标记或向用户询问；
- 需要外部核查时，只有用户明确授权才浏览或查证；
- 不擅自删除可能影响结论的段落。

#### 5. 缺少编辑目标和优先级

“improve” 可能意味着清晰、简洁、说服力、技术准确性、SEO、风格统一或语气变化。建议让编辑目标按以下优先级处理：

1. 保持原意和事实边界；
2. 修复结构和论证依赖；
3. 改善清晰度、连贯性和可读性；
4. 按用户指定的读者、语气和格式调整；
5. 最后才做压缩、修辞和风格优化。

#### 6. 缺少局部修改与全文修改的边界

建议支持三种范围：

- `proofread`：拼写、语法、标点和明显表达问题；
- `revise`：结构、清晰度、连贯性和段落重写；
- `rewrite`：允许较大幅度重组，但必须先确认目标读者、保留内容和结构方案。

默认使用 `revise`，不要把所有“edit”都理解成全文重写。

### 下一步修正规格

后续修正 `skills/edit-article/SKILL.md` 时，建议补成以下流程：

1. 读取文章并识别标题、段落、列表、表格、代码、引用和链接。
2. 判断编辑范围：`proofread`、`revise` 或 `rewrite`；如果用户没有说明，默认 `revise`。
3. 根据文章目标、读者、语气和约束建立结构草案。
4. 分析章节的信息依赖，找出前置定义、论据、结论和重复内容。
5. 结构变化明显时先请求确认；结构清楚且用户要求直接编辑时，说明假设后继续。
6. 逐节修改，保持事实、引用、限定条件和原意；不确定内容用 `%%` 标记。
7. 处理跨章节衔接，避免局部改写破坏全局论证。
8. 输出修改后的文章、编辑摘要、未解决的 `%%` 问题和需要用户决定的取舍。

### 静态验收清单

- [ ] 支持 proofread/revise/rewrite 三种范围。
- [ ] 默认不会把编辑任务扩展成全文重写。
- [ ] 章节结构和信息依赖在改写前得到处理。
- [ ] 用户明确授权直接编辑时不会无谓阻塞；结构有重大歧义时会确认。
- [ ] 保留原文事实、引用、限定条件和原意。
- [ ] 不确定事实、冲突和缺失引用使用 `%%` 或请求用户确认。
- [ ] 240 字符限制不再作为无条件硬规则。
- [ ] 明确处理非正文内容和格式元素。
- [ ] 输出包含修改结果、摘要和未解决问题。
- [ ] 不自动发布、提交、归档或修改外部系统。

### 当前结论

`edit-article` 的基本方向正确，但内容明显不完整。建议保留“先结构、后逐节”的核心做法，重新补充编辑范围、结构确认策略、事实保真、格式处理和输出完成标准。它应继续保持轻量的用户主动调用 skill，不需要引入复杂的 agent 或工程生命周期流程。

本章节只记录评审建议，尚未修改 `skills/edit-article/SKILL.md`。

## `eng-review-finance`

目标文件：`skills/eng-review-finance/SKILL.md`

参考文件：未找到第三方对应版本。以下评审基于当前 skill、`references/` 下的四个参考文件、`scripts/validate_eng_review.py` 和测试脚本。

### 当前评价

质量约为 8.5/10。当前 skill 是一个边界清楚、结构完整的金融系统架构评审器，尤其适合权限、审计、数据契约、失败模式和发布影响容易被遗漏的后台/报表/数据系统。它已经比一般的“给一段架构建议”更可检查，但仍需解决输出方式、状态门禁、AIW/OpenSpec 落点和资源范围几个一致性问题。

### 当前版本做得好的地方

1. 明确这是 implementation 前的只读 architecture review，不写代码、迁移、部署脚本或 PR。
2. 明确了不适用场景，并将 metrics、business value、release readiness 分流给对应 Skill。
3. 对金融系统关键风险设置硬门禁：指标、边界、数据契约、权限、审计、失败模式、可观测性、测试和发布影响。
4. 对敏感权限和审计字段要求具体，避免用“需要权限控制”“需要审计”这类空泛表达。
5. 使用 `READY / INCOMPLETE / HIGH RISK` 与 `GO / GO WITH RISK / NO GO / NOT YET REVIEWED` 两条独立状态轴，避免工程准备度和发布许可混淆。
6. 输出格式使用表格覆盖边界、模块、数据流、权限、审计、失败模式、监控和测试，适合机器解析和后续 handoff。
7. references 与 validator 采用渐进披露，主 skill 保留工作流和门禁，模板细节外置。
8. 与 `office-hours-finance`、`business-review`、`metrics-review`、`release-review`、`autoplan-finance` 的上下游关系已经写清楚。

### 需要重新增加或调整的内容

#### 1. 解决“emit ENG_REVIEW”与“只返回消息”的冲突

开头写“emits a structured `ENG_REVIEW.md`”，Outputs 又规定默认只返回消息、不写磁盘。建议明确三种输出模式：

- `message`：默认，直接返回结构化 Markdown，不写文件；
- `artifact`：用户明确要求时，生成指定路径的 `ENG_REVIEW.md`；
- `openspec`：用户明确要求时，只返回或写入映射后的 design/permissions/audit/release 内容。

每种模式都应明确是否允许文件写入，避免 agent 误把“输出文档”当成“修改仓库”。

#### 2. 与 AIW/OpenSpec 工作流对齐

当前 skill 可以读取 OpenSpec，但没有明确：

- 若作为已有 AIW Task 的规划步骤运行，应读取对应的 `task.toml`、proposal、design、spec 和 tasks；
- review 结论默认作为消息或当前 Task 的 planning input，不直接覆盖 OpenSpec 文件；
- 用户明确要求 OpenSpec 输出时，应回到当前 change，不能创建平行 `.scratch` 文档；
- 设计评审完成后，下一步应由 `/to-spec` 或 `/to-tickets` 接管，而不是直接进入实现；
- `ENG_REVIEW.md` 的生成文件若被用户要求保存，应在当前分支提交，并让后续 AIW worktree 继承。

#### 3. 将 TODO 统一为项目的 `%%` 风险语义

当前项目约定未解决风险和问题使用 `%%` notes，而本 skill 多处要求 `TODO`。建议：

- 面向人和 AIW/OpenSpec 的未决问题使用 `%%`；
- 如果 validator 需要 `TODO` 作为结构占位，可以在模板中保留，但说明它不是风险记录的唯一格式；
- 不得用 TODO 掩盖缺失的 owner、权限、来源、阈值或 rollback 决策。

#### 4. 增加有限度 review 模式

金融评审的 13 个章节可能产生很长输出。建议增加：

- `focused`：只评审用户点名的风险轴，例如 permissions + audit 或 data flow + failure modes；
- `standard`：执行当前 12 个 review passes，默认模式；
- `full`：额外读取相关 references、历史设计和发布资料，仅用户明确要求时使用。

无论模式如何，失败的早期 gate 仍应阻塞后续结论，不能为了“完整输出”而猜测缺失信息。

#### 5. 加强敏感数据处理规则

这是金融系统 review，建议补充：

- 不要求用户粘贴生产凭证、完整客户数据、账户号、支付信息或敏感日志；
- 示例数据默认脱敏，使用字段类型和数据分类描述；
- 真实数据 replay、生产查询和外部系统访问需要单独授权；
- 权限矩阵应评审角色/范围/动作，不保存不必要的个人数据。

#### 6. 增加门禁结果的证据要求

每个 gate 不应只输出状态，至少要有：

- 输入证据或引用来源；
- 缺失决策；
- 风险影响；
- owner；
- 下一步动作；
- 是否阻塞后续 gate。

这样 `INCOMPLETE` 和 `HIGH RISK` 才能直接交给 `metrics-review`、`release-review` 或 `/to-spec`。

#### 7. 明确测试策略是设计评审，不是运行授权

当前输出包含 testing strategy，这是正确的，但应明确：

- 评审只定义应覆盖的 unit/integration/e2e/data/permission/audit/migration 场景；
- 不因此自动运行测试或构建；
- 如果用户明确授权，只能交给 focused TDD 或指定测试命令执行；
- migration + rollback 的设计门禁不等于已经验证过 migration。

#### 8. 校验器和模板应成为真正的完成标准

已有 validator 是优点，但主 skill 应明确：

- `message` 模式也必须生成完整的 13 个 heading 结构；
- 空章节保留 heading 并标记缺失决策；
- `--strict` 校验失败时不能返回 READY；
- validator 只能验证结构，不能替代人工判断或证明风险已解决。

### 不应恢复或引入的内容

- 不应自动写入 `openspec/`、`ENG_REVIEW.md` 或其他文件，除非用户选择 artifact/openspec 模式。
- 不应自行补全指标、数据契约、权限、审计、owner、阈值或监管要求。
- 不应把 `READY` 当成发布批准，也不应把 `GO` 当成工程设计完成。
- 不应因为模板字段很多而要求所有金融系统都拥有不适用的复杂组件。
- 不应运行生产查询、真实数据 replay、迁移或发布命令。
- 不应自动触发 TDD、code-review、archive、merge 或 worktree 清理。

### 下一步修正规格

后续修正 `skills/eng-review-finance/SKILL.md` 时，建议：

1. 明确 `message / artifact / openspec` 输出模式及写入权限。
2. 增加 AIW Task/OpenSpec change 解析和交接规则。
3. 将未决风险统一为 `%%`，兼容 validator 所需的 TODO 占位。
4. 增加 `focused / standard / full` review 范围。
5. 增加金融敏感数据、生产访问和真实 replay 的安全边界。
6. 要求每个 gate 输出证据、缺失项、owner、动作和阻塞关系。
7. 明确 testing strategy 不等于测试运行授权。
8. 将 validator 的结构检查纳入完成标准，但不把它当作风险判断替代品。

### 静态验收清单

- [ ] 输出模式和文件写入权限没有歧义。
- [ ] 评审可挂接到现有 AIW Task 和 OpenSpec change。
- [ ] 指标缺失、权限缺失和审计缺失仍会阻塞 READY。
- [ ] 未决问题使用 `%%`，不会用 TODO 掩盖风险。
- [ ] 支持 focused review，默认不会无必要地产生完整长报告。
- [ ] 不要求或泄露生产敏感数据。
- [ ] 每个 gate 有证据、owner、动作和状态影响。
- [ ] testing strategy 与测试执行权限分离。
- [ ] validator 结构失败时不会返回 READY。
- [ ] 不自动写文件、运行命令或执行 AIW 生命周期操作。

### 当前结论

`eng-review-finance` 本身已经是高质量的金融架构评审 skill，主要问题不是缺少检查项，而是输出模式、项目工作流、敏感数据边界和有限范围执行还不够明确。建议小幅修正，不需要重写其金融风险门禁模型。

本章节只记录评审建议，尚未修改 `skills/eng-review-finance/SKILL.md`。

## `grill-me`

目标文件：`skills/grill-me/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\productivity\grill-me\SKILL.md`

### 当前评价

质量约为 8/10，作为 wrapper skill 是合理的。当前版本与参考版本一致，只做一件事：把没有代码库或不需要持久化文档的计划/设计问题转交给 `/grilling`。它的短小本身是优点，不应把完整提问逻辑复制进来。

### 参考版本中值得保留的做法

1. description 清楚说明这是一个 relentless interview，用于 sharpen plan/design。
2. `disable-model-invocation: true` 合理，避免每轮上下文自动加载一个高交互成本的提问器。
3. 通过 `/grilling` 复用唯一的提问原语，保持单一事实来源。
4. `grill-me` 与 `grill-with-docs` 的职责可以保持分离：前者无代码库/无持久化文档，后者面向代码库并留下上下文记录。

### 当前版本已经做好的地方

- 没有复制 `/grilling` 的提问规则，避免 duplication 和漂移。
- 没有自动写文件、创建 Task、创建 worktree、运行命令或发布外部内容。
- 作为 standalone skill 足够轻量，符合 progressive disclosure。

### 需要重新增加或调整的内容

#### 1. 明确适用边界

当前正文只有 `Run a /grilling session.`，没有告诉 agent 什么时候不能用它。建议补充：

- 没有代码库或不需要把讨论写入代码库时使用 `/grill-me`；
- 有代码库且需要保留 CONTEXT/ADR 时使用 `/grill-with-docs`；
- 已经有明确需求、只需要开始规划时，不要无条件启动 relentless interview；
- 用户只要求直接回答或执行一个明确动作时，不要先进入 grilling。

#### 2. 明确完成后的去向

`grill-me` 不应只产生一串问题。完成后应输出：

- 已确定的目标、范围和关键决策；
- 未决问题和 `%%` 风险；
- 推荐的下一步 Skill；
- 如果后续进入工程流程，交给 `/to-spec` 或 `/wayfinder`；
- 如果仍然是普通设计讨论，不创建 AIW Task 或 OpenSpec change。

#### 3. 控制 interview 成本

“Relentless” 不应意味着无限提问。建议增加有限模式：

- `focused`：只追问阻塞当前决策的 3–5 个问题；
- `deep`：用户明确要求时，继续遍历依赖和边界；
- 用户回答已经足够明确时立即结束，不为了完成轮数继续提问；
- 每一轮只问一个高价值问题，并说明为什么该问题影响决策。

#### 4. 与 `/grilling` 的交接规则

如果提问逻辑由 `/grilling` 负责，`grill-me` 应明确传递：

- 当前目标和用户原始问题；
- 是否有代码库、是否需要持久化文档；
- 期望的输出：计划、设计决策、风险或下一步路由；
- 不创建 AIW/OpenSpec/worktree 的 standalone 边界。

这样 wrapper 不只是一个无上下文的别名，也不会把错误的持久化行为传给底层 skill。

### 不应恢复或引入的内容

- 不应把完整 `/grilling` 内容复制到 `grill-me`。
- 不应在没有用户授权时创建 CONTEXT.md、ADR、Task、OpenSpec change 或 worktree。
- 不应无条件进行无限提问。
- 不应把 `grill-me` 当成 `grill-with-docs` 的替代品。
- 不应在用户已经给出明确实现授权时，用 grilling 拖延执行。

### 下一步修正规格

后续修正 `skills/grill-me/SKILL.md` 时，建议只增加少量 wrapper 元信息：

1. 说明适用于无代码库或不需要持久化文档的讨论。
2. 说明不创建 AIW/OpenSpec/worktree 等资源。
3. 说明完成后输出决策、`%%` 风险和下一步 Skill。
4. 传递 focused/deep 模式和用户原始问题给 `/grilling`。
5. 保持提问逻辑只存在于 `/grilling`。

### 静态验收清单

- [ ] 仍然是轻量 wrapper，没有复制 `/grilling` 规则。
- [ ] 与 `grill-with-docs` 的适用边界清楚。
- [ ] 有 focused/deep 或等价的提问成本边界。
- [ ] 完成后能输出决策、`%%` 风险和下一步路由。
- [ ] 不创建文件、Task、OpenSpec change 或 worktree。
- [ ] 用户已有明确执行请求时不会无条件启动 interview。

### 当前结论

`grill-me` 的最重要优点是极简和单一事实来源，建议保留 wrapper 形态，只补充适用边界、完成输出和有限提问模式。不要把它扩展成第二份 grilling 实现。

本章节只记录评审建议，尚未修改 `skills/grill-me/SKILL.md`。

## `grill-with-docs`

目标文件：`skills/grill-with-docs/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\engineering\grill-with-docs\SKILL.md`

### 当前评价

质量约为 8/10，作为 `/grilling` + `/domain-modeling` 的组合 wrapper 设计是合理的。与 `grill-me` 一样，它保持了单一事实来源；不同点是它明确会把领域结论持久化到 glossary 和 ADR。当前正文过于简短，缺少写入范围、何时写入、如何确认和完成后的主流程衔接。

### 参考版本中值得保留的做法

1. 只通过 `/grilling` 复用提问流程，不复制 interview 逻辑。
2. 通过 `/domain-modeling` 处理 glossary 冲突、模糊术语、场景边界、代码矛盾和 ADR 条件。
3. 与 `grill-me` 明确形成互补：有代码库且需要持久化领域知识时使用本 skill。
4. 让重要领域结论在对话过程中及时进入 `CONTEXT.md` 或 ADR，避免最终凭记忆批量补写。

### 当前版本已经做好的地方

- `disable-model-invocation: true` 适合一个会写入长期项目文档的用户主动流程。
- 没有复制 `/grilling` 或 `/domain-modeling` 的详细规则。
- 没有自动创建外部 Issue、PR、worktree 或运行命令。
- wrapper 依赖关系非常清楚，维护成本低。

### 需要重新增加或调整的内容

#### 1. 明确持久化边界

“creates docs as we go” 太宽泛。建议明确：

- 只写领域 glossary 到 `CONTEXT.md` 或 context-specific context 文件；
- 只把满足 `/domain-modeling` 三条件的决策写入 ADR；
- 需求、范围、用户故事、实现计划和验收标准进入 OpenSpec，而不是 CONTEXT/ADR；
- 临时探索、候选方案和未决问题保留在对话或 `%%` notes，不污染稳定文档。

#### 2. 增加 AIW/OpenSpec 主流程衔接

建议在 session 完成后明确：

- 如果只是领域讨论，输出 glossary/ADR 变更和 `%%` 风险即可；
- 如果要构建功能，把已经稳定的领域结论交给 `/to-spec`；
- `/to-spec` 生成的规范文件和本 skill 产生的领域文档都应在当前分支提交，后续 AIW worktree 直接继承，不手工复制；
- 不在本 skill 中创建第二个 Task 或 `.scratch` 规划层；
- `grill-with-docs` 不负责实现、TDD、code review、archive、merge 或清理。

#### 3. 增加 focused/deep 交互模式

参考版本使用 relentless interview，但当前成本控制需要有限模式：

- `focused`：只解决阻塞当前设计的 3–5 个问题，适合普通功能；
- `deep`：用户明确要求时，继续探索跨 context、边界和长期决策；
- 结构和领域语言已经足够清楚时立即结束，不为“relentless”而继续提问；
- 每轮只问一个高价值问题，并指出它会改变哪个设计决策或文档。

#### 4. 增加写入前检查点

虽然用户主动调用本 skill 已经表达了写入意图，但长期文档变更仍需可检查：

- 新建 `CONTEXT.md`、context map 或 ADR 前确认目标 context 和文件位置；
- 覆盖既有术语前说明旧定义、新定义和影响范围；
- 创建 ADR 前列出三个触发条件和被拒绝的替代方案；
- 同一轮产生多个候选定义时，先保留 `%%` 未决状态，不选一个伪装成确定事实。

#### 5. 明确输出完成标准

完成时至少输出：

- 已解决的 canonical terms；
- glossary/ADR 的变更文件和摘要；
- 用户确认过的关键决策；
- 未决问题和 `%%` 风险；
- 是否可以进入 `/to-spec`、`/wayfinder` 或其他下一步 Skill。

### 不应恢复或引入的内容

- 不应把完整 grilling 或 domain-modeling 内容复制进 wrapper。
- 不应把所有讨论写入 CONTEXT.md 或为普通选择创建 ADR。
- 不应把 OpenSpec proposal/design/tasks 写进领域 glossary。
- 不应在没有 context 归属时创建全局术语。
- 不应无限提问或自动运行测试、TDD、code review。
- 不应在当前 skill 中自动创建 Task、worktree、archive、merge 或删除资源。

### 下一步修正规格

后续修正 `skills/grill-with-docs/SKILL.md` 时，建议只增加 wrapper 元信息：

1. 说明适用于有代码库且需要持久化领域知识的场景。
2. 说明 glossary、ADR、OpenSpec 各自的写入边界。
3. 增加 focused/deep 模式和结束条件。
4. 增加新建/覆盖领域文档前的检查点。
5. 说明完成后如何把稳定结论交给 `/to-spec`。
6. 保持提问和领域建模逻辑只存在于对应底层 Skill。

### 静态验收清单

- [ ] 与 `grill-me` 的代码库/持久化边界清楚。
- [ ] 只通过底层 Skill 处理提问和领域建模。
- [ ] glossary、ADR、OpenSpec 的所有权和写入范围清楚。
- [ ] 有 focused/deep 提问成本边界。
- [ ] 新建或覆盖长期文档前有 context/定义检查。
- [ ] 完成后能输出文档变更、决策、`%%` 风险和下一步路由。
- [ ] 不自动执行实现、测试、review 或 AIW 完成协议。
- [ ] 文档变更能在当前分支提交并被后续 worktree 继承。

### 当前结论

`grill-with-docs` 的 wrapper 方向正确，建议保留极简结构，只补充持久化边界、AIW/OpenSpec 衔接、有限提问模式和完成标准。不要把它膨胀成第二份 `grilling` 或 `domain-modeling` 实现。

本章节只记录评审建议，尚未修改 `skills/grill-with-docs/SKILL.md`。

## `grilling`

目标文件：`skills/grilling/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\productivity\grilling\SKILL.md`

### 当前评价

质量约为 7/10。核心提问纪律很清楚：一次一个问题、事实查环境、决策交给用户、确认共识前不执行。但“relentlessly / every aspect / each branch”没有预算或终止条件，容易产生无限采访、重复问题和用户负担。由于它是 `grill-me` 与 `grill-with-docs` 的底层原语，边界问题会被两个 wrapper 放大。

### 参考版本中值得保留的做法

1. 一次只问一个问题，等待用户回答，避免多问题并行造成认知负担。
2. 能从环境获得的事实先查证，不把可发现事实转化成用户问答。
3. 把决策权留给用户，并为每个问题提供推荐答案。
4. 在用户确认共享理解前不执行后续动作，避免未批准的设计被当作事实。
5. 用决策树逐步解决依赖关系，而不是一次性抛出长问卷。

### 当前版本已经做好的地方

- description 触发条件明确，适合模型自动调用。
- 没有复制到 `grill-me` 或 `grill-with-docs`，保持单一事实来源。
- 明确区分事实和决策。
- 明确要求等待反馈，保留用户控制权。
- 没有自动写文件、创建 Task/worktree、运行测试或发布外部内容。

### 需要重新增加或调整的内容

#### 1. 增加可控的 interview 模式和预算

建议把 relentless 解释为深入而不是无限：

- `focused`：3–5 个会改变当前方案的关键问题；
- `deep`：用户明确要求时，继续走依赖和边界分支；
- 每轮最多一个问题；
- 用户已经给出足够信息时立即总结，不为了“每个分支”继续扩展；
- 连续两轮没有产生新决策或约束时停止并总结。

#### 2. 增加明确的完成标准

共享理解达成至少应包含：

- 目标和成功标准；
- 范围与非范围；
- 关键参与者/约束；
- 已确定的决策及其理由；
- 依赖顺序和未决分支；
- `%%` 风险或待确认问题；
- 推荐下一步 Skill。

没有这些内容时，不能只因为问了很多问题就宣告完成。

#### 3. 区分“事实查找”与高成本/高风险操作

“If a fact can be found by exploring the environment” 应增加边界：

- 优先使用廉价、只读、局部的文件和元数据查询；
- 不运行测试、构建、网络请求、生产查询或长时间脚本来回答普通事实问题；
- 如果事实需要运行命令或访问敏感数据，说明命令、范围和成本后请求授权；
- 查到的事实要给出来源路径或命令，避免把推断写成事实。

#### 4. 明确推荐答案不是替用户做决定

每个问题建议使用固定格式：

```text
问题：……
我的推荐：……
原因：……
如果选择其他方案，会影响：……
```

用户仍需确认；推荐答案不应被自动写入 spec、ADR、Task 或代码。

#### 5. 增加用户跳过和提前结束规则

- 用户说“先给结论/停止提问”：立即整理当前共识和剩余风险；
- 用户拒绝回答：记录 `%%`，不要反复施压；
- 用户要求直接实现：停止 grilling，并交给合适的执行 Skill；
- 用户的问题已明确且无需设计取舍：不要强行启动 interview。

#### 6. 明确持久化和后续动作边界

`grilling` 本身应保持无状态：

- `grill-me` 决定不持久化；
- `grill-with-docs` 通过 `domain-modeling` 持久化 glossary/ADR；
- `grilling` 不直接写文件，也不直接创建 AIW/OpenSpec 资源；
- 用户确认后，wrapper 或下一步 Skill 才能把结果落到文档或 Task。

#### 7. 处理分支爆炸

“Walk down each branch” 可能导致复杂决策树失控。建议按优先级处理：

1. 先解决阻塞主路径的决策；
2. 再处理高风险、难以逆转或影响多个下游的分支；
3. 低风险可延后的分支记录为 `%%`，不在当前 interview 展开；
4. 每次只保留对当前目标有影响的分支。

### 不应恢复或引入的内容

- 不应无限提问、重复同一问题或强迫用户回答非阻塞问题。
- 不应把推荐答案当作用户决定。
- 不应为了查事实自动运行测试、构建、网络、生产查询或昂贵脚本。
- 不应在共识确认前写入代码、spec、ADR、Task 或外部系统。
- 不应把所有可能分支都展开，造成决策树爆炸。
- 不应在用户要求直接执行时继续 grilling。

### 下一步修正规格

后续修正 `skills/grilling/SKILL.md` 时，建议：

1. 保留一次一个问题、事实查环境、推荐答案和用户确认四项核心纪律。
2. 增加 `focused/deep` 模式和问题预算。
3. 增加共享理解的完成标准。
4. 增加低成本只读事实查询与高成本操作授权边界。
5. 增加跳过、拒答、提前结束和转交执行 Skill 的规则。
6. 用阻塞性、风险和可逆性控制分支展开顺序。
7. 保持底层 skill 无状态，不直接持久化或执行生命周期操作。

### 静态验收清单

- [ ] 一次只问一个问题并等待回答。
- [ ] 默认有明确的问题预算和停止条件。
- [ ] 事实查询优先使用廉价、只读、局部证据。
- [ ] 高成本或高风险查询需要授权。
- [ ] 推荐答案与用户决定明确分离。
- [ ] 共享理解包含目标、范围、决策、依赖和 `%%` 风险。
- [ ] 支持用户跳过、拒答、提前结束或转入执行。
- [ ] 不在确认前写文件或执行生命周期操作。
- [ ] 不自动运行 TDD、code review、测试、构建或网络操作。

### 当前结论

`grilling` 的核心纪律值得保留，但需要从“无限深入的采访”调整为“预算内、依赖优先、可提前结束的决策访谈”。这是后续 `grill-me` 和 `grill-with-docs` 稳定性的关键底层改进。

本章节只记录评审建议，尚未修改 `skills/grilling/SKILL.md`。

## `handoff`

目标文件：`skills/handoff/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\productivity\handoff\SKILL.md`

### 当前评价

质量约为 8.5/10。当前版本在参考版本基础上正确加入了 AIW Task、worktree、Session、OpenSpec change 和 `aiw task agent next` 约束，已经适合本项目的跨 Session 工程协作。主要还需要补充 handoff 文档的结构、完成校验、状态边界和与自动完成闭环的关系。

### 参考版本中值得保留的做法

1. 把 handoff 定义为当前对话到新 agent/session 的上下文压缩，而不是继续在原 Thread 中强行工作。
2. 要求包含 `suggested skills`，帮助新 agent 选择正确的后续能力。
3. 避免复制 specs、plans、ADR、issues、commits 和 diffs，只通过路径或 URL 引用权威材料。
4. 对 API key、密码和 PII 做脱敏。
5. 根据用户传入的 handoff 目标裁剪文档，而不是保存一份无边界的全量 transcript。

### 当前版本已经做好的地方

- 已要求先解析 AIW Task、worktree、Session 和 matching OpenSpec change。
- 优先使用 AIW Session artifact location，避免临时目录成为 canonical work。
- 只有用户明确要求交接执行时才调用 `aiw task agent next`，不自动创建 Thread。
- 已明确不重复 OpenSpec 和其他权威工件。
- 已保留敏感信息脱敏要求。
- 已将 handoff 与 Task/worktree/lease/lineage 绑定，避免新 agent 在错误目录继续工作。

### 需要重新增加或调整的内容

#### 1. 明确 handoff 文档的最小结构

当前 skill 只要求“总结当前对话”，建议规定最小章节：

- `Task and Session`：Task ID、Session、worktree、branch、parent branch 和当前阶段；
- `Goal and scope`：目标、已确认范围、明确 out of scope；
- `Current state`：已完成、进行中、阻塞项；
- `Decisions`：已确认决策及来源路径；
- `Next action`：新 agent 的第一项可执行动作；
- `Open risks`：`%%` 未决问题、假设和外部依赖；
- `Referenced artifacts`：proposal、design、spec、tasks、ADR、commits、diff；
- `Suggested skills`：下一步建议调用的 Skill 和理由。

这样新 agent 可以快速恢复，不需要猜测 handoff 文档的字段含义。

#### 2. 增加“恢复就绪”完成标准

handoff 完成至少要验证：

- 文档已写入 AIW Session artifact location；
- 文档可被新 agent 读取；
- Task、Session、worktree、branch 和 OpenSpec change 互相匹配；
- 下一步动作只有一个明确主动作；
- 所有路径是相对于当前仓库或明确的绝对路径；
- 敏感信息已脱敏；
- 文档没有复制权威 artifacts 的大段内容；
- 新 agent 不需要依赖当前对话中未记录的隐含信息。

#### 3. 区分“保存 handoff”和“启动新 Thread”

建议明确两种模式：

- `save-only`：只生成 handoff 文档，当前 agent 停止；
- `continue`：用户明确要求继续执行时，先保存并校验 handoff，再调用 `aiw task agent next <task-id>`。

默认使用 `save-only`。保存失败时不能启动新 Thread；启动后也不能删除原 handoff。

#### 4. 增加 Session/Task 状态门禁

handoff 前应检查：

- Task 没有被 archive 或删除；
- Session 状态允许 handoff；
- worktree 路径存在且与 metadata 一致；
- 当前没有另一个 agent 持有冲突的 execution lease；
- 如果有未提交修改，handoff 明确列出它们，不能假装已提交。

如果状态不一致，应停止并报告，不能通过 handoff 文档掩盖生命周期冲突。

#### 5. 与自动完成闭环衔接

handoff 发生在任务未完成时，不能触发 sync/archive/merge/cleanup。只有后续 `/implement` 检查 `tasks.md` 全部完成后，才进入自动完成协议。

如果 handoff 发生在实现提交之后但 tasks 尚未全部完成：

- 保留 Task branch 和 worktree；
- 在 handoff 中记录最新 commit；
- 新 agent 继续使用同一 Task/worktree/Session lineage；
- 不创建平行 Task 或复制 OpenSpec artifacts。

#### 6. 增加敏感信息和外部路径边界

- 不把 token、密码、cookie、PII、完整生产 payload 或内部凭证写入 handoff；
- 外部路径或 URL 只在新 agent 确实需要且用户允许时引用；
- 临时目录只能作为无 AIW artifact store 时的 fallback；
- handoff 应标记引用文件是否存在，避免新 agent 追逐失效路径。

### 不应恢复的内容

- 不应默认把 handoff 写入临时目录而绕过 AIW Session artifact store。
- 不应自动启动新 Thread。
- 不应复制完整 transcript、diff 或 OpenSpec 文档。
- 不应在 handoff 阶段 archive、merge、删除 worktree/branch 或清理 Task。
- 不应通过 handoff 绕过 active lease、Task 状态或 worktree 校验。
- 不应将未提交修改描述成已完成提交。

### 下一步修正规格

后续修正 `skills/handoff/SKILL.md` 时，建议：

1. 保留 AIW artifact store 优先和 `aiw task agent next` 的显式启动规则。
2. 增加最小 handoff 文档结构和恢复就绪完成标准。
3. 增加 `save-only / continue` 模式。
4. 增加 Task、Session、worktree、lease 和未提交修改的状态门禁。
5. 明确 handoff 不触发自动完成闭环。
6. 强化敏感信息脱敏和引用路径可用性检查。

### 静态验收清单

- [ ] 默认写入 AIW Session artifact location。
- [ ] handoff 文档包含 Task/Session、当前状态、下一动作、风险、引用和 suggested skills。
- [ ] 保存完成前验证文档可读、路径有效、状态一致和信息已脱敏。
- [ ] `save-only` 与 `continue` 行为清楚。
- [ ] 不自动启动新 Thread。
- [ ] 不复制权威 artifacts 或完整 transcript。
- [ ] 未提交修改、lease 冲突和失效 Task 状态会阻塞恢复。
- [ ] handoff 不触发 sync、archive、merge 或清理。

### 当前结论

`handoff` 已经很好地完成了 AIW 化适配，建议做增量增强：定义最小文档结构、恢复就绪标准、save-only/continue 模式和状态门禁。不要退回到只写临时文件的通用 handoff 行为。

本章节只记录评审建议，尚未修改 `skills/handoff/SKILL.md`。

## `implement`

### 评审对象

- 本地版本：`skills/implement/SKILL.md`
- 参考版本：`D:\03_projects\third-part\skills\skills\engineering\implement\SKILL.md`
- 配套约束：`skills/work-management.md`、仓库 `AGENTS.md`。

### 当前评价

当前质量约为 **8.5/10**。参考版本非常简洁，保留了“按 spec/ticket 实现、可以在约定 seam 使用 TDD、定期检查、完成后 code review、提交”的基本闭环，但它默认当前分支、默认运行检查、默认 TDD 和 code review，也没有 AIW Task/worktree 或 OpenSpec 生命周期。本地版本已经完成了关键适配，并加入了用户明确要求的自动完成协议；目前主要缺口是把前置解析、单项实现、全量完成判断、失败恢复和证据报告写成更可检查的状态机。

### 参考版本值得保留的做法

1. 以一个明确的 work item 为实现单位，避免一轮实现混入多个目标。
2. 将实现依据限定为 spec 或 tickets，而不是仅凭用户描述自由扩展范围。
3. 保留 TDD 作为可选的质量路径，而不是把测试设计从实现流程中删除。
4. 原版本要求定期做小范围 typecheck 和测试、结束时做完整测试，体现了分阶段反馈的工程意图；当前仓库可保留这个意图，但必须受显式授权和资源预算约束。
5. 完成后进行 code review 的意图值得保留，但在本仓库应改成显式、有限度的 opt-in。
6. 最终提交实现结果，保证 work item 有可追溯交付点。

### 当前版本已有的优点

- 明确解析一个 AIW Task、匹配的 OpenSpec change 和一个 `tasks.md` checklist item。
- 明确多个匹配 Task 时停止并请求 Task ID，避免在错误上下文中实现。
- 使用 `aiw wt`，验证 branch/path 与 Task metadata，避免静默切换工作区。
- 明确以最小完整改动实现单个 checklist item，并限制 sub-agent 数量。
- 已移除自动调用 `/tdd`、`/code-review` 和自动运行测试，符合当前账单控制和资源规则。
- 已要求更新 checklist、TODO、Verification 和 `%%` 风险，并在静态 review 后自动 commit。
- 已实现用户要求的完成闭环：全部 checklist 完成后 sync、archive、合并到记录的 `parent_branch`、验证合并、清理 worktree 和删除 Task branch。
- 对 sync、archive、merge 或验证失败，以及冲突场景，要求保留 branch/worktree，具备基本恢复能力。

### 需要调整的内容

#### 1. 增加可检查的 preflight 状态

当前 Resolve Work 和 Prepare The Workspace 已覆盖主要动作，但建议在开始修改前明确输出并验证以下快照：

- Task ID、OpenSpec change ID、当前 checklist item；
- `parent_branch`、Task branch、worktree 路径和当前 checkout；
- proposal、design、capability spec、`tasks.md`、task metadata 是否一致存在；
- 当前 item 的前置依赖是否完成；
- 当前工作区是否确实位于 AIW Task worktree。

任何身份不一致、缺少规范或依赖未完成都应在 preflight 停止，不应先修改代码再补上下文。

#### 2. 严格区分“单项完成”和“Task 完成”

技能同时要求实现一个 selected item，又要求完成后检查所有 items 并触发协议。建议显式定义：

- 单项完成：实现、静态检查、更新该 item 的状态、Verification 和剩余 `%%`；
- Task 完成：重新读取同一 change 的完整 `tasks.md`，确认所有 implementation checklist items 都完成，没有未解决 blocker；
- 只有 Task 完成才进入 sync/archive/merge/cleanup；
- 任意 item 未完成时只提交当前增量并保留 Task worktree，不能因为当前 item 完成就归档。

这样能防止“当前任务做完”被误判为“整个 change 做完”。

#### 3. 统一 `TODO` 与 `%%` 语义

正文要求更新 TODO 和 Verification，但仓库约定 unresolved risks or questions 使用 `%%`。建议改为：

- 若 `tasks.md` 已有项目定义的 TODO，保留其原结构并更新实际完成状态；
- 新增未决事项统一使用 `%% NEEDS_INPUT`、`%% NEEDS_VALIDATION` 或 `%% BLOCKS_TASK`；
- Verification 只记录已执行或静态确认的证据，不把未运行测试写成通过；
- 完成 Task 前必须检查是否仍有阻塞性 `%%`，并把非阻塞风险交给归档记录。

#### 4. 把自动完成协议写成原子性与恢复状态

当前顺序符合用户要求，但需要进一步规定每一步的成功条件和可恢复点：

1. sync 成功，且没有 OpenSpec/AIW ownership conflict；
2. archive 成功，且目标 change 状态可确认；
3. 在记录的 `parent_branch` 上 merge 成功；
4. 验证 merge commit/状态确实包含 Task branch 的交付；
5. 最后才删除 worktree 和 Task branch。

若 archive 成功但 merge 失败，应保留 branch/worktree 并报告 archive 状态；若 cleanup 部分失败，不应重做可能产生重复副作用的 archive/merge，而应从已完成的阶段恢复。删除前应再次确认路径和 branch 正是当前 Task 的资源。

#### 5. 明确自动 commit 的范围和提交失败处理

用户已明确希望实现完成后自动提交，因此本地版本可以保留自动 commit。但建议补充：

- 只提交当前 Task worktree 中属于本次 Task 的变更；
- 先做静态 diff 检查和敏感文件/秘密扫描，再生成提交；
- 提交信息包含 Task ID 和 change ID，便于回溯；
- 发现无关修改、冲突或提交失败时停止，不强行 stage/覆盖；
- commit 成功后仍须更新并核对 `tasks.md`，不能以 commit 存在代替 Task 完成。

这条自动 commit 规则是本仓库用户明确要求的工作流，不应泛化为所有项目的默认 Git 行为。

#### 6. 明确 sub-agent 资源档位

当前“最多两个 bounded sub-agents”比参考版本安全，但仍缺少何时使用的参考值。建议加入：

- `off`：0 个，默认用于简单单文件或高风险变更；
- `focused`：1 个，只做代码定位或一个静态分析问题；
- `standard`：最多 2 个，处理互不依赖的定位/静态片段；
- `expanded`：仍不超过 2 个，但允许更长上下文或更多主 agent 读取；需要更多并行资源时另行确认，不在 skill 内突破硬上限。

所有 sub-agent 不得测试、构建、联网、commit、archive、merge 或操作 worktree；主 agent 负责整合和生命周期变更。

#### 7. 将 TDD 和 code review 保持为显式 opt-in

参考版本的 TDD 和结束 review 意图可保留，但当前 skill 应明确：

- 普通实现不自动启动 `/tdd` 或 `/code-review`；
- 用户明确要求 test-first、focused TDD 或 code review 时，说明范围、命令、成本和预计时长；
- focused TDD 只处理一个 seam/行为和最小命令；
- focused code review 只检查选定 diff 的 Standards + Spec；
- review 发现 blocker 时不得触发完成协议；
- broader test/build/review 需要再次授权。

#### 8. 完善最终报告与验证授权

实现结束后建议固定报告字段：

- Task/change/item 与实际修改路径；
- 静态检查证据和提交结果；
- 未运行的测试、构建、lint、review 及原因；
- 剩余 `%%` 风险；
- Task 是否全部完成；
- 完成协议各阶段状态，或失败时的恢复入口；
- 可选 focused test 的精确命令、范围和预期时长。

“ask once whether the user wants one focused test”应保留，但不能把未回复解释为测试通过，也不能在用户未授权时自行执行。

### 不建议恢复的做法

- 不恢复在当前分支直接实现而不解析 AIW Task/worktree 的方式。
- 不恢复默认运行完整测试、typecheck、formatter、lint、build 或 vet。
- 不恢复自动调用 TDD 或 code review。
- 不恢复没有失败保护的自动 archive、merge、cleanup。
- 不在 archive/merge 未成功时删除 Task branch 或 worktree。
- 不把一个 checklist item 的完成当作整个 OpenSpec change 的完成。

### 后续修正建议

下一步可在 `skills/implement/SKILL.md` 中补充：

1. preflight 快照与明确停止条件。
2. item completion 与 Task completion 的双层判定。
3. 自动 commit 的范围、提交信息和失败恢复。
4. sync/archive/merge/cleanup 的阶段状态与幂等恢复。
5. `%%`、Verification 和未运行命令的报告格式。
6. sub-agent 资源档位和 TDD/code-review opt-in 规则。

### 静态验收清单

- [ ] 开始修改前能确认 Task、change、item、branch、worktree 和 `parent_branch`。
- [ ] 只实现一个选定 item，不会扩展到无关任务。
- [ ] item 完成与全部 tasks 完成有明确区别。
- [ ] 自动 commit 只包含当前 Task 变更，并处理提交失败。
- [ ] 只有全部 tasks 完成且 sync/archive/merge/验证成功后才清理资源。
- [ ] 失败或冲突时保留 branch/worktree，并能从已完成阶段恢复。
- [ ] 默认不运行测试、构建、lint、vet、TDD 或 code review。
- [ ] 最终报告区分静态证据、未运行检查和剩余 `%%` 风险。

### 结论

本地 `implement` 已经正确吸收了 AIW/OpenSpec、Task worktree 和用户要求的自动完成闭环，是比参考版本更适合当前仓库的实现。下一步不需要恢复参考版本的默认测试、TDD 或 code review；应补齐 preflight、双层完成判定、自动提交范围和完成协议的阶段化恢复信息。

本章节只记录评审建议，尚未修改 `skills/implement/SKILL.md`。

## `improve-codebase-architecture`

目标文件：`skills/improve-codebase-architecture/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\engineering\improve-codebase-architecture\SKILL.md`

关联参考：`HTML-REPORT.md`

### 当前评价

质量约为 8/10。当前版本与参考版本一致，核心设计是先扫描代码库中的 deepening opportunities，再用可视化报告让用户选择，最后进入 grilling 和 domain-modeling。它很好地避免了“发现架构问题就直接重构”，但存在三个实际问题：HTML 所谓 self-contained 却依赖 CDN、探索范围没有成本边界、报告生成与打开存在环境副作用。

### 参考版本中值得保留的做法

1. 先找 architectural friction 和 shallow modules，再提出 deepening opportunity，而不是从抽象偏好出发重构。
2. 使用近期 commit hot spots 限定探索范围，体现 YAGNI。
3. 使用 CONTEXT.md 的领域语言和 codebase-design 的 module/interface/depth/seam/adapter/leverage/locality 词汇。
4. 用 deletion test 判断模块是否真正隐藏复杂度。
5. 要求每个候选包含文件、问题、方案、收益、before/after 图和推荐强度。
6. 不在候选报告阶段提前设计 interface，先让用户选择值得探索的方向。
7. 用户选择候选后才进入 grilling、domain-modeling 和可选的 codebase-design，阶段边界清楚。
8. ADR 冲突只有在真实 friction 足够强时才提出重开，避免列出理论上的所有反对意见。

### 当前版本已经做好的地方

- `disable-model-invocation: true` 合理，因为扫描可能较重且会产生临时报告。
- 明确了探索、报告、候选选择、grilling loop 的顺序。
- 已要求报告写入 OS temp，而不是污染仓库。
- 已保留外部 HTML scaffold，避免主 skill 被大量展示代码撑大。
- 没有直接修改业务代码、创建 Task、创建 worktree 或运行测试。
- 已明确用户选择候选后才继续深入设计。

### 需要重新增加或调整的内容

#### 1. 修正“self-contained”与 CDN 的矛盾

当前要求使用 Tailwind CDN 和 Mermaid CDN，但这不是真正 self-contained，也会触发网络依赖。建议改为：

- 默认使用内联 CSS 和不依赖网络的原生 SVG/HTML 图；
- 如果要使用 Mermaid/CDN，必须标记为 optional enhancement，并在离线时仍提供可读报告；
- 不因渲染失败而丢失候选文本；
- 不自动下载依赖或发起网络请求。

#### 2. 增加扫描资源档位

扫描全仓库和调用 Explore agent 可能成本较高，建议增加：

- `focused`：只扫描用户指定模块/最近变更路径；
- `standard`：扫描近期 hot spots 和直接依赖；
- `wide`：用户明确要求时，扩大到跨模块和历史热点分析。

默认使用 `standard`，但用户指定方向时优先 `focused`。每个档位应限制读取范围、子 agent 数量和报告候选数量。

#### 3. 明确只读探索边界

Explore agent 应只能：

- 读取代码、测试、文档、git 历史和配置；
- 记录证据、路径、调用关系和 friction；
- 不运行测试、构建、网络、格式化或性能命令；
- 不修改代码、文档、CONTEXT、ADR 或 OpenSpec；
- 不创建 branch/worktree 或提交。

如果需要运行命令验证架构假设，先停下并请求授权，或将其列为 `%%` 待验证风险。

#### 4. 增加候选质量和停止标准

报告不应以列出很多“可能重构”作为完成。每个候选至少需要：

- 证据路径和调用关系；
- 当前 friction 的用户/维护者影响；
- deletion test 结果；
- 当前 seam 和可能的 deepening 方向，但不提前固定 interface；
- 预期 locality/leverage/testability 收益；
- 风险、迁移成本和 ADR 冲突；
- Strong/Worth exploring/Speculative 推荐理由。

候选数量建议控制在 3–5 个；如果没有足够证据，应报告“没有可靠候选”，不要填充 speculative refactor。

#### 5. 处理 HTML 临时文件和打开动作

建议区分：

- `report-only`：生成临时 HTML 并返回路径；
- `open-report`：用户明确要求时才调用系统打开动作；
- `text`：无法生成或打开 HTML 时，返回同样的候选 Markdown。

当前环境不支持 GUI 或打开动作时，不应重试其他 shell，也不应阻塞报告生成。

#### 6. 明确用户选择后的 AIW/OpenSpec 边界

选择候选后：

- 先通过 `/grilling` 和 `/domain-modeling` 形成决策；
- 需要实现时回到 `/to-spec`，由 AIW Task/OpenSpec change 承载；
- 不在扫描 skill 中直接创建第二个 Task 或 `.scratch` 规划层；
- 如果领域词汇或 ADR 被更新，按 domain-modeling 规则提交并让后续 worktree 继承；
- 代码实现、focused TDD、focused code review 和完成协议由对应 Skill 负责。

#### 7. 控制 design-it-twice 的资源消耗

当前最后一段直接引用 codebase-design 的并行设计模式。应遵守当前资源档位：默认 focused/bounded，超过两个 sub-agents 需要用户明确授权，并说明预计成本和收益。

### 不应恢复或引入的内容

- 不应默认发送 CDN 网络请求或下载 Tailwind/Mermaid。
- 不应因为生成视觉报告而自动打开 GUI 或要求用户安装工具。
- 不应扫描全仓库、运行测试或启动多个 agent 而不说明范围。
- 不应把 speculative candidate 当作事实或强制推荐。
- 不应在候选选择前提出完整 interface 或开始实现。
- 不应自动修改代码、CONTEXT、ADR、OpenSpec、Task、branch 或 worktree。
- 不应在用户选择候选后绕过 `/to-spec` 直接实现。

### 下一步修正规格

后续修正 `skills/improve-codebase-architecture/SKILL.md` 时，建议：

1. 增加 focused/standard/wide 扫描范围和资源边界。
2. 将报告改为真正离线可读，CDN 只作为可选增强。
3. 明确 Explore agent 的只读操作和运行授权边界。
4. 为候选增加证据、deletion test、风险、成本和停止条件。
5. 增加 report-only/open-report/text 输出模式。
6. 明确候选选择后回到 grilling/domain-modeling，再由 to-spec 进入 AIW/OpenSpec。
7. 约束 design-it-twice 的 sub-agent 资源档位。

### 静态验收清单

- [ ] 默认扫描范围和 agent 资源预算明确。
- [ ] 报告离线可读，不依赖 CDN 才能查看核心内容。
- [ ] Explore 阶段只读，不运行测试、构建、网络或修改文件。
- [ ] 每个候选有证据、deletion test、收益、风险和推荐强度。
- [ ] 候选数量受控，证据不足时可以返回无可靠候选。
- [ ] report-only、open-report、text 行为清楚。
- [ ] 不在候选选择前设计 interface 或实现代码。
- [ ] 选中候选后通过 grilling/domain-modeling/to-spec 进入正式工作流。
- [ ] 不自动触发 TDD、code review 或 AIW 完成协议。

### 当前结论

`improve-codebase-architecture` 的核心路线值得保留，但需要把视觉报告从网络依赖改成离线可靠，把全仓扫描改成有预算的分层探索，并明确只读和后续 AIW/OpenSpec 交接边界。它应继续负责“发现候选”，不负责直接改造代码。

本章节只记录评审建议，尚未修改 `skills/improve-codebase-architecture/SKILL.md`。

## `metrics-review`

目标文件：`skills/metrics-review/SKILL.md`

参考文件：未找到第三方对应版本。以下评审基于当前 skill、`references/` 下的治理规则、模板、OpenSpec mapping 和 validator。

### 当前评价

质量约为 8.5/10。当前 skill 已经建立了很强的 metric-registry-first 纪律：先定义业务含义，再确认来源、金融正确性、一致性和 owner，最后决定状态。它能够有效阻止“先写 SQL 再争论数字含义”的常见错误。主要需要补充的是输出模式、数据血缘/版本、指标验证证据、`%%` 风险语义和有限范围执行。

### 当前版本做得好的地方

1. 把 business definition 放在 query/source 之前，避免把技术可计算误当成业务正确。
2. 对 currency、FX timestamp、precision、rounding、timezone、cut-off、settlement/value date、snapshot-vs-transaction、dedup 和过滤范围设置了明确门禁。
3. 要求 business owner 和 technical owner 同时存在，避免指标无人负责。
4. 对 finance/risk/regulatory/management reporting 冲突设置 `CONFLICT`，不擅自选择口径。
5. 使用 per-metric status，并用最差状态汇总多指标文档。
6. 与 `eng-review-finance`、`release-review`、`autoplan-finance` 的 handoff 关系清楚。
7. 通过模板、治理规则和 OpenSpec mapping 渐进披露细节，主 skill 结构清楚。
8. validator 和 strict 模式提供了可检查的文档结构完成标准。

### 需要重新增加或调整的内容

#### 1. 解决输出模式与写入权限的歧义

当前同时写“emits a structured METRICS_SPEC.md”和“return as a message; do not write to disk unless explicitly asks”。建议明确：

- `message`：默认返回结构化指标规范，不写文件；
- `artifact`：用户明确要求时生成 `METRICS_SPEC.md`；
- `openspec`：用户明确要求时生成当前 change 或稳定 capability 的映射内容。

如果写入 OpenSpec，必须先解析当前 AIW Task/change，不能创建平行指标注册表或 `.scratch` 文件。

#### 2. 将 TODO 与项目 `%%` 风险语义统一

当前 skill 要求用 TODO 标记不确定内容，但项目约定使用 `%%` 记录未解决风险和问题。建议：

- OpenSpec/AIW 面向的未决问题使用 `%%`；
- validator 需要结构占位时可以保留 TODO，但说明它不是治理结论；
- 所有 INCOMPLETE/CONFLICT 的原因必须进入 `## 8. Open Issues`，并包含 owner、影响、决策和目标日期。

#### 3. 增强指标血缘和版本语义

当前 Source Mapping 已覆盖 system.table.field 和 transformation，但金融指标还需要：

- 数据粒度和聚合 grain；
- 数据有效期和 metric definition version；
- schema/contract version；
- backfill/reprocessing 规则；
- late-arriving data、correction、reversal 和 restatement 处理；
- null、zero、negative、missing 和 unknown 的语义；
- reconciliation tolerance 和对账频率；
- 指标变更的 effective date 与历史可比性。

这些内容不必全部塞进主表，可以通过 `references/metrics-governance-rules.md` 或附加章节按需披露。

#### 4. 增加证据与可复核性要求

每个 READY 指标应能指出：

- 定义来源和批准人；
- source mapping 的证据或 schema 引用；
- 公式中的独立 worked example；
- reconciliation/sample validation 计划或结果；
- rounding、FX、cut-off 和 timezone 的示例边界；
- 最后确认时间和 definition version。

不要因为表格字段已填写就把指标标记为 READY；字段完整性和业务正确性是两件事。

#### 5. 增加 focused/standard/full review 范围

完整金融指标 review 可能很长，建议增加：

- `focused`：只审查一个 metric 和点名的风险维度；
- `standard`：执行当前定义、来源、金融正确性、一致性和 owner 全流程；
- `full`：增加历史口径、跨系统对账、监管/管理报表和回填影响，仅用户明确要求时使用。

默认使用 `standard`，但用户明确只想解决 cut-off 或 currency 时应支持 focused，而不是强制填完整长表。

#### 6. 明确敏感数据与外部查询边界

- 不要求用户提供完整生产交易、账户号、客户身份或凭证；
- 优先使用 schema、字段名、脱敏样例和数据分类；
- 真实数据查询、生产 reconciliation、外部监管定义检索需要用户授权；
- 不运行 SQL、ETL、dashboard refresh 或回填命令；
- Illustrative SQL 只能作为说明，不能被误认为已验证的查询。

#### 7. 明确状态门禁的证据关系

- `INCOMPLETE`：缺信息，不代表指标错误；
- `CONFLICT`：已有权威定义冲突，不代表可以由 agent 选择一方；
- `READY`：所有 gate 通过且证据/owner/版本完整；
- 如果 metric 已通过但 source 数据质量尚未验证，应保持 READY 的治理状态与 release 风险分离，不能混用 `GO`。

每个 gate 应输出缺失项、影响、owner、下一步和是否阻塞下游 `eng-review-finance`/`release-review`。

#### 8. 与 AIW/OpenSpec 的交接

- 业务定义未稳定时，回到 `office-hours-finance` 或 `business-review`；
- 指标定义稳定后，`metrics-review` 产出 registry/spec；
- 工程数据流和权限审查交给 `eng-review-finance`；
- 实现需求进入当前 OpenSpec change 的 design/spec/tasks，而不是直接进入 SQL 或 dashboard；
- metric registry 的稳定内容可进入 `openspec/specs/<capability>/`，变更记录仍需关联当前 Task/change；
- 本 skill 不负责 commit、sync、archive、merge 或清理。

### 不应恢复或引入的内容

- 不应只根据 SQL 可写出来就批准指标。
- 不应猜测 source、owner、监管口径、cut-off 或 FX 来源。
- 不应把 TODO/空白表格当成已处理的风险。
- 不应运行生产 SQL、ETL、dashboard 或回填操作。
- 不应把 `READY` 和 release `GO` 混为一谈。
- 不应默认生成或覆盖稳定 registry 文件。
- 不应自动触发 TDD、code review 或 AIW 完成协议。

### 下一步修正规格

后续修正 `skills/metrics-review/SKILL.md` 时，建议：

1. 明确 `message/artifact/openspec` 输出模式。
2. 将 `%%` 纳入未决风险语义，并保留 validator 所需的结构占位。
3. 补充 grain、版本、backfill、late data、reversal、null/zero 和 effective date 规则。
4. 为 READY 指标增加证据、worked example 和确认版本要求。
5. 增加 focused/standard/full review 模式。
6. 增加敏感数据、真实查询和外部资料授权边界。
7. 明确指标规范到 AIW/OpenSpec 和 eng-review-finance 的交接。

### 静态验收清单

- [ ] 输出模式和写入权限没有歧义。
- [ ] 每个 metric 有定义、公式、来源、owner、时间和金融正确性信息。
- [ ] grain、版本、回填、修正、null/zero 和 effective date 有规则。
- [ ] READY 指标有可复核证据，而非只有填满的表格。
- [ ] INCOMPLETE/CONFLICT 都有 Open Issues、owner、影响和目标日期。
- [ ] 支持 focused review，默认不会无必要地产生完整长报告。
- [ ] 不要求或泄露生产敏感数据。
- [ ] 不运行 SQL、ETL、回填或发布操作。
- [ ] OpenSpec/AIW 交接路径明确，不创建平行 registry。
- [ ] 不自动触发测试、code review 或完成协议。

### 当前结论

`metrics-review` 已经具备高质量金融指标治理骨架，建议增量增强血缘版本、证据复核、输出模式、`%%` 风险和有限范围执行。核心 gate 不应削弱。

本章节只记录评审建议，尚未修改 `skills/metrics-review/SKILL.md`。

## `office-hours-finance`

目标文件：`skills/office-hours-finance/SKILL.md`

参考文件：未找到第三方对应版本。以下评审基于当前 skill、intake template、OpenSpec mapping 和 validator。

### 当前评价

质量约为 8.5/10。当前 skill 很好地把金融需求 intake 从“加页面/加按钮/做报表”拉回到“谁在什么情况下看到什么信号、做什么决定、采取什么行动、造成什么下游影响”。决策流、范围削减、owner 和 unknown gates 都很实用。主要问题是 TODO/AIW/OpenSpec 语义、输出文件映射和有限访谈模式还需要明确。

### 当前版本做得好的地方

1. 强制从 business problem 和 decision flow 开始，不提前设计架构、数据库、API 或 UI。
2. 要求 Actor / Situation / Sees / Decides / Acts / Downstream Impact 全部填充。
3. 将 Must Have、Should Have、Nice To Have 和 Explicit Non-Goals 分开，能有效压缩范围。
4. 用 Feature-Centric、Decision Flow、Business Impact、Owner 和 Scope gates 约束 `PROCEED/HOLD/REDUCE/NEEDS_VALIDATION`。
5. 要求 frequency、severity、cost 和 operational owner，避免只描述“很重要”。
6. 下游路由到 business-review、metrics-review、eng-review-finance 和 manual validation 的关系清楚。
7. 结构化输出、模板和 validator 适合后续自动汇总到 autoplan-finance。

### 需要重新增加或调整的内容

#### 1. 统一 TODO 与 `%%` 风险语义

当前 skill 使用 TODO，但项目约定使用 `%%` 记录未解决风险和问题。建议：

- Problem Brief 中的 unknown 和阻塞问题使用 `%%`；
- validator 需要空章节占位时可以保留 TODO，但不能用 TODO 替代 owner、影响或决策日期；
- 每个 `Blocks Next Step? yes` 的 unknown 必须有 owner、下一步、目标日期和阻塞的下游 Skill。

#### 2. 解决 OpenSpec `tasks.md` 与 AIW `tasks.md` 的关系

当前 OpenSpec Handoff 同时输出 `tasks.md` 和 `tasks.md`，容易形成两个任务清单。建议明确：

- `Problem Brief` 是 intake 产物；
- AIW Task identity 使用 `task.toml`；
- 实现 checklist 只使用 `tasks.md`；
- 如果需要保存 brief，应放在当前 OpenSpec change 的 proposal 或指定的 intake artifact，而不是再建立独立 task tracker；
- `tasks.md` 只能作为兼容性映射或被明确要求的 intake 文档，不能被 `/implement` 当作实现 checklist。

#### 3. 增加 focused/standard/deep intake 模式

当前流程要求完整填写很多栏位，可能对小需求过重。建议：

- `focused`：只澄清 actor、decision、action、impact 和一个最小范围；
- `standard`：执行当前完整 Problem Brief，默认模式；
- `deep`：用户明确要求时，增加多 stakeholder、合规、下游系统和运营例外分析。

默认 `standard`，但当用户已经提供完整 decision flow 时应直接整理，不重复问已知问题。

#### 4. 优化“ask once then proceed”

当前缺少必需输入时问一次后返回 HOLD，这对防止无限访谈很好，但应明确：

- 缺少事实时可以从现有仓库/文档中查找，只做廉价只读查询；
- 缺少用户决策时不能猜测；
- 缺少敏感业务数据时使用脱敏字段和范围描述，不要求生产数据；
- 用户未回答时保持 HOLD/NEEDS_VALIDATION，不自动进入 business-review 或 engineering review。

#### 5. 增加事实、假设和决策的区分

Problem Brief 应明确标记：

- 已知事实及来源；
- 用户确认的决策；
- agent 的推断或建议；
- 未确认假设；
- `%%` 未决问题。

否则“频率、成本、风险”可能被 agent 误填成看似精确的数字。

#### 6. 细化业务影响门禁

`frequency × severity × cost` 是好的方向，但不应强制制造数字。建议允许：

- 已知量化值；
- 可信范围或数量级；
- 定性影响 + 明确的轻量验证计划；
- 说明 impact 尚未量化时推荐 `NEEDS_VALIDATION`，而不是假设通过。

#### 7. 明确 AIW/OpenSpec 生命周期边界

- Intake 阶段默认只返回 Problem Brief，不创建 AIW Task、worktree 或实现分支；
- 用户确认问题和范围后，才由 `/to-spec` 或 `autoplan-finance` 创建/关联 AIW Task 和 OpenSpec change；
- OpenSpec Profile 输出必须使用当前 Task/change 的 `task.toml` 和 `tasks.md` 规则；
- 本 skill 不直接 commit、sync、archive、merge 或清理。

#### 8. 增加金融敏感信息边界

- 不要求用户提供客户身份、账户号、交易明细、凭证或生产日志；
- 角色和 stakeholder 可以用岗位/团队名；
- 合规信息缺失时标记为 unknown，不自行推断监管义务；
- 如果需要外部规则查询，单独说明来源和授权，不把检索结果伪装成用户确认。

### 不应恢复或引入的内容

- 不应接受“加一个页面/按钮/报表”作为完成需求。
- 不应在 intake 阶段设计架构、schema、API 或 implementation tasks。
- 不应把 TODO、空表格或假设数字当成已确认信息。
- 不应创建第二套 task tracker 或让 `tasks.md` 与 `tasks.md` 并行管理实现。
- 不应自动创建 AIW Task、worktree、branch、commit 或运行测试。
- 不应因为缺少 business impact 就猜测成本、频率或严重度。

### 下一步修正规格

后续修正 `skills/office-hours-finance/SKILL.md` 时，建议：

1. 保留 decision-centric intake 和现有 decision gates。
2. 统一 `%%` 风险、unknown owner 和目标日期语义。
3. 明确 Problem Brief、task.toml、proposal 和 tasks.md 的关系。
4. 增加 focused/standard/deep 模式。
5. 区分事实、用户决策、建议、假设和未决问题。
6. 允许定性影响和轻量验证计划，不强行伪造精确数字。
7. 增加金融敏感信息和外部查询边界。
8. 明确 intake 完成后如何交给 `/to-spec`、business-review 或 metrics-review。

### 静态验收清单

- [ ] Decision Flow 六列和 operational owner 仍是必要条件。
- [ ] Feature-centric 请求不能直接 PROCEED。
- [ ] 业务影响缺失时不会制造数字，正确返回 NEEDS_VALIDATION。
- [ ] Unknown 包含 owner、影响、目标日期和是否阻塞。
- [ ] `%%` 与 TODO 的职责清楚。
- [ ] `tasks.md` 不会与 `tasks.md` 形成第二套实现清单。
- [ ] 支持 focused intake，默认不会重复问已知信息。
- [ ] 不要求或泄露生产敏感数据。
- [ ] Intake 不自动创建 Task、worktree、branch 或执行命令。
- [ ] 下游路由和 OpenSpec/AIW 交接路径明确。

### 当前结论

`office-hours-finance` 的决策中心和范围削减方法值得保留，已经是较高质量的金融需求 intake skill。建议主要修正文档映射、事实/假设语义、有限访谈模式和 AIW/OpenSpec 边界，不需要削弱现有 HOLD/REDUCE 门禁。

本章节只记录评审建议，尚未修改 `skills/office-hours-finance/SKILL.md`。

## `prototype`

目标文件：`skills/prototype/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\engineering\prototype\SKILL.md`

关联参考：`LOGIC.md`、`UI.md`

### 当前评价

质量约为 7.5/10。核心原则很强：原型必须回答一个具体设计问题，先选择 logic/UI 分支，保持 throwaway、单命令、内存状态、低保真和可见状态。主要问题是它会直接写代码并要求运行，却没有与当前 AIW worktree、运行授权、自动提交/清理闭环和原型结果吸收规则对齐。

### 参考版本中值得保留的做法

1. 让问题决定原型形状，而不是先搭一个泛用 demo。
2. 明确 logic/state model 与 UI 两个分支，避免用错误的原型回答错误的问题。
3. 原型从第一天就标记为 throwaway，防止误认为生产实现。
4. 要求一个无需思考即可运行的命令，降低验证摩擦。
5. 默认内存状态、不引入 persistence、不做 polish、不写无关 abstraction。
6. 每次 logic action 或 UI variant switch 都展示完整相关状态，保证原型真的能回答问题。
7. 完成后捕获 verdict、问题和 validated decision，而不是只留下没有结论的代码。

### 当前版本已经做好的地方

- 分支判断清晰，且 `LOGIC.md`/`UI.md` 通过外部参考实现渐进披露。
- 明确禁止默认 persistence 和过度 polish。
- 明确要求显示状态变化。
- 没有把 prototype 伪装成 production-ready code。
- 没有自动调用测试、code review 或发布外部系统。

### 需要重新增加或调整的内容

#### 1. 增加问题定义和完成标准

原型开始前至少要写清楚：

- 要回答的一个问题；
- 成功判据或需要观察的信号；
- 不回答的相邻问题；
- 预计保留时间和资源预算；
- 结果如何影响后续 design/spec。

完成时必须输出：

- 实际观察到的行为；
- 结论：支持、否定或仍无法判断；
- 被验证的设计决定；
- 未解决的 `%%` 风险；
- 原型代码的去向：删除、保留为 primary source 或吸收进正式实现。

#### 2. 运行命令需要显式授权和预算

当前规则要求“一条命令”，但没有说明可以直接运行。建议：

- 写原型可以在用户明确授权后进行；
- 第一次运行前展示命令、范围、预计时长和依赖；
- 默认只运行一次 focused 命令，相关代码或环境变化后最多重跑一次；
- UI server、browser automation、长时间运行、外部网络或真实数据需要单独授权；
- 不自动安装依赖、下载 CDN 或访问生产系统。

#### 3. 与 AIW worktree/branch 对齐

参考版本的“commit to a throwaway branch”不能直接使用 raw Git。建议：

- 原型只为当前 AIW Task 服务时，放入该 Task 的 AIW worktree，并用明显的 PROTOTYPE 标记；
- 原型需要独立生命周期时，先经过用户批准，创建独立 AIW Task/worktree，而不是手工 throwaway branch；
- 不把原型代码混入生产实现 commit，除非用户确认吸收；
- 原型结论应通过 handoff、OpenSpec design 或 ADR 记录，而不是依赖 issue tracker；
- 完成协议只清理已明确标记为可删除的原型资源，不能删除包含 validated decision 的唯一证据。

#### 4. 保护项目环境和数据

- 默认使用 fixture、内存状态、mock adapter 和本地 sandbox；
- 不使用真实客户、支付、生产或外部服务数据；
- 数据库 prototype 使用独立 scratch DB/文件，并明确 `PROTOTYPE - WIPE ME`；
- UI prototype 不修改真实路由、生产配置或持久化 schema；
- prototype 依赖和临时文件必须有清单，避免污染正式环境。

#### 5. 增加 focused/full prototype 模式

建议：

- `focused`：一个问题、一个路径/状态模型、一个命令、少量场景；默认模式；
- `comparative`：UI 或接口需要 2–3 个明显不同候选时使用；
- `deep`：用户明确要求时探索更多边界，但仍必须设时间、场景和依赖预算。

不要因为“prototype”就顺手加入完整错误处理、测试、认证、持久化、部署或通用抽象。

#### 6. 明确 prototype 与 TDD 的边界

- prototype 用来回答设计问题，不替代 regression test；
- 不因为 prototype 能运行就宣称实现正确；
- validated decision 吸收后，再由 focused TDD 或 `/implement` 建立正式行为测试；
- prototype 本身是否保留为 primary source，要在结论中明确。

### 不应恢复或引入的内容

- 不应使用 raw Git 创建 throwaway branch/worktree。
- 不应在没有运行授权时执行 prototype 命令。
- 不应访问生产数据、外部服务或真实支付/账户系统。
- 不应把 prototype 代码直接当成生产代码或自动合并。
- 不应自动安装依赖、引入 persistence、部署或长时间运行服务。
- 不应只留下代码而不记录 verdict 和设计决定。
- 不应为了 prototype 自动触发 TDD、code review 或 AIW 完成协议。

### 下一步修正规格

后续修正 `skills/prototype/SKILL.md` 及其 references 时，建议：

1. 增加问题、成功判据、非目标和完成输出。
2. 增加运行授权、单命令预算和外部副作用边界。
3. 用 AIW worktree/Task 规则替代 raw throwaway branch 语义。
4. 增加 fixture/sandbox/scratch data 安全边界。
5. 增加 focused/comparative/deep 模式。
6. 明确 prototype 结果如何进入 handoff、OpenSpec design、ADR 或正式实现。
7. 保持 prototype 与 TDD、code review、生产实现的边界。

### 静态验收清单

- [ ] 原型开始前有一个明确问题、成功判据和非目标。
- [ ] logic/UI 分支选择清楚，歧义时会确认或记录假设。
- [ ] 默认 focused，运行命令、重试次数和依赖有边界。
- [ ] 运行前需要用户授权，不自动访问网络、生产或真实数据。
- [ ] 原型在 AIW worktree/Task 中隔离，或独立 Task 已获批准。
- [ ] 所有 prototype 资源有 PROTOTYPE 标记和去向。
- [ ] 完成输出包含观察、verdict、validated decision 和 `%%` 风险。
- [ ] 不把 prototype 代码自动视为生产实现。
- [ ] 不自动触发 TDD、code review 或完成协议。

### 当前结论

`prototype` 的设计理念值得保留，但需要补齐运行授权、AIW 隔离、数据安全、问题完成标准和结果吸收路径。它应继续是快速回答单一设计问题的工具，不应演变成低质量的临时生产实现。

本章节只记录评审建议，尚未修改 `skills/prototype/SKILL.md`。

## `publish-github-issue`

目标文件：`skills/publish-github-issue/SKILL.md`

参考文件：未找到第三方对应版本。以下评审基于当前 skill、`scripts/projection.py`、测试脚本和 AIW/OpenSpec 工作管理规则。

### 当前评价

质量约为 8.5/10。当前 skill 已经清楚地区分本地权威资料与 GitHub 外部投影，并具备显式授权、映射文件、managed marker、保留人工内容和单向发布等关键安全措施。主要需要补充的是网络请求前的确认、敏感内容筛选、并发/幂等、映射文件提交和失败恢复边界。

### 当前版本做得好的地方

1. 只有用户明确要求时才允许发布，且 `disable-model-invocation: true` 防止意外外发。
2. 明确 AIW 管生命周期、OpenSpec 管需求与 checklist，GitHub 只是 projection。
3. 要求解析唯一 AIW Task 和 matching OpenSpec change，避免发布错误对象。
4. 使用 `external/github.json` 保存版本化映射，避免重复创建 Issue。
5. 更新时只替换 managed marker block，保留 marker 外人工内容。
6. 要求通过 `--body-file`/stdin 传递正文，避免 shell quoting 和敏感内容泄露。
7. 明确不关闭 Issue、不更新本地 Task 状态、不导入远程评论，单向边界清楚。
8. 使用 projection helper 和测试脚本，减少 Markdown 拼接漂移。

### 需要重新增加或调整的内容

#### 1. 增加外部发布前的最终确认摘要

即使用户已经明确要求发布，也建议在网络请求前展示一次不可歧义的摘要：

- AIW Task ID 和 OpenSpec change ID；
- GitHub owner/repo；
- 创建还是更新；
- Issue number/URL（更新时）；
- 将公开的标题、scope、requirements 和 task progress；
- 是否包含敏感或内部信息；
- 将执行的具体命令和请求数量。

用户明确要求“立即发布”时可以省略二次确认，但必须在结果中报告这些信息。

#### 2. 强化敏感信息和内部内容过滤

投影到公开 Issue 前应检查并默认排除：

- API key、token、密码、cookie 和签名 URL；
- 客户、账户、交易、内部 IP、生产日志和 PII；
- 内部安全细节、临时调查结果和未批准的监管判断；
- 详细 design notes、未决内部讨论和不适合公开的 `%%` 内容。

如果无法判断某段内容是否可公开，停止并请求用户决定，不要默认外发。

#### 3. 明确 credential provider 边界

当前要求 `GITHUB_TOKEN`，建议补充：

- 不在命令行、日志、handoff 或 Issue body 中输出 token；
- 优先使用 AIW/GitHub credential provider；
- token 缺失或权限不足时停止，不要求用户把 token 粘贴到聊天中；
- 只请求所需 repo 的最小权限；
- 不自动更换账号、仓库或 token 重试。

#### 4. 增加幂等和并发安全

创建/更新流程应明确：

- 先读取并校验 mapping；
- mapping 缺失时只允许一次 create 请求；
- create 成功但本地 mapping 写入失败时，不要再次 create，先报告 Issue 信息并恢复 mapping；
- update 前再次确认 Issue number、repository、URL 和 marker 属于同一对象；
- 如果远程 body 在读取后被其他人更新，更新失败时保留远程内容，不覆盖重试；
- `external/github.json` 写入采用原子方式，并记录 content hash/timestamp。

#### 5. 明确本地映射文件的提交边界

外部发布成功后会修改 `external/github.json`，建议规定：

- mapping 是本地外部投影元数据，不改变 AIW Task 或 OpenSpec 需求所有权；
- 发布成功后将 mapping 作为当前分支的普通变更提交；
- 如果当前工作区属于 AIW Task，mapping 应在对应 Task 分支中提交，并由完成协议处理；
- 发布失败时不修改 Task 状态、不 archive、不 merge、不清理。

#### 6. 处理网络成本和失败重试

- 默认一次读取/创建/更新流程，不自动循环重试；
- 网络超时、429、权限错误和远程 404 应分别报告；
- 只有明确的瞬时网络错误才允许一次有限重试；
- 不因为生成内容不满意而反复更新 Issue；
- 失败时报告本地 projection body 和 mapping 状态，方便人工恢复。

#### 7. 增加投影内容完成标准

投影至少应包含：

- change ID；
- 目标和 scope；
- 核心 requirement summary；
- tasks progress；
- AIW/OpenSpec ownership statement；
- 本地 canonical artifact 路径；
- 更新时间或 content hash。

同时不得把远程 Issue 当作实现 checklist 或规范来源。

### 不应恢复或引入的内容

- 不应在用户未明确要求时发布 GitHub Issue。
- 不应把 GitHub Issue、评论或标签反向当作本地需求和 Task 状态来源。
- 不应自动关闭 Issue、同步评论、修改标签或更新本地生命周期状态。
- 不应在无法验证 mapping 时静默创建 replacement Issue。
- 不应把 token 或敏感内部内容写入日志、body 或 mapping。
- 不应在网络失败时无限重试或重复创建。
- 不应在发布失败时触发 archive、merge、worktree 清理或删除 branch。

### 下一步修正规格

后续修正 `skills/publish-github-issue/SKILL.md` 时，建议：

1. 增加发布前摘要和公开内容检查。
2. 增加 credential provider、最小权限和 token 脱敏规则。
3. 增加 create/update 幂等、并发和 mapping 恢复流程。
4. 明确 `external/github.json` 的提交边界。
5. 为网络失败增加有限重试和错误分类。
6. 明确 projection 最小内容和本地 canonical 路径。
7. 保持单向、显式、marker-only 更新边界。

### 静态验收清单

- [ ] 用户明确授权后才会发起网络请求。
- [ ] 发布前能明确展示目标仓库、Issue、动作和公开内容范围。
- [ ] 敏感信息和不适合公开的内部内容有过滤/停止规则。
- [ ] credential 不会进入命令输出、日志、handoff 或 Issue body。
- [ ] mapping 缺失、损坏或远程不匹配时不会静默创建 replacement。
- [ ] create/update 具备幂等和失败恢复规则。
- [ ] managed marker 外的人类内容始终保留。
- [ ] 失败时不改变 Task 状态、不 archive、不 merge、不清理。
- [ ] 成功后的 mapping 变更能进入当前分支提交。
- [ ] GitHub 仍然只是单向 projection，不是本地权威来源。

### 当前结论

`publish-github-issue` 已经是安全边界较好的外部发布 skill，核心设计不需要重写。建议补充公开内容审查、凭据边界、幂等恢复、有限重试和 mapping 提交规则。

本章节只记录评审建议，尚未修改 `skills/publish-github-issue/SKILL.md`。

## `release-review`

目标文件：`skills/release-review/SKILL.md`

参考文件：未找到第三方对应版本。以下评审基于当前 skill、release-gate references、OpenSpec mapping 和 validator。

### 当前评价

质量约为 8.5/10。当前 skill 已经提供了较完整的金融发布门禁：scope、migration、data、metrics、permission、audit、rollback、observability、ownership 和 GO/GO WITH RISK/NO GO 决策。它明确是 read-only gate，不执行部署。主要需要补充输出模式、证据新鲜度、`%%` 风险语义、OpenSpec 映射和有限范围评审。

### 当前版本做得好的地方

1. 对 schema/data/permission/audit/rollback/monitoring 的缺失使用强制 `NO GO`，适合金融发布。
2. 明确区分 `GO`、`GO WITH RISK` 和 `NO GO`，并要求业务与工程 owner。
3. 对 migration lock、backward/forward compatibility、backfill idempotency、reconciliation tolerance 设门禁。
4. 对敏感 action 的 before/after、reason、trace 和 retention 设审计要求。
5. 将 metrics-review 和 eng-review-finance 的状态接入 release decision，避免发布审查重新发明指标口径。
6. checklist、open risks、final recommendation 和 validator 结构适合机器读取和 autoplan-finance 聚合。
7. 明确不写代码、migration 或 deployment script，也不修改仓库。

### 需要重新增加或调整的内容

#### 1. 解决输出模式与写入权限歧义

当前同时写“emits RELEASE_REVIEW.md”和“return as a message; do not write to disk”。建议明确：

- `message`：默认返回结构化 release review，不写文件；
- `artifact`：用户明确要求时生成 `RELEASE_REVIEW.md`；
- `openspec`：用户明确要求时返回或写入当前 change 的 release artifact。

任何模式都不得直接执行 deploy、migration、rollback、feature flag 或生产命令。

#### 2. 将 TODO 与 `%%` 风险语义统一

当前使用 TODO 标记不确定事项，但项目约定使用 `%%`。建议：

- `## 10. Open Risks` 中的未决风险使用 `%%`；
- validator 所需的空章节可保留 TODO，但必须同时给出 blocker/owner/action；
- 没有 owner 的风险不能通过 `GO WITH RISK`。

#### 3. 增加证据新鲜度和可追溯性

每个 gate 应记录：

- evidence 来源路径、commit、版本或系统；
- 检查时间和适用 launch window；
- evidence owner；
- 是设计承诺、已验证结果还是待执行计划；
- 是否覆盖 staged rollout、全量 rollout 和 rollback。

不能因为设计文档存在就认为 migration、监控或 rollback 已经验证。

#### 4. 细化 GO WITH RISK 的门槛

当前定义方向正确，但建议明确 `GO WITH RISK` 不能用于：

- permission unknown；
- audit missing；
- rollback 不存在；
- metrics 为 `INCOMPLETE/CONFLICT`；
- 没有 release/rollback/on-call owner；
- 数据 reconciliation 没有 tolerance 或验证计划。

它只能用于已知、量化、有人负责、可快速检测且可恢复的非 blocker 风险。

#### 5. 增加 focused/standard/full review 模式

建议：

- `focused`：只评审用户点名的 migration、permission、rollback 等单一高风险轴；
- `standard`：执行当前全部 release gates，默认模式；
- `full`：增加历史数据、分阶段 rollout、监管证据、演练记录和跨系统 reconciliation，仅明确要求时使用。

无论模式如何，关键 gate 缺失都不能被范围缩小绕过。

#### 6. 增加 staged rollout 和 kill-switch 规则

除了 feature flag 是否存在，还应评审：

- 默认状态和目标人群/数据范围；
- 灰度比例和扩大条件；
- kill-switch 的权限、响应时间和审计；
- 观测窗口和停止阈值；
- staged rollout 到 full rollout 的二次 gate。

#### 7. 处理 OpenSpec/AIW 映射中的任务文件问题

当前 OpenSpec Handoff 示例同时列出 `task.toml`、`tasks.md`、`tasks.md`，可能产生双重任务记录。建议：

- release review 只生成 release artifact；
- AIW Task identity 由 `task.toml` 管理；
- 实现 checklist 继续由 `tasks.md` 管理；
- `tasks.md` 若保留，只能作为兼容性 intake/summary 文档，不能被 `/implement` 当作 checklist；
- 归档、sync、merge 和 worktree 清理由 AIW 完成协议负责，不由 release review 执行。

#### 8. 增加敏感信息与外部操作边界

- 不要求生产凭证、客户数据、完整交易、内部网络细节或真实 secrets；
- rollback/monitoring evidence 可以使用脱敏截图、配置摘要和验证记录；
- 不执行 deploy、migration、rollback、feature flag、backfill 或生产查询；
- 发现 release 已经开始或发生事故时，转入 incident/operations 流程，不把 review 当作应急执行器。

### 不应恢复或引入的内容

- 不应在缺少关键 gate 时给出 GO WITH RISK 来“先上线再补”。
- 不应把设计文档、计划或 feature flag 存在当作已验证 evidence。
- 不应运行部署、迁移、回滚、回填、生产查询或 flag 操作。
- 不应把 GO、APPROVE、READY 混为一个状态。
- 不应让 `tasks.md` 与 `tasks.md` 形成第二套实现清单。
- 不应自动写文件、commit、archive、merge 或清理 worktree。
- 不应泄露生产敏感数据或凭证。

### 下一步修正规格

后续修正 `skills/release-review/SKILL.md` 时，建议：

1. 明确 `message/artifact/openspec` 输出模式。
2. 统一 `%%` 风险记录和 validator 占位规则。
3. 增加 evidence 来源、版本、时间、owner 和验证状态。
4. 收紧 GO WITH RISK 的不可豁免 blocker。
5. 增加 focused/standard/full review 范围。
6. 增加 staged rollout、kill-switch 和二次 gate。
7. 清理 OpenSpec tasks.md/tasks.md 映射歧义。
8. 保持 release review 只读，不执行任何发布或生命周期操作。

### 静态验收清单

- [ ] 输出模式和文件写入权限没有歧义。
- [ ] 关键 gate 缺失时稳定返回 NO GO。
- [ ] GO WITH RISK 只接受已知、量化、有人负责且可恢复的非 blocker 风险。
- [ ] 每个结论都有 evidence 来源、时间、owner 和验证状态。
- [ ] 支持 focused review，但不能绕过关键门禁。
- [ ] staged rollout、kill-switch 和 full rollout gate 有明确规则。
- [ ] 不执行 deploy、migration、rollback、backfill 或生产查询。
- [ ] AIW/OpenSpec 映射不会产生第二套 checklist。
- [ ] 不自动触发 TDD、code review、archive、merge 或 worktree 清理。

### 当前结论

`release-review` 已经具备高质量的金融发布门禁骨架，建议增量增强证据新鲜度、GO WITH RISK 边界、staged rollout、输出模式和 OpenSpec 映射。关键的 NO GO 规则不应削弱。

本章节只记录评审建议，尚未修改 `skills/release-review/SKILL.md`。

## `research`

### 评审对象

- 本地版本：`skills/research/SKILL.md`
- 参考版本：`D:\03_projects\third-part\skills\skills\engineering\research\SKILL.md`
- 本地版本与参考版本完全一致，尚未加入 AIW/OpenSpec 和当前资源授权规则。

### 当前评价

当前质量约为 **6.5/10**。参考版本把 research 的核心职责压缩得很清楚：针对一个问题、优先使用高信任 primary sources、每个结论保留来源、输出单一 Markdown 文件。但它默认每次启动 background agent、访问外部资料并写入 repo；这与当前仓库默认 0 个 sub-agent、网络关闭、运行/写入边界受控的规则冲突。

### 参考版本值得保留的做法

1. 将研究问题限制为一个明确的问题，避免泛泛收集资料。
2. 优先使用官方文档、源代码、标准、第一方 API 等 primary sources，而不是二手总结。
3. 要求每个 claim 回溯到拥有该事实的来源，减少无依据推断。
4. 将 findings 汇总到一个 Markdown 产物，便于后续 Session 或 planning skill 消费。
5. 保存到仓库已有的 notes convention，并在没有 convention 时说明选择的路径，体现了对项目结构的尊重。

### 需要调整的内容

#### 1. 不应默认启动 background agent

“Spin up a background agent”是当前 skill 的强制第一步，但仓库默认 sub-agent budget 为 0。建议增加资源模式：

- `static`：0 个 sub-agent，优先读取本地文件和已有文档，默认模式；
- `focused`：1 个 sub-agent，处理一个清晰、独立的研究问题；
- `parallel`：最多 2 个 sub-agent，仅处理互不依赖的来源集合；
- 超过 2 个、需要长期运行或需要多个外部系统时先取得单独授权。

sub-agent 必须有问题边界、来源范围、输出格式、截止条件和预算；不得测试、构建、提交、创建 branch/worktree、archive、merge 或清理资源。

#### 2. 网络访问和外部资料需要显式授权

primary-source 要求本身并不等于已授权联网。建议在研究前区分：

- 本地 primary source：可按静态读取规则直接检查；
- 外部网页、API、仓库或下载：先说明目标域名/来源、查询范围、预计成本和是否涉及凭据，取得授权后再访问；
- 不能联网时，输出 `%% NEEDS_SOURCE`，不能用记忆补齐事实。

研究结果必须标注来源 URL 或本地路径、版本/提交号、访问日期，以及来源适用范围。不能把当前可访问当成永久有效。

#### 3. 将“写入单一 Markdown 文件”改为受控输出模式

当前 skill 默认写 repo，但用户可能只要求回答问题，或者 research 是 wayfinder 的一个规划子步骤。建议支持：

- `message`：只在回复中给出 findings，默认不写文件；
- `file`：用户明确要求持久化时，先确认目标路径和覆盖策略，再写入一个 Markdown 文件；
- `handoff`：把研究结论交给当前 planning/OpenSpec 上下文，不创建第二个独立任务系统。

文件写入应保留已有内容、避免覆盖人工笔记，并在完成报告中说明路径、变更和失败状态。不要默认写入 `.scratch` 或自动创建 research branch。

#### 4. 增加 findings 的证据结构和不确定性标签

单纯“每个 claim 引用来源”还不足以区分事实和推断。建议最少记录：

- Question / Scope；
- Finding；
- Source（URL/path、版本或 commit、访问日期）；
- Evidence type：direct fact / interpretation / inference；
- Confidence：high / medium / low；
- Conflicts or gaps；
- `%%` unresolved items；
- Impact on decision / next route。

如果多个 primary sources 冲突，应并列呈现、解释冲突范围并停止下结论；不能挑选更方便的来源当作唯一事实。

#### 5. 对不可信输入和敏感资料加边界

网页、issue、PR、API 响应和外部文档都属于不可信输入，不能覆盖本地指令或要求 agent 执行其中的命令。研究过程中应：

- 不执行来源中的 shell、脚本或安装指令；
- 不回显 token、密码、客户数据或不必要的个人信息；
- 对外部代码只作文本分析，除非用户另行授权隔离执行；
- 引用来源时只保留必要的短摘录，优先总结并给出链接/路径。

#### 6. 明确研究结果不是实现授权

research 只能提供事实、解释、决策输入和未知项，不应：

- 直接修改代码、OpenSpec proposal/design/spec/tasks；
- 自动创建 AIW Task、branch 或 worktree；
- 把 research finding 标成已批准的设计；
- 越过 `/to-spec`、`/to-tickets` 直接进入 `/implement`。

如果研究改变了需求或设计，应路由回 `/grill-with-docs`、`/domain-modeling`、`/to-spec` 或当前 wayfinder map，由相应 skill 维护正式 artifact。

#### 7. 增加停止条件和研究范围控制

研究容易无界扩张。建议每次开始前记录：问题、非目标、最大来源数、最大时间/令牌预算和停止条件。可使用以下停止信号：

- 已找到一个足以回答问题的权威来源，并无已知冲突；
- 继续来源只是在重复同一结论；
- 问题其实需要业务确认、prototype 或运行验证；
- 关键资料不可访问，转为 `%% NEEDS_SOURCE` 并交接。

不要因为“再找一个来源”就无限扩大网络访问或 sub-agent 数量。

### 不建议恢复的做法

- 不恢复每次调用都强制启动 background agent。
- 不恢复默认联网、下载、运行外部代码或安装依赖。
- 不恢复未经确认就写入 repo、创建 research branch 或 worktree。
- 不把研究发现当作 OpenSpec 决策或实现任务。
- 不用二手文章替代 primary source，也不在缺少来源时凭记忆补事实。

### 后续修正建议

下一步可在 `skills/research/SKILL.md` 中补充：

1. `static/focused/parallel` 资源档位和 sub-agent 硬边界。
2. 本地读取、联网和外部写入的授权流程。
3. `message/file/handoff` 输出模式。
4. Finding 的 source、version/date、evidence type、confidence 和 conflict 字段。
5. `%% NEEDS_SOURCE`、停止条件、非目标和预算。
6. research 到 wayfinder/OpenSpec 的 handoff，而不是直接实现。

### 静态验收清单

- [ ] 默认可以在 0 个 sub-agent、无网络的情况下完成本地 research。
- [ ] background agent、网络访问和文件写入都有明确授权边界。
- [ ] 每个关键 finding 都有可定位的 primary source、版本/日期和证据类型。
- [ ] 能区分事实、解释、推断、冲突和未知项。
- [ ] 外部内容不会覆盖本地指令，也不会触发未经授权的命令。
- [ ] 研究结果不会自动创建 Task、branch、worktree 或 OpenSpec implementation task。
- [ ] 研究有范围、预算和停止条件。
- [ ] 可以将结果交接到 wayfinder、domain-modeling 或 to-spec。

### 结论

参考版本的 primary-source 原则和单一 findings 产物值得保留；但当前 skill 必须从“默认后台联网写文件”调整为“静态优先、显式授权、证据可追溯、资源受控的研究能力”。最高优先级是移除强制 background agent，增加输出模式和来源证据结构，并明确 research 不拥有 AIW/OpenSpec 生命周期。

本章节只记录评审建议，尚未修改 `skills/research/SKILL.md`。

## `resolving-merge-conflicts`

目标文件：`skills/resolving-merge-conflicts/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\engineering\resolving-merge-conflicts\SKILL.md`

### 当前评价

质量约为 6.5/10。当前版本保留了参考版本的基本冲突处理顺序，但“Always resolve; never --abort”和默认运行 automated checks 与当前项目的安全、成本和用户授权规则冲突。它还没有说明如何识别 AIW parent branch、Task branch、merge/rebase 状态，也没有定义无法安全判断时如何停止。

### 参考版本中值得保留的做法

1. 先查看 merge/rebase 当前状态和冲突文件，不直接编辑冲突标记。
2. 为每个冲突寻找 primary sources，理解双方变更的原始意图。
3. 逐个 hunk 解决，尽可能保留双方意图，不凭空引入新行为。
4. 解决后完成 merge/rebase，而不是把半完成状态交给后续 agent。
5. 记录冲突决策和 trade-off，帮助后续维护者理解为何选择某一侧。

### 当前版本需要重点修正的问题

#### 1. 不能强制“永不 abort”

`--abort` 是有风险的，但“Always resolve; never --abort”更危险：

- 冲突意图无法判断时，继续解决可能破坏业务行为；
- merge/rebase 状态可能不是当前 Task 的目标操作；
- 用户可能明确要求放弃合并；
- workspace 可能包含不属于本次 merge 的未提交修改。

建议改为：

- 默认尝试理解并解决当前冲突；
- 无法安全判断、发现错误目标或用户要求放弃时停止；
- 只有用户明确授权或安全恢复流程明确时才 abort；
- abort 前记录当前状态和未决冲突，避免静默丢失用户工作。

#### 2. 运行检查必须受授权和预算控制

当前要求自动发现并运行 typecheck、tests、format，这与当前规则冲突。建议：

- 先做静态检查：冲突标记清除、文件状态、差异、ours/theirs 语义和语法结构；
- 运行测试、构建、格式化前展示准确命令、范围、时长并请求授权；
- 默认最多运行一次最小 focused check；
- 相关代码/环境变化后最多重跑一次；
- 不自动扩大到全仓测试或构建。

#### 3. 增加 AIW 合并上下文识别

冲突处理前应确认：

- 当前是 merge 还是 rebase；
- 当前 workspace 是 AIW Task worktree 还是 parent branch；
- Task branch、`parent_branch`、Session 和 OpenSpec change 是否匹配；
- 冲突是否属于自动完成协议的 Task branch → parent branch 合并；
- 是否存在未提交的非冲突修改。

如果当前状态不匹配，不应直接处理冲突，应停止并报告。

#### 4. 增强 primary source 的本地优先级

参考版本提到 PR 和 Issues，但当前项目的权威顺序应是：

1. 当前 Task 的 `task.toml` 和 `parent_branch`；
2. OpenSpec proposal、design、capability specs、`tasks.md`；
3. 冲突双方的 commit message 和 diff；
4. `CONTEXT.md`、ADR 和领域/架构规则；
5. 外部 Issue 或 PR，仅在用户明确提供且允许读取时使用。

远程评论不能自动覆盖本地 OpenSpec 或 AIW 状态。

#### 5. 增加解决完成标准

不能只看不到 `<<<<<<<` 就宣告完成。至少应确认：

- 所有 unmerged paths 已处理；
- 冲突解决保留了双方必要意图，或记录了明确取舍；
- 没有引入超出 merge 目标的新行为；
- `git diff --check` 等静态检查通过；
- 冲突决策已记录在 commit message、handoff 或 Task notes；
- 若用户授权，focused tests/checks 通过；
- merge/rebase 状态已完成，工作区没有半完成状态。

#### 6. 与自动完成协议衔接

当冲突发生在 Task branch 合并到 `parent_branch` 的自动完成协议中：

- 自动协议必须暂停，不得删除 worktree 或 Task branch；
- 进入本 skill 解决后，先提交冲突解决结果；
- 再由完成协议验证 merge、sync/archive 状态和清理条件；
- 如果冲突解决失败，保留 Task branch、worktree 和 handoff；
- 不因为冲突而创建第二个 Task。

### 不应恢复或引入的内容

- 不应无条件执行 `--abort`，也不应无条件禁止 abort。
- 不应自动运行全仓测试、构建或格式化。
- 不应在错误的 workspace、branch 或 merge 状态下解决冲突。
- 不应把远程 PR/Issue 当作本地权威来源。
- 不应只删除冲突标记就完成，不记录取舍或验证状态。
- 不应在冲突未解决时 commit、archive、merge、删除 worktree 或删除 branch。
- 不应把冲突解决扩展成无关重构。

### 下一步修正规格

后续修正 `skills/resolving-merge-conflicts/SKILL.md` 时，建议：

1. 保留先观察状态、查 primary source、逐 hunk 解决和最终完成 merge/rebase 的主流程。
2. 将 `never --abort` 改为安全停止/用户授权 abort 规则。
3. 增加 AIW Task/parent_branch/worktree/Session 状态确认。
4. 将自动检查改为静态优先、focused、需授权的运行模式。
5. 增加冲突解决完成标准和决策记录要求。
6. 明确与自动完成协议的暂停、恢复和清理顺序。

### 静态验收清单

- [ ] 能识别 merge/rebase 类型和当前 AIW workspace。
- [ ] primary source 优先使用 Task/OpenSpec/ADR，而不是默认远程 Issue。
- [ ] 无法安全判断时会停止，不强行选择 ours/theirs。
- [ ] abort 有明确授权和状态保存规则。
- [ ] 测试、构建、格式化需要命令、范围和授权。
- [ ] 所有 unmerged paths、冲突决策和完成状态有检查。
- [ ] 冲突期间不会 archive、清理 worktree 或删除 branch。
- [ ] 解决成功后能安全回到 AIW 自动完成协议。

### 当前结论

`resolving-merge-conflicts` 的核心顺序正确，但必须修正“永不 abort”和“自动运行检查”这两个危险默认值，并补足 AIW 上下文、停止条件和自动完成协议衔接。它应优先保证不丢失工作和不引入错误行为。

本章节只记录评审建议，尚未修改 `skills/resolving-merge-conflicts/SKILL.md`。

## `resume-ext`

目标文件：`skills/resume-ext/SKILL.md`

参考文件：未找到第三方对应版本。以下评审基于当前 skill 和 AIW/Codex Session 工作流要求。

### 当前评价

质量约为 8.5/10。当前 skill 已经正确把 `aiw cxs` 作为会话来源，避免直接扫描内部 Session 文件，并明确不在当前 Codex 进程中执行 `codex resume`，不启动 nested Codex。主要还需要补充路径/Task 安全校验、敏感信息处理、`all` 范围确认和恢复后的 AIW 状态衔接。

### 当前版本做得好的地方

1. 使用 `aiw cxs list --current-workspace --json`，避免依赖非稳定的本地 Session 存储格式。
2. 只从最近一次展示的结果解析用户选择，防止编号过期或跨列表误选。
3. 同时展示 alias、title、updated time 和 `original_cwd`，足够用户区分会话。
4. 选择不明确或已不存在时重新展示，不猜测目标 Session。
5. `original_cwd` 缺失时拒绝生成不安全的恢复命令。
6. 明确只提供用户可复制的命令，不在 active Codex session 中执行 resume。
7. 提示和选项使用 Easy English，符合当前用户界面约束。

### 需要重新增加或调整的内容

#### 1. 增加 Session 与 AIW Task/worktree 的一致性信息

如果 Session 绑定 AIW Task，列表/预览最好同时显示：

- Task ID；
- branch 和 worktree；
- Session 状态和 lease 状态；
- matching OpenSpec change；
- parent branch（如果属于自动完成中的 Task）。

恢复前确认这些信息仍然匹配。不能只因为 `original_cwd` 相同，就认为可以在正确 Task 上继续。

#### 2. 验证恢复目录和安全范围

生成命令前应检查：

- `original_cwd` 是存在的目录；
- 它属于当前 workspace 或用户明确选择的 workspace；
- 不是已删除、已归档或不再属于当前 Task 的 worktree；
- 路径没有被截断、转义或被用户输入注入命令；
- command 使用结构化参数展示，不把 title/alias 拼接进可执行 shell 片段。

如果目录不匹配，显示原因并不生成 resume 命令。

#### 3. 明确 `all` 的范围扩展风险

`all` 会取消 `--current-workspace` 过滤，可能展示其他项目或用户不期待的 Session。建议：

- 默认只显示当前 workspace；
- 只有用户明确输入 `all` 才扩展；
- 扩展前说明会展示跨 workspace Session；
- 展示时标明每个 Session 的 workspace，不把它们混成同一项目；
- `all` 结果也必须经过同样的路径、Task 和状态检查。

#### 4. 增加刷新和过期选择的竞态处理

在列出 Session 到用户选择之间，Session 可能已更新、结束、归档或被其他 agent 占用。建议：

- 选择后再次用稳定 Session ID 查询状态；
- 如果状态、alias、路径或 lease 发生变化，重新展示并请求选择；
- 不使用旧列表中的 path 直接生成命令；
- 不自动夺取 lease、切换 Task 或关闭其他 Session。

#### 5. 保护 Session 内容和显示字段

Session title、alias、cwd 或 handoff 摘要可能包含敏感信息。建议：

- 列表默认只显示必要元数据，不显示完整 prompt、日志或 token；
- 对 title/alias/cwd 中的 credential、PII 和内部密钥做脱敏；
- 输出命令时不携带 prompt 或 session 内容；
- 用户明确要求预览时，仍只展示脱敏摘要。

#### 6. 明确恢复后的责任边界

resume-ext 只负责发现和准备命令，不负责：

- 自动 handoff；
- 自动切换到 worktree；
- 自动启动新 Thread；
- 修改 Task、Session、lease 或 branch 状态；
- 运行测试、构建、commit、archive 或清理。

恢复到新 Thread 后，由 `/handoff`、`/implement` 或其他对应 Skill 重新解析当前 AIW 状态。

#### 7. 增加输出模式和完成标准

建议支持：

- `list-only`：仅列出会话；
- `preview`：显示选中会话的脱敏元数据和状态；
- `command`：生成可复制的 `codex resume` 命令，不执行。

完成标准是：用户能明确选择一个合法 Session，看到目标目录、Task/worktree 状态和可复制命令；如果任一关键字段不可信，就明确停止。

### 不应恢复或引入的内容

- 不应自动执行 `codex resume` 或启动 nested Codex。
- 不应默认展示跨 workspace 的所有 Session。
- 不应从旧列表直接恢复已变化的 Session。
- 不应自动夺取 lease、切换 worktree、修改 Task 或清理资源。
- 不应显示完整 Session prompt、日志、token 或敏感路径细节。
- 不应把 `original_cwd` 单独当作 AIW Task/worktree 身份证明。

### 下一步修正规格

后续修正 `skills/resume-ext/SKILL.md` 时，建议：

1. 保留 `aiw cxs` JSON 来源、最近列表选择和不执行 resume 的核心边界。
2. 增加 Task、branch、worktree、Session、lease 和 OpenSpec change 预览。
3. 增加路径存在性、workspace 归属和命令安全检查。
4. 明确 `all` 的跨 workspace 范围扩展和提示。
5. 增加选择后的状态刷新和竞态处理。
6. 增加敏感元数据脱敏。
7. 支持 list-only/preview/command 三种输出模式。

### 静态验收清单

- [ ] 只从最新 `aiw cxs` 结果解析选择。
- [ ] Session ID、Task、branch、worktree、lease 和 OpenSpec 状态可交叉验证。
- [ ] `original_cwd` 存在且属于允许的 workspace。
- [ ] `all` 会明确提示跨 workspace 范围。
- [ ] 选择后会处理 Session 状态变化和 lease 竞态。
- [ ] title、alias、cwd 和预览内容有脱敏规则。
- [ ] 只生成可复制命令，不自动执行或启动 nested Codex。
- [ ] 不自动修改 AIW 生命周期或清理资源。

### 当前结论

`resume-ext` 已经有很好的安全核心，尤其是“不在当前会话中嵌套 resume”和“只基于最新列表选择”。建议增量补充 AIW 身份校验、路径安全、跨 workspace 提示、竞态刷新和敏感元数据脱敏。

本章节只记录评审建议，尚未修改 `skills/resume-ext/SKILL.md`。

## `setup-matt-pocock-skills`

目标文件：`skills/setup-matt-pocock-skills/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\engineering\setup-matt-pocock-skills\SKILL.md`

### 当前评价

质量约为 8.5/10。当前版本是对参考版本很有价值的 AIW/OpenSpec 适配：移除了第二套 issue tracker 配置，把 AIW Task 和 OpenSpec 作为本地权威，并保留了 domain docs、triage labels 和 Agent instructions 的初始化能力。主要需要补充的是幂等性、写入前差异检查、配置提交边界和 setup 完成后的验证。

### 参考版本中值得保留的做法

1. 先探索仓库状态，再做配置决策，不假设项目从空白开始。
2. 按 issue tracker、triage labels、domain docs 分段处理，每段一次确认。
3. 先展示 Agent skills block 和文档草稿，再写入文件。
4. 选择已有的 `CLAUDE.md` 或 `AGENTS.md`，不同时创建两个指令文件。
5. 更新已有 `## Agent skills` block 而不是重复追加，保留用户周围内容。
6. 根据真正的 monorepo 边界决定 single-context 或 multi-context，而不是默认拆分。
7. 完成后报告写了哪些文件、哪些 Skill 会读取这些配置。

### 当前版本已经做好的地方

- 正确把 AIW/OpenSpec ownership split 作为核心配置。
- 明确不创建独立 issue-tracker 配置，符合当前项目权威边界。
- 将 `.scratch` 降级为 legacy data，避免继续建立第二套 tracker。
- 只在 triage 已安装时配置默认 label。
- 对 `AGENTS.md`/`CLAUDE.md` 选择、重复 block 和周围内容保护有明确规则。
- 明确 setup 不创建 Task、worktree、外部 Issue，也不运行测试。
- 使用 `docs/agents/work-management.md`、`domain.md` 和 triage labels 作为可维护配置文件。

### 需要重新增加或调整的内容

#### 1. 增加幂等 setup 检查

重复运行 setup 不应重复创建或破坏用户配置。建议在写入前比较：

- ownership split 是否已存在且内容是否冲突；
- `docs/agents/domain.md`、triage labels 和 Agent skills block 是否已存在；
- 当前配置是否由本 skill 管理，还是用户手工维护；
- `AGENTS.md` 与 `CLAUDE.md` 是否存在互斥或重复规则。

冲突时停止并展示 diff，不要静默覆盖。

#### 2. 明确 setup 的所有写入都需要用户确认

当前已经要求展示草稿，但应把确认门槛写成硬规则：

- 探索阶段只读；
- 写入前展示将修改的文件、diff 和文件不存在时的创建动作；
- 用户确认后才写入；
- 用户拒绝某一部分时，只跳过该部分，不擅自替代方案；
- 没有 `AGENTS.md`/`CLAUDE.md` 时不能自行选择创建哪一个。

#### 3. 增加 runtime capability 的验证边界

`aiw help --json`、Task 状态和 skill 安装状态可以验证，但：

- 只执行低成本、只读命令；
- 不因为 setup 而安装 AIW/OpenSpec、下载依赖或运行测试；
- command 不可用时记录 `%%` 风险和人工后续动作；
- 不把“配置文件写好了”误报成“AIW runtime 已可用”。

#### 4. 增加 domain docs 与 AIW/OpenSpec 的关系

建议明确：

- `CONTEXT.md`/`domain.md` 保存领域词汇和消费者规则，不保存实现 spec；
- ADR 只记录满足 domain-modeling 三条件的长期决策；
- OpenSpec proposal/design/spec/tasks 继续由 OpenSpec 管理；
- setup 只配置路径和规则，不为每个项目自动生成空 CONTEXT、空 ADR 或空 change。

#### 5. 明确配置文件的提交和 worktree 继承

setup 会修改仓库级文档和 Agent instructions。建议：

- 用户确认并写入后，将配置变更作为当前分支的普通提交候选；
- 如果 setup 在已有 AIW Task 中执行，应保持与 Task/worktree 绑定，不复制到另一个 worktree；
- 如果 setup 是首次工程流之前的 bootstrap，则在创建后续 Task worktree 前提交配置，使新 worktree 直接继承；
- setup 不自动 archive、merge 或清理。

#### 6. 增加跨平台和权限失败处理

- 识别 Windows/macOS/Linux 路径和临时目录差异；
- 无法写入 docs/agents 或指令文件时，报告具体路径和最小人工操作；
- 不用更高权限或替代 shell 绕过失败；
- 保留已成功写入的文件列表，避免报告“全部成功”。

#### 7. 完成标准要包含实际验证

setup 完成至少要报告：

- ownership split 文件是否存在且内容一致；
- Agent skills block 位于哪个文件；
- triage 是否安装、label 是否配置；
- domain layout 是 single-context 还是 multi-context；
- AIW/OpenSpec runtime 是否可发现；
- 未完成项和 `%%` 风险。

### 不应恢复或引入的内容

- 不应恢复 GitHub/GitLab/local markdown issue tracker 作为 AIW/OpenSpec 的第二权威系统。
- 不应把 `.scratch` 重新设为 canonical work。
- 不应无确认覆盖 AGENTS、CLAUDE、docs/agents 或用户自定义 labels。
- 不应同时创建 AGENTS.md 和 CLAUDE.md。
- 不应安装依赖、下载工具、运行测试或创建 Task/worktree。
- 不应自动创建空 CONTEXT、空 ADR 或空 OpenSpec change。
- 不应把 setup 成功等同于 runtime capability 已验证。

### 下一步修正规格

后续修正 `skills/setup-matt-pocock-skills/SKILL.md` 时，建议：

1. 保留 AIW/OpenSpec ownership split 和三类配置探索。
2. 增加幂等性、冲突 diff 和写入前用户确认。
3. 增加低成本 runtime 验证和失败报告。
4. 明确 domain docs、ADR、OpenSpec 的职责边界。
5. 明确配置提交和后续 worktree 继承规则。
6. 增加跨平台路径、权限和部分成功处理。
7. 将 setup 完成报告扩展为可检查的验证摘要。

### 静态验收清单

- [ ] 重复运行不会重复或覆盖用户配置。
- [ ] 写入前展示文件和 diff，并等待确认。
- [ ] 冲突配置会停止，不静默覆盖。
- [ ] 没有 AGENTS/CLAUDE 时不会自行选择创建文件。
- [ ] 不创建第二套 issue tracker 或 `.scratch` canonical work。
- [ ] runtime 验证只读、低成本，失败会记录 `%%`。
- [ ] domain docs、ADR 和 OpenSpec 的职责清楚。
- [ ] 配置变更可被后续 Task worktree 继承。
- [ ] 不自动运行测试、创建 Task、archive、merge 或清理。
- [ ] 完成报告列出成功、跳过、失败和风险。

### 当前结论

`setup-matt-pocock-skills` 已经完成了关键的 AIW/OpenSpec 迁移，建议做增量增强，不应恢复旧的 issue tracker 初始化模型。最重要的改进是幂等、确认、冲突保护、配置提交继承和真实验证报告。

本章节只记录评审建议，尚未修改 `skills/setup-matt-pocock-skills/SKILL.md`。

## `teach`

目标文件：`skills/teach/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\productivity\teach\SKILL.md`

### 当前评价

质量约为 8/10。当前版本的学习模型很完整：mission、资源、lesson、reference、learning records、retrieval practice、spacing、interleaving 和 zone of proximal development 都有明确位置。主要风险是它默认在当前 workspace 大量写入 HTML/Markdown 文件，并要求寻找高可信资源，但没有定义网络授权、目录隔离、文件幂等和学习阶段完成标准。

### 参考版本中值得保留的做法

1. 将学习视为跨多个 Session 的 stateful request，而不是一次性回答。
2. 用 `MISSION.md` 约束学习目标，用 learning records 记录已掌握内容和下一步难度。
3. 区分 knowledge acquisition、skill practice 和 wisdom/community interaction。
4. 使用 retrieval practice、spacing 和 interleaving 对抗“当下会做但长期不会”的 fluency illusion。
5. 每个 lesson 只教一个紧凑、可完成的能力，并与 mission 和 zone of proximal development 绑定。
6. 将 lesson 与长期 reference documents 分离，reference 保存可重复查阅的压缩知识。
7. 使用 assets 复用样式、quiz、模拟器和图表组件，避免每课重复实现。
8. 要求可信资源、引用和 primary source，提升知识的可追溯性。

### 当前版本已经做好的地方

- 目录和文件职责清晰，格式文件外置，支持渐进披露。
- 明确 mission 变化需要用户确认，避免悄悄重写学习目标。
- 明确每个 lesson 的编号和独立性。
- 已要求 lesson 链接其他 lesson/reference，并包含 follow-up 提示。
- 已把共享 assets 作为默认复用点。
- `disable-model-invocation: true` 合理，教学是用户主动选择的长流程。

### 需要重新增加或调整的内容

#### 1. 增加教学 workspace 隔离规则

当前目录可能是代码仓库，直接创建 `MISSION.md`、`lessons/`、`assets/` 和 `learning-records/` 可能污染工程。建议：

- 先确认当前目录是否是专用 teaching workspace；
- 如果是代码仓库，默认使用明确的教学子目录或用户指定目录；
- 写入前展示将创建的目录和文件；
- 不覆盖同名的工程文件、README、CONTEXT 或 Agent instructions；
- 教学资料不得进入生产代码的构建、测试或发布路径。

#### 2. 增加资源获取和网络授权边界

“Never trust parametric knowledge” 是质量目标，但不代表可以无条件联网。建议：

- 先读取已有 `RESOURCES.md` 和本地资料；
- 需要外部资源时说明搜索范围、来源类型和预期成本；
- 网络访问需要用户授权或当前环境明确允许；
- 优先官方文档、教材、论文和高信誉机构；
- 记录 URL、作者、发布日期/版本和访问时间；
- 无法核查时标记 `%%`，不要把记忆内容伪装成引用。

#### 3. 增加课程阶段和完成标准

每个 Session 应明确：

- 本次 lesson 的一个学习目标；
- 先备知识和当前 zone；
- 学习内容；
- 用户需要完成的 retrieval/practice；
- 反馈结果；
- 下一次的 spacing/interleaving 安排；
- 是否写入 learning record 或 reference。

不能只因为生成了漂亮 HTML 就宣告学会；应以用户能解释、回忆或完成练习作为 lesson 完成信号。

#### 4. 增加文件幂等和编号安全

- 创建 lesson/reference/learning record 前检查编号和文件名；
- 不覆盖已有课程或学习记录；
- 如果已有同编号文件，重新计算下一个编号或请求用户选择；
- 更新 mission 前必须确认；
- 更新 `NOTES.md` 时区分用户偏好和临时 agent notes，不把两者混在一起。

#### 5. 限制 HTML/浏览器副作用

- HTML 默认离线可读，不依赖 CDN 才能学习核心内容；
- 运行交互式 quiz/simulator 前说明命令和资源；
- 不自动打开 GUI 或启动长时间 server；
- 不使用真实敏感数据作为练习材料；
- lesson 失败或部分生成时保留已生成文件并报告状态，不重复覆盖。

#### 6. 调整“wisdom/community”规则

寻找论坛、subreddit、课程或社区可能需要网络和外部推荐，也可能引入隐私/安全风险。建议：

- 只有用户明确需要现实社区/实践建议时才进入该分支；
- 先询问地区、语言、预算和隐私偏好；
- 不把社区建议当作事实或课程权威来源；
- 高风险领域（医疗、金融、法律、身体训练）优先转向专业来源和安全提示。

#### 7. 避免过度格式化成为 no-op

“beautiful”“Tufte”和 quiz 答案等长等要求有价值，但不应压过学习目标。建议：

- 内容、练习和反馈优先于视觉装饰；
- 等长答案只在不泄露答案且不牺牲自然语言时使用；
- 复杂主题可以使用短 lesson 序列，不把所有内容塞进一页 HTML；
- 视觉组件只在确实改善理解时新增。

#### 8. 与 AIW/OpenSpec 代码工作流隔离

- teach 默认不是工程 Task，不创建 AIW Task/worktree；
- 不把 learning records 当作 ADR 或 OpenSpec spec；
- 如果学习结果转化为代码需求，由用户另行进入 `/to-spec`；
- 教学资料的提交、归档和清理应与代码 Task 分开，不触发自动完成协议。

### 不应恢复或引入的内容

- 不应无确认在代码仓库根目录生成教学目录和文件。
- 不应无授权联网或把模型记忆当作可信资源。
- 不应覆盖 mission、lesson、reference 或 learning record。
- 不应把 HTML 外观、引用数量或文件数量当作学习完成证明。
- 不应自动打开 GUI、启动 server、运行长时间练习或访问真实数据。
- 不应把教学文档混入 AIW/OpenSpec 工程生命周期。

### 下一步修正规格

后续修正 `skills/teach/SKILL.md` 时，建议：

1. 增加教学 workspace 目录确认和代码仓库隔离。
2. 增加外部资源获取、引用记录和网络授权规则。
3. 增加 lesson/session 的学习完成标准。
4. 增加文件编号、幂等和不覆盖规则。
5. 增加 HTML、GUI、server 和交互练习的副作用边界。
6. 收紧 wisdom/community 分支的触发和安全条件。
7. 明确 teach 与 AIW/OpenSpec 工程生命周期完全分离。

### 静态验收清单

- [ ] 当前目录是否为 teaching workspace 有明确判断。
- [ ] 代码仓库不会被默认写入教学文件。
- [ ] 网络资源需要授权或有明确环境许可。
- [ ] 资源有 URL、作者、版本/日期和访问记录。
- [ ] lesson 有目标、练习、反馈和完成标准。
- [ ] 文件编号和更新操作幂等，不覆盖已有内容。
- [ ] HTML/GUI/server/真实数据副作用有边界。
- [ ] mission 变更需要用户确认。
- [ ] 不把教学资料当作 ADR、OpenSpec 或 AIW Task。

### 当前结论

`teach` 的教学设计理念较完整，建议重点补充 workspace 隔离、资源获取授权、文件幂等和基于学习表现的完成标准。它应继续保持独立的多 Session 学习系统，不与工程 Task 生命周期混合。

本章节只记录评审建议，尚未修改 `skills/teach/SKILL.md`。

## `to-spec`

目标文件：`skills/to-spec/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\engineering\to-spec\SKILL.md`

### 当前评价

质量约为 9/10。当前版本已经把参考版本从“写完 spec 并发布到 issue tracker”正确迁移为 AIW/OpenSpec 工作流，并补上了 `task.toml`、Task ID、parent branch、规范文件集合、提交继承和 `/implement` 解析门禁。它是目前评审过的核心工程 Skill 中适配最完整的之一。

### 参考版本中值得保留的做法

1. 不通过 interview 重新发明需求，而是综合当前对话和代码库理解。
2. 先用 domain glossary 和 ADR，再写 proposal/design/spec。
3. 先识别最高层 test seam，优先复用既有 seam，减少跨代码库的测试切口。
4. 使用结构化模板区分 Problem、Solution、User Stories、Implementation Decisions、Testing Decisions 和 Out of Scope。
5. 不把具体文件路径和易过时代码片段塞进用户-facing spec，长期决策放进 design。
6. 将 prototype 中真正表达决策的状态机、reducer 或 schema 片段作为例外保留。

### 当前版本已经做好的地方

- 已移除默认 issue tracker 发布和 triage label 操作，符合 AIW/OpenSpec 本地权威规则。
- 已要求先解析 AIW Task 和 matching OpenSpec change。
- 已要求 `task.toml` 与 change directory、Task ID 一致。
- 已要求 proposal、design、capability spec、tasks 和 AIW mapping 同时存在。
- 已明确 `/implement` 使用 `tasks.md`，避免实现内容只留在 proposal/design。
- 已明确不创建 implementation worktree，先把规范提交到当前分支，再由 AIW worktree 继承。
- 已明确只同步 AIW 粗粒度 title/goal/planning status，不覆盖 OpenSpec-owned content。
- 已明确规范阶段不运行测试，并使用 `%%` 记录未决风险。

### 需要重新增加或调整的内容

#### 1. 将 parent branch 设为完成闭环的必需字段

当前表述是 parent branch “when those fields exist”，但新的自动完成协议需要可靠合并目标。建议：

- 创建/解析 Task 时必须记录 `parent_branch`；
- `parent_branch` 必须在创建 Task worktree 前验证存在并与当前分支匹配；
- `/implement` 完成时只合并到这个记录的 branch，不根据当前 checkout 猜测；
- 缺失或冲突时停止，不创建 worktree 或进入实现。

#### 2. 增加规范阶段的确认边界

to-spec 虽然不是完整 interview，但以下内容仍需用户确认或已有上下文明确：

- test seam；
- unresolved `%%` risks；
- out-of-scope；
- 是否是新 Task/change 还是已有 Task/change；
- 是否需要拆成多个独立 lifecycle。

不要在缺少关键决策时用“synthesis”掩盖不确定性，也不要因为要求 no interview 而跳过必要的确认点。

#### 3. 增加需求与规范完整性门禁

报告完成前应检查：

- proposal 的问题、目标、范围与 user stories 一致；
- design 中的决定能解释 capability spec 的要求；
- spec 中每个 requirement 有可观察 scenario；
- tasks.md 的每个 item 都有前置条件、验收标准和可执行边界；
- testing decisions 与选择的 seam 一致；
- out-of-scope 没有被 tasks 或 spec 间接重新带入。

不是文件存在就代表规范可实现；内容之间必须相互一致。

#### 4. 明确 capability spec 的命名和变更方式

建议补充：

- 复用已有 capability 时更新对应 change delta，不创建同义 capability；
- 新 capability 使用领域 glossary 命名，不使用临时文件名；
- stable `openspec/specs/` 与 change-scoped `openspec/changes/<task-id>/specs/` 的关系要明确；
- 发现现有 spec 与新要求冲突时停止并记录，而不是覆盖 stable spec。

#### 5. 增加自动提交的安全边界

当前已允许规范完成后提交，但建议明确提交前检查：

- 只 stage 当前 Task/change 相关的规范文件；
- 不包含用户无关的工作区修改、secrets 或生成产物；
- 提交信息包含 Task/change ID 和 spec 阶段；
- commit 成功后再交给 `aiw wt` 创建 worktree；
- commit 失败时保留文件并报告，不手工复制到 worktree。

#### 6. 增加无 AIW/OpenSpec backend 时的停止策略

- AIW 能力缺失时不手工伪造 `task.toml`、branch 或 worktree 映射；
- OpenSpec CLI 缺失但本地 artifacts 可安全更新时，明确报告未执行的 CLI 操作；
- Task/change 冲突时停止，不覆盖任一方；
- backend 返回部分成功时记录已创建资源和恢复动作。

#### 7. 明确与 `to-tickets` 的边界

- to-spec 负责规范和高层任务方向，不负责最终 tracer-bullet 切片；
- `tasks.md` 可以包含粗粒度 checklist，但实现前由 `/to-tickets` 细化为垂直切片；
- `/implement` 只选择一个已确认的 checklist item；
- 不因为 to-spec 生成了 tasks.md 就跳过必要的 ticket granularity review。

#### 8. 明确输出完成后的下一步

完成后应报告：

- Task ID、change ID、parent branch；
- 生成/更新的文件；
- commit ID；
- 是否需要 `/to-tickets`；
- 可开始的 checklist item；
- 未解决的 `%%` 风险；
- 未运行的测试/构建/CLI 验证。

### 不应恢复或引入的内容

- 不应恢复默认发布 issue tracker 或自动加 triage label。
- 不应创建 `.scratch` 或第二套 Task/change 记录。
- 不应在规范阶段创建 implementation worktree 或手工复制 artifacts。
- 不应把文件存在、模板填满或用户故事数量当作规范正确的证明。
- 不应猜测缺失的 domain、owner、权限、数据契约或 rollback 决策。
- 不应自动运行测试、构建、code review、archive、merge 或 worktree 清理。
- 不应在 Task/change 冲突时覆盖 AIW 或 OpenSpec 内容。

### 下一步修正规格

后续修正 `skills/to-spec/SKILL.md` 时，建议：

1. 将 `parent_branch` 设为必需且可验证的 Task 元数据。
2. 增加规范内容一致性和 requirement scenario 门禁。
3. 明确 capability 命名、stable spec 与 change delta 的关系。
4. 增加提交前 staging、secret 排除和 commit 结果报告。
5. 明确 backend 缺失、部分成功和冲突时的停止策略。
6. 强化 to-spec、to-tickets、implement 的职责边界。
7. 保留现有 AIW/OpenSpec artifact set 和自动 worktree 继承流程。

### 静态验收清单

- [ ] Task ID、change ID、目录名和 `task.toml.id` 一致。
- [ ] `parent_branch` 存在、可验证并记录在 Task metadata。
- [ ] proposal/design/spec/tasks 内容相互一致。
- [ ] capability requirements 有可观察 scenarios。
- [ ] tasks.md item 可被 `/implement` 解析，必要时可由 `/to-tickets` 细化。
- [ ] test seam、out-of-scope 和 `%%` 风险已确认或明确记录。
- [ ] 规范提交只包含当前 Task/change 相关文件。
- [ ] commit 成功后才创建 implementation worktree。
- [ ] backend 冲突或失败不会覆盖或手工复制 artifacts。
- [ ] 未运行的命令和未解决风险已报告。

### 当前结论

`to-spec` 已经很好地完成了从第三方 issue-tracker 流程到 AIW/OpenSpec 的迁移。建议只增强 parent branch 必需性、内容一致性门禁、提交安全和 to-tickets 交接，不需要回退到外部 tracker 发布模式。

本章节只记录评审建议，尚未修改 `skills/to-spec/SKILL.md`。

## `to-tickets`

目标文件：`skills/to-tickets/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\engineering\to-tickets\SKILL.md`

### 当前评价

质量约为 8.5/10。当前版本正确地把第三方“发布到 issue tracker 的 ticket”迁移为同一 AIW Task 下的 OpenSpec `tasks.md` checklist，保留了 tracer-bullet vertical slices、用户确认和 wide refactor 的 expand-contract 思路。主要需要补充的是自动提交、任务完成标准、独立 change 模板清理和依赖/并行语义。

### 参考版本中值得保留的做法

1. 用 tracer-bullet vertical slice，而不是按 schema/API/UI/test 做 horizontal slice。
2. 每个 ticket 必须从用户行为角度描述一个可演示或可验证的完整路径。
3. 每个 ticket 都有明确的前置依赖，blocker-first 排序。
4. 先展示 breakdown，让用户确认粒度、依赖和 merge/split，再写入。
5. 对 wide refactor 使用 expand-contract，并按 blast radius 分批迁移。
6. 将具体文件路径和代码片段留在实现上下文，避免 ticket 文案快速过期。

### 当前版本已经做好的地方

- 已要求先解析 AIW Task 和 matching OpenSpec change。
- 已禁止 `.scratch` ticket hierarchy 和第二套任务追踪系统。
- 已把普通实现 slice 留在同一 AIW Task/worktree/lifecycle 下。
- 已将普通依赖表达为 `tasks.md` 的顺序和 prerequisites，而不是虚构外部 blocking links。
- 已要求用户批准 breakdown 后才写入 `tasks.md`。
- 已明确不创建 worktree、不运行测试，并保留 independent lifecycle 需要用户批准的边界。
- 已保留 wide refactor 的 expand-contract 例外。

### 需要重新增加或调整的内容

#### 1. 增加 tasks.md 的结构和完成标准

每个 implementation item 至少应有：

- 唯一编号和短标题；
- 用户可观察的 What to build；
- 明确 prerequisites；
- 可验证的 acceptance criteria；
- 与 proposal/design/spec 的关联；
- 相关测试 seam 或验证方式；
- `%%` 风险和未决依赖。

“可开始”不等于“足够实现”；`/implement` 必须能只读取该 item 和相关 OpenSpec artifacts 完成工作。

#### 2. 增加任务写入后的自动提交规则

当前新 AIW 流程要求规范和 checklist 在创建 worktree 前存在于当前分支。建议明确：

- 用户批准 breakdown 后，只更新当前 change 的 `tasks.md` 和必要的粗粒度 AIW progress；
- 写入后执行静态一致性检查；
- 只提交当前 Task/change 相关的 tasks 文件，排除无关修改和 secrets；
- commit 成功后才交给 `aiw wt` 创建/解析实现 worktree；
- 不手工复制 `tasks.md` 到 worktree。

#### 3. 恢复“frontier”概念，但不恢复外部 tracker

参考版本的 frontier 很有用：当一个 ticket 的 prerequisites 都完成，它才可被 implement。当前可以改写为：

- `tasks.md` 中明确哪些 item 是 `ready`、`blocked` 或 `done`；
- `/implement` 只允许选择 ready item；
- 同一 AIW Task 可以有多个 ready item，但是否并行必须遵守 worktree/lease 规则；
- 不把 frontier 变成第二个 tracker 或外部 blocking link。

#### 4. 清理 independent-change 模板的旧 tracker 语义

当前模板仍然包含 `Parent` 和 independent change 概念，容易让 agent 想回到 issue tracker。建议改成：

- Parent AIW Task/change；
- 独立 lifecycle 的理由：独立 worktree、delivery、archive 或权限边界；
- 用户批准记录；
- 新 Task 的 `task.toml`、`parent_branch` 和 OpenSpec change 关联。

普通 ticket 不应使用 independent-change 模板。

#### 5. 明确 ticket 数量和粒度预算

建议：

- 默认生成 3–7 个 vertical slices；
- 少于 3 个时说明为什么不能进一步切分；
- 多于 7 个时先按 capability 或 milestone 分组，避免一次确认几十项；
- 每个 item 应适合一个 fresh context，但不能为了短而拆成没有用户价值的水平步骤；
- 用户确认“太粗/太细”后重新生成，不在未确认状态直接写入。

这些是参考值，不是绝对限制；大型工作可以由 `/wayfinder` 或多个 change 处理。

#### 6. 修正“CI green”假设

wide refactor 说明中提到 CI green，但当前项目默认不运行测试/构建。建议改成：

- ticket 的 acceptance criteria 可以要求“在授权的 focused checks 下通过”；
- 不声称 CI green，除非实际运行并记录了结果；
- expand-contract 的价值是降低破坏范围，不等于自动获得验证证据。

#### 7. 明确与 to-spec/implement/完成协议的交接

- `to-spec` 提供 requirements/design；
- `to-tickets` 将其转换为可执行 vertical slices；
- `/implement` 一次选择一个 ready item；
- 实现完成后更新 item、TODO/Verification 和 `%%` 风险；
- 全部 item 完成后由 `/implement` 触发 sync、archive、merge 和 cleanup；
- `to-tickets` 本身不触发完成协议。

### 不应恢复或引入的内容

- 不应恢复 `.scratch`、GitHub/Linear blocking links 或独立 issue 文件作为默认实现清单。
- 不应在用户批准 breakdown 前写入 `tasks.md`。
- 不应把一个 horizontal layer 当作一个用户可验证 ticket。
- 不应为共享 lifecycle 的普通 slice 创建独立 AIW Task。
- 不应自动创建 worktree、运行测试或进入实现。
- 不应使用 `task.md` 替代 `tasks.md`。
- 不应宣称未运行的 CI/test/build 已通过。

### 下一步修正规格

后续修正 `skills/to-tickets/SKILL.md` 时，建议：

1. 增加 tasks.md item 的最小字段和 `/implement` 可执行标准。
2. 增加用户批准后的静态检查和自动提交/继承规则。
3. 恢复 ready/blocked/done frontier 语义，但保持在同一 tasks.md 中。
4. 将 independent-change 模板改为 AIW/OpenSpec lifecycle 语义。
5. 给 ticket 数量和 fresh-context 粒度提供参考预算。
6. 删除未运行验证的 CI green 暗示。
7. 明确 to-spec、to-tickets、implement 和完成协议的交接。

### 静态验收清单

- [ ] 每个 item 有用户行为、prerequisites 和 acceptance criteria。
- [ ] 每个 item 能被 `/implement` 独立解析和执行。
- [ ] 用户确认 breakdown 后才写入 tasks.md。
- [ ] tasks.md 更新在 worktree 创建前提交。
- [ ] ready/blocked/done frontier 语义清楚。
- [ ] 普通 slice 仍归属于同一 AIW Task/change。
- [ ] independent change 只有独立 lifecycle 且用户批准时才创建。
- [ ] ticket 数量和粒度有合理参考范围。
- [ ] 不运行的测试、构建和 CI 不会被宣称为通过。
- [ ] 全部 item 完成后由 implement 进入自动完成协议。

### 当前结论

`to-tickets` 已经正确完成了从外部 tracker ticket 到 OpenSpec checklist 的迁移。建议补充 item 完成标准、自动提交、frontier 状态、独立 lifecycle 模板和粒度预算，不应恢复第二套 ticket tracker。

本章节只记录评审建议，尚未修改 `skills/to-tickets/SKILL.md`。

## `triage`

### 评审对象

- 本地版本：`skills/triage/SKILL.md`
- 参考版本：`D:\03_projects\third-part\skills\skills\engineering\triage\SKILL.md`
- 参考版本与本地版本基本一致，主要差异不在文字内容，而在当前仓库已经采用 AIW + OpenSpec 作为任务与规范的权威来源。

### 当前评价

当前质量约为 **6.5–7/10**。参考实现的状态机和人工确认边界比较清楚，但它默认存在外部 issue tracker，并且包含标签、评论、关闭 issue、写入 `.out-of-scope` 等外部或持久化操作。这些假设与本仓库的 AIW 工作流不完全一致；如果直接使用，容易产生第二套任务系统，或在未明确授权时改变外部状态。

### 参考版本值得保留的做法

1. 用 `bug` / `enhancement` 作为分类，用 `needs-triage`、`needs-info`、`ready-for-agent`、`ready-for-human`、`wontfix` 作为状态，形成有限状态机。
2. 所有写入 tracker 的评论使用固定的 AI 免责声明，便于区分自动生成内容与人工意见。
3. 先按发现、具体 issue/PR、冗余检查、既往拒绝记录分桶，再决定后续路径，避免立即修改状态。
4. 对具体 issue/PR 先提出建议并等待维护者确认；同时保留已确认后的 quick state override，适合处理明确的人工指令。
5. 将“agent brief”和“out-of-scope”分开，避免把实现说明和范围拒绝理由混在同一文档里。
6. 对 bug 和 PR 都要求证据，而不是仅凭标题或主观判断；已实现、重复和超范围事项有明确处理路径。

### 当前版本已有的优点

- 已明确分类与状态不能重复、每次操作要保留理由，并且有重复检查与历史拒绝检查。
- 已要求在建议后等待维护者确认，避免把 triage 建议误当成最终决策。
- 已覆盖 issue、PR、发现项和已实现项，基本流程完整。

### 需要调整的内容

#### 1. 明确外部 tracker 不是默认权威来源

`setup-matt-pocock-skills` 已明确避免引入独立 issue tracker。triage 不应默认要求 tracker 配置，也不应把 tracker 标签当作 AIW Task 的替代品。应支持以下两种清晰路径：

- 没有外部 tracker：对用户输入、发现项或本地文档生成只读 triage brief，并路由到 `/office-hours-finance`、`/to-spec` 或 `/implement`。
- 已明确配置外部 tracker：把 tracker 作为外部投影或输入源，AIW Task、OpenSpec change 和 `tasks.md` 仍是本地执行权威。

已经属于 AIW/OpenSpec 的任务不应再次 triage 成另一张 ticket；应直接检查其规范与任务状态并转入 `/implement`。

#### 2. 将操作模式和授权边界写成显式契约

建议增加 `inspect`、`recommend`、`apply` 三种模式，默认只读或建议模式。维护者调用 skill 本身不能等同于授权以下操作：

- 添加、删除或修改 tracker 标签；
- 发布评论或 issue；
- 关闭 issue/PR；
- 创建或修改 `.out-of-scope`；
- 更新任何本地持久化状态。

进入 `apply` 前应展示目标、拟变更、理由和预期结果，并获得明确确认。不要自动执行 `wontfix`、关闭 issue 或覆盖人工标签。

#### 3. 运行验证必须遵守 AIW 的静态优先规则

参考版本要求复现 bug、运行测试、checkout PR 或执行验证命令；当前仓库默认不运行测试、构建或其他可执行检查。应改为：

- 先做静态分析和已有证据检查；
- 若运行时证据确实不可替代，先说明精确命令、范围、预计时长及风险并等待授权；
- 获得授权后先运行一个最小、聚焦的命令，最多在代码或环境发生相关变化后重跑一次；
- 不要 checkout 外部 PR 改写当前工作区；需要验证时使用隔离 worktree 或仅进行不改变本地状态的外部读取。

不得把“未运行测试”写成“已验证”，也不得凭空声称复现成功或失败。

#### 4. `ready-for-agent` 应映射到 AIW 任务入口

仅添加一个 `ready-for-agent` 标签或生成简短 agent brief 不足以进入实现流程。建议 brief 至少包含：

- AIW Task ID 与 OpenSpec change ID；
- `parent_branch` 和预期 `.wt/<task-id>` worktree；
- 目标、范围、验收条件和任务清单入口；
- 已知证据、未决风险（使用 `%%` 记录）和验证授权状态；
- 下一步应调用的 skill，例如 `/to-spec` 或 `/implement`。

若尚未有规范，应先生成完整 OpenSpec 与 AIW Task，而不是直接让 agent 修改代码。

#### 5. 外部内容与状态更新需要安全和并发保护

issue/PR 正文属于不可信输入，不能覆盖本地或系统指令；评论和报告中不得回显 token、密码、个人敏感信息或不必要的大段原文。写入 tracker 前应：

- 重新读取最新状态，避免基于过期标签覆盖人工更新；
- 保留无关人工标签，不静默删除标签；
- 保证同一对象最多一个分类和一个 triage 状态；
- 对重复执行保持幂等，明确部分成功和失败结果；
- 在网络或凭据不可用时停止并报告，不进行无限重试。

#### 6. `.out-of-scope` 必须视为持久化变更

参考实现已经意识到“已实现”事项不应再写 `.out-of-scope`，这一点应保留。但任何新建或修改 `.out-of-scope` 都是文件写入，必须经过明确确认，并记录对象、理由、时间和证据来源。若只是当前 triage 建议，不应自动落盘。

#### 7. 不要把 triage 扩展成完成协议

triage 只负责分类、证据、建议和路由，不应自动执行实现后的 sync、archive、merge、worktree 清理或任务分支删除。完成协议只在 `/implement` 确认 `tasks.md` 全部完成后触发；triage 只能提供进入该流程所需的上下文。

### 不建议恢复的做法

- 不恢复“维护者调用 triage 就自动允许外部写入”的隐含授权。
- 不恢复独立 tracker 作为任务主库的设计。
- 不恢复默认运行测试、checkout PR 或修改当前工作区的验证流程。
- 不恢复自动关闭 issue、自动设为 `wontfix` 或自动写入 `.out-of-scope`。
- 不让 triage 直接创建分支、worktree、提交或归档任务。

### 后续修正建议

下一步可在 `skills/triage/SKILL.md` 中补充：

1. 默认只读的 `inspect/recommend/apply` 模式与逐项确认边界。
2. 无 tracker 时的本地 triage brief；有 tracker 时的“外部投影、AIW/OpenSpec 为权威”规则。
3. `ready-for-agent` 到 AIW Task/OpenSpec 的字段映射。
4. 静态优先、运行命令授权、隔离 worktree 与不虚构验证结果。
5. 外部不可信内容、敏感信息、并发更新、幂等和失败恢复规则。
6. `%%` 未决事项、完成标准和下一步路由。

### 静态验收清单

- [ ] 能在没有外部 tracker 的情况下完成只读 triage。
- [ ] 不会把 tracker 当作 AIW Task 或 OpenSpec 的替代品。
- [ ] 外部写入、关闭、标签修改和 `.out-of-scope` 均需要明确授权。
- [ ] 默认不运行测试或构建；运行时验证有精确授权和最小范围。
- [ ] `ready-for-agent` 输出包含 Task、change、分支、worktree 和验收上下文。
- [ ] 不会因 triage 触发 sync、archive、merge 或清理。
- [ ] 报告能区分事实、推断、未决风险和未执行的验证。

### 结论

参考版本的状态机、证据分桶、人工确认和 AI 免责声明值得保留；但它需要从“外部 issue tracker 操作器”调整为“AIW/OpenSpec 上下文中的只读优先 triage 与受控外部投影”。最高优先级是消除第二套任务权威、补足写操作授权、增加运行验证限制，并把 `ready-for-agent` 连接到真实的 Task/OpenSpec 入口。

本章节只记录评审建议，尚未修改 `skills/triage/SKILL.md`。

## `wayfinder`

### 评审对象

- 本地版本：`skills/wayfinder/SKILL.md`
- 参考版本：`D:\03_projects\third-part\skills\skills\engineering\wayfinder\SKILL.md`
- 本地版本只在开头补充了 AIW Task、worktree、测试和最多两个 research sub-agent 的约束，主体流程仍基本沿用参考版本。

### 当前评价

当前质量约为 **6/10**。参考版本对“巨大且决策路径尚不清晰的工作”有很好的思维模型：先命名 destination，再维护 frontier，用 fog of war 控制未知范围，并把 research、prototype、grilling、task 区分开。但它的持久化模型完全建立在 issue tracker 上，且包含创建 child issue、加标签、分配、设置 blocking、发表评论、关闭 issue 和 throwaway branch。这些操作与本仓库的 AIW/OpenSpec 工作流不兼容，不能仅靠文件开头的约束修复。

### 参考版本值得保留的做法

1. 明确“wayfinder 负责规划和消除决策迷雾，不直接冲向实现”，默认产出决定而不是代码交付物。
2. 先定义 destination，再决定 scope；这是避免在错误目标上提前拆解任务的有效顺序。
3. 使用 `Decisions so far`、`Not yet specified`、`Out of scope` 三类信息区分已决策、尚不可精确表述的未知项和明确排除项。
4. 区分“现在能精确描述的问题”和“只能粗略感知的问题”，避免把 fog 过早切成伪精确的 tickets。
5. 把 decision ticket 限定为一个可在单次 Session 内处理的问题，并用 frontier 表达当前可推进边界。
6. 区分 HITL 与 AFK，并区分 research、prototype、grilling、task 四种决策前置工作。
7. 采用 breadth-first 的初始制图方式，避免一开始在单个局部问题上过度深入。
8. 一次 Session 最多解决一个 decision ticket，便于控制上下文和降低决策漂移。
9. 地图只做索引、不重复存储 ticket 细节，以及“先创建、后建立依赖”的两阶段思路，都是不错的信息组织原则。

### 当前版本已有的优点

- 已明确 `disable-model-invocation: true`，不会被普通请求隐式触发。
- 已声明读取 `skills/work-management.md`，并把 AIW Task、worktree 和测试规则置于正文之前。
- 已把 research sub-agent 默认上限收敛到两个，避免参考版本按 research ticket 数量无限扩展。
- 仍保留了 destination、frontier、fog、scope、HITL/AFK 和“一个 Session 处理一个 ticket”的核心结构。

### 需要调整的内容

#### 1. 将 issue tracker 地图改为 OpenSpec 内的规划产物

当前项目不应默认创建第二套 issue tracker 或 local-markdown tracker。`wayfinder:map`、child issue、tracker label、assignment、blocking edge、resolution comment 和关闭 issue 都会形成独立任务生命周期。

建议改为：

- 在尚未形成明确 change 时，使用一个受控的规划文档，例如 `docs/wayfinding/<effort>.md`，只记录 destination、decision log、fog、out-of-scope 和 frontier；
- 一旦 destination 和主要决策清晰，回流到 `to-spec`，由 `openspec/changes/<change-id>/` 的 proposal、design、spec、tasks 承担正式规范；
- 不在规划阶段创建 AIW implementation Task，除非用户已经确认要进入可执行工作流；
- 不把 `docs/wayfinding` 变成新的 ticket 系统，不维护独立的 ticket ID、状态机和任务完成状态。

如果未来确实接入外部 tracker，应明确它只是外部投影，AIW Task 和 OpenSpec 仍是执行权威；没有显式配置时不能假设 tracker 存在。

#### 2. 明确 wayfinder 与 AIW Task、OpenSpec 的边界

建议把边界写成可检查的路由：

- 目标和关键决策仍不清晰：wayfinder 只产出规划上下文和决策问题；
- 目标清晰但规范尚未形成：转入 `/to-spec`；
- 规范已有、任务清单已有：转入 `/to-tickets` 或 `/implement`；
- 已有 AIW Task：wayfinder 不重复创建任务，不改变其 branch/worktree；
- 任务完成后的 sync、archive、merge、worktree 清理由 `/implement` 完成，wayfinder 不参与。

wayfinder 的完成条件应是“路线足够清晰，可以生成 OpenSpec”，而不是“所有 issue 都已关闭”。

#### 3. 删除 throwaway branch 假设，改用受控资源边界

参考版本要求 research ticket 使用 `research/<name>` throwaway branch。当前工作流要求通过 AIW 管理 Task、branch 和 worktree，不能在 skill 内创建平行分支体系。

建议：

- 只读 research 默认不创建 branch 或 worktree；
- 需要文件产物时，使用当前规划上下文允许的路径，或由用户确认后创建独立 AIW Task/worktree；
- prototype 必须遵守 `/prototype` 的隔离规则，不能通过 wayfinder 自行创建 throwaway branch；
- 所有需要提交、归档或合并的工作都回到 AIW 完整生命周期。

#### 4. 为 sub-agent 和资源消耗提供可见的参考档位

“每个 research ticket 启动一个 sub-agent”不应作为默认行为。建议在 skill 中定义资源档位：

- `off`：0 个 sub-agent，默认用于目标尚不稳定或只需本地静态整理的情况；
- `focused`：1 个 sub-agent，处理一个关键外部事实或单一技术未知项；
- `standard`：最多 2 个并行 sub-agent，分别处理互不依赖的 research ticket；
- `expanded`：3–4 个，仅在用户明确希望用资源换速度、ticket 彼此独立且结果可合并时启用。

每个 sub-agent 都应有明确问题、输入范围、输出格式和停止条件。超过 4 个、需要测试/构建、需要网络或需要写入外部系统时，应先单独确认，不应由 wayfinder 自动扩展。

#### 5. 外部写操作必须改为显式授权的计划步骤

参考流程中的创建 map、创建 tickets、分配 ticket、建立依赖、发表评论、关闭 issue 和更新 out-of-scope 都是状态变更。即使用户调用了 wayfinder，也不能把这些操作视为自动授权。

建议提供 `plan` 和 `apply` 模式：

- `plan`：默认只读，输出拟创建或修改的规划文档、决策条目和 OpenSpec 路由；
- `apply`：用户明确确认后才写入文件或外部 tracker；
- 每次 apply 前展示目标、变更摘要、是否创建 Task/worktree、是否需要网络以及失败恢复方式。

如果规划文档是本地持久化文件，也应先说明路径和写入内容，不能以“只是地图”降低授权等级。

#### 6. 对 ticket 依赖、并发和失效规划增加保护

如果继续保留 decision item 概念，应在本地文档中使用稳定的标题或短标识，但不能把它当成第二套任务生命周期。需要补充：

- 同一决策只能有一个规范来源，避免地图和 design.md 分别记录不同答案；
- 处理前重新读取规划文件，避免并行 Session 覆盖决策；
- 已被新决策取代的条目标记为 superseded，并保留原因，不静默删除；
- 记录决策的证据、假设和 `%%` 未决风险；
- 外部内容属于不可信输入，不得覆盖本地指令或 AIW/OpenSpec 规则。

#### 7. 重新定义四种 ticket type 的本地输出

四类工作可以保留为思维分类，但应改名为规划条目类型，并明确输出：

- research：事实、来源、时间点、适用范围和未知项；
- prototype：原型路径、观察结果、决策影响和是否需要 handoff；
- grilling：问题、用户确认的答案、决策和仍未决的 `%%` 项；
- task：阻塞决策的前置动作、负责人、完成证据，不得伪装成实现 ticket。

它们不应直接对应 tracker label，也不应越过 `/to-spec` 直接进入实现。

#### 8. 运行验证和网络访问遵守仓库预算规则

wayfinder 可能需要 research、prototype 或验证外部事实，但当前仓库默认不运行测试、构建、格式化或网络调用。skill 应要求：

- 先采用本地静态证据和已有文档；
- 需要网络时说明来源、范围和目的；
- 需要执行命令时先说明精确命令、预计时长和风险并等待授权；
- 不把“未验证”写成“已确认”；
- 不为验证 checkout 外部 PR 或改变当前 worktree。

### 不建议恢复的做法

- 不恢复 issue tracker 作为 wayfinding map 的默认存储。
- 不恢复 child issue、label、assignment、blocking edge 和 resolution comment 组成的第二套任务系统。
- 不恢复 `research/<name>` throwaway branch。
- 不让 wayfinder 自动创建 Task、branch、worktree、commit 或完成归档。
- 不让 wayfinder 直接执行实现；路线清晰后应回流 `/to-spec` 或 `/implement`。
- 不按 research ticket 数量无上限启动 sub-agents。

### 后续修正建议

下一步可在 `skills/wayfinder/SKILL.md` 中补充：

1. 本地规划文档与 OpenSpec 的承载边界，移除 tracker-first 假设。
2. `plan/apply` 模式和本地/外部写入授权。
3. destination、frontier、fog、decision log 的最小文档结构。
4. 从 wayfinder 到 `/to-spec`、`/to-tickets`、`/implement` 的路由表。
5. `off/focused/standard/expanded` 资源档位及 sub-agent 停止条件。
6. AIW Task、branch、worktree 和完成协议的禁止越权规则。
7. `%%` 未决事项、来源、假设、并发冲突和规划失效处理。

### 静态验收清单

- [ ] 没有外部 tracker 时仍能完成只读 wayfinding。
- [ ] 不会创建第二套 issue/ticket 生命周期。
- [ ] destination、fog、frontier、decisions 和 out-of-scope 有清晰的本地承载方式。
- [ ] 能明确路由到 `/to-spec`、`/to-tickets` 或 `/implement`。
- [ ] 不会创建 throwaway branch，也不会绕过 AIW 创建 Task/worktree。
- [ ] sub-agent 资源档位可见，默认不会按 ticket 数量无限扩展。
- [ ] 网络、测试、构建和其他可执行验证均受授权规则约束。
- [ ] 不会触发 sync、archive、merge 或 worktree 清理。

### 结论

wayfinder 的决策地图、fog of war、frontier 和 HITL/AFK 分类是参考版本中最有价值的部分，应保留为规划方法；但 issue tracker-first 的实现方式必须重构为 AIW/OpenSpec 兼容的本地规划与受控路由。最高优先级是移除第二套任务系统、消除 throwaway branch、定义资源档位，并确保路线清晰后只能回流到正式 OpenSpec/AIW 工作流。

本章节只记录评审建议，尚未修改 `skills/wayfinder/SKILL.md`。

## `writing-great-skills`

### 评审对象

- 本地版本：`skills/writing-great-skills/SKILL.md`
- 配套参考：`skills/writing-great-skills/GLOSSARY.md`
- `.agents` 镜像：`.agents/skills/writing-great-skills/SKILL.md`
- 第三方参考目录未提供可独立确认的不同版本；现有内容与本地版本的主体原则一致。

### 当前评价

当前质量约为 **8.5/10**。这是目前 `skills/` 中较强的元技能之一：它提供了 predictability、context load、cognitive load、information hierarchy、progressive disclosure、leading words、completion criterion、pruning 和 failure modes 等可复用词汇，能帮助评审者从“感觉写得不错”转向检查技能是否可预测、可维护和可完成。

它本身是 reference skill，不是一个要逐步执行的业务流程，因此没有大量步骤并不构成缺陷。主要不足是缺少针对本仓库的落地验收协议：如何评审一个具体 skill、如何处理 AIW/OpenSpec 约束、何时只写 review、何时才允许修改，以及如何控制昂贵的验证动作。

### 值得保留的做法

1. 把根目标定义为 predictability，而不是要求每次输出完全相同；这个区分准确且适合指导 agent 工作流。
2. 对 model-invoked 与 user-invoked 做 context load / cognitive load 的成本分析，能解释 `disable-model-invocation` 的实际取舍。
3. 用 information hierarchy 区分 in-skill step、in-skill reference 和 external reference，并把 progressive disclosure 与 context pointer 联系起来。
4. 强调每个步骤必须有可检查的 completion criterion，直接针对 premature completion。
5. 用 granularity、single source of truth、relevance、no-op 和 co-location 约束 skill 的维护成本。
6. leading words 的方法可以把反复解释的行为压缩为稳定的共享词汇，适合本仓库多个 skill 共用 AIW/OpenSpec 术语。
7. 对 duplication、sediment、sprawl、no-op、negation 等 failure modes 给出了诊断方向，而不是只要求“写得更清楚”。
8. `GLOSSARY.md` 作为渐进披露的配套 reference，避免把完整定义全部塞进 `SKILL.md`。

### 当前版本已有的优点

- `disable-model-invocation: true` 与它作为人工调用的评审参考相匹配，避免每轮自动增加上下文负担。
- `GLOSSARY.md` 确实存在，`SKILL.md` 的 context pointer 没有指向缺失文件。
- 内容以原则和诊断词汇为主，和具体工程 skill 解耦，适合被 `ask-matt`、review 流程和后续 skill 评审共同引用。
- 没有默认调用测试、构建、网络或 sub-agent，不会触发当前仓库的资源成本问题。

### 需要调整的内容

#### 1. 增加“如何评审一个 skill”的最小协议

当前文档解释了什么是好 skill，但没有明确本仓库正在执行的评审动作。建议增加一个很短的评审协议，避免不同评审者只挑自己熟悉的原则：

1. 读取目标 skill、其直接引用的 reference，以及对应的 AIW/OpenSpec 约束。
2. 判断 skill 类型：步骤型、参考型或混合型；不要用步骤型标准误判纯 reference skill。
3. 检查 invocation、信息层级、分支、完成标准、单一事实来源和失败路径。
4. 对仓库特有的 Task、branch、worktree、Session、OpenSpec、验证和授权边界做兼容性检查。
5. 输出保留项、缺口、不应恢复项、修正建议和静态验收清单。

每一项评审都应明确“只记录 review”还是“已实际修改”，避免建议被误认为已经落地。

#### 2. 将仓库级约束纳入信息层级，但不要复制整份规则

该 skill 的通用原则没有提到 AIW/OpenSpec。建议只增加一个 context pointer，指向 `skills/work-management.md` 和仓库 `AGENTS.md`，并说明它们覆盖：

- Task、branch、worktree 和 Session 的权威关系；
- OpenSpec artifact 与 `tasks.md` 的规范边界；
- 测试、构建、网络和 sub-agent 的授权规则；
- 实现完成后的 sync、archive、merge 和清理协议。

不要把完整仓库规则复制进本 skill，否则会制造第二个 single source of truth，也会增加 sediment 和 sprawl。

#### 3. 补充“静态评审优先”的成本边界

writing skill 的评审动作通常可以静态完成。建议明确：

- 默认只读取和分析 skill 文本及直接引用；
- 不默认运行脚本、测试、构建、格式化或网络检索；
- 需要验证链接、渲染、工具调用或跨文件一致性时，先说明精确命令和成本并取得授权；
- sub-agent 不是默认的质量证明，只有在用户明确希望提高并行度时才按资源档位启用。

这样可以让“legwork”表示必要的阅读与推理，而不是无界的执行成本。

#### 4. 用正向规则指导 negation 的修正

文档正确指出 negation 可能强化被禁止行为，但仓库目前有许多必须表达的硬边界，例如不得自动测试、不得创建第二套 tracker、不得越过 AIW 生命周期。建议给出写法模板：

- 先写期望行为：“默认只做静态检查，并报告未执行的运行验证”；
- 再写不可违反的守门条件：“只有获得明确授权后才可运行指定命令”；
- 同时提供失败后的替代路径，而不是只写禁止事项。

这能把一般提示原则转化为适合安全工程流程的正向 guardrail。

#### 5. 明确 description 评审不能脱离实际调用入口

当前 description 原则关注 leading word 和 trigger pruning，这是正确的；还应补充仓库级检查：

- user-invoked skill 应保持简短的人类入口描述，不要写成自动触发清单；
- model-invoked skill 才需要完整的触发分支，并且描述必须与实际可用路径一致；
- description 不应声称不存在的 skill、tracker、脚本或工具；
- 路由器应承担跨 skill 选择，单个 skill 不要复制整个路由表。

这对当前 `ask-matt`、`wayfinder`、`triage` 等存在路由关系的 skill 尤其重要。

#### 6. 为 completion criterion 增加“证据”维度

文档要求 completion criterion 可检查且必要时 exhaustive，建议再明确完成证据的三种状态：

- `observed`：从文件、diff 或明确命令结果中直接观察到；
- `inferred`：基于静态分析推断，必须标注为推断；
- `not-run` / `unknown`：没有执行或无法判断，不能写成完成。

这样能与仓库要求的“不得声称未运行测试已通过”一致，也能减少 skill review 中的过早结论。

### 不建议恢复的做法

- 不把本 skill 改成自动运行所有 skill 检查的脚本型流程。
- 不把 `AGENTS.md`、`work-management.md` 或 OpenSpec 规则完整复制进来。
- 不默认启动 sub-agents、测试、构建或网络检索来证明 skill 质量。
- 不把所有原则都提升到 `SKILL.md`，破坏现有 `GLOSSARY.md` 的 progressive disclosure。
- 不为了追求“更完整”继续堆积没有可检查行为影响的通用写作建议。

### 后续修正建议

下一步可在 `skills/writing-great-skills/SKILL.md` 中增加一个简短的仓库适配段：

1. 评审 skill 前读取直接引用和 AIW/OpenSpec 约束。
2. 区分步骤型、参考型和混合型 skill 的验收方式。
3. 默认静态评审，运行验证必须有明确授权和精确范围。
4. completion criterion 同时记录证据状态。
5. 对 user/model invocation、router、外部引用和资源成本增加仓库级检查。

### 静态验收清单

- [ ] 能判断目标 skill 是步骤型、参考型还是混合型。
- [ ] 能检查 invocation、information hierarchy、progressive disclosure 和 single source of truth。
- [ ] 能发现 premature completion、duplication、sediment、sprawl、no-op 和 negation 风险。
- [ ] 能检查 AIW Task、OpenSpec、worktree、授权和验证边界，而不复制其完整规则。
- [ ] 评审默认不运行测试、构建、网络或 sub-agent。
- [ ] completion criterion 能区分 observed、inferred 和 not-run/unknown。
- [ ] 评审报告明确哪些是建议，哪些已经实际修改。
- [ ] `GLOSSARY.md` 的 context pointer 保持有效。

### 结论

`writing-great-skills` 的通用方法质量较高，尤其是 predictability、information hierarchy、completion criterion 和 pruning。需要增加的不是更多通用写作理论，而是一个很薄的 AIW/OpenSpec 适配层：规定如何评审仓库内 skill、如何控制验证成本、如何标记证据，以及如何避免把仓库规则复制成第二套事实来源。

本章节只记录评审建议，尚未修改 `skills/writing-great-skills/SKILL.md`。

## `ask-matt`

目标文件：`skills/ask-matt/SKILL.md`

参考文件：`D:\03_projects\third-part\skills\skills\engineering\ask-matt\SKILL.md`

### 当前评价

质量约为 7/10。当前版本已经正确表达 AIW/OpenSpec 的所有权、主工程流程和执行边界，但相比参考版本，路由模型和上下文管理信息有所削弱。

### 参考版本中值得恢复的做法

1. 用明确的层次组织路由：
   - Main flow：正常的 idea → ship 流程；
   - On-ramps：bug、原始请求、大型模糊项目等特殊入口；
   - Vocabulary underneath：`domain-modeling`、`codebase-design` 等词汇层；
   - Crossing sessions：`handoff`、`compact`；
   - Standalone：不进入主工程流程的独立能力。

2. 增加主流程分支判断：
   - 当前问题是否能在一个 Session 内完成；
   - 是否需要多个实现 Session；
   - 是否需要 prototype 或可运行结果来回答设计问题；
   - 是否需要先 handoff 到新 Session。

3. 明确多 Session 工作的判断：
   - 小型、已有明确实现项的工作可直接进入 `/implement`；
   - 需要多个 Session、多个实现切片或阶段交接时，进入 `/to-spec`；
   - 已有规范但没有垂直切片时，进入 `/to-tickets`；
   - 已有明确 checklist item 时，进入 `/implement`。

4. 恢复上下文卫生规则：
   - `grill-with-docs`、`to-spec`、`to-tickets` 尽量保持在同一个 Session；
   - 每个实现 ticket 可在新的 Session 中执行；
   - `/handoff` 用于跨 Session 保留完整工作上下文；
   - `/compact` 用于同一 Session 内的阶段性压缩，不应在阶段中途随意使用。

5. 明确 `/triage` 的边界：
   - 外部进入的 bug、原始需求和请求使用 `/triage`；
   - `/to-tickets` 已生成的 implementation item 直接进入 `/implement`；
   - 不对自己刚生成的 ticket 重复执行 `/triage`。

6. 明确 `/wayfinder` 的触发条件：
   - 不是“功能很大”就使用 `/wayfinder`；
   - 只有当关键决策未确定、无法看见通往目标的路径时才使用；
   - `/wayfinder` 产出决策，不直接产出实现；
   - 决策清晰后必须回到 `/to-spec`，再进入 `/to-tickets` 和 `/implement`。

7. 对需要运行结果的问题保留 prototype 分支：
   - UI、状态模型或业务逻辑无法仅凭文字确定时，使用 `/prototype`；
   - 通过 `/handoff` 将问题带入 prototype Session，再把结论带回原工作流；
   - prototype 的结果应作为设计输入，而不是绕过 AIW Task 直接实现。

### 不应原样恢复的内容

1. `.scratch/<feature>/issues/` ticket 层级不应恢复。当前项目由 AIW 管理 Task 生命周期，由 OpenSpec 管理 `tasks.md`，不能建立第二套任务追踪系统。

2. 不应恢复“`/implement` 自动驱动 `/tdd` 和 `/code-review`”。当前应改成显式、有限度的 opt-in：
   - `/implement` 不自动调用 `/tdd`；
   - `/implement` 不自动调用 `/code-review`；
   - 用户可以单独授权一次 focused TDD 或一次 focused code review；
   - focused TDD 只处理一个 seam、一个行为和一个最小测试命令；
   - focused code review 只检查当前选定 diff/路径和明确的 Standards + Spec 范围；
   - 测试、构建、检查需要用户明确授权；
   - 扩大测试、构建或 review 范围必须再次获得授权；
   - 不自动 commit、archive 或删除 worktree/branch。

3. 应区分提交与生命周期清理：AIW 工作流可以在规范完成后提交当前分支的
   OpenSpec/Task artifacts，并在实现完成后提交 Task 分支的实现结果。这样
   aiw wt 创建的分支和 worktree 能直接继承规范文件，不需要手工复制。
   完成协议成功后，Task 可自动 archive，代码合并回记录的父分支，并清理
   worktree 和 Task branch；任一步失败或出现冲突，都必须保留现场供恢复。
   创建 Task worktree 前必须在 Task metadata 中记录 `parent_branch`，完成时
   只合并到该已校验的父分支，不能根据当前 checkout 推断目标。

### 建议的修正方向

将 `ask-matt` 重构为真正的路由器，但保留当前 AIW/OpenSpec 边界：

1. 增加“根据当前状态选择 Skill”的决策表；
2. 明确“仅咨询”和“开始执行”两种模式；仅咨询时不创建 Task、不修改文件；
3. 为每个路由结果定义完成标准；
4. 增加 AIW Task 与 OpenSpec change 不匹配时的停止条件；
5. 将 `work-management.md` 已定义的通用规则压缩为引用，避免重复；
6. 对新增的 `/grill-me`、`/prototype`、`/diagnosing-bugs`、`/research` 等 Skill 建立可执行路由；
7. 保证所有推荐的 Skill 都存在于当前 `./skills` 目录，避免产生死链接。

### 当前结论

`ask-matt` 不需要完全重写。建议保留现有 AIW/OpenSpec 生命周期规则，恢复参考版本的“主流程、入口、词汇层、Session 管理、独立能力”结构，并补充明确的分支条件和完成标准。

本章节只记录评审建议，尚未修改 `skills/ask-matt/SKILL.md`。

### 下一步修正规格

后续修正可以只依据本节执行，不要求重新读取第三方参考文件。

#### 目标结构

将 `skills/ask-matt/SKILL.md` 组织为以下章节，顺序保持稳定：

1. `# Ask Matt`：说明这是一个用户主动调用的路由器；
2. `## Routing rule`：先判断用户是在咨询路线，还是已经授权开始工作；
3. `## Main flow`：`grill-with-docs` → AIW Task → `to-spec` → `to-tickets` → `implement`；
4. `## Main-flow branches`：单 Session、多 Session、prototype、handoff；
5. `## On-ramps`：`triage`、`diagnosing-bugs`、`wayfinder`；
6. `## Codebase health`：`improve-codebase-architecture` 和 `codebase-design`；
7. `## Vocabulary layer`：`domain-modeling`、`codebase-design`；
8. `## Crossing sessions`：`handoff` 和 `compact`；
9. `## Standalone skills`：`grill-me`、`prototype`、`research`、`teach`、`writing-great-skills`；
10. `## AIW/OpenSpec boundaries`：只保留本路由器特有的生命周期边界，其他细节引用 `skills/work-management.md`。

#### 必须实现的路由判断

修正后的 skill 至少要能回答以下情况：

| 当前情况 | 推荐路由 | 关键条件 |
| --- | --- | --- |
| 想法或需求仍然模糊 | `/grill-with-docs` | 先形成可追踪的上下文和决策 |
| 外部进入的原始 bug/需求 | `/triage` | 不对已经生成的 implementation item 重复 triage |
| 困难、间歇性或难以复现的缺陷 | `/diagnosing-bugs` | 先建立针对该问题的紧反馈回路 |
| 关键决策尚未确定、路径不可见 | `/wayfinder` | 产出决策，之后回到 `/to-spec` |
| 需要可运行结果回答设计问题 | `/prototype` | 必要时通过 `/handoff` 往返 |
| 当前 Session 可完成且已有明确实现项 | `/implement` | 必须解析 AIW Task、OpenSpec change 和 worktree |
| 需要多 Session 或多个实现切片 | `/to-spec` | 随后进入 `/to-tickets`，再逐项 `/implement` |
| 已有规范但没有实现切片 | `/to-tickets` | 生成可执行的垂直 checklist |
| 需要继续另一 Session 的工作 | `/handoff` | 保留可恢复的上下文和状态 |
| 只是压缩当前 Session | `/compact` | 不跨 Session，不替代 handoff |
| 代码库健康或模块边界问题 | `/improve-codebase-architecture` 或 `/codebase-design` | 选定方向后再进入主流程 |

#### AIW/OpenSpec 不可违反的约束

- 仅咨询路由时，不创建 AIW Task、不创建 OpenSpec change、不创建 worktree。
- 开始工程工作时，使用一个 AIW Task 关联一个 OpenSpec change。
- AIW 管理 Task、branch、worktree、Session 和 handoff 状态。
- OpenSpec 管理 proposal、design、capability specs 和 `tasks.md`。
- 不创建 `.scratch` 或其他第二套任务追踪系统。
- `/implement` 必须在 AIW 管理的 Task worktree 中执行。
- `/implement` 不自动调用 `/tdd`、`/code-review`，不自动运行测试，不自动 commit。
- 不推荐当前 `skills` 目录中不存在的 Skill；如依赖缺失，应明确报告缺失能力。
- `/wayfinder`、`prototype` 等规划或探索能力的产出必须回流到 AIW/OpenSpec 主流程，不能绕过 Task 生命周期直接实现。

#### 每个路由步骤的完成标准

- 路由器最终推荐一个主 Skill，并说明推荐理由。
- 如果缺少前置条件，明确指出缺少什么以及应先执行什么。
- 如果存在多个 AIW Task，停止并请求 Task ID。
- 如果 AIW Task 与 OpenSpec change 不匹配，停止，不自行创建平行记录。
- 如果只是咨询，明确说明不会修改文件或创建生命周期资源。
- 推荐的 Skill 必须能在当前 `./skills` 中找到对应目录和 `SKILL.md`。

#### TDD 与 code review 的有限度路由

`ask-matt` 后续修正时，不应把这两个 Skill 从工作流中删除，而应把它们作为显式 opt-in 的质量步骤：

- 用户明确要求 test-first、TDD 或 red-green-refactor：推荐 `/tdd`，默认采用 `focused` 模式；
- 用户明确要求检查当前改动：推荐 `/code-review`，默认只审查当前选定 diff 或路径；
- 用户只要求普通实现：不自动追加 `/tdd` 或 `/code-review`；
- 用户要求有限度质量保障：可以推荐一次 focused TDD，或一次 focused code review，但先说明范围、命令/检查内容和预期成本；
- 用户要求扩大范围时，再单独确认是否运行更大测试集、构建或完整 review。

这里的“有限度”是路由策略，不是禁止策略：保留能力，但防止每个实现任务默认触发昂贵的附加流程。

#### 修正后的静态验收清单

- [ ] skill 仍保留 `disable-model-invocation: true`。
- [ ] 主流程、入口、词汇层、Session 管理和独立能力均有独立章节。
- [ ] 单 Session 与多 Session 有明确分支。
- [ ] `triage`、`diagnosing-bugs`、`wayfinder` 有不同触发条件。
- [ ] `prototype` 与 `handoff` 的关系有说明。
- [ ] 没有恢复 `.scratch`、自动 TDD、自动 review 等过时约定；自动提交和完成闭环应遵循 AIW 的提交、sync、archive、merge、清理顺序。
- [ ] 没有重复抄写完整的 `work-management.md`。
- [ ] 所有被推荐的 Skill 都存在于当前 `./skills`。
- [ ] 路由结果和停止条件可由 agent 检查，而不是依赖主观判断。
- [ ] 修改后与本章节的 AIW/OpenSpec 约束一致。
