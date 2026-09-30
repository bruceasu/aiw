# Issue Plan: 将 aiw-cz 从 TypeScript 迁移到 Python

## Decision

将 `plugins/aiw-cz` 从 TypeScript/Node.js 迁移为 Python 插件，同时保持
`aiw cz` 的外部命令、配置、国际化、人工审阅和提交行为兼容。

## Scope

- 使用 Python 作为插件实现语言和运行时。
- LLM 回退顺序固定为：Codex CLI → Copilot CLI → OpenAI HTTP API → 交互式向导。
- Codex 和 Copilot 通过 Python 标准库进程调用本机 CLI，不引入对应 SDK。
- OpenAI 通过 HTTP API 调用，不引入 OpenAI Python SDK。
- 目标机器预装 Python；缺少必要第三方依赖时支持 `pip install -r requirements.txt`。
- 交互式环境检测到可用 GUI 时使用 GUI，否则使用 TUI。
- 保留现有国际化；未显式配置语言时使用操作系统语言，无对应翻译时回退英语。
- 保留提交前人工审阅，LLM 不得直接提交。

## Non-goals

- 不改变 `aiw cz` 的命令名称和用户配置语义。
- 不改变 Codex → Copilot → OpenAI 的顺序或全部失败后的交互式回退。
- 不将 GUI 作为强制依赖；GUI 不可用时必须回退到 TUI。
- 不修改 Go AI provider 工作流。

## Acceptance

- `aiw cz` 能从现有插件入口启动 Python 实现。
- 各 provider 失败或返回无效候选时按确定顺序继续尝试。
- 全部 provider 失败后，本地交互环境使用 GUI；无 GUI 或非交互环境使用 TUI。
- 现有 `[cz]`、provider 配置、语言选择、候选校验、人工审阅和 Git 提交行为保持兼容。
- 系统语言没有对应 locale 时显示英语。
- CLI 不可用、HTTP 错误、超时或响应无效时不会绕过人工审阅。

## Constraints and risks

%% NEEDS_INPUT: FD 阶段需确认 Codex CLI 和 Copilot CLI 的实际命令、参数和结构化输出协议。
%% NEEDS_INPUT: FD 阶段需验证不同操作系统上的 GUI 检测、tkinter 可用性和 TUI 回退。
%% NEEDS_INPUT: FD 阶段需确认 OpenAI Responses API 的请求与 JSON Schema 响应格式。

## Recommended next action

创建编号 FD，设计 Python 插件入口、CLI provider 适配器、OpenAI HTTP 适配器、
GUI/TUI 分层、locale 解析和兼容发布流程。
