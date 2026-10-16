# FD-075: Add bounded optional consultation to ask-asu navigation

**Status:** Open
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户确认ask-asu应保持默认只读导航，同时可按需咨询宿主子Agent或aiw ask。当前Skill仅导航，与命名和用户第二意见需求存在落差；咨询的授权、副作用、上下文和次数缺少合同。

## Options and decision

保留一个入口，默认导航，显式第二意见或已授权的重要不确定性才使用一次有界咨询。通常选择一个渠道：宿主子Agent或aiw ask，不串行/并行叠加。不是每次导航自动外调，不修改aiw ask产品代码或全局配置。

## Solution

导航继续读取实际状态，只输出下一步，不修改文件/事件或派发角色。咨询是独立授权模式，可能产生模型费用、传输数据和本地历史，不等于导航只读；任务明确允许咨询才可在其限制内调用，缺少授权只准备问题/建议渠道。当前confirm仅授权本地实现定义，不实际调用模型。

宿主Agent能力决定能否指定模型/推理强度，不能将子Agent或reasoning名称视为更强模型，记录实际可知值而非伪造。独立咨询最多一个有界只读子Agent，不递归，不实施/派发FD角色/执行runtime操作。aiw ask使用现有配置和支持参数，无--model/provider/reasoning单次flag；可使用已允许的进程级配置但不改持久配置，无法验证能力则声明。单次--json调用保留Answer协议，不用chat/resume循环，无自动重试/切换后端。

aiw ask在auto配置下会逐后端尝试，因此咨询前必须有已明确且允许的固定provider；无法确认固定时不调用，避免一次命令隐含多次模型请求。记录一次宿主咨询请求与已知provider重试限制，不保证供应商内部传输次数。CLI会写~/.aiw/ask历史、目录访问能力依后端而异，--allow-path目录不视作强隔离；只传已授权最小问题/证据摘要，不自动继承当前会话或发送秘密。

结果区分建议与确认事实、缺失材料/失败/未知；回当前FD/讨论，不作为独立fd-review或人类决定。失败/无效结果/能力缺失保留已知结果并结束咨询，不改模型/渠道重试。已有诊断/测试/评审预算不因咨询重置，diagnosis-only不经咨询实施候选修正或重启执行。

## Scope

ask-asu入口、README、development-workflow引用、按需consultation引用、UI简述、目录README、项目副本和稳定FD规格。只文档/metadata；无CLI/API/依赖/持久化/状态修改，无真实模型/网络/安装/角色咨询。方法授权界限澄清不新增运行权限，不另加设计评审；正常独立实现评审保留。

## Work items

- [ ] 1.1 明确导航和可选咨询合同。Size: S; Difficulty: Low; Dependencies: none. Completion: 触发、授权、channel选择、一次咨询与失败停止、结果边界清晰。
- [ ] 1.2 更新aiw ask/子Agent能力参考及说明。Size: S; Difficulty: Low; Dependencies: 1.1. Completion: 当前参数/固定provider/历史副作用/最小上下文准确，源项目同步。
- [ ] 1.3 静态证据和独立评审交接。Size: S; Difficulty: Low; Dependencies: 1.2. Completion: 有界静态核查、双格式报告、实际独立Reviewer；交付另见TODO。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- 默认导航无额外模型、Agent或持久写入，仍输出实际阶段/下一步/阻断，不执行建议命令。
- 咨询需具体授权及有价值问题，通常只选一种渠道且一次；失败不改渠道/模型重试、不递归。
- 上下文包含目标/问题/约束/必要证据/版本和未知，不自动继承当前会话或暴露秘密，结果作为建议回原流程。
- 子Agent不承诺更强模型或高推理；aiw ask无不存在的参数、固定provider避免auto fallback、历史副作用和目录隔离限制明确。
- 咨询不替代人类决定/独立评审、不写FD事件、不扩大runtime权限或重置diagnosis-only/已消耗预算。
- 源项目副本/链接/metadata/规格一致，真实模型遵从和网络未验证不冒充通过。

## Verification

- Compile-only check and static review by default; record unrun tests explicitly.
- For explicitly requested test support (any wording/language), record suitable
  TDD scope, independent fd-test scope, command/revision authorization and finite
  budget here. Otherwise no testing stage is added. On exhausted unexpected-
  failure budget, stop affected test repair/execution, diagnose statically and
  continue independent work; unresolved acceptance defects still block delivery.

## Sources

- 用户confirm导航/咨询两模式建议。
- src/skills/ask-asu、src/internal/commands/ask/{command,config,path_policy}.go与src/internal/ai/llm.go。
- openspec/specs/ai-routing/spec.md、fd-workflow/spec.md：模型名称不代表能力，阶段与派发权威保留。

## TODO

- [ ] 完成1.1–1.3及独立实现评审。
- [ ] 本地交付与归档。

- Issue: none

- 本次纯Markdown/metadata不编译无关产品、不跑测试/模型/网络/脚本/安装/格式化。一批只读核对副本/链接/范围和实际参数路径；静态情景包含仅导航、宿主咨询、固定provider ask、auto后端阻断、缺权限/材料、失败结束及诊断预算不重置。
