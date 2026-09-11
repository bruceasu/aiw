# Requirement Artifact

## Metadata
- Artifact Type: Problem Brief
- Requirement ID: aiw-ask
- Stage: intake
- Status: CLEAR
- Based On: 用户希望通过 AIW 内置 LLM 获取 AIW 使用方案，并沉淀个人问答记忆
- Human Approval Required: yes

## Facts
- 新增 `aiw ask "<prompt>"` 命令，调用 AIW 内置 LLM。
- LLM 回答如何使用 AIW 完成指定工作；AIW 不支持时提供替代方案。
- 问答保存到用户 HOME 下的 `$HOME/.aiw/ask/<date>/`，不提交 Git。
- 文件名为 `<datetime>-<sha256(prompt)>.md`，内容包含 session、时间、问题和回答。
- 支持 `--resume` 续接最近可恢复的相同 session；找不到时直接开启新 session。
- 支持 `--chat` 直接进入 chat 模式；chat 模式使用 `promptui`。
- 非 chat 模式输出友好结果。
- 预留 system prompt 参数，可指定内容或文件；默认从配置文件读取，也可为空。
- `ask` 只提供问答，不修改任何文件。
- 默认不可读取当前目录以外的文件；明确授权后才可读取。
- LLM 失败或中断时不写入 answer 内容；chat 模式输出提示。

## Assumptions
- “最近可恢复 session”按当前用户本地 ask 记录确定。
- session 可以跨日期目录继续。
- system prompt 文件也遵守目录访问限制，除非明确授权。

## Decision Flow
| Actor | Situation | Sees | Decides | Acts | Downstream Impact |
|---|---|---|---|---|---|
| AIW 用户 | 不确定如何用 AIW 完成工作 | LLM 给出的 AIW 操作方案或替代方案 | 是否采纳方案、是否继续追问 | 执行建议中的后续命令 | 问答沉淀为个人 memory，供后续分析并评估插件或核心功能 |

## Stakeholders
- AIW 个人用户
- AIW 功能维护者（后续分析问答记录）

## Current Workaround
- 用户自行查阅 AIW 文档、尝试命令，或向 Agent 重复询问；没有统一的本地问答记忆。

## Scope
### Smallest Useful Scope
- `ask` 单轮问答及 Markdown 记录。
- `--chat` 和 `--resume` 会话交互。
- system prompt 配置入口。
- 本地存储、失败提示和目录读取边界。

### Explicit Non-Goals
- 不直接修改项目文件。
- 不提交问答记录到 Git。
- 不在本需求中自动把问答提炼为插件或核心功能。

## Open Questions
- %% 工程设计阶段确定具体错误提示文案。
- %% 工程设计阶段确定 session 选择和并发写入细节。

## Suggested Next Stage
- 进入工程可行性与边界澄清，重点确认现有 LLM 接口、promptui 集成、配置读取和 HOME 路径兼容性。
