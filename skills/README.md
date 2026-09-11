# AIW Skills 使用指南

本目录包含 AIW/Codex 可复用的开发 Skills。Skill 是给 Agent 使用的工作流程说明，不是独立的 CLI 子命令。CLI 负责发现、安装和管理 Skill；Agent 根据 Skill 的 frontmatter 和正文决定何时调用它。

## 目录中的实际 Skills

### 路由与探索

- `ask-asu`：判断当前处于哪个 AIW/OpenSpec 阶段，并给出下一步建议。
- `wayfinder`：把不确定的工作拆成可推进的决策、研究和实现路径。
- `triage`：整理问题、报告和待处理事项，明确证据与下一步。
- `grilling`：逐个问题压力测试方案或决定。
- `grill-me`：持续追问一个计划或想法，直到关键假设暴露。
- `grill-with-docs`：在深度追问过程中同步记录 ADR、术语和设计文档。
- `prototype`：制作一次性原型，验证状态模型、逻辑或 UI 方向。
- `research`：基于高可信一手资料研究，并将结果记录为 Markdown。
- `diagnosing-bugs`：按证据驱动方式诊断故障、回归、异常和性能问题。

### 需求、领域与设计

- `requirement-management`：管理需求讨论、决策、批准和向 Task 的移交。
- `finance-requirement-intake`：澄清金融、运营、报表和分析类需求。
- `finance-value-assessment`：评估需求价值、成本、风险和范围。
- `finance-metric-brief`：定义指标公式、来源、时间维度、精度、刷新和所有权。
- `finance-engineering-options`：比较技术方案、边界、权限、审计、失败处理和可观测性。
- `finance-requirement-synthesis`：将金融需求讨论整理为完整的 Requirement Plan。
- `domain-modeling`：统一领域术语、概念关系和边界决策。
- `codebase-design`：设计更清晰、更深的模块接口和测试边界。
- `improve-codebase-architecture`：发现架构改进机会并对选定机会深入分析。
- `fd-workflow`：在存在实质性设计选择时深化 Feature Design。

### OpenSpec 与实现

- `to-spec`：把已经讨论清楚的需求和方案写成 OpenSpec change。
- `to-tickets`：将 spec 或 plan 拆成有顺序和验收标准的 checklist item。
- `implement`：实现一个已选中的 AIW Work Item；修改后允许并要求执行
  compile-only 检查，但不自动执行测试、广义 build 或发布。
- `tdd`：按 red-green-refactor 方式围绕一个测试边界开发。
- `code-review`：从 Standards 和 Spec 两个维度审查变更。
- `finance-release-gate`：检查金融变更的迁移、权限、审计、回滚和运行准备度。

### 会话、协作与维护

- `handoff`：生成或消费交接信息，使后续 Agent 能继续工作。
- `resume-ext`：查找并恢复已有的 Codex 会话。
- `resolving-merge-conflicts`：处理进行中的 Git merge/rebase 冲突。
- `setup-project`：为项目准备 AIW/OpenSpec 文档、模板和约定。
- `edit-article`：编辑文章、技术文档或说明文字。
- `teach`：用循序渐进的方式解释技术主题或代码。
- `writing-great-skills`：编写和改进 Skill 的参考规范。
- `github-issues-management`：管理 GitHub Issue 的整理和流程。
- `publish-github-issue`：在用户明确要求时，将结果发布为 GitHub Issue。

## Skill 的调用方式

Skill 通过 Skill 名称调用，例如：

```text
$ask-asu 我现在有一个需求和一部分代码，下一步应该做什么？
$diagnosing-bugs 这个请求偶发超时，请按证据驱动方式诊断。
$to-spec 请把已经确认的方案写成 OpenSpec change。
$implement 实现当前 Task 中选中的 checklist item。
```

具体调用语法取决于宿主 Agent。`$name` 表示调用 Skill，不是 PowerShell 命令，也不是 `aiw name` CLI 命令。

部分 Skill 使用 `disable-model-invocation: true`，只能由用户或其他明确流程显式调用；不要假设每个 Skill 都会自动触发。`implement` 允许宿主 Agent 在自动 Work Item 执行时根据 managed handoff 隐式选择，但仍需要先安装到宿主可发现的 Skill 目录。

## CLI 管理 Skills

以下命令管理本目录中的 canonical Skills，并将它们复制到项目的 `.agents/skills/`：

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

默认作用域是当前项目。安装到用户级目录时使用：

```powershell
aiw skills install tdd --scope user
aiw skills discover --scope user
```

也可以从包含 `SKILL.md` 的目录、ZIP 或 Skill 集合安装：

```powershell
aiw skills install .\skills\my-skill
aiw skills install .\skill-bundle.zip --dry-run
```

安装规则：

- 每个 Skill 目录必须包含合法的 YAML frontmatter、`name` 和 `description`。
- 目录名必须与 frontmatter 中的 `name` 相同。
- 默认不会覆盖同名的 unmanaged Skill。
- `adopt` 只登记已有目录，不会把它转换成 canonical source。
- `sync` 只更新已经登记为 managed 的目标。
- `--dry-run` 只显示计划，不写入目标目录。

当前实现会拒绝 Skill 中的符号链接和非普通文件。安装不可信 ZIP 前应先检查来源；ZIP 路径校验仍是需要优先修复的安全事项。

## 推荐工作流

### 需求还不清楚

```text
requirement-management
  -> grilling / grill-with-docs
  -> domain-modeling
  -> finance-value-assessment（金融或运营需求需要时）
  -> finance-metric-brief（涉及指标时）
  -> finance-engineering-options
```

### 需求已经明确

```text
to-spec -> to-tickets -> implement
                       \-> tdd（用户要求测试先行时）
```

完成实现后，按用户授权选择 `code-review`、`finance-release-gate` 或发布流程。`implement` 不会自动调用 `tdd` 或 `code-review`。

### 遇到 Bug

```text
triage -> diagnosing-bugs -> implement
```

如果需要先确认模块边界，可在实现前使用 `codebase-design`。

## Skill 的统一边界

所有 reviewed Skill 都应遵循：

- 使用 `skills/reviewed-skill-contract.md` 和 `skills/work-management.md`。
- 缺少关键事实时写 `%% NEEDS_INPUT: ...`，阻塞时明确 `BLOCKED` 或 `INCOMPLETE`。
- 报告实际产生的 artifacts、Evidence、Gates 和推荐的下一步。
- 不伪造 Task 状态、Attempt 结果、Lease 或发布结果。
- 自动实现时允许 compile-only 检查：优先使用 `scripts/` 或仓库根目录的
  `compile` 脚本；没有脚本时使用语言级编译命令。不得运行 `build` 脚本、
  测试、格式化、Lint、vet、验证流程、网络操作或发布；编译失败时修复并重试，
  无法解决则报告阻塞。
- 结束时区分已完成的静态工作和没有执行的运行验证。

编写新 Skill 时，至少说明：触发条件、非触发条件、输入、输出、完成标准、未决输入处理和验证范围。

## 相关文档

- [Reviewed Skill Contract](reviewed-skill-contract.md)
- [Work Management](work-management.md)
- [AIW Work Management](../docs/agents/work-management.md)
- [AIW 使用说明](../README.md)
