# FD-005: 将 aiw-cz 从 TypeScript 迁移到 Python

**Status:** Pending Verification
**Revision:** 3
**Priority:** Medium

## Problem

`aiw-cz` 当前依赖 TypeScript、Node.js 和三个 JavaScript SDK。目标环境希望减少运行时依赖，并直接复用目标机器上已安装的 Codex CLI、Copilot CLI。现有 `aiw cz` 的命令、配置、国际化、人工审阅和提交行为必须保持兼容。

## Options and decision

### 方案 A：继续 TypeScript 和 SDK

改动最小，但继续携带 Node.js、npm 依赖和平台 SDK 资源，不满足减少依赖和直接调用 CLI 的目标。

### 方案 B：Python + CLI/HTTP（选定）

迁移到 Python；Codex 和 Copilot 通过 `subprocess` 调用 CLI；OpenAI 通过 Python 标准库 HTTP 调用 Responses API；交互层只使用标准库 TUI。

### 方案 C：Python + SDK

仍需安装 Python SDK，不能直接复用 CLI 登录和配置，依赖更多，因此不选。

## Solution

Python 插件入口替换现有 `aiw-cz.js`，保持 `aiw cz` 的参数和插件发现契约。内部划分为配置/locale、Git 上下文、LLM provider、候选校验、交互界面和提交审阅模块。

LLM 回退顺序固定为：Codex CLI → Copilot CLI → OpenAI HTTP API → 交互式 TUI。provider 失败、超时、CLI 不存在、HTTP 错误或候选无效时进入下一个 provider。全部失败后进入 TUI。所有候选在 TUI 预览并允许编辑或取消；用户接受后自动执行本地 commit，回车默认接受。插件不执行 push。

OpenAI 使用标准库 HTTP 和 JSON，不引入 OpenAI SDK。`requirements.txt` 只列出无法由标准库替代的依赖；若不需要第三方包则不安装第三方包。

配置优先级、provider 配置、CLI 参数和环境变量保持现有语义。显式语言配置优先；未配置时读取操作系统 locale；没有对应翻译时使用英语。

## Scope

- Python 插件入口和内部模块。
- Codex CLI、Copilot CLI、OpenAI HTTP provider 适配器。
- JSON 候选解析、校验、回退和错误提示。
- TUI 交互向导、locale 选择和现有配置兼容。
- 构建、发布和安装脚本从 Node.js 产物迁移到 Python 插件发布内容。
- 更新受影响的稳定规格和使用文档。

不包含：Go AI provider 工作流、`aiw cz` 命令名称、用户配置格式的无关重设计、GUI 框架、自动 push。

## Work items

- [x] 1.1 固化 Python 插件入口、配置兼容层和 locale 解析；验收：现有 `aiw cz` 参数、配置优先级和系统语言回退均有静态证据。已新增 `plugins/aiw-cz/aiw-cz.py` 与 `cz_config.py`，使用标准库解析配置和 locale。
- [x] 1.2 实现 Git staged context、提示构造、候选 JSON 解析和 Conventional Commit 校验；验收：无效候选不会跳过人工审阅。已新增 `plugins/aiw-cz/cz_core.py`。
- [x] 1.3 实现 Codex CLI 和 Copilot CLI 适配器；验收：记录实际 CLI 命令、参数、超时、退出码、stdout/stderr 和结构化输出兼容策略。已由 `cz_providers.py` 接入 `cz_llm.py`。
  设计决策：Codex 从配置或 PATH 自动发现，Copilot 仅在显式设置
  `CZ_COPILOT_COMMAND` 后探测；未配置时直接跳过，不引导安装。适配器调用
  版本命令记录版本，随后读取 help/能力信息，
  选择可用的非交互模式。默认要求 provider 输出一个 JSON 对象，格式为
  `{"candidates":[{"type":"...","scope":"...","subject":"...","body":"","breaking":"","footer":""}]}`。
  Codex 使用 `exec --sandbox read-only --json -`，通过 stdin 提交提示，并优先读取
  `--output-last-message` 的最终文本；不可用时解码 JSONL 中最后一条 agent_message。
  Copilot 使用 `--output-format text --prompt`，从模型文本提取候选 JSON。
  stderr 只用于诊断；解析后仍执行字段、类型、长度和 issue 引用校验。CLI 不支持
  结构化输出时，可以请求纯文本并由适配器提取 JSON，但不得把未经校验的文本当作
  候选。版本未知或能力探测失败时按命令失败处理并回退下一个 provider。
- [x] 1.4 实现 OpenAI Responses HTTP 适配器；验收：支持 API key、model、base URL、超时、HTTP 错误和结构化候选响应，不依赖 OpenAI SDK。已新增 `plugins/aiw-cz/cz_openai.py`。
- [x] 1.5 实现 Codex → Copilot → OpenAI → 交互式的回退编排；验收：provider 失败或候选无效时按顺序回退，显式 provider 失败时不调用其他 provider。已新增 `plugins/aiw-cz/cz_llm.py`。
- [x] 1.6 实现 TUI 交互向导；验收：候选在 TUI 预览，可编辑或取消；接受后自动本地 commit，不执行 push。已更新 `plugins/aiw-cz/cz_ui.py`。
- [x] 1.7 替换构建、发布和插件入口；验收：发布内容不再要求 Node.js/npm，Python、locale 和 requirements 安装路径清晰。已更新 `build.bat` 并新增 `plugins/aiw-cz/requirements.txt`。
- [x] 1.8 更新 CZ 配置规格、Python runtime 规格及使用文档；验收：稳定规格与 Python/CLI/HTTP 行为一致。

### CLI capability detection

- Codex 命令可由环境变量覆盖，未配置时使用 PATH 查找；Copilot 只接受显式配置的 `CZ_COPILOT_COMMAND`，未配置时跳过。
- 检测必须是只读的：版本/help/能力查询不得修改仓库、提交文件或启动交互式会话。
- CLI 探测和调用不得读取交互式输入；不可用时不能弹出安装提示。
- 每次调用设置超时、工作目录和只读权限语义；捕获退出码、stdout、stderr 和超时状态。
- 适配器保存版本和选择的调用模式到诊断信息，但不得记录 API key 或完整敏感环境变量。
- CLI 参数由 provider 适配器集中维护，不能散落在业务候选逻辑中。
- JSON 契约由 AIW 定义，CLI 的原生 JSON/JSONL 输出只作为输入格式；最终必须转换
  为统一候选结构后再进入公共校验器。

## Acceptance

- `aiw cz` 的外部命令、参数、配置和 locale 行为保持兼容。
- LLM 顺序为 Codex CLI → Copilot CLI → OpenAI HTTP → TUI。
- provider 不存在、失败、超时或返回无效候选时可解释地回退。
- 未安装、版本未知或不支持非交互结构化输出的 CLI 会被视为该 provider 不可用，
  并自动继续回退。
- 交互始终使用 TUI，不依赖 `tkinter` 或显示环境。
- 系统语言无对应翻译时使用英语。
- 所有候选均在 TUI 预览；接受后自动执行本地 Git commit，插件不执行 push。
- 不再要求 Node.js、npm 或三个 JavaScript SDK。

## Verification

- Design only：静态阅读现有 `plugins/aiw-cz`、Python 插件执行器、GUI 参考插件、构建脚本和相关稳定规格。
- 1.1：运行 Python compile-only 检查通过；未运行插件、GUI、CLI provider 或提交流程。
- 1.1 locale 跟进：Windows 默认语言改读用户区域设置（与 `Get-Culture` 对应），配置文件的 `i18n.default_language` 按现有文件优先级由后者覆盖前者；未在本轮运行插件。
- 1.2：新增 staged diff/history 读取、统一 JSON 候选结构和字段/类型/长度/Issue 引用校验。
- 1.3：新增 `cz_providers.py`，支持 PATH/环境变量命令发现、版本/help 探测、超时、退出码和统一 JSON 提取，并已接入回退编排。
- 1.3 跟进：Copilot 改为仅显式配置后探测，探测和调用关闭标准输入；未配置时直接进入下一 provider。本轮只做静态检查，未运行真实 CLI。
- 1.3 Go 对照修复：Codex 改为 stdin 提示、read-only sandbox、最终消息文件或 JSONL agent_message 解码；Copilot 改为文本输出并提取候选 JSON；两个 CLI 均传入 provider model。本轮未运行真实 LLM。
- 1.4：新增标准库 `urllib` Responses API 客户端和 JSON Schema；未执行网络请求。
- 1.5：三个 provider 已接入统一顺序，失败后进入 TUI。
- 1.6：移除 GUI 路径；TUI 统一预览、编辑、取消与默认接受，接受后本地 Git commit。未在本轮运行交互式流程。
- 1.7：构建脚本已改为准备和安装 Python 发布目录；未运行构建或安装流程。
- 1.8：已同步 `openspec/specs/cz-configuration-priority/spec.md`，新增 `openspec/specs/cz-python-runtime/spec.md` 和 Python runtime 使用说明。
- Review fixes：保留安装目录中的 `cz.toml`/`.cz.toml`；修复 OS locale 选择、staged 前置检查、`--retry`、多候选选择和 CLI help 能力门控；同步 CZ 插件规格入口。
- FD Review：`docs/features/reviews/FD-005-review.md`；独立静态审查结论 `CHANGES_REQUESTED`。未发现有效 Worker `implementation-ready` 移交，因此未发出生命周期事件；FD 保持 Pending Verification。

## Review follow-up gate

- 静态修复已处理：用户配置保留、系统 locale、staged 前置检查、`--retry`、多候选选择、CLI help 能力门控和规格入口同步。
- `%% NEEDS_INPUT: 目标环境中 Codex/Copilot CLI 的实际版本、非交互参数和 JSON/JSONL 输出证据。`
- `%% NEEDS_INPUT: 目标环境中 Python 发布安装、TUI 和用户配置保留的运行证据。`
- 在上述证据完成并由独立 Reviewer 复核前，FD 不得进入 Verification Passed 或 Complete。
- 本轮未运行测试、构建、CLI provider、TUI 或 OpenAI 请求验证。
- 实施阶段按单个 Work Item 执行 compile-only 检查；provider 和 TUI 的运行验证需记录结果。

## Sources

- Issue: REQ00004-aiw-cz-python-cli
- `plugins/aiw-cz/src/index.ts`
- `plugins/aiw-cz/src/llm.ts`
- `plugins/aiw-cz/src/config.ts`
- `plugins/aiw-cz/aiw-cz.js`
- `internal/plugin/discover.go`
- `internal/plugin/exec.go`
- `plugins/aiw-cxs.py`
- `openspec/specs/cli-and-plugins/spec.md`
- `openspec/specs/cz-configuration-priority/spec.md`
