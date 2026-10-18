# aiw — AI Worker

**让 AI 以 Worker 的方式参与工程：理解目标、按计划实现、交接审查、留下证据。**

aiw 是面向 AI 辅助开发的命令行工具。它将需求讨论、工程设计、Skills、角色交接和 Git 工作区组织在同一套流程中，让人和 AI 围绕明确的工作项协作。

## AI Worker 如何工作

人决定目标、范围和授权。Planner 明确方案与工作项，Worker 实现已准备好的范围，独立 Reviewer 检查实现与证据。PM 管理流程状态和需要人决定的事项。宿主可以承担多个职责，独立审查仍需由另一会话完成。

```text
需要澄清的目标 → Issue 讨论与批准 → 编号 FD
明确的工程请求 ─────────────────→ 编号 FD
                                     ↓
                               Planner 设计
                                     ↓
                               Worker 实现
                                     ↓
                            独立 Reviewer 审查
                                     ↓
                               交付与归档
```

编号 FD（Feature Design）是工作计划的主记录，包含设计决策、Work Items、进度、验收要求和 Verification。明确的请求可以直接创建 FD；新工作不要求先创建 Task。OpenSpec 保存稳定能力规格；只有用户明确要求时才创建 change。

## aiw 提供什么

| 协作需要 | AI Worker 的支持 |
|---|---|
| 把想法说清楚 | Issue 保存问题、讨论产物、范围和批准决定 |
| 把目标变成可执行工作 | 编号 FD 记录方案、工作项、验收要求和待决问题 |
| 让角色接得上 | 显式 handoff、回执和日志支持交接与恢复 |
| 给实现划定工作区 | `aiw git wt` 管理 FD 专用分支、worktree 和本地交付 |
| 复用工作方法 | Skills 为宿主 Agent 提供澄清、设计、实现和审查流程 |
| 接入辅助工具 | 插件扩展 Git、AI 使用问答、翻译和其他开发工具 |

CLI 管理记录和工具操作，Skills 指导 Agent 的工作方式，宿主与配置的 provider 提供 AI 执行能力。配置了角色 runner 时，交接可以启动下一角色；未配置时保留待处理交接，由 Skill 或后续 `aiw fd resume` 继续。

## 独立辅助对话

除工程工作流外，AIW 提供 `research`、`prototype`、`edit-article` 和 `teach`。
先通过对话明确目标，再生成研究记录、一次性原型、文章副本或课程材料；teach 支持
多个课程及暂停／继续。辅助任务本身不创建 Issue／FD。见
[辅助对话使用指南](docs/usage/aiw-auxiliary.md)。

```powershell
aiw teach new "Learn system design"
aiw teach list
aiw research "Understand this module" --source src/internal
```

## 快速开始（工程）

从源码编译主程序需要 Go 1.25.1 或兼容版本。在仓库根目录执行：

```powershell
cd src
go build -o ../aiw.exe ./cmd/aiw
cd ..
.\aiw.exe --help
```

Linux / macOS 可将输出名改为 `../aiw`，使用 `./aiw` 调用。将可执行文件所在目录加入 `PATH` 后，即可使用下文中的 `aiw`。插件按需准备对应运行时和配置；只编译主程序不会安装所有插件。参见 [构建、插件与运行时参考](docs/usage/cli-reference.md)。

在目标项目根目录初始化项目并准备 Skills：

```powershell
aiw init
aiw pkg skill list
aiw pkg skill install fd-workflow --dry-run
aiw pkg skill install fd-workflow
```

Skills 需要支持它们的宿主 Agent 加载。可按需要安装 `issue-management`、`implement` 和 `fd-review`；安装 Skill 本身不会启动工程工作。见 [Skills 指南](src/skills/README.md)。

从一个明确目标开始：

```powershell
aiw fd new "Improve report export"
aiw fd list
```

使用创建命令返回的 FD ID 查看计划，例如 `aiw fd show FD-001`。在宿主 Agent 中调用 `$fd-workflow` 完善方案和工作项，再按交接进入实现与审查。`$fd-workflow` 是 Skill 调用写法，具体语法取决于宿主。

如果目标还需要讨论或批准，先使用 `aiw issue chat` 或 `$issue-management`。Issue 获得明确批准后，`aiw issue promote <issue-id>` 创建关联 FD 并交给 Planner。见 [Issue 使用手册](docs/usage/aiw-issue.md)。

## 按 FD 执行、审查和交付

默认在 `feature/<fd-id>` 与 `.wt/<fd-id>` 中实现。创建 worktree 前，先提交准备好的 FD 计划并保持父工作区干净，确认 `.wt/` 和 `.ai/` 已被 Git 忽略。

`aiw git wt add <fd-id>` 创建并记录工作区；审查通过后，`aiw git wt local-merge <fd-id>` 将改动 squash 到记录的父分支，并在核对成功后清理工作区与分支。这些命令会修改 Git 状态，执行需遵守目标仓库的授权规则。

工作完成、Git 交付和 FD 归档分别记录；具体步骤与恢复方式见 [FD 工作流](docs/usage/aiw-fd.md)。

## 工作记录放在哪里

下表是记录各自创建后的位置，不表示初始化时已生成所有文件。

| 位置 | 用途 |
|---|---|
| `docs/issues/<issue-id>/` | 新 Issue 的讨论产物、计划与批准 |
| `docs/features/FD-XXX_SLUG.md` | FD 的设计、工作项、状态与 Verification |
| `docs/features/FEATURE_INDEX.md` | 从 FD 派生的索引 |
| `.ai/fd/<fd-id>/` | 交接回执、日志、工作区信息与操作审计 |
| `.wt/<fd-id>/` | FD 的隔离实现工作区 |
| `docs/features/reports/`、`docs/features/reviews/` | 活跃 FD 的报告与独立审查证据 |
| `docs/features/archive/<FD-ID>/` | 归档后的设计与证据 |
| `openspec/specs/` | 稳定能力规格 |

旧 Task、REQ 和 Workflow Core 记录继续保留其兼容入口与历史路径。具体命令见 [CLI 详细参考](docs/usage/cli-reference.md)。

## 完成工作需要真实证据

Worker 应记录实际修改、检查结果和未验证项。编译通过、独立审查通过、Git 合并和发布是不同事实；正常的 Complete 归档需要当前实现的独立审查通过记录。PM 的显式流程覆盖也必须如实保留跳过的阶段与缺失证据。

测试遵守仓库授权规则，默认 FD 流程不设置独立 Tester 阶段。可显式调用 `$fd-test` 生成黑盒场景或事实报告；请求设计用例不代表授权执行测试，其报告也不作为默认 FD 验收门槛。网络调用、提交和发布同样遵守各自授权范围。

## 继续阅读

- [文档导航](docs/README.md)：按需求选择指南。
- [项目上下文与术语](CONTEXT.md)：理解 AI Worker、角色和记录归属。
- [FD 工作流](docs/usage/aiw-fd.md)：创建、交接、审查、恢复与归档。
- [Issue 使用手册](docs/usage/aiw-issue.md)：讨论、批准与推广。
- [Skills 指南](src/skills/README.md)：可复用的 Agent 工作方法。
- [CLI 详细参考](docs/usage/cli-reference.md)：命令、插件与历史兼容说明。
- [稳定能力规格](openspec/specs/README.md)：可追踪到源码的能力契约。
