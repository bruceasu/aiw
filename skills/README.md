# AIW Skills 使用手册

本目录收集了一组面向 Codex/AIW 的工作技能。它们不是一组互相独立的命令，而是把一次工作拆成不同阶段：理解问题、形成方案、建立规格、拆分任务、实现代码、审查交付，以及维护会话和协作状态。

本手册回答四个问题：

1. 每个 skill 解决什么问题？
2. 什么时候应该使用它？
3. 它通常产生什么结果？
4. 它应该和哪些 skill 配合？

## 一、先理解三类 Skill

### 1. 思考与设计类

这些 skill 帮助我们把模糊想法变成可讨论、可决策的内容：

`ask-matt`、`grilling`、`grill-with-docs`、`office-hours-finance`、`business-review`、`metrics-review`、`eng-review-finance`、`domain-modeling`、`codebase-design`、`improve-codebase-architecture`、`autoplan-finance`。

### 2. OpenSpec 与实现类

这些 skill 把结论变成工程产物并推进代码：

`to-spec`、`to-tickets`、`implement`、`tdd`、`code-review`、`release-review`、`publish-github-issue`。

### 3. 协作、维护与学习类

这些 skill 处理任务路由、会话连续性、Git 状态、环境配置和知识传递：

`handoff`、`resume-ext`、`resolving-merge-conflicts`、`triage`、`wayfinder`、`setup-matt-pocock-skills`、`teach`、`writing-great-skills`。

## 二、最常用的完整路线

一般工程需求可以按下面的顺序推进：

```text
模糊想法
  -> ask-matt
  -> grilling / grill-with-docs
  -> office-hours-finance（如果是财务/运营问题）
  -> business-review
  -> metrics-review
  -> eng-review-finance
  -> to-spec
  -> to-tickets
  -> implement / tdd
  -> code-review
  -> release-review
  -> publish-github-issue（仅在明确需要时）
```

不需要每次都使用全部 skill。上面的路线是“完整覆盖”，实际工作应从当前最缺失的阶段开始。

`autoplan-finance` 是财务场景的编排器：当你希望一次得到完整的 `PLAN.md`，可以直接使用它；如果只想解决一个局部问题，就使用对应的单项 review。

## 三、逐个 Skill 教程

### `ask-matt`：不知道下一步用什么

作用：根据当前情况选择合适的工程 Skill 或 AIW/OpenSpec 流程。

什么时候用：需求很短、上下文混乱，或者你不确定应该先澄清、先写 spec，还是直接实现。

典型结果：得到下一步建议和相应的工作顺序。它是路由器，不负责替代后续 skill 的完整工作。

配合方式：通常作为入口，之后转到 `grill-with-docs`、`office-hours-finance`、`to-spec` 或 `implement`。

### `grilling`：逐题压力测试想法

作用：通过一次只问一个问题的方式，逐步暴露假设、边界和决策依赖。

什么时候用：你想验证一个计划是否真的想清楚，或者担心遗漏了关键场景。

典型结果：一组经过追问的决策、边界和待解决问题。它偏“访谈”，不会自动承担完整的文档编排。

配合方式：之后可用 `domain-modeling` 固化术语和决策，再用 `to-spec` 形成 OpenSpec。

### `grill-with-docs`：压力测试并同步形成文档

作用：在 `grilling` 的访谈过程中，同时沉淀 ADR 和 glossary 等设计文档。

什么时候用：讨论还不成熟，而且术语、架构选择或关键决策需要留下记录。

典型结果：澄清后的设计理解，以及随讨论更新的架构决策和领域词汇。

配合方式：它通常位于 `to-spec` 之前；如果是财务需求，可再交给 `office-hours-finance` 或 `autoplan-finance`。

### `office-hours-finance`：澄清财务/运营问题

作用：识别真实业务问题、利益相关者、决策流程、范围和未知项。

什么时候用：有人提出“做一个报表、后台、风险看板或运营工具”，但还没有说明谁使用、做什么决定、成功是什么。

典型结果：问题定义、用户/角色、决策流程、范围缩减建议和未知项。

配合方式：通常先于 `business-review`；它负责把问题说清楚，不负责最终批准项目。

### `business-review`：判断是否值得做

作用：从业务价值、成本、风险和替代方案判断 `APPROVE`、`REDUCE` 或 `HOLD`。

什么时候用：需要决定某个需求、报表、工作流或平台能力是否值得投入。

典型结果：业务决策及其理由、范围调整和继续推进的前提。

配合方式：通常接在 `office-hours-finance` 后面，并把结论交给 `metrics-review` 或 `eng-review-finance`。

### `metrics-review`：定义和审核指标

作用：明确指标名称、公式、来源、口径、时间窗口、刷新频率、负责人和一致性风险。

什么时候用：要做 KPI、财务报表、经营看板、风险指标，或发现不同系统的数字对不上。

典型结果：指标定义表、数据来源映射、口径冲突和治理责任。

配合方式：通常在业务价值确认后使用，之后交给 `eng-review-finance` 设计数据流和实现方式。

### `eng-review-finance`：审核技术设计

作用：检查系统边界、数据流、模块职责、权限、审计、失败模式、可观测性和测试策略。

什么时候用：财务或运营需求已经明确，需要确认技术方案是否可构建、可审计、可恢复。

典型结果：技术设计评审意见、风险清单、数据和权限方案、测试与监控要求。

配合方式：通常接在 `metrics-review` 后面；接近上线时再用 `release-review`。

### `release-review`：审核是否可以上线

作用：作为发布门禁，检查 schema/migration、数据影响、指标、权限、审计、回滚、监控和运维准备。

什么时候用：实现基本完成，准备部署或发布，而不是在需求刚开始时使用。

典型结果：发布风险、阻塞项、上线前检查项和是否具备发布条件的结论。

配合方式：它是审核，不是定义需求；通常位于 `eng-review-finance` 和 `code-review` 之后。

### `autoplan-finance`：编排完整财务计划

作用：把 intake、业务价值、指标、工程设计和发布准备组织成一份完整的 `PLAN.md`。

什么时候用：你要规划一个财务后台、运营系统、报表、风险看板或数据项目，并希望一次得到端到端计划。

典型结果：包含问题、范围、决策、指标、架构、风险和发布门槛的计划文档。

配合方式：它是编排器，不等于“直接写代码”。计划稳定后，使用 `to-spec` 转成 OpenSpec，再使用 `to-tickets` 和 `implement`。

### `domain-modeling`：建立领域模型

作用：统一术语，识别概念关系，记录 glossary 和架构决策，并主动用边界场景检验模型。

什么时候用：同一个词在不同人或系统中含义不同，或者设计决策需要成为长期共享知识。

典型结果：领域词汇、概念关系、决策记录和明确的边界案例。

配合方式：常与 `grilling`、`grill-with-docs`、`codebase-design` 和 `to-spec` 配合。

### `codebase-design`：改进模块和代码边界

作用：使用 deep module 等设计语言分析接口、职责、封装和可测试性。

什么时候用：不知道功能应该放在哪个模块，模块接口过于泄漏，或代码难以测试和被 AI 导航。

典型结果：模块边界建议、接口设计、职责调整和可测试性改进方向。

配合方式：可以在 `implement` 前用于设计，也可以在 `code-review` 中用于解释结构性问题。

### `improve-codebase-architecture`：发现架构深化机会

作用：扫描代码库，生成可视化 HTML 报告，再针对选中的机会进行深入追问。

什么时候用：面对大型或陌生代码库，想系统发现哪些模块值得重构，而不是只修一个局部 bug。

典型结果：架构机会报告，以及对某个机会的深入分析。

配合方式：通常先用它发现方向，再用 `codebase-design`、`grilling` 或 `to-spec` 固化具体改动。

### `to-spec`：把讨论变成 OpenSpec 变更

作用：综合当前对话和代码库理解，生成 OpenSpec change，包括 proposal/design/spec/tasks 等工件。

什么时候用：问题和关键方案已经讨论过，不需要再进行访谈，下一步是形成可实现规格。

典型结果：一个可审查、可实现、可追踪的 OpenSpec 变更目录。

配合方式：通常在 `grill-with-docs`、财务 review 或架构讨论之后使用；不要把它当成“从一句模糊需求直接猜完整方案”的工具。

### `to-tickets`：把方案拆成实现切片

作用：把计划、spec 或当前讨论拆成有顺序的 tracer-bullet 实现项。

什么时候用：方案已经存在，但 `tasks.md` 太粗，或者任务之间的依赖和验收条件不清楚。

典型结果：顺序明确、范围较小、可逐项完成的 OpenSpec tasks。

配合方式：通常在 `to-spec` 后、`implement` 前使用。

### `implement`：实现一个选定任务

作用：根据 AIW Task 和对应 OpenSpec change，完成一个明确的实现项。

什么时候用：已经知道要实现哪个 task，不应在这里重新发明需求或扩大范围。

典型结果：代码、必要的测试或文档更新，以及任务状态和验证记录。

配合方式：接在 `to-spec`/`to-tickets` 后；如果要求测试优先，可与 `tdd` 配合。

### `tdd`：用测试驱动实现

作用：按 red-green-refactor 循环，把一个可观察行为作为一个小实现闭环。

什么时候用：用户明确要求 TDD、测试优先，或功能边界适合先写行为测试。

典型结果：先失败的测试、最小实现、再整理代码的开发过程。

配合方式：通常是 `implement` 的实现方法，而不是替代 OpenSpec；先确定 task，再用 TDD 完成它。

### `code-review`：审核代码变更

作用：相对于指定基准检查变更，一方面看仓库规范，另一方面看是否符合原始 spec/需求。

什么时候用：需要 review 分支、PR 或某个固定点之后的工作时。

典型结果：按严重程度列出的 bug、回归风险、规范问题和 spec 偏差。

配合方式：通常在 `implement` 后、`release-review` 前使用。它是代码审核，不是发布审核。

### `publish-github-issue`：发布 OpenSpec 到 GitHub Issue

作用：将一个 OpenSpec change 发布成受管理的 GitHub Issue 投影，同时保留本地 OpenSpec 作为权威来源。

什么时候用：用户明确要求把某个 change 发布到 GitHub Issues。

典型结果：GitHub 上的 issue 投影和本地变更的关联信息。

配合方式：它不是普通的“创建 issue”技能，也不能替代 `to-spec`；只有在本地规格准备好且明确需要外部协作时使用。

### `handoff`：为下一次会话准备交接

作用：压缩当前对话，形成下一个 agent 可以继续使用的上下文交接文档。

什么时候用：会话过长、需要换 Thread，或希望把当前进度交给另一个 agent。

典型结果：当前任务、已完成工作、未完成工作、风险、下一步和关键文件的摘要。

配合方式：通常在暂停实现或切换会话前使用，可和 `resume-ext` 配合。

### `resume-ext`：寻找并恢复历史会话

作用：列出当前工作区的本地 Codex 会话，帮助选择并恢复某个会话。

什么时候用：你知道工作曾经在另一个会话中进行，但不想重新启动或丢失上下文。

典型结果：可选择的历史会话和恢复命令。

配合方式：`handoff` 负责留下交接，`resume-ext` 负责找回会话；两者解决的是不同方向的问题。

### `resolving-merge-conflicts`：解决 Git 合并冲突

作用：处理已经发生的 merge 或 rebase 冲突，理解双方意图后保留正确行为，并完成冲突流程。

什么时候用：Git 已经处于冲突状态，有冲突文件和未完成的 merge/rebase。

典型结果：冲突被解决、代码一致、相关检查完成，并可继续或完成 merge/rebase。

配合方式：它不是一般的代码修改或 review skill；只在 Git 冲突现场使用。

### `triage`：分诊 issue 和外部 PR

作用：按状态机处理 issue/PR，完成分类、验证、必要的追问，并写出 agent-ready brief。

什么时候用：收到很多 issue 或外部 PR，需要先判断类型、优先级、是否可执行和下一步负责人。

典型结果：分类标签、验证结论、待澄清问题和可直接执行的任务简报。

配合方式：triage 后可进入 `grilling`、`to-spec`、`to-tickets` 或 `implement`。

### `wayfinder`：规划跨多个会话的大型工作

作用：把超出单个 agent 会话容量的大任务拆成共享的决策 tickets，并逐个解决依赖。

什么时候用：大型迁移、跨模块重构、长期项目或多个 agent 需要协作的工作。

典型结果：一张通往目标的决策地图、依赖顺序和可独立推进的工作项。

配合方式：它位于大型项目的上游；每个决策或子任务可以再使用 `to-spec`、`implement` 和 `handoff`。

### `setup-matt-pocock-skills`：配置工程工作流

作用：配置 AIW Task 生命周期、OpenSpec 工件管理、triage 标签和领域文档约定。

什么时候用：首次在一个项目中启用这套工程 Skills，或需要修复/统一项目级工作流配置。

典型结果：项目中的任务、OpenSpec、标签和领域文档约定被建立或更新。

配合方式：通常只在项目初始化或工作流迁移时使用，不是日常每个需求都要运行的 skill。

### `teach`：学习一个概念或技能

作用：以持续、多次会话的方式教授用户一个概念或工作方法。

什么时候用：你不只是想让 agent 执行，而是想理解 OpenSpec、指标治理、架构设计或某个 Skill 的使用方式。

典型结果：循序渐进的解释、练习、反馈和后续学习上下文。

配合方式：可以用来学习本手册中的任意工作流，再回到实际 skill 执行任务。

### `writing-great-skills`：编写和改进 Skill

作用：提供编写 Skill 的原则、结构和可预测性标准。

什么时候用：要新建 skill、修改现有 skill，或发现 agent 经常误用某个 skill。

典型结果：更清晰的触发条件、步骤、边界、输出约定和验证方式。

配合方式：可和 `teach`、`code-review` 配合；如果要创建/更新 skill，应先读它的指导内容。

## 四、按目标选择

| 你的目标 | 首选 skill | 下一步 |
| --- | --- | --- |
| 不知道从哪里开始 | `ask-matt` | 按建议进入具体流程 |
| 澄清一个模糊想法 | `grilling` 或 `grill-with-docs` | `to-spec` 或领域/财务 review |
| 规划财务产品或报表 | `autoplan-finance` | `to-spec` |
| 判断业务上是否值得做 | `business-review` | `metrics-review` 或 `eng-review-finance` |
| 定义 KPI 或报表口径 | `metrics-review` | `eng-review-finance` |
| 设计模块和架构边界 | `codebase-design` | `to-spec` 或 `implement` |
| 扫描大型代码库的重构机会 | `improve-codebase-architecture` | 选定方向后深入设计 |
| 写 OpenSpec 变更 | `to-spec` | `to-tickets` |
| 拆细 tasks.md | `to-tickets` | `implement` |
| 开始实现一个明确任务 | `implement` | `code-review` |
| 测试优先开发 | `tdd` | 配合 `implement` |
| 审核代码变更 | `code-review` | 修复后再做发布审核 |
| 准备上线 | `release-review` | 发布或回滚 |
| 发布到 GitHub Issue | `publish-github-issue` | 仅在明确授权后执行 |
| 处理 merge/rebase 冲突 | `resolving-merge-conflicts` | 完成 Git 流程 |
| 分诊 issue/PR | `triage` | 进入对应工程流程 |
| 规划大型跨会话项目 | `wayfinder` | 拆分并逐项推进 |
| 切换或恢复会话 | `handoff` / `resume-ext` | 继续原任务 |
| 初始化项目工作流 | `setup-matt-pocock-skills` | 再开始日常流程 |
| 学习某个概念 | `teach` | 再使用对应 skill |
| 编写一个新 skill | `writing-great-skills` | 编写、审查、试用 |

## 五、三个完整示例

### 示例 A：一个模糊的财务报表需求

```text
ask-matt
  -> grill-with-docs（术语和关键决策仍不清楚）
  -> office-hours-finance（明确谁看、做什么决定）
  -> business-review（APPROVE / REDUCE / HOLD）
  -> metrics-review（统一公式和数据源）
  -> eng-review-finance（数据流、权限、审计）
  -> autoplan-finance 或 to-spec
  -> to-tickets
  -> implement
  -> code-review
  -> release-review
```

### 示例 B：一个已经讨论清楚的工程改动

```text
to-spec
  -> to-tickets
  -> implement
  -> tdd（如果采用测试优先）
  -> code-review
```

这里不需要重新运行 `grilling`。`to-spec` 的职责是综合已有结论，而不是再次采访用户。

### 示例 C：大型重构或长期迁移

```text
wayfinder
  -> triage / 决策 tickets
  -> codebase-design
  -> to-spec
  -> to-tickets
  -> implement
  -> handoff / resume-ext
  -> code-review
```

如果实现期间发生 merge/rebase 冲突，临时切换到 `resolving-merge-conflicts`，解决后再回到原任务。

## 六、常见误区

### “review”是不是用来定义内容？

通常不是。`business-review`、`metrics-review`、`eng-review-finance` 和 `release-review` 都会产出定义或建议，但它们的核心职责是对某一层内容进行检查、质疑和做门禁决策：

```text
office-hours-finance = 先把问题定义清楚
business-review      = 审核业务价值
metrics-review       = 审核指标口径
eng-review-finance   = 审核技术方案
release-review       = 审核上线条件
```

### `autoplan-finance` 和 `to-spec` 是否重复？

不重复。`autoplan-finance` 面向“我要一份完整的决策和执行计划”；`to-spec` 面向“这些结论已经够清楚了，请把它们转成 OpenSpec 变更”。前者是规划编排，后者是规格落地。

### `grill-with-docs` 和 `autoplan-finance` 是否重复？

不重复。`grill-with-docs` 深挖不确定性，并在过程中记录 ADR/glossary；`autoplan-finance` 汇总财务工作流的多个评审阶段，形成完整计划。前者偏探索，后者偏编排。

### 是否应该每次都把全部 skill 跑一遍？

不应该。Skill 是按缺口选择的工具：需求清楚就跳过澄清，非财务问题就跳过 finance review，没有上线计划就跳过 `release-review`。只有跨阶段、风险高或需要完整记录时，才使用完整路线。

### `publish-github-issue` 是否等同于普通发 Issue？

不是。它用于把 OpenSpec change 作为受管理的 GitHub Issue 投影发布，前提是用户明确要求外部发布。没有这个明确要求时，继续维护本地 OpenSpec 即可。

## 七、建议的工作习惯

1. 先判断当前缺的是“问题理解、方案决策、规格、实现还是审核”。
2. 一次只推进一个明确的阶段，不要在 `implement` 中重新讨论未决需求。
3. 让 `to-spec` 保存稳定需求，让 `to-tickets` 保存执行顺序，让 `implement` 修改代码。
4. 将 review 的阻塞项回写到 spec、design 或 tasks，而不是只留在聊天记录里。
5. 长任务在切换会话前使用 `handoff`，恢复时使用 `resume-ext`。
6. 只有在确实需要外部协作时才使用 `publish-github-issue`。

## 八、快速记忆

```text
不清楚       ask-matt / grilling
需要文档     grill-with-docs / domain-modeling
财务规划     autoplan-finance
写规格       to-spec
拆任务       to-tickets
写代码       implement
测试优先     tdd
审代码       code-review
准上线       release-review
发 GitHub    publish-github-issue
冲突         resolving-merge-conflicts
大项目       wayfinder
换会话       handoff / resume-ext
学东西       teach
写 Skill     writing-great-skills
```
