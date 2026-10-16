# FD-068: Integrate domain modeling decision dependencies and intake triage

**Status:** Open
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

当前 domain-modeling 偏重词汇和 ADR，缺少概念关系、状态、不变量与具体例子的明确产出；wayfinder 的独立地图、决策票据与会话规则和 Issue/FD 主流程重复；triage 的 Agent brief 被描述为权威执行规格，与 Issue 软件需求和 FD 唯一工程计划冲突。用户要求吸收有价值的方法，减少重复流程和入口。

## Options and decision

| 方案 | 决定 | 理由 |
| --- | --- | --- |
| 三个独立流程全部保留并默认串联 | 不采用 | 重复澄清、状态和计划，增加日常成本。 |
| 删除三个技能及方法 | 不采用 | 丢失术语/规则澄清、大型决策依赖与多来源分流价值。 |
| 共享方法、按阶段整合、收窄入口 | 采用 | 保留 domain-modeling 与队列 triage，将 wayfinder 方法吸入 Issue/Planner，移除独立入口，不留薄兼容入口。 |

## Solution

在已有 workflow-methods.md 增加领域澄清、决策依赖和请求分流章节，入口只加载相关章节。无需增加安装器白名单、CLI 参数、角色或状态。

domain-modeling 保留独立能力并继续作为 Issue 的 TermsUnclear 补充方法。需求会话只使用宿主快照，不扫描实现或写稳定文件；返回定义、关系/状态/不变量候选、正常/边界例子、来源与待确认问题。工程模式可在获准范围内核对代码并维护文档。词汇归 CONTEXT；稳定行为归 specs；本次方案和依赖归 FD；长期重要取舍才记 ADR，各处以链接相连，不复制规则。

大型 Issue 按需在同一 Plan 记录“目标、未决问题、前置依赖、可推进项、结论位置、未明确区域、范围外”，不创建第二份 map/ticket/任务状态。grilling/grill-with-docs 负责讨论，方法负责保留依赖与跨会话恢复；可查事实由宿主在预算内调查，涉及取舍的人类决策不代答。只有满足既有业务子 Issue 标准的独立成果才拆分；工程问题由 Planner 在原 FD 延续，关键未知阻止就绪。

triage 专注多来源用户反馈与队列整理：请求来源、用户目标、已知事实、信息缺口、重复/相关候选、优先级理由与下一步。单项深入澄清交回 Issue；工程设计交 FD；PR 正确性和合并结论交独立 review。已有实现或历史拒绝不自动关闭；临时延期与长期拒绝分开。外部标签/评论/关闭与网络仅在明确授权时操作，默认本地建议，不发送任何消息。

这是指令和文档整合，不改公开 API、依赖、权限实现、数据格式或并发实现；不需额外独立设计评审，正常独立实现评审必做。

## Scope

包含：共享方法；domain-modeling 及其文档格式；Issue 入口/引用；FD planning；triage 及 brief/拒绝说明；setup-project 的相关引用；README、稳定规格、源技能与项目副本；移除 wayfinder 的受跟踪入口/资源及安装索引条目，修正现有调用位置。

不包含：CLI 及 Go 实现改动、新审批、新 tracker、外部 Issue/PR 操作、历史记录或用户级技能清理、自动批量迁移、测试/安装/联网。旧的外部地图只在明确引用时读取并提供继续处理建议，不自动删除或迁移。

## Work items

- [ ] 1.1 领域建模方法与产物归属。Size: M; Difficulty: Medium; Dependencies: none. Completion: 术语、关系、状态/不变量、例子、来源和未知有明确定义，Requirement mode 保持快照只读，工程写入仅限授权范围。
- [ ] 1.2 决策依赖吸收与 wayfinder 入口退役。Size: M; Difficulty: Medium; Dependencies: 1.1. Completion: Issue/Planner 按需共享方法，原 Plan/FD 唯一归属，不再有新独立 map/ticket/会话限额或 wayfinder 入口，清理安装索引和当前引用。
- [ ] 1.3 triage 收窄到用户诉求队列。Size: M; Difficulty: Medium; Dependencies: 1.1. Completion: 保留来源/重复/信息/优先级分析，brief 非权威执行计划，无强制复现、自动外部操作、自动关闭或重复深度访谈。
- [ ] 1.4 文档、安装副本与证据一致。Size: M; Difficulty: Medium; Dependencies: 1.2,1.3. Completion: 相关 stable specs/setup/README/项目副本一致，一批静态检查与独立评审后正常交付归档。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- Issue 的 TermsUnclear 补充入口仍有效；不依赖旧 wayfinder；domain-modeling 单独安装可读共享方法，未确认规则不写成事实。
- 领域澄清覆盖定义、关系/状态、不变量、正常和边界例子，并明确各类信息的权威归属；日常读词汇表不触发完整建模。
- 大型不确定需求按需记录决策依赖，简单 Issue 不强制建图；恢复时不重问已确认结论，不用“一会话只能一个问题”限制工作。
- 用户目标可追溯至软件需求与验收；业务拆分、决策问题与工程 Work Items 不混用；关键缺口仍阻止设计就绪。
- triage 处理多输入队列，不替代 Issue Plan、FD 或 PR review；已有实现和旧拒绝是核实输入，不是自动关闭依据。
- 无默认网络、测试、子 Agent 调研或外部写入；已授权外部动作不重复询问，但不扩大范围；静态结论不宣称运行时复现。
- wayfinder 源/项目受跟踪文件和索引条目移除，无薄入口；历史记录/用户级安装不变，相关当前引用完整更新。

## Verification

- Compile-only check and static review by default; record unrun tests explicitly.
- For explicitly requested test support (any wording/language), record suitable
  TDD scope, independent fd-test scope, command/revision authorization and finite
  budget here. Otherwise no testing stage is added. On exhausted unexpected-
  failure budget, stop affected test repair/execution, diagnose statically and
  continue independent work; unresolved acceptance defects still block delivery.

- 本 FD 只编辑技能/规格/元数据，无可编译源码变更，不运行编译、测试、安装、构建、lint 或网络。
- 一批静态只读检查：源/项目字节比较、共享资源声明与自包含、wayfinder 退役、活动引用与元数据、审批/产物归属一致性。
- 静态推演术语混淆、大型依赖恢复、重复反馈三个场景，不声称宿主运行效果。
- 独立 Reviewer 必须检查实际 diff，记录中文 Markdown 与同名 JSON；通过后自动本地交付和归档。

## TODO

- [ ] 完成 1.1–1.4 与独立评审。
- [ ] 正常本地合并及归档。

## Sources

- Issue: none

- 本会话已确认对三种技能的定位、grilling 与决策依赖的区别，以及吸收整合要求。
- src/skills/domain-modeling/、wayfinder/、triage/、issue-management/、fd-workflow/references/planning.md。
- src/skills/workflow-methods.md、setup-project/、openspec/specs/requirement/spec.md 与 fd-workflow/spec.md。
- src/internal/issue/context_methods.go 的 TermsUnclear 补充加载规则保持不变。
