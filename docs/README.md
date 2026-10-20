# AI Worker 文档导航

AIW = **AI Worker**。这里按一次工作的需要组织文档：明确目标、准备计划、执行工作项、交接独立审查，再交付与归档。

## 从哪里开始

| 你现在需要什么 | 阅读入口 |
|---|---|
| 了解 aiw 并开始使用 | [项目首页与快速开始](../README.md) |
| 理解项目定位、角色与术语 | [项目上下文](../CONTEXT.md) |
| 澄清 Bug、功能或修改请求 | [Issue 使用手册](usage/aiw-issue.md) |
| 为明确目标制定计划并推进实现 | [FD 工作流](usage/aiw-fd.md) |
| 选择和安装 Agent 工作方法 | [Skills 指南](../src/skills/README.md) |
| 查看命令、初始化、插件和历史 Task | [CLI 详细参考](usage/cli-reference.md) |
| 查询源码所支持的稳定行为 | [OpenSpec 能力索引](../openspec/specs/README.md) |
| 配置部署环境、查询环境变量含义和默认值 | [环境变量运维手册](usage/environment-variables.md) |

Issue 用于需要讨论与批准的目标；明确的工程请求可直接进入 FD。FD 保存工作计划和状态，Skills 提供工作方法，交接回执记录角色之间的传递。默认流程由 Planner、Worker、独立 Reviewer 接力，自动派发取决于宿主和 runner 配置。

## 按需使用辅助工具

| 场景 | 指南 |
|---|---|
| 询问 aiw 用法 | [AIW Ask](usage/aiw-ask.md) |
| 研究、原型、文章修改、多课程教学与恢复 | [辅助对话](usage/aiw-auxiliary.md) |
| 管理 GitHub Issues | [GitHub Issues 集成](usage/aiw-github-issues.md) |
| 导出或应用 Git patch | [Git patch](usage/aiw-git-patch.md) |
| 使用 AI 辅助 Git 操作 | [AI Git 命令](usage/aiw-git-ai.md) |
| 配置提交向导 | [CZ 配置](usage/cz-configuration.md)、[Python 运行时](usage/cz-python-runtime.md) |
| 翻译文本 | [AIW Say](../src/programs/aiw-say/README.md) |
| 使用 cxs、bcc、tcc 或 vc 工具 | [cxs](usage/aiw-cxs.md)、[bcc](usage/aiw-bcc.md)、[tcc](usage/aiw-tcc.md)、[vc](usage/aiw-vc.md) |
| 继续旧 REQ 记录 | [Requirement 兼容入口](usage/aiw-requirement.md) |

插件有各自的运行时和 provider 配置。需要模型的命令按配置调用对应服务；具体数据使用和操作范围以各指南为准。

## 阅读工作产物

- [FD 索引](features/FEATURE_INDEX.md)：查看设计与进展；状态以 FD 本身为准。
- `features/reports/` 与 `features/reviews/`：活跃 FD 的实施报告和独立审查记录。
- `features/archive/<FD-ID>/`：已归档的设计与证据，保留其历史语义。
- [报告模板](templates/)：按报告类型选择模板；新 FD 报告使用中文 Markdown 和同名 JSON。
- [改进反馈](feedback.md)：已记录的操作问题与后续改进建议。

报告应区分实际完成的检查、跳过的检查和未决风险。测试、网络、Git 操作与发布遵守目标仓库的授权规则；文档中的命令示例本身不授予执行权限。
