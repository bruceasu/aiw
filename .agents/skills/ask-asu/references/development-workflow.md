# 开发流程知识库

本知识库为 `ask-asu` 提供本仓库内置 Skill 和 AIW 命令的路由信息。它不授予执行权限；需要执行时，先读取目标 Skill 的 `SKILL.md` 或运行对应的只读帮助。

## 内置 Skill 路由

| 场景 | 首选 Skill | 何时使用 |
| --- | --- | --- |
| 需求发现、范围与干系人 | `requirement-management` | 新需求、模糊目标或需要记录 Requirement |
| 缺陷初分流 | `triage` | 报错、异常行为或待分类请求 |
| 难题诊断 | `diagnosing-bugs` | 需要建立假设、证据和复现路径 |
| 大型不确定计划 | `wayfinder` | 选择、依赖或范围尚未收敛 |
| Feature Design 与决策 | `fd-workflow` | Task 尚有重大设计决策 |
| OpenSpec 规格 | `to-spec` | 需要 proposal、design 或 capability specs |
| 实现切片 | `to-tickets` | 将设计转为有序 `tasks.md` 条目 |p
| 单一 checklist 实现 | `implement` | 已选中 Task 条目并具备工作区 |
| 测试先行 | `tdd` | 用户明确要求 red-green-refactor 或集成测试 |
| 变更评审 | `code-review` | 用户明确要求对分支、PR 或固定点之后的改动评审 |
| 模块设计 | `codebase-design` | 接口、模块边界、seam、可测试性需改进 |
| 领域语言与 ADR | `domain-modeling` | 术语、实体关系或架构决策需要固定 |
| 原型验证 | `prototype` | 用一次性原型回答 UI 或状态模型问题 |
| 架构健康 | `improve-codebase-architecture` | 识别并规划代码库健康改进 |
| 交接与恢复 | `handoff`、`resume-ext` | 迁移到新 Thread 或寻找本地会话 |
| 项目工作流初始化 | `setup-project` | 仓库尚未建立 AIW 工作管理 |
| 合并冲突 | `resolving-merge-conflicts` | 正在进行 merge 或 rebase 且存在冲突 |
| 外部 Issue 操作 | `github-issues-management`、`publish-github-issue` | 用户明确要求 GitHub 读取或发布 |
| 知识研究 | `research` | 需要对可信来源做研究并写入 Markdown |
| 技能作者工作 | `writing-great-skills` | 创建或改进仓库内的 Skill |
| 教学与文章 | `teach`、`edit-article` | 多会话学习或文章编辑 |
| 决策压力测试 | `grill-me`、`grill-with-docs`、`grilling` | 用户要求质询或压测方案 |

金融、指标、发布、工程评审等专用 Skill 只在请求确属财务、运营、风险或分析领域时推荐；选择前读取对应 `SKILL.md`。

## AIW 命令知识库

| 目的 | 命令 | 边界 |
| --- | --- | --- |
| 获取准确命令语法 | `aiw help [command|topic]` | 先用于陌生或变更状态的操作；帮助本身不替代授权 |
| 初始化项目约定 | `aiw init` | 会写入脚手架，需用户请求 |
| 创建受管变更 | `aiw new <task-id> --backend auto` | 建立 AIW Task 并可委托 OpenSpec；不要以 `openspec new change` 代替 |
| 查看、列出 Task | `aiw show <task-id>`、`aiw list` | 只读状态检查 |
| 改变 Task 状态 | `aiw status <task-id> <status>`、`aiw done <task-id>` | 需要明确状态转换请求 |
| 归档 | `aiw archive <task-id>` | 完成不代表合并或发布；归档需单独授权 |
| 管理隔离工作树 | `aiw wt add|status|list|lock|unlock|repair ...` | 仅在隔离确有必要且已授权时；不得用 raw Git worktree 绕过 |
| 单次会话交接 | `aiw turn <task-id>` | 用户明确要求继续/交接时使用；保留 Session 与 lineage |
| 规划或同步 Workflow Core | `aiw task workflow plan|sync|advance <task-id>` | 会改变受管状态时需要授权 |
| 预览或执行下一个工作项 | `aiw task workflow run <task-id>`；`--execute` | 无 `--execute` 用于预览；执行需明确授权，默认隔离工作树 |
| 持续监督 | `aiw task workflow supervise <task-id> start|status|stop` | `start`/`stop` 是状态性操作，需明确授权 |
| 诊断和修复受管状态 | `aiw task workflow diagnose|recover|repair <task-id>` | `diagnose` 用于检查；`recover`/`repair` 需明确授权 |
| Evidence、Gate、Work Item | `aiw task workflow evidence|gate|complete ...` | 仅记录真实证据和获批决定，不能伪造结果 |

## 建议原则

- 一个共同目标、分支、工作树、交付和归档生命周期对应一个 AIW Task 与一个 OpenSpec change；独立生命周期才建议拆分，并先取得用户同意。
- AIW 管理 Task、工作树、分支、Session 和生命周期；OpenSpec 管理 proposal、design、规格与人类维护的 checklist。
- 检查、测试、构建、格式化、发布、提交、合并和清理是不同授权边界。建议它们时，明确说明目的、范围、预计耗时和风险；不得默认执行。
- 每次建议优先给出最小的可逆步骤；信息不足则用 `%% NEEDS_INPUT:` 询问会实质改变方案的事实。
