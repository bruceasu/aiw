# AIW Skills 指南

本目录存放可复用的 AIW Skills。Skill 为 Agent 描述工作流程；`aiw`
CLI 负责发现、安装和同步 Skill。Skill 名称不是 CLI 子命令。

## Skill 目录

以下条目对应本目录中的 Skill。具体触发条件和边界以各自的
`SKILL.md` 为准。

### 路由、探索与问题澄清

- `ask-asu`：根据当前阶段建议下一步 Skill 或 AIW 操作，不代替执行。
- `wayfinder`：将较大的不确定工作组织成可逐步解决的问题。
- `triage`：按证据和状态整理问题或外部 PR。
- `grilling`、`grill-me`、`grill-with-docs`：压力测试方案；后两者分别
  持续追问、同步记录设计材料。
- `research`：基于高可信资料调查问题，并在仓库记录发现。
- `prototype`：构建一次性原型以检验设计想法。
- `diagnosing-bugs`：根据证据诊断故障、回归或性能问题。

### Issue、FD 与领域设计

- `issue-management`：澄清并决策一个 AIW Issue，可交接到 FD。
- `fd-workflow`：创建或细化编号 FD、记录工程决策和有序 Work Items。
- `domain-modeling`：整理领域术语、概念关系和边界。
- `codebase-design`：设计模块接口、职责和代码边界。
- `improve-codebase-architecture`：发现并分析架构改进机会。
- `finance-requirement-intake`、`finance-value-assessment`、
  `finance-metric-brief`、`finance-engineering-options`、
  `finance-requirement-synthesis`：澄清金融及运营需求、价值、指标、
  技术选项并整理结论。
- `requirement-management`：继续处理旧版 AIW Requirement 记录，保留其
  REQ ID 和历史；新需求使用 `issue-management`。

### 实现、规格与审查

- `implement`：按 FD 实现选定的 Work Item；旧 AIW Task 按其现有记录继续。
- `fd-review`：独立审查编号 FD 的实现和证据。
- `to-spec`：更新 OpenSpec 稳定规格；只有用户明确要求时才创建 change。
- `to-tickets`：将 Issue 或设计拆为 AIW Task work items，可选 OpenSpec
  change；新工作默认采用编号 FD 流程。
- `tdd`：在用户要求测试优先时采用 red-green-refactor。
- `code-review`：按仓库标准和原始规格审查变更。
- `finance-release-gate`：在发布前评估金融变更的上线准备情况。

### 会话、维护与集成

- `handoff`：准备供另一 Agent 或会话继续工作的交接文档。
- `resume-ext`：列出当前工作区的 Codex 会话并准备可复制的恢复命令。
- `resolving-merge-conflicts`：处理进行中的 Git merge/rebase 冲突。
- `setup-project`：提议并配置 AIW 工程约定和 Agent 指令；写入前展示变更并等待确认。
- `edit-article`、`teach`：编辑文章或循序渐进地讲解技术内容。
- `writing-great-skills`：编写和改进 Skill 的参考规范。
- `github-issues-management`：通过 AIW GitHub 集成管理 GitHub Issues。
- `publish-github-issue`：仅在用户明确要求时发布 GitHub Issue。

## 调用 Skill

在支持 Skill 的宿主 Agent 中按名称调用，例如：

```text
$issue-management 梳理这个功能请求并决定是否进入设计
$fd-workflow 为已批准的 Issue 创建编号 FD
$implement 实现当前 FD 中已选定的 Work Item
```

具体语法由宿主决定。`$name` 表示调用 Skill，不是 `aiw name` 命令。
带有 `disable-model-invocation: true` 的 Skill 需要用户或已定义的流程
显式调用；不要假设 Agent 会自动选择它。

## 推荐工作流

### 新工程工作

```text
issue-management（需要澄清或批准时）
  -> fd-workflow
  -> implement
  -> fd-review
```

可以直接为明确请求创建 FD；新 FD 不要求先创建 AIW Task。FD 是
`docs/features/FD-XXX_SLUG.md` 中的权威计划，包含决策、Work Items、状态
和 Verification。`docs/features/FEATURE_INDEX.md` 仅作索引；`.ai/fd/<id>/`
存放 AIW 生成的交接收据和日志。

OpenSpec 管理稳定能力规格。仅当用户明确要求时创建 OpenSpec change。
既有 AIW Task 和 Core 状态仍按原记录继续，不要为了迁移改写其历史。

### 其他入口

- 想先了解阶段和下一步：`ask-asu`。
- 仅需诊断故障：`diagnosing-bugs`；需要改动时再进入 Issue/FD 流程。
- 金融需求：按需要组合 `finance-requirement-intake`、
  `finance-value-assessment`、`finance-metric-brief`、
  `finance-engineering-options` 和 `finance-requirement-synthesis`。
- 用户明确要求 OpenSpec change：`to-spec`，再按需要使用 `to-tickets`。

Skill 不会因为出现在工作流中就自动获得额外授权。实现、审查、测试、
发布和工作流状态操作仍受仓库指令及各 Skill 边界约束。

## 使用 CLI 管理 Skills

以下命令管理仓库中的 canonical Skills，并将它们安装到项目的
`.agents/skills/`：

```powershell
aiw skills list
aiw skills list --json
aiw skills install implement --dry-run
aiw skills install implement
aiw skills install --all --dry-run
aiw skills install --all
aiw skills discover
aiw skills discover --json
aiw skills adopt
aiw skills sync implement
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
- [AIW Work Management](../docs/agents/work-management.md)
- [AIW 使用说明](../README.md)
