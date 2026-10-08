# FD-005: 将 aiw-cz 从 TypeScript 迁移到 Python

**Status:** Complete
**Revision:** 10
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
- [x] 1.3 实现 Codex CLI 和 Copilot CLI 适配器；验收：记录实际 CLI 命令、参数、超时、退出码、stdout/stderr 和结构化输出兼容策略。静态核对 `cz_providers.py` 的命令发现、版本/help 探测、只读调用、超时、结果捕获和 JSON 提取，并确认接入 `cz_llm.py`；用户决定不运行此项测试。
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
- [x] 1.4 实现 OpenAI Responses HTTP 适配器；验收：支持 API key、model、base URL、超时、HTTP 错误和结构化候选响应，不依赖 OpenAI SDK。静态核对 `cz_openai.py` 的配置、JSON Schema、请求超时与错误处理，并确认使用标准库；用户决定不运行此项测试。
- [x] 1.5 实现 Codex → Copilot → OpenAI → 交互式的回退编排；验收：provider 失败或候选无效时按顺序回退，显式 provider 失败时不调用其他 provider。`cz_llm.py` 的顺序、异常捕获与显式 provider 分支已有逐项静态证据，见 Verification。
- [x] 1.6 实现 TUI 交互向导；验收：候选在 TUI 预览，可编辑或取消；接受后自动本地 commit，不执行 push。用户提供 `go run cmd\aiw\main.go cz --no-llm` 的目标环境输出：TUI 录入、预览 `fix(cz): fix cz`，回车接受后创建本地提交 `879a072`。编辑、取消和不执行 push 由 `cz_ui.py` 的分支与提交调用静态核对。
- [x] 1.7 替换构建、发布和插件入口；验收：发布内容不再要求 Node.js/npm，Python、locale 和 requirements 安装路径清晰。用户提供 `build cz`、`build plugins` 输出；只读核对 `c:\green\aiw\plugins\aiw-cz` 已安装 Python 入口、模块、`locales/` 和 `requirements.txt`。插件发现优先选 `.py`。已有用户配置保留尚缺目标环境样本，作为独立验收风险保留。
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
- 1.4 评审修复：`cz_openai._text` 对非对象顶层、非数组 `output` 和非数组 `content` 做形状检查；无效响应转为 `OpenAIUnavailable`，由 `cz_llm.candidates` 的现有异常分支回退。仅静态复核，不运行 HTTP 测试。
- 1.5：`cz_llm.candidates` 未指定 provider 时依次尝试 `codex`、`copilot`、`openai`；指定时列表只含所选 provider。`run_cli`、`openai_generate` 或 `parse_candidates` 抛出的 provider/校验异常被捕获后继续下一项；列表耗尽返回 `None`，`aiw-cz.py` 随后进入 `wizard`，故无效候选与显式 provider 失败均不会直接提交。此项以静态调用路径验收，未运行真实 provider。
- 1.6：移除 GUI 路径；TUI 统一预览、编辑、取消与默认接受，接受后本地 Git commit。用户提供的目标环境输出证明 `--no-llm` 进入 TUI、显示预览并在默认接受后创建本地提交 `879a072`（1 file changed, 8 insertions, 6 deletions）；编辑、取消和无 push 路径经静态核对。本会话未重复运行交互式流程。
- 1.7：用户提供 `build cz` 的发布目录准备输出，以及 `build plugins` 的两段 ROBOCOPY 输出（失败数均为 0）。本会话只读核对 `c:\green\aiw\plugins\aiw-cz` 含 Python 入口、模块、`locales/` 和 `requirements.txt`；`internal/plugin/discover.go` 优先选 `.py`，`internal/plugin/exec.go` 为 `.py` 选择 Python。安装目录仍有旧 JavaScript 文件，但无需 Node.js 启动当前入口。`build.bat` 用 `/E` 而非镜像删除，并用 `/XF cz.toml .cz.toml` 排除覆盖，静态支持已有用户配置保留；目标目录无配置样本，未做重新安装的运行验证。本会话未运行构建或安装。
- 1.8：已同步 `openspec/specs/cz-configuration-priority/spec.md`，新增 `openspec/specs/cz-python-runtime/spec.md` 和 Python runtime 使用说明。
- Review fixes：保留安装目录中的 `cz.toml`/`.cz.toml`；修复 OS locale 选择、staged 前置检查、`--retry`、CLI 参数契约、多候选选择和 CLI help 能力门控；同步 CZ 插件规格入口。静态修复已复核通过，运行证据仍待补。
- 本轮跟进：CLI 版本和 help 必须各自成功且有输出；Codex 必须声明 `--sandbox` 和 `--json`，Copilot 必须声明 `--output-format` 和 `--prompt`，否则按 provider 不可用回退。`docs/usage/cz-configuration.md` 已改为 Python/CLI/HTTP 发布与配置说明。用户已明确跳过 1.3 和 1.4 的运行测试；真实 CLI 版本兼容性与 HTTP 响应仍未经运行验证。
<<<<<<<< HEAD:docs/features/archive/FD-005_AIW_CZ_TYPESCRIPT_PYTHON.md
- FD Review：`docs/features/reviews/FD-005-review.md` 和 `docs/features/reviews/FD-005-review-r2.md` 的 `CHANGES_REQUESTED` 是修订前结论。其后已补入 1.6/1.7 用户证据、1.5 逐项静态证据，并记录用户不运行 1.3/1.4 测试的决定；修订后的 FD 仍需独立 Reviewer 重新判断，不能把旧报告当作通过。
========
- FD Review：`docs/features/archive/FD-005/reviews/FD-005-review.md` 和 `docs/features/archive/FD-005/reviews/FD-005-review-r2.md` 的 `CHANGES_REQUESTED` 是修订前结论。其后已补入 1.6/1.7 用户证据、1.5 逐项静态证据，并记录用户不运行 1.3/1.4 测试的决定；修订后的 FD 仍需独立 Reviewer 重新判断，不能把旧报告当作通过。
>>>>>>>> main:docs/features/archive/FD-005/FD-005_AIW_CZ_TYPESCRIPT_PYTHON.md

## Review follow-up gate

- 当前验收依据为上面的静态调用路径、用户提供的 TUI/安装输出，以及用户明确跳过 1.3/1.4 运行测试的决定。旧评审报告中的运行测试要求已被此决定取代；Reviewer 仍需核对实现是否满足行为要求。
- `%% RISK: Codex/Copilot CLI 的目标版本、非交互参数和 JSON/JSONL 输出，以及 OpenAI HTTP 实际响应未做运行验证；用户已明确跳过 1.3 和 1.4 测试。`
- `%% RISK: 用户配置保留由 build.bat 的非镜像复制和 /XF 排除规则静态支持；目标目录没有既存配置样本，未验证重新安装。`
- 在独立 Reviewer 对当前 FD 与报告给出 `verification-passed` 前，FD 不得进入 Complete。
- 本会话未运行测试、构建、CLI provider、TUI 或 OpenAI 请求验证；TUI 证据由用户提供。
- 实施阶段按单个 Work Item 执行 compile-only 检查；独立复核仍需记录结果。
<<<<<<<< HEAD:docs/features/archive/FD-005_AIW_CZ_TYPESCRIPT_PYTHON.md
- 独立复核 r2（修订前）：`docs/features/reviews/FD-005-review-r2.md`，结论 `CHANGES_REQUESTED`。本次已更新其指出的过期运行门槛和 1.5 证据缺口；尚无新的生命周期评审结论。
- 独立复核 r3：`docs/features/reviews/FD-005-review-r3.md`，结论 `CHANGES_REQUESTED`。已接受 1.3/1.4 的免测决定；`cz_openai.py` 对合法 JSON 非对象响应抛出未捕获异常，阻断无效候选回退 TUI。修复后需重新评审。
- 独立复核 r4：`docs/features/reviews/FD-005-review-r4.md`，结论 `VERIFICATION_PASSED`。r3 的异常响应回退缺口已静态复核修复；1.3/1.4 免测及配置保留运行验证缺失继续作为上述残余风险记录。
========
- 独立复核 r2（修订前）：`docs/features/archive/FD-005/reviews/FD-005-review-r2.md`，结论 `CHANGES_REQUESTED`。本次已更新其指出的过期运行门槛和 1.5 证据缺口；尚无新的生命周期评审结论。
- 独立复核 r3：`docs/features/archive/FD-005/reviews/FD-005-review-r3.md`，结论 `CHANGES_REQUESTED`。已接受 1.3/1.4 的免测决定；`cz_openai.py` 对合法 JSON 非对象响应抛出未捕获异常，阻断无效候选回退 TUI。修复后需重新评审。
- 独立复核 r4：`docs/features/archive/FD-005/reviews/FD-005-review-r4.md`，结论 `VERIFICATION_PASSED`。r3 的异常响应回退缺口已静态复核修复；1.3/1.4 免测及配置保留运行验证缺失继续作为上述残余风险记录。
>>>>>>>> main:docs/features/archive/FD-005/FD-005_AIW_CZ_TYPESCRIPT_PYTHON.md

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

**Completed:** 2026-10-01
