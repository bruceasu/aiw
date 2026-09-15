---
name: ask-asu
description: 在整个 AIW/OpenSpec 开发流程中提供阶段建议，并路由到合适的内置 Skill 或 AIW 命令；不执行实现或生命周期变更。
---

# Ask Asu

在用户提出需求、澄清范围、设计、规划、实现、验证、交付或交接时使用本 Skill，帮助其判断当前阶段、最小下一步和风险。它是顾问与路由器，不是执行器。

遵循 `skills/reviewed-skill-contract.md` 和 `skills/work-management.md`。

## 输入与输出

从用户已有上下文识别目标、当前阶段、Task / OpenSpec change、已完成证据、未知项和是否授权运行命令或改变外部状态。不要为获得信息而改变任何状态。

返回简明建议，包含：

1. **当前判断**：所处阶段及其依据；无法判断时标记 `%% NEEDS_INPUT: ...`。
2. **建议路径**：一个首选 Skill 或 AIW 命令，以及只在确有价值时列出的备选方案。
3. **开始条件**：需要的输入、授权或现有工件。
4. **下一可观察交接**：将产生或检查的工件、Evidence、Gate 或用户决策。
5. **风险与边界**：指出范围、兼容性、迁移、权限、运行时验证或交付授权中的阻塞项。

建议应随开发进展更新：先消除会改变方案的未知项，再建议最小可交付切片；不要把“完成实现”误说成已测试、已合并、已发布或已归档。

## 开发阶段建议

- **新想法、问题不清或需求冲突**：先澄清问题、受众、成功条件和范围；推荐 `requirement-management`。复杂或跨域决策可用 `wayfinder`。
- **缺陷、报错或性能退化**：先以 `triage` 分类；问题难以定位时用 `diagnosing-bugs`，并把运行时复现作为需显式授权的后续步骤。
- **需求已稳定但尚未可实施**：确认一个 AIW Task 与一个 OpenSpec change 的生命周期映射；设计、规格和任务拆分依次推荐 `fd-workflow` / `to-spec` / `to-tickets`。
- **需求或设计中的领域语言不稳定**：先使用 `domain-modeling` 固化术语、实体关系和边界，再继续需求或 OpenSpec 设计。
- **准备编码**：确认选中的 `tasks.md` 条目、工作区和授权边界；推荐 `implement`。只有明确要求测试先行时才推荐 `tdd`。
- **实现中或实现后**：建议检查变更是否覆盖当前 checklist、静态证据和未决 Gate；测试、构建、格式化和评审均须用户明确授权。用户明确要求审查时才推荐 `code-review`。
- **交付、并行或跨会话**：先确认 Task 的工作区、分支和父分支；隔离工作区使用 `aiw wt`，跨会话使用 `handoff` 后由用户明确要求的 `aiw turn <task-id>`。合并、发布、清理和归档必须单独授权。

读 [开发流程知识库](references/development-workflow.md) 查询内置 Skills、AIW 命令边界和阶段映射。知识库只作路由速查；执行某项 Skill 或命令前，仍须读取其自身说明或 `aiw help <command>`。

## AIW / OpenSpec 主流程

1. 使用 `requirement-management` 处理新需求、模糊需求或待决的业务/技术选择。
2. Requirement 获批后，将其提升为一个 AIW Task；该步骤创建或复用 Task 及其初始 OpenSpec 规划工件。
3. 若 Task 仍缺设计、提案、能力规格或任务轮廓，使用 `/fd-workflow` 或 `/to-spec`；所需工件和 ID 一致性未满足前，流程仍不完整。
4. 使用 `/to-tickets` 在 `tasks.md` 中形成有序的实现切片。
5. 在 Task 工作区内使用 `/implement` 完成一个选中的条目；受管自动执行可使用 `aiw task workflow run|supervise <task-id>`，但只有用户明确授权执行时才建议实际运行。
6. 开发后询问一次用户是否要运行一个聚焦测试命令；默认不测试。
7. 仅在用户明确要求评审时使用 `/code-review`。
8. 全部 checklist 完成后，报告派生的 Workflow 摘要、Evidence、未关闭 Gate 和用户请求的下一状态转换。Git 交付、归档、合并、清理及删除分支均须单独授权。

保持共享目标、分支、工作树、交付和归档生命周期的 checklist 在同一 AIW Task 内；仅在生命周期独立且用户同意后拆分为另一个 AIW Task 与 OpenSpec change。

## 边界与完成条件

- 不自动调用 `/tdd`、`/code-review`，也不运行命令、修改文件、创建 Task/change、管理工作树或改变生命周期。
- 它在用户获得清晰下一步、前置条件和风险边界后完成；若关键事实未知，返回 `INCOMPLETE` 和 `%% NEEDS_INPUT: ...`。
- 编写测试可以被建议；运行测试需要用户明确指令。主代理最多使用两个有界子代理；子代理不能运行测试、构建、网络操作、权限升级、提交、归档或工作树操作。
- 手动实现通常使用主工作区；自动 Task 执行默认隔离至 `.wt/<task-id>`。只有明确授权时才使用主工作区例外；工作树只能经 `aiw wt` 创建和解析。

## 补充路线

- 来自用户的缺陷或请求：`/triage`，随后进入主流程。
- 困难缺陷：先静态诊断；仅在用户明确授权后才建议运行时复现。
- 大型或不确定的工作：用 `/wayfinder` 解决决策，再进入 `/to-spec`、`/to-tickets` 和 `/implement`。
- 代码库健康：`/improve-codebase-architecture`，获批后再进入主流程。

## 跨会话

当新 Thread 需要当前上下文时使用 `/handoff`。在 AIW Session 中，将 handoff 保存到 Session 工件；只有用户要求在新 Thread 中继续时，才使用 `aiw turn <task-id>` 以保留 Task、工作树、Session、lease 和 lineage。

使用 `/compact` 仅在有意的阶段边界继续同一会话。

## 常用支持 Skill

- `/domain-modeling`：领域语言和 ADR。
- `/codebase-design`：模块边界、seam 和接口。
- `/tdd`：用户明确要求且已授权运行测试的测试先行工作。
- `/code-review`：针对固定点的显式静态评审。
- `/publish-github-issue`：显式外部发布。
- `/teach`：多会话学习。
- `/writing-great-skills`：Skill 编写指导。

当仓库尚未配置工作管理时，在首次工程流程前运行 `/setup-project`。
