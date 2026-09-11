# Requirement Artifact

## Metadata
- Artifact Type: Requirement Plan
- Requirement ID: aiw-ask
- Stage: synthesis
- Status: READY_FOR_HUMAN_DECISION
- Based On: Problem Brief, Business Case, Engineering Options
- Human Approval Required: yes

## Facts
- AIW 处于推广期，用户查找 help/README 了解用法成本较高，需要即时 LLM 助手。
- 问答记录作为个人 memory 保存到 `$HOME/.aiw/ask/<date>/`，不进入 Git。
- `aiw ask` 支持单轮、简洁 `--chat` 和 `--resume`；Chat 支持多行输入、快捷键提示和 `/exit`。
- LLM 已在 AIW 核心实现，直接复用，不提取 provider 接口。
- `ask` 只读；默认只读取当前目录及子目录中与问题相关的内容，目录外需显式授权。
- 回答使用 JSON 规范数据，并渲染为友好终端文本。
- 不做自动分析、复杂编辑或工具调用；未来插件可承载外部集成和分析能力。

## Assumptions
- 现有核心 LLM 调用入口可安全复用。
- 初期每周由维护者直接分析 Markdown 记录。
- 首期价值通过推广期问答频率、重复问题、无法回答问题和任务完成效率验证。

## Evidence Index
- `problem-brief.md`：问题、用户、决策流程和最小范围。
- `business-case.md`：推广期价值假设与 VALIDATE_FIRST 建议。
- `engineering-options.md`：核心/插件边界、权限、失败处理和结构化回答契约。

## Confirmed Workflow

1. 在 AIW 核心新增只读的 `aiw ask "<prompt>"` 命令，并复用现有 LLM。
2. 支持 `--chat`、`--resume`、多行输入、基础快捷键提示和 `/exit`；不提供复杂编辑或工具调用。
3. 从配置读取可为空的默认 system prompt，并支持文本与文件形式的命令行覆盖。
4. 默认仅在问题相关时读取当前目录及子目录；目录外读取须经命令行路径授权和对话确认。
5. 返回并校验版本化 JSON 结构化回答，将其渲染为友好终端输出。
6. 在 `$HOME/.aiw/ask/<date>/` 保存个人 Markdown session memory；不写项目文件、不提交 Git。
7. LLM 失败或中断时记录问题和状态但不写 answer；`--resume` 无可恢复会话时新建 session。

## Scope
### In Scope
- 核心 `aiw ask` 命令与 LLM 调用。
- `--chat`、`--resume`、多行输入、`/exit` 和基础快捷键提示。
- system prompt 文本/文件配置入口。
- Session 与 Markdown memory 持久化。
- JSON 结构化回答及终端渲染。
- 只读上下文和路径授权边界。

### Out of Scope
- 自动问答分析或插件自动生成。
- 复杂文本编辑。
- Chat 内工具调用。
- 新的 provider 抽象或 LLM 重构。

## Design Readiness
- Status: BLOCKED
- Reason: 正式 OpenSpec 设计仍须确定现有 LLM 调用入口、路径授权参数名称，以及 JSON schema 校验失败时的重试或降级策略。
- Implementation Handoff: blocked until the above decisions are resolved in `design.md`.

## Risks and Open Questions
- %% 现有核心 LLM 调用入口和 system prompt 配置格式需在 OpenSpec 设计阶段确认。
- %% 路径授权参数和 JSON schema 校验失败策略待设计确认。
- %% 每周人工汇总的最小输出格式暂不固定。

## Earliest Blocking Decision
- Requirement 已批准并可创建 Task；但 OpenSpec Design Readiness 为 BLOCKED，Task 不得进入实现，直至设计决策解决。

## Human Decision Requested
- 已记录 APPROVED；下一步推广为 AIW Task 并准备 OpenSpec 规划材料。

## Suggested Next Stage
- 创建 Task 和初始 OpenSpec 规划材料；在 `design.md` 解决所有 `%% NEEDS_INPUT` 后再交给实现。

## Release
Status: NOT_STARTED
Trigger: after an AIW Task exists, implementation evidence is available, and a release is planned.
