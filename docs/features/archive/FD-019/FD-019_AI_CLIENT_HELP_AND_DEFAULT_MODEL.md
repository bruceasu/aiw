# FD-019: AI client help and default model

**Status:** Complete
**Revision:** 8
**Priority:** Medium
**Test policy:** External
**Evidence policy:** Dual

## Problem

用户在启动 ai 客户端时缺少配置指引，当前帮助仅一行且模型必填。

## Options and decision

按用户指定，客户端默认逻辑模型 gpt-6-luna，环境变量可覆盖。
扩展现有帮助和错误提示，无需依赖、请求参数或网关授权机制变化。

## Solution

增加 -h 支持，使 aiw help ai 可复用相同帮助；帮助列出最少客户端和网关设置、
平台示例、默认值和输入输出规则。认证 Key 仍必填，不为用户生成或披露凭据。

## Scope

客户端代码、README、稳定客户端 spec。用户已有网关配置仍保留，文档说明
默认逻辑模型必须在 models 和 allowed_models 中配置，后端可用性由运营者确认。

## Work items

- [x] 1.1 默认模型与配置错误提示；规模小、难度低，无依赖。完成标准：无模型变量时使用 gpt-6-luna，非空变量覆盖，Key 错误指向帮助。
- [x] 1.2 完善帮助；规模小、难度低，依赖 1.1。完成标准：-h/--help 共享最少配置、默认值、选项及平台示例，帮助不请求。
- [x] 1.3 同步 README 与 spec；规模小、难度低，依赖 1.1/1.2。完成标准：客户端默认与网关映射要求一致。
- [x] 1.4 提供 Windows 交互哈希 BAT；规模小、难度低，依赖 1.3。完成标准：隐藏输入，UTF-8 无换行哈希，只输出哈希，无配置或凭据文件写入。
- [x] 1.5 拼接多个提示词参数；规模小、难度低，依赖 1.1。完成标准：非选项参数按空格拼接，保留选项和单独 stdin 标记。
- [x] 1.6 EDITOR 临时文件编辑；规模小、难度中，无依赖。完成标准：解析引号路径及参数、直接启动编辑器、有界 UTF-8 读取、finally 清理，失败不请求。
- [x] 1.7 接入交互 Ctrl+E；规模小、难度中，依赖 1.6。完成标准：携带当前输入、关闭 readline 后转交终端、正常退出后提交，Ctrl+C 取消。
- [x] 1.8 同步参数与编辑器说明；规模小、难度低，依赖 1.5/1.7。完成标准：帮助、README 与稳定 spec 对齐，同步本机插件文件。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

默认模型 gpt-6-luna，显式模型覆盖；Key 必填且错误不显示凭据。
帮助无需环境变量、stdin 或服务。兼容现有 URL、输入、stdout/stderr 和请求契约。

## TODO

- 独立复审已确认 reviews/FD-019-review-r5.md 的 R1 消除：帮助、README 和客户端稳定规格使用 `principals[].keys`，明确两端相同字符串，当前配置不需要哈希工具。结论见 `docs/features/reviews/FD-019-review-r7.md` 及同名 JSON。
- %% 哈希 BAT 的历史工作项 1.4 保留；README 已澄清当前配置无需计算哈希。此轮只修复仓库指引，本机安装副本未同步，需运营者后续更新。
- 实际运行时认证和模型调用仍待验证。
- Ctrl+E 终端接管、编辑器等待和多平台运行行为未执行测试，待用户安排。

## Verification

- 2026-10-04 独立 Reviewer `codex-reviewer-FD019-20261004-r7-82de` 领取
  `FD-019-000007-implementation-ready`，复审修订 7，基线为 HEAD
  `c9b6d51cd2308e1ffbaf51755d697f6acda3ad83`，审查当前未提交的定向差异。
  结论：静态审查通过，上一轮 R1 已消除；新增改变仅涉及 HELP、README 与客户端规格的认证指引。
  八项既有实现结合前次静态证据与当前源码追踪获得支持；历史 BAT 保留但不是当前认证配置步骤。
  本轮仓库文档修复不重做历史本机同步，安装副本未更新，需运营者另行更新；External 运行风险继续保留。
  Reviewer 未执行测试、编译、客户端/编辑器/BAT、网络、构建或 Git 写操作。
  报告：`docs/features/reviews/FD-019-review-r7.md` 及同名 JSON；通过交接由 CLI 收据确认。

- 2026-10-04 用户再次授权处理本 FD。Worker `codex-worker-FD019-20261004-r6-a71c`
  领取 `FD-019-000006-changes-requested`，仅修复 R1 的帮助/README 内容，不改变认证实现。
  明确 `keys` 与客户端 Key 使用同一字符串；旧摘要只按不透明字符串使用。
  保留历史报告和工作项编号；本轮不修改安装文件，不执行模型请求、测试、构建或网络。
- 本轮 `node --check plugins/aiw-ai/aiw-ai.mjs` 退出码 0，仅语法编译无产物。
  根据用户“key_hashes 已经废除，请更新文档”的补充要求，移除 README 中当前配置步骤形式的
  哈希工具调用，并在客户端稳定规格明确两端相同字符串；后续仅改帮助文字，未重复编译。
  实现报告：`docs/features/reports/FD-019-implementation-r6.md` 及同名 JSON。

- 2026-10-04 独立 Reviewer `codex-reviewer-FD019-20261004-r5` 领取
  `FD-019-000005-review-requested`，审查修订 5、提交
  `c9b6d51cd2308e1ffbaf51755d697f6acda3ad83` 及其父提交到当前实现的定向差异。
  结果：请求修改，R1 阻塞工作项 1.2、1.8 的当前最少配置与文档对齐要求。
  八项既有勾选保留为历史实现记录，不表示当前独立审查通过；后续 Worker 修复由正式交接跟踪。
  报告：`docs/features/reviews/FD-019-review-r5.md`，同名 JSON 为机器证据。
  当前证据为源码/提交/稳定规格静态审查；下列编译、认证核对和本机同步仍仅是历史记录。
  本次未执行测试、编译、客户端/编辑器/BAT、模型调用、网络请求、构建或 Git 写操作。

- 2026-10-04 用户一次性授权处理 FD-019 状态。修订 4 的八项工作已勾选，
  最新交接仍为修订 2 的待领取 Planner 请求 `FD-019-000002-design-requested`。
  本次恢复为 Pending Verification，并通过 `aiw fd request-review` 正式创建当前
  Reviewer 请求、取消并保留旧交接；不补造 Worker 完成记录。
- 本次读取发现客户端帮助和 README 的 `key_hashes` 指引与当前网关 `keys`
  稳定规格不一致，交由独立 Reviewer 评估。以下编译、同步和认证核对均为历史记录，
  本次未重复执行，未修改认证实现或本机安装文件。Test policy 保持 External。

- 本轮静态追踪多个参数解析、readline close/keypress 生命周期、编辑器直接 spawn、
  有界文件读取和 finally 清理；不执行 shell 命令字符串。
- 已分别运行 `node --check plugins/aiw-ai/aiw-ai.mjs` 和
  `node --check plugins/aiw-ai/editor.mjs`，均退出码 0，不执行模块或保留产物。
- 已同步 aiw-ai.mjs、editor.mjs 和 README 到本机安装目录。
- 编辑器会短暂写入本机临时 Prompt 文件；正常/可捕获失败清理，强制退出和编辑器备份
  可能残留。此用户明确要求的输入方式补充原“不持久化输入”契约。

- 静态追踪 parseArgs、config、main 和 aiw help ai 的 -h 调用。
- 已运行 `node --check plugins/aiw-ai/aiw-ai.mjs`，退出码 0，仅语法编译，不生成产物。
- 同步客户端入口和 README 到当前 aiw 安装目录 C:/green/aiw/plugins/aiw-ai。
- 后续用户反馈认证失败：离线计算用户此前提供 Key 的 SHA-256，与工作区及
  安装目录配置中的启用主体匹配；未输出 Key 或哈希。认证逻辑未改动。
- 修正工作区和安装目录 gateway.json，补充 gpt-6-luna 映射及 team-a 权限，
  保留 gateway-default 和既有 Key 哈希。静态读取确认映射和授权存在。
- 检查时未发现 agent-gateway.exe 进程；用户/系统持久环境中均未设置客户端 Key。
  无法读取用户当前 PowerShell 的临时环境，不能据此确认请求所发送 Key。
- 未执行 CLI、网关、模型请求、测试或最终产物构建。
- 哈希脚本静态检查输入在 PowerShell 内读取，不经 cmd 展开；校验 Key 长度及空白，
  SHA-256 输入由 UTF8.GetBytes 获取，末尾不追加换行。未执行交互脚本。

## Sources

- Issue: none

用户本轮三项要求；openspec/specs/agent-proxy-client/spec.md；FD-017 历史实现。

**Completed:** 2026-10-04
