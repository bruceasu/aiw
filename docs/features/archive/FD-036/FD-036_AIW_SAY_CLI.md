# FD-036: AIW Say 核心 CLI 翻译闭环

**Status:** Complete
**Revision:** 36
**Priority:** Medium  
**Test policy:** Independent  
**Evidence policy:** Dual

## Problem

`docs/aiw-say.md` 要求日常使用的中、日、英文本翻译工具，并明确先完成第一阶段的可运行闭环。当前 `aiw` 通过 `aiw-<name>` 分派外部插件，但仓库尚无 `aiw say`。直接实施整份 V1 文档会同时触及服务请求、跨平台剪贴板、AHK、Zenity 和分发，难以逐项评审。本 FD 先交付无图形 CLI 核心，为后续适配器提供稳定边界。

## Options and decision

| 方案 | 判断 |
| --- | --- |
| 在 `cmd/aiw/main.go` 增加内置 `say` 分支 | 绕过现有插件分派约定，扩大主程序改动。 |
| 新建独立 Go 模块 | 重复仓库模块与依赖管理。 |
| 在现有 Go 模块下提供 `cmd/aiw-say` 可执行插件 | 复用 `aiw-<name>` 约定，保持主程序边界；选用此方案。 |

首个服务适配器使用 OpenAI API。定义小型翻译提供方接口，以便后续接入 Codex CLI；本 FD 不实现第二个提供方。按用户 2026-10-08 的决定，基础配置改用程序安装目录的 `aiw.toml`，profile 改用用户配置目录中的独立 TOML 文件；这取代源文档最初的 YAML 方案。模型名由配置或参数提供，不在源码中猜测默认可用模型。

按用户对未知键及 TOML 语法的决定，并响应 Reviewer r3，使用 `github.com/BurntSushi/toml` 解析完整文档。完整语法错误报错退出；未知键忽略；已知配置字段类型或取值无效时不覆盖较低优先级值。

## Solution

- `aiw say` 由现有插件发现机制启动 `aiw-say`。核心翻译逻辑放在独立 Go 包；CLI 只处理输入、配置、输出与退出状态，不要求图形会话。
- 接受单个位置文本或 stdin；二者同时提供时明确拒绝，空输入报错。stdin/stdout 使用 UTF-8；成功时 stdout 只有译文，失败时 stderr 给出错误、stdout 为空并以非零状态退出。
- 基础配置读取程序安装目录的 `aiw.toml` 中的 `[say]` 与 `[say.llm]`；`--config` 可显式指定另一份基础 TOML，指定文件不存在或无效时报错。缺失默认文件时采用内置翻译默认值。
- profile 读取 Windows `%APPDATA%/aiw/profiles/<name>.toml`；Linux/WSL 读取 `$XDG_CONFIG_HOME/aiw/profiles/<name>.toml`，若未设置 XDG 则读取 `~/.config/aiw/profiles/<name>.toml`。只允许安全的 profile 名，禁止路径分隔符或目录穿越。发行物预置 `ja-business`、`ja-teams`、`en-simple`、`en-document` 的示例文件，用户按需复制到上述目录；程序不得自动覆盖用户文件。合并顺序：内置默认值、安装目录 `aiw.toml`、选定的用户 profile、显式 CLI 参数。选定 profile 不存在或无效时报错；API 凭据只从环境变量读取。
- 支持 `zh`、`ja`、`en` 和 `auto` 源语，目标缺省 `ja`，实时模式缺省。参数包括 `--target/-t`、`--source/-s`、`--mode`、`--style`、`--polite/-p`、`--simple`、`--profanity`、`--profile`、`--provider`、`--model`、`--timeout`、`--config`、`--help`、`--version`。校验枚举、输入冲突和未知提供方。
- 提示词固定翻译规则；运行配置作为可信参数，源文本作为独立的不可信用户内容传递。只返回翻译结果，不回答源文本中的问题或执行指令；保留数字、标识符、URL、命令、专有名词与段落结构。模糊源语言交给提供方按提示词检测，不增加本地猜测器。
- OpenAI 适配器通过 HTTPS 请求，使用上下文超时和有界瞬时错误重试；认证、限流、超时、无效响应及缺少凭据给出不泄露文本或密钥的错误。重试不得产生部分 stdout。具体 API 调用及模型可用性在实现时依据当时的官方文档核对。
- `--clipboard`、`--copy`、`--file`、`--output`、`--dialog`、`--pair`、术语表、Codex CLI 适配器、AHK、Zenity、剪贴板及安装包属于后续 FD；遇到未实现选项须明确报错。源文档的 AHK v1 标题与 v2 正文冲突，后续按正文的 v2 要求设计。

兼容性：保持现有 `aiw` 命令和插件分派、stdout/stderr 与退出码契约；只新增 `say` 插件及支持文件。隐私：默认不持久化或记录原文、译文、凭据；请求只发送给显式配置的翻译提供方。

## Scope

- 包含：第一阶段的文本参数/stdin → 配置及 profile → 翻译提示词 → OpenAI 提供方 → stdout 闭环、确定性模拟提供方测试、配置样例与使用说明。
- 不包含：原文第二至四阶段的 GUI、剪贴板、文件模式、双向语言对、术语表、打包与真实 LLM 质量评估。后续 FD 继续以 `docs/aiw-say.md` 的 V1 目标为依据。

## Work items

- [x] 1.1 建立 `cmd/aiw-say` 插件入口与核心包边界。Size: S；difficulty: Low；dependencies: none；完成标准：入口接收参数，主程序不变。
- [x] 1.2 实现位置参数/stdin、UTF-8 文本和互斥/空输入校验。Size: S；difficulty: Low；dependencies: 1.1；完成标准：合法输入形成统一请求，错误输入不调用提供方。
- [x] 1.3 实现安装目录 `aiw.toml` 的 `[say]` 配置、`--config` 覆盖、内置默认值、TOML 解析与字段校验。Size: M；difficulty: Medium；dependencies: 1.1；完成标准：缺省、有效及无效文件行为明确，配置样例不含凭据。
- [x] 1.4 实现用户目录 profile 查找、名称安全校验、CLI 覆盖顺序及参数枚举/模型/提供方/超时校验。Size: M；difficulty: Medium；dependencies: 1.2, 1.3；完成标准：四层优先级和缺失/无效 profile 行为可独立检查。
- [x] 1.5 实现翻译请求结构、稳定核心提示词和独立源文本消息。Size: S；difficulty: Medium；dependencies: 1.4；完成标准：源文本内伪指令不进入可信配置，风格及保留规则可检查。
- [x] 1.6 实现 OpenAI 适配器、凭据读取、超时、错误分类和有界重试。Size: M；difficulty: Medium；dependencies: 1.5；完成标准：适配器只返回完整译文或错误，诊断不泄露敏感内容。
- [x] 1.7 串联 stdout、stderr、退出码、`--help`、`--version`，并拒绝未实现选项。Size: S；difficulty: Low；dependencies: 1.2, 1.4, 1.6；完成标准：成功 stdout 仅译文，失败无部分译文。
- [x] 1.8 已取消：该项将 Worker 编写模拟黑盒代码与独立 Tester 场景盘点合在一起；拆分为 1.8.1 和 1.8.2，分别保留代码交付及独立验证职责。
- [x] 1.8.1 编写根目录确定性模拟 API 黑盒用例，覆盖位置/stdin 输入、UTF-8、提示词边界、认证失败、限流重试及未实现选项。Size: M；difficulty: Medium；dependencies: 1.7；完成标准：Python 黑盒用例通过本地模拟 HTTP 服务，不需要真实凭据或外部网络；已编写，未执行。
- [x] 1.8.2 已取消：该项重复描述 Independent Test policy 下必需的 Pending Test 阶段交付，且无法在 Worker handoff 前完成。独立 Tester 的场景清单和覆盖报告仍是后续 Tester 阶段必需证据，并未豁免。
- [x] 1.9 编写安装目录 `aiw.toml` 示例、四个独立 profile TOML 示例和第一阶段 README，说明构建、插件安装、profile 复制位置、环境变量、限制与故障排查。Size: S；difficulty: Low；dependencies: 1.7；完成标准：示例字段与实际解析一致，不覆盖已有用户配置，文档标明后续 V1 功能尚未交付。
- [x] 1.10 根据首轮 Tester 场景清单扩充确定性黑盒用例，覆盖配置/profile/CLI 优先级、无效输入、凭据和响应错误、重试耗尽以及帮助/版本输出，并将各验收场景映射到具体用例。Size: M；difficulty: Medium；dependencies: 1.8.1；完成标准：每项可运行验收行为都有明确用例或书面排除理由；本轮仅编写并静态检查，执行仍由获授权的独立 Tester 负责。
- [x] 1.11 接受合法 TOML 表头后的行内注释，并添加配置回归用例。Size: S；difficulty: Low；dependencies: 1.3, 1.10；完成标准：`[say] # ...` 与 `[say.llm] # ...` 加载成功并应用设置；普通非法表头仍报错。
- [x] 1.12 按用户决定使无效配置字段值回退到较低优先级有效值、最终内置默认值，并忽略未知键；保留损坏 TOML 与无效 CLI 参数的报错。Size: M；difficulty: Medium；dependencies: 1.3, 1.4, 1.8.1；完成标准：黑盒用例验证基础配置/profile 回退、未知键忽略、损坏 TOML 报错以及无效 CLI 参数仍失败；核心用例已由 Tester r4 执行。
- [x] 1.13 使用符合 TOML 语法的解析器读取完整文档，再按字段类型应用 `[say]`、`[say.llm]` 和 `[profile]` 配置；未知键跳过，错误类型不覆盖低优先级有效值。添加未知数组键和 TOML 非法转义回归。Size: M；difficulty: Medium；dependencies: 1.12；完成标准：合法未知数组忽略、Go 专有转义导致格式错误退出、已有回退语义保持；用例由独立 Tester 执行。
- [x] 1.14 为 Tester r5 的 S04 `ERROR` 准备聚焦诊断 handoff：命令须保存完整 stdout/stderr 并输出退出码，先取得用户明确批准，再绑定当前 FD 修订与独立 Tester session。Size: S；difficulty: Medium；dependencies: 1.13；完成标准：诊断命令、范围、副作用与用户批准已记录，交由独立 Tester 按精确授权执行一次。

## Acceptance

- 在插件可发现的安装布局执行 `aiw say -t ja "你好"` 或从管道输入中文，可获得日文译文；正常输出不含说明文字。真实 API 效果须在获授权的联网验证后才能宣称通过。
- 安装目录 `aiw.toml` 和用户目录 profile 可选择语言、模式、风格、礼貌等级、粗话处理及简化选项；显式 CLI 值覆盖选定 profile，profile 覆盖安装配置，安装配置覆盖内置默认值。配置字段值无效时忽略该值并保留较低优先级有效值，最终使用内置默认值；未知键忽略。损坏 TOML 语法报错退出。四个预置示例可复制后通过 `--profile` 选择；不得覆盖已有用户文件。
- 源文本包含换行、引号、Unicode、命令或伪指令时，原文作为数据传递，不作为程序命令或可信提示词；原文和密钥不进入默认日志。
- 缺失密钥、损坏 TOML、缺失 profile、不安全的 profile 名、空输入、无效 CLI 选项、认证失败、超时、限流与无效响应均给出非零状态和可理解的 stderr；失败时 stdout 为空。配置字段无效值回退默认值，未知配置键忽略。
- 无图形环境可使用 CLI；未实现的剪贴板、文件、对话框等选项明确报错。
- 根目录 `tests/` 的模拟提供方用例无需联网；实际执行结果须由授权的独立 Tester 记录。

## Verification

- 设计阶段：静态核对 `cmd/aiw/main.go`、`internal/plugin/discover.go` 与 `openspec/specs/cli-and-plugins/spec.md` 的插件契约；未执行测试。
- Worker：静态检查安装目录解析、用户 profile 目录及名称校验、配置优先级、参数到提示词/提供方/stdout 的调用路径、错误分支、敏感数据处理和最终 diff；执行 `go build -o NUL ./cmd/aiw-say`，首次因请求礼貌字段类型错误失败，修正后相同命令通过，未保留发布产物。
- Worker 处理首轮测试拒绝：扩展 `tests/test_fd036_say_blackbox.py`，新增空输入、配置/profile/CLI、参数校验、凭据和响应错误、超时、重试耗尽、help/version 等用例，并加入 S01–S15 场景映射。静态检查后提交 Worker 报告；执行证据由后续独立 Tester 报告为准。
- Worker 处理 Reviewer r1：`internal/say/config.go` 在验证 TOML 表头前先剥离行内注释，并将 `[say] # translation settings` 与 `[say.llm] # provider settings` 加入现有显式配置回归用例。运行 `go build -o NUL ./cmd/aiw-say` 通过；Python 行为测试留给新实现版本的独立 Tester。
- 独立 Tester（历史 r2）：针对实现交接 `FD-036-000014-implementation-ready`（Revision 14，digest `914ae9b1043fa69226b2c7e00fed2ff4c2efe7b7e9ffad7f84c56ae8f83ad537`）记录 Planner 授权 r4 命令。修正测试夹具后，黑盒用例 19 项通过、0 项行为失败；报告及其覆盖数字详见 `docs/features/reports/FD-036-test-report-r2.md`，授权：`docs/features/reports/FD-036-test-authorization-r4.md`。仅使用 loopback mock，没有真实 API/网络请求。
- PM 测试决策：`docs/features/reports/FD-036-test-decision-r2.md` 采用 `adaptive-v1/escalated`。A/C 两票 `accept-with-risk`，B 一票 `repair`；接受进入 Reviewer，并明确保留 S05/S08/S09/S14/S15、Linux/WSL profile 路径及 branch coverage 的验证例外。未覆盖行为未计作通过。
- PM 测试决策 r3：事件 `FD-036-000021-test-report-ready` 的三份有效独立评估为两票 `repair`、一票 `accept-with-risk`，决策 `rejected`（事件 `FD-036-000022-test-rejected`）。19/19 行为测试通过但 11/16 场景完整覆盖；用户随后明确配置值无效时回退到较低优先级值/内置默认值，未知键忽略，损坏 TOML 报错退出。现有 r3 测试证据未验证该更新后的配置行为。
- Worker 当前变更：PM r3 拒绝事件 `FD-036-000022-test-rejected` 后，用户澄清合法 TOML 中无效字段值回退到低优先级有效值/内置默认值、未知键忽略、损坏 TOML 报错退出。Work Item 1.12 将 `applyConfig` 改为逐字段校验并只应用有效值；最终 CLI 校验保持不变，因此非法 CLI 参数仍报错。黑盒回归新增基础配置默认回退、profile 低优先级回退、未知键和语法损坏案例；这些用例已由独立 Tester r4 执行并通过。`go build -o NUL ./cmd/aiw-say` 通过，未保留构建产物。
- 独立 Tester r4：对实现事件 `FD-036-000023-implementation-ready`（Revision 23，digest `ec7adbbb1e50c75d8e95834321a3f5d71f0765299d3934ddcee80694851f1c27`）执行 Planner r6 授权命令。黑盒用例 21 项通过、0 项行为失败，18 个场景中 14 个完整覆盖（77.78%）；branch coverage 未测量。报告：`docs/features/reports/FD-036-test-report-r4.md`；授权：`docs/features/reports/FD-036-test-authorization-r6.md`。仅使用 loopback mock，没有真实 API/网络请求。
- PM 测试决策 r4：事件 `FD-036-000024-test-report-ready` 经三份独立评估后以三票 `accept-with-risk` 接受（事件 `FD-036-000025-test-accepted`），进入 Reviewer。显式保留 S08/S09/S14/S15 覆盖缺口及 branch coverage 未测量；实际 API、跨平台行为及翻译质量未由 mock 证明。
- Reviewer r1：对 `FD-036-000017-review-requested` 的报告为 `docs/features/reviews/FD-036-review-r1.md`，结果 `changes-requested`（事件 `FD-036-000018-changes-requested`）。发现合法 `[say] # comment` 表头被拒绝，关联 Work Item 1.3 与 TOML 配置 Acceptance；本轮按新 Work Item 1.11 修复并加回归断言。该缺陷不属于 PM 已接受的测试覆盖例外。
- Reviewer r2：修复并完成 Tester/PM 收据后重新请求独立审查；当前已由 r4 测试验收，等待基于最新 FD 修订的独立 Reviewer 结果。
- Reviewer r3：审查事件 `FD-036-000026-review-requested`（revision 26，digest `7e4977b143e589f74c2e18fd7547e4a5ed3aadab866283f782c77477bdc34232`），报告 `docs/features/reviews/FD-036-review-r3.md`，结果 `changes-requested`（事件 `FD-036-000027-changes-requested`）。发现自定义 TOML 标量解析器拒绝合法未知键值且接受 TOML 不允许的 Go 字符串转义；需修复解析与过滤顺序，再按独立 Tester 授权流程验证。
- Worker r6：用户授权新增 TOML 解析依赖并获取。Work Item 1.13 以 BurntSushi/toml 完整解析文档，将已知表抽取为类型化配置值；合法未知值不做字段转换，未知键忽略；已知字段类型不符或值不合法时不覆盖低优先级配置。黑盒回归增加合法数组型未知键和 TOML 不支持的 `\\a` 转义案例。`go build -o NUL ./cmd/aiw-say` 在关闭模块代理后通过；黑盒测试未运行，需新版本独立 Tester 获得 Planner 授权。
- Worker compile-only：`GOPROXY=off GOSUMDB=off go build -o NUL ./cmd/aiw-say` 通过；未运行行为测试、最终构建或格式化工具。
- 独立 Tester r5：事件 `FD-036-000029-implementation-ready`（revision 29，digest `64a3b0ec060f0d1d492f6ac3e8acf7d4c3ad20184449064c0b6962596051f522`）按 Planner r7 授权命令执行一次，20 项通过、1 项 `ERROR`；`test_missing_default_config_uses_defaults_and_requires_explicit_model` 未提供 traceback、最终摘要或退出码，S04 标为 blocked。场景覆盖 13/18（72.22%），branch coverage 未测。报告：`docs/features/reports/FD-036-test-report-r5.md`。未重跑。
- PM 测试决策 r5：对事件 `FD-036-000030-test-report-ready`（revision 30，digest `8fc52ef381ac87a4932b0784c36a338ac540785cb732fd479e3d719ec7e05510`）进行 `adaptive-v1/escalated` 评估；A/B/C 三票均为 `repair`，拒绝测试验收（事件 `FD-036-000031-test-rejected`）。S04 ERROR 未归因且不能视为通过；S08/S09/S14/S15、branch coverage、真实 API/平台行为及翻译质量风险继续保留。
- Worker r7：处理事件 `FD-036-000031-test-rejected`。静态追踪 S04 默认文件缺失、内置默认配置及缺少 model 错误路径，未能解释原始 `ERROR`，未重跑。用户批准一次仅运行 S04 的诊断命令；命令把完整输出写入 `%TEMP%` 唯一日志并显式输出退出码。阻塞记录 `docs/features/reports/FD-036-blocker-test-diagnostic-20261008T093704Z.md` 已更新为已解决；新的 Tester 结果仍待本次 handoff。
- 独立 Tester r6 / PM r6：事件 FD-036-000033-test-report-ready（revision 33，digest 54862b4294872e269be7ea94f4b8dd4d3467b9e2e4b50a85d69776f306dd05e0）仅重跑 S04 指定用例，1/1 通过、退出码 0；r5 原始 ERROR 未复现但原因未知，未重跑完整套件。A/B/C 三票 ccept-with-risk，PM 决策 docs/features/reports/FD-036-test-decision-r6.md 以重大证据缺口升级评估后接受风险，进入独立 Reviewer；S08/S09/S14/S15、branch coverage、真实 API/平台行为和翻译质量风险保留。

## TODO

- [x] 完成 Worker 负责的实现 Work Items，并在独立提交中记录；1.8.2 留给 Pending Test 阶段。
- [x] 根据实现更新 Verification、剩余风险和后续 FD 边界。
- [x] 处理 PM 测试拒绝并补充配置行为用例；独立 Tester r4 已执行，剩余场景缺口按 PM r4 风险决策记录。
- [x] 根据 Tester 实际运行和 PM 风险决策更新 Verification；未覆盖场景已显式记录并进入独立 Reviewer 审查。
- [x] 修复 Reviewer r1 发现的合法 TOML 表头注释问题；用例已由 Tester r4 执行。
- [x] 完成 Work Item 1.12 的配置回退与未知键行为；新增黑盒回归已由 Tester r4 执行并通过。
- [x] 根据 Reviewer r3 增加 Work Item 1.13：使用 TOML 解析库并补充解析边界回归；compile-only 通过，数组型未知键、类型回退和非法转义场景均由 Tester r5 记录为通过。
- [x] 根据 Tester r5 的 S04 ERROR 完成 Work Item 1.14：用户批准的单用例 S04 诊断已由独立 Tester r6 执行并通过；原 r5 ERROR 未复现但原因仍未知，完整套件未重跑。

## Sources

- Issue：无；本 FD 由用户指定 `docs/aiw-say.md` 创建。
- `docs/aiw-say.md`：V1 目标及先完成 Phase 1 的顺序。
- `openspec/specs/cli-and-plugins/spec.md`：内置命令优先、未知命令按 `aiw-<name>` 分派的契约。
- `cmd/aiw/main.go`、`internal/plugin/discover.go`、`go.mod`：现有 Go 模块、插件入口和依赖边界。
- `internal/plugin/runtime_config.go`：现有运行时已采用安装目录 `aiw.toml` 和平台用户配置目录，供 `say` 配置路径保持一致。

**Completed:** 2026-10-08
