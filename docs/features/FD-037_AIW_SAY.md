# FD-037: AIW Say 跨平台剪贴板与图形入口

**Status:** Open
**Revision:** 3
**Priority:** Medium
**Test policy:** Independent
**Evidence policy:** Dual

## Problem

FD-036 只设计无图形的 `aiw say` 翻译闭环。日常使用还需要从 Windows、Linux、WSL 剪贴板取文和回写结果；Windows 用户需要 AHK 快捷键，三个平台的图形会话都需要可选的 Zenity 输入。若各入口自行实现翻译或在失败时先改写剪贴板，会造成行为不一致、Unicode 损坏或原文丢失。本 FD 设计这些适配器和共同的成功/失败语义，依赖 FD-036 的核心翻译接口。

## Options and decision

| 方案 | 判断 |
| --- | --- |
| AHK、Zenity、CLI 分别调用 LLM | 配置和错误处理重复，难以保证同一翻译行为；不采用。 |
| GUI 脚本直接操作剪贴板并传给 `aiw say`，CLI 不支持剪贴板 | `--clipboard` 与 `--copy` 的公开用法无法实现，Linux/WSL 逻辑重复；不采用。 |
| CLI 统一拥有翻译与剪贴板适配器；AHK 经 stdin 调 CLI，Zenity 作为可选输入/结果适配器 | 复用 FD-036 的配置、profile、提供方和输出契约；采用。 |

Windows 选择系统剪贴板接口；Linux 按实际显示环境选择已安装的 Wayland 或 X11 工具；WSL 明确访问 Windows 剪贴板，而不是悄悄改用 WSLg 的 Linux 剪贴板。Zenity 在 Windows 调用用户安装的原生可执行文件，在 Linux/WSL 调用图形环境中的 Zenity；缺失工具或无图形会话时报清楚错误。所有外部命令均通过固定程序与参数调用，文本只走 stdin/stdout，不拼接进 shell 命令。

## Solution

- 在 FD-036 的翻译入口外增加 `Clipboard` 边界，提供 `Read(ctx)`、`Write(ctx, text)`；根据运行平台选择 Windows、Linux、WSL 适配器。WSL 识别优先于通用 Linux 选择，避免误选 X11/Wayland。适配器只处理文本和错误，不持久化剪贴板内容。
- CLI 支持 `--clipboard` 输入、`--copy` 成功后回写、`--dialog zenity` 图形输入。位置参数、stdin、剪贴板、Zenity 输入互斥；取消或空输入在调用提供方之前结束。普通 CLI 默认不改写剪贴板；`--clipboard --copy` 仅在获得完整有效译文后写入。回写失败返回非零状态，不能把错误文本写入剪贴板。
- Windows 通过系统剪贴板读写 Unicode 文本；Linux 在 Wayland 优先使用 `wl-paste`/`wl-copy`，在 X11 使用 `xclip`，必要时使用 `xsel`，仅选择当前会话可用且具备读写配对的工具。找不到合适工具则报错，不要求无图形 CLI 安装它们。
- WSL2 使用 Windows 剪贴板互操作：读使用可靠的 Windows 侧读取通道，写使用 Windows 侧写入通道；不得假定 `clip.exe` 能读取，也不得把中文、日文、换行转换错误。具体通道由适配器封装，并在具备 WSL 环境时验证双向 UTF-8/Unicode 往返；若前置工具不可用，报出依赖及恢复办法。
- Zenity 在 Windows、Linux、WSL 的受支持图形会话中提供可编辑多行输入，成功翻译后显示结果并默认复制到剪贴板。启动失败、取消输入和结果窗口取消均有明确结果；输入取消不调用 LLM，翻译失败不写剪贴板。`--dialog zenity` 不成为核心 CLI 的运行前提。Windows 未安装 Zenity，或 WSL 缺少 WSLg/X 图形环境时，直接报清楚前置条件。
- Windows AHK v2 提供 Ctrl+Alt+J/E/B/T；J/E/B 从剪贴板读取，分别翻译为日文、英文、商务日文；T 启动 `aiw say --dialog zenity`。脚本经子进程 stdin 传递任意 Unicode 文本，等待退出并检查状态，仅成功时回写译文；不模拟键入，不把文本拼入命令行，不把错误写入剪贴板。为避免并发热键覆盖用户后来复制的内容，回写前核对剪贴板仍是本次读取的原文；不一致则保留现值并提示用户。
- GUI 路径的译文可进入结果窗口或剪贴板；普通 CLI 成功 stdout 仍只包含译文、stderr 只包含错误。对外新增的 flag 和适配器行为需与 FD-036 的配置/profile 及退出码保持兼容。不得自动上传、记录或持久化剪贴板内容。

此 FD 涉及操作系统剪贴板、外部程序调用、GUI 与 CLI 公开选项。实施前应确认 FD-036 的实际入口、错误类别和配置契约，若接口与此设计有实质差异，先修订 FD 再编码。

## Scope

- 包含：Windows/Linux/WSL 剪贴板读写、CLI 的 `--clipboard`/`--copy`、Windows AHK v2 快捷键、Windows/Linux/WSL Zenity 可选输入及结果展示、错误与 Unicode 处理、平台安装前置条件文档。
- 不包含：文件输入输出、语音、自动抓取任意应用的选中文本、模拟键入、翻译历史、Bundled Zenity 安装包、其他 GUI 框架或新的 LLM 提供方。FD-036 的核心翻译语义不在此 FD 重做。

## Work items

- [ ] 1.1 在 FD-036 入口增加剪贴板边界、输入源互斥及成功后回写流程。Size: M；difficulty: Medium；dependencies: FD-036；完成标准：普通 CLI 默认不触碰剪贴板，失败不回写。
- [ ] 1.2 实现 Windows Unicode 剪贴板读写适配器。Size: M；difficulty: Medium；dependencies: 1.1；完成标准：中日文、引号、换行可读写，访问失败有明确错误。
- [ ] 1.3 实现 Linux Wayland/X11 工具选择和读写适配器。Size: M；difficulty: Medium；dependencies: 1.1；完成标准：按当前显示会话选具备读写配对的工具，缺失时报错。
- [ ] 1.4 实现 WSL2 Windows 剪贴板互操作适配器。Size: M；difficulty: Medium；dependencies: 1.1；完成标准：Windows 剪贴板双向读写且 Unicode 不损坏，前置工具缺失时给出恢复提示。
- [ ] 1.5 实现 Zenity 子进程边界和输入取消语义。Size: S；difficulty: Medium；dependencies: 1.1；完成标准：输入取消或空输入不请求 LLM，无 Zenity 不影响普通 CLI。
- [ ] 1.6 接入 Windows 原生 Zenity 的输入和结果窗口。Size: S；difficulty: Medium；dependencies: 1.5, 1.2；完成标准：受支持 Windows 图形会话可运行，未安装时明确报错。
- [ ] 1.7 接入 Linux/WSL Zenity 图形会话的输入和结果窗口。Size: M；difficulty: Medium；dependencies: 1.5, 1.3, 1.4；完成标准：Linux 与 WSL 图形会话可运行，无图形会话时明确报错。
- [ ] 1.8 编写 Windows AHK v2 快捷键脚本。Size: M；difficulty: Medium；dependencies: 1.1, 1.2；完成标准：四个热键正确调用 CLI、检查状态，失败和并发剪贴板变更均不覆盖原内容。
- [ ] 1.9 更新平台使用说明和静态示例，列出 Zenity/剪贴板工具安装前置条件、AHK v2、WSL 图形与编码限制。Size: S；difficulty: Low；dependencies: 1.2–1.8；完成标准：文档命令与实现一致，不宣称未验证的平台行为。

## Acceptance

- 三平台 `aiw say --clipboard -t ja --copy` 读取原剪贴板内容，翻译成功后写入完整日文译文；普通 `aiw say` 不改写剪贴板。空剪贴板、翻译失败或剪贴板写入失败有非零状态，翻译失败时原内容不变。
- Windows、Linux、WSL 的 Zenity 图形会话可输入多行中日英文本、查看完整译文，并在成功时复制结果；取消输入不调用提供方，图形工具缺失时普通 CLI 仍可使用。
- Windows AHK v2 的 J/E/B/T 四个热键分别执行指定翻译或 Zenity 输入；引号、换行、非 ASCII、命令字符均按原文传递；失败不覆盖剪贴板，热键运行期间用户更换剪贴板时不覆盖新内容。
- WSL2 使用 Windows 剪贴板；中日文、表情、换行的往返保持原样。若实施时无法在目标环境验证，Tester 必须如实列出未覆盖场景及前置条件，不能声明通过。
- 不论入口，翻译结果和错误均不被持久化；错误不得写入剪贴板，普通 CLI stdout 仍只输出译文。

## Verification

- 设计阶段：静态读取 `docs/aiw-say.md`、FD-036 和现有插件分派契约；未执行平台程序。
- Worker：检查输入互斥、平台适配器选择、子进程参数与 stdin、取消/失败/回写顺序、输出通道及最终 diff；按仓库规则执行一个窄范围 compile-only 检查。AHK/Zenity 脚本只做静态检查，执行另需授权。
- 独立 Tester：按每个平台列出正常、取消、缺失工具、Unicode、翻译失败与剪贴板竞争场景；任何可执行命令须先经过 Planner 对具体环境、命令和副作用的版本绑定授权。缺少目标平台或图形会话时记为未运行，不推断成功。
- 独立 Reviewer：对照当前 FD、Worker 报告、Tester/PM 证据审查，不把未运行的跨平台场景视为通过。

## TODO

- [ ] 在 FD-036 核心 CLI 可用后，按 Work items 实施适配器并记录实际平台证据。
- [ ] 实施后更新 Verification、未运行场景、前置工具和残余风险。

## Sources

- Issue：无；本 FD 由用户指定的新平台集成要求创建。
- `docs/aiw-say.md`：剪贴板、AHK、Zenity、输出、隐私和 V1 Phase 2 要求；Windows/WSL Zenity 范围按用户最新要求扩展。
- `docs/features/FD-036_AIW_SAY_CLI.md`：核心 CLI、配置/profile、翻译提供方与失败输出契约；本 FD 的前置依赖。
- `openspec/specs/cli-and-plugins/spec.md`：现有 `aiw-<name>` 插件分派契约。
