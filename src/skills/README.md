# AI Worker Skills 指南

AIW = **AI Worker**。本目录存放可复用的工作方法，帮助 AI 澄清目标、制定计划、实现工作项并交接独立审查。Skill 为宿主 Agent 描述工作流程；`aiw`
CLI 负责发现、安装和同步 Skill。Skill 名称不是 CLI 子命令。

## Skill 目录

以下条目对应本目录中的 Skill。具体触发条件和边界以各自的
`SKILL.md` 为准。

### 路由、探索与问题澄清

- `ask-asu`：根据当前阶段建议下一步 Skill 或 AIW 操作，不代替执行。
- 大型不确定需求：Issue／Planner 按需读取共享决策依赖方法，由现有 grilling 方法澄清问题，不再有独立 wayfinder 入口或地图票据流程。
- `triage`：整理多来源用户诉求、重复候选、信息缺口和下一步；深入需求交 Issue，工程计划交 FD，PR 正确性交独立评审。
- `grilling`、`grill-me`：按用户要求通过追问检验方案。
- `grill-with-docs`：在 Issue 初步理解、关键规则讨论和决策交接时，
  按复杂度检验需求质量；确认结论交回原 Issue Plan，由宿主确认和记录。
- `research`：基于高可信资料调查问题，并在仓库记录发现。
- `prototype`：构建一次性原型以检验设计想法。
- `diagnosing-bugs`：根据证据诊断故障、回归或性能问题。

### Issue、FD 与领域设计

- `issue-management`：澄清并决策一个 AIW Issue，可交接到 FD。
- `fd-workflow`：统一编号 FD 的设计、实现、独立审查交接及自动交付归档；支持只实现或部分工作项等明确限制。
- `domain-modeling`：澄清术语、概念关系、状态和不变量，给出正常／边界例子；Issue 快照模式只返回建议，稳定文档写入遵守所属阶段与授权范围。
- `codebase-design`：设计模块接口、职责和代码边界。
- `improve-codebase-architecture`：发现并分析架构改进机会。
- `issue-management/references/`：随 Issue 入口按需读取的问题澄清、
  价值、指标、技术选项与综合决策方法，不作为独立 Skill 安装或调用。
- `requirement-management`：继续处理旧版 AIW Requirement 记录，保留其
  REQ ID 和历史；新需求使用 `issue-management`。

### 实现、规格与审查

- `fd-workflow/references/implementation.md`：由 Worker 阶段按需加载的实现方法，不作为独立 Skill 安装或调用。
- `fd-review`：独立审查编号 FD 的实现和证据。
- `to-spec`：更新 OpenSpec 稳定规格；只有用户明确要求时才创建 change。
- Issue 负责按业务结果拆子 Issue；Planner 负责拆 FD Work Items。
  共同原则按需读取 `skills/workflow-methods.md` 的领域澄清、决策依赖、业务拆分、规格或评审章节；triage 只用队列分流章节，不加载全套。
- `tdd`：显式测试优先或 FD 测试支持时，对合适的核心规则使用有效 Red/Green，记录独立预期依据与回归保护。
- `fd-test`：独立推导验收场景、复用或补充测试，在授权范围内执行并产出版本绑定的事实报告。
- FD 测试支持按语义选择：`with test`、`带测试`、`需要测试支持` 等均可；默认不增加测试。更具体的只生成用例、不运行或不用 TDD 等限制优先。
- 测试共享 `skills/testing-methods.md`。达到非预期失败预算后自动停止该测试相关修复/执行，静态诊断并继续独立工作；已确认缺陷和关键未知仍阻止交付。
- `code-review`：独立审查分支或 PR；与 `fd-review` 共享实现审查方法，普通 FD 不重复启动两套审查。
- `finance-release-gate`：跨领域评估已实现变更的上线准备情况。

### 会话、维护与集成

- `handoff`：准备供另一 Agent 或会话继续工作的交接文档。
- `resume-ext`：列出当前工作区的 Codex 会话并准备可复制的恢复命令。
- `resolving-merge-conflicts`：处理进行中的 Git merge/rebase 冲突。
- `setup-project`：提议并配置 AIW 工程约定和 Agent 指令；写入前展示变更并等待确认。
- `edit-article`、`teach`：编辑文章或循序渐进地讲解技术内容。
- `writing-great-skills`：创建、改进、只读审查和适配 Skill；按需读取
  通用原则、Codex/Claude 平台参考与可选效果评估，区分静态证据和实际效果。
- `github-issues-management`：通过 AIW GitHub 集成管理 GitHub Issues。
- `publish-github-issue`：仅在用户明确要求时发布 GitHub Issue。

## 调用 Skill

在支持 Skill 的宿主 Agent 中按名称调用，例如：

```text
$issue-management 梳理这个功能请求并决定是否进入设计
$fd-workflow 为已批准的 Issue 创建编号 FD
$fd-workflow FD-XXX 只实现工作项 1.1，不审查、不合并
$fd-workflow FD-XXX auto with test
$fd-workflow FD-XXX 自动执行，需要测试支持
```

具体语法由宿主决定。`$name` 表示调用 Skill，不是 `aiw name` 命令。
带有 `disable-model-invocation: true` 的 Skill 需要用户或已定义的流程
显式调用；不要假设 Agent 会自动选择它。

## 推荐工作流

### 新工程工作

```text
issue-management（需要澄清或批准时）
  -> fd-workflow
  -> fd-workflow 的 Worker 阶段（按需加载 implementation.md）
  -> fd-review
  -> PM：local-merge、归档与清理（条件满足时自动继续）
```

可以直接为明确请求创建 FD。FD 是
`docs/features/FD-XXX_SLUG.md` 中的权威计划，包含决策、Work Items、状态
和 Verification。`docs/features/FEATURE_INDEX.md` 仅作索引；`.ai/fd/<id>/`
存放 AIW 生成的交接收据和日志。

编号 FD 实现请求默认包含本地聚焦提交、独立审查、通过后的自动交付及
归档，无需再补发 `auto`。只规划、只实现、只审查、部分工作项或暂不合并等
明确限制优先；脏工作区、过期审查、未解决 blocker、冲突和失败会停止受影响的
步骤并保留现场。该默认不授权测试、构建、网络、push 或部署。

OpenSpec 管理稳定能力规格。仅当用户明确要求时创建 OpenSpec change。
FD 是唯一工程计划，历史 FD 和审查证据保留；当前技能不再使用旧 Task 流程。

### 其他入口

- 想先了解阶段和下一步：`ask-asu`。
- 仅需诊断故障：`diagnosing-bugs`；需要改动时再进入 Issue/FD 流程。
- 各领域 Issue：使用 `issue-management`，由它按当前缺口选择内部
  references；共享 `grill-with-docs` 和发布评估保持独立。
- 用户明确要求 OpenSpec change：`to-spec`；FD 仍拥有工程执行清单。

Planner 负责设计方案、验收和工程拆分，按能力规则变化加载共享规格方法。
重要设计或用户要求时，按 `fd-workflow/references/planning.md` 进行独立设计反馈；
它不替代实现后的 `fd-review`。简单改动不固定增加设计审查轮次。

`fd-review` 检查真实代码/配置、正确性、Standards、Spec 和证据，独立生成
FD 验收结论。共享方法不自动启动额外 Agent、测试或完整 `code-review`。
`finance-release-gate` 仅在计划发布时检查运维准备；本地合并和归档不是发布，
GO 也不授权部署。

Skill 不会因为出现在工作流中就自动获得额外授权。实现、审查、测试、
发布和工作流状态操作仍受仓库指令及各 Skill 边界约束。

## 使用 CLI 管理 Skills

以下命令管理仓库中的 canonical Skills，并将它们安装到项目的
`.agents/skills/`：

```powershell
aiw skills list
aiw skills list --json
aiw skills install fd-workflow --dry-run
aiw skills install fd-workflow
aiw skills install --all --dry-run
aiw skills install --all
aiw skills discover
aiw skills discover --json
aiw skills adopt
aiw skills sync fd-workflow
```

默认作用域为当前项目。安装或检查用户级 Skills 时使用 `--scope user`：

```powershell
aiw skills install tdd --scope user
aiw skills discover --scope user
```

也可以安装包含 `SKILL.md` 的目录或 ZIP：

```powershell
aiw skills install .\skills\my-skill
aiw skills install .\skill-bundle.zip --dry-run
```

- Skill 目录名必须与 frontmatter 中的 `name` 相同，并包含 `description`。
- 默认不会覆盖同名的 unmanaged Skill。
- `adopt` 登记已有目标；不会把它转成 canonical source。
- `sync` 只更新已登记为 managed 的目标，并从权威文件重新生成共享引用。
- `--dry-run` 只显示计划，不写入目标目录。

## 统一边界与验证

Engineering Skills 遵循 `skills/reviewed-skill-contract.md` 和
`skills/work-management.md`。缺少关键事实时记录
`%% NEEDS_INPUT: ...`；存在阻塞时标明 `BLOCKED` 或 `INCOMPLETE`。报告真实
产生的文件、证据、未决事项和实际运行的命令，不伪造状态或验证结果。

按仓库资源预算执行验证：默认不运行测试、最终构建、格式化、Lint、vet、
验证脚本或网络操作。代码修改后允许一次 compile-only 检查；遵从更具体的
仓库或 Skill 指令，并在总结中说明实际运行和跳过的检查。

## 相关文档

- [Reviewed Skill Contract](reviewed-skill-contract.md)
- [Work Management](work-management.md)
- [AI Worker 文档导航](../../docs/README.md)
- [AI Worker 使用说明](../../README.md)
