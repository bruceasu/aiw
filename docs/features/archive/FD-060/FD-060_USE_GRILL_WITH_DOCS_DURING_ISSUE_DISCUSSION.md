# FD-060: 在 Issue 讨论中使用 grill-with-docs 提高需求质量

**Status:** Complete
**Revision:** 8
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户确认 grill-with-docs 应在必要的需求讨论节点常规参与，不应只在讨论阻塞后启用。目前 Issue 指引没有调用约定，Go 加载器也未装载该方法。

## Options and decision

选择保留五个主讨论方法，以 grill-with-docs 作为共用的讨论质量方法；按阶段和复杂度使用，不新增独立审批。只修改指引无法覆盖 CLI；新增方法选择 JSON 字段会扩展契约，本次不采用。保持既有主方法和术语补充字段，由 Methods 列表记录额外方法的实际装载情况。

## Solution

有效方法建议的主方法和必要术语补充装载完成后，尝试从固定项目路径加载 grill-with-docs。初步理解形成后检验问题、目标、范围和假设；关键规则或方案讨论时检验边界、例外和后果；准备人工决策或 FD 交接时检验一致性与验收。每次聚焦一个决策，简单需求少问，已确认事项不重复。结论交回同一份 Issue Plan，宿主仍拥有确认与写入。

缺失、空内容或预算不足时在 Methods 记录诊断，不阻塞主流程，不宣称已使用。无建议或无效建议仍只使用内置基线。保留只读、证据、路径、上下文预算、审批及主方法缺失阻塞边界；不改变模型调用次数或 JSON 结构。

## Scope

范围：issue-management、grill-with-docs 及其项目副本和界面说明，Issue 方法加载与提示、技能 README 和 requirement 稳定规格。不修改 grill-me，不创建新审批阶段、不改 CLI/持久化/公共 API/依赖，无用户级安装更新。

## Work items

- [x] 1.1 定义讨论质量方法及 Issue 路由节点，同步项目副本和界面说明。规模小、难度低，无依赖；触发、深度、结束和记录边界清晰。
- [x] 1.2 接入可选方法装载、模型提示和稳定规格。规模小、难度中，依赖 1.1；有效建议才尝试装载，缺失/空/预算不足不阻塞，既有边界保留。
- [x] 1.3 完成一次离线库包编译及静态检查，记录双格式证据并交接独立审查。规模小、难度低，依赖 1.2；如实记录执行和未执行检查。

## Acceptance

- 有效的五种 gap 建议均可同时获得 grill-with-docs 正文，跨领域一致。
- 方法文本要求按需选择讨论节点和追问深度，不为每个回合强制完整访谈。
- 无效建议或无建议不新增方法；主方法及术语补充优先于可选质量方法。
- 可选方法缺失、空或超预算只记录诊断；不改变主方法已选状态，不假称使用。
- 不自动调用 grill-me，不新增模型调用、文件写入或审批权限。

## TODO

- 实施与双格式报告已记录，文档引用修复经独立 r2 审查通过；已 squash 交付至 develop（05ba46a）并归档 Complete，专用 worktree 和 feature/FD-060 分支已清理。

## Verification

- 一次静态 diff/调用路径检查完成：有效建议和主方法优先、可选方法失败不阻塞、空内容诊断、Methods 来源输出、JSON/审批边界及技能描述/提示/规格一致。
- 三个变更文件的源/项目副本 Get-FileHash 比较均为 True。
- 在 src 执行 `$env:GOPROXY='off'; $env:GOSUMDB='off'; $env:GOTOOLCHAIN='local'; go build ./internal/issue`，退出码 0；仅编译库包，无最终分发产物。
- 未创建或运行测试，未运行 formatter/linter/vet、全量构建、网络或下载。
- 实施证据：reports/FD-060-implementation-r1.md 与同名 JSON。
- 独立审查：reviews/FD-060-review-r1.md 与同名 JSON；Reviewer `fd060-reviewer-20261017`，来源事件 `FD-060-000004-implementation-ready`，范围 `d9bd08b..69e335c`。追加发现新共享契约 `.agents/skills/reviewed-skill-contract.md` 缺失，结论更正为请求修改；先前通过事件 `FD-060-000005-verification-passed` 不覆盖此发现，待 PM 重新路由。其余静态检查未发现阻塞；Reviewer 未运行测试或编译。
- 剩余风险：真实模型追问质量未运行验证；可选方法占用剩余上下文；已安装 CLI 和用户级技能未更新。
- 修复证据：reports/FD-060-implementation-r2.md 与同名 JSON；修复提交 `b7f375f` 不修改 Go，沿用原编译证据。
- 最新独立审查通过：reviews/FD-060-review-r2.md 与同名 JSON，Reviewer `fd060-reviewer-r2-20261017`，来源 `FD-060-000007-review-requested`，HEAD `e60a5e7`。本轮仅复核文档修复与 r2 证据，r1 晚发现原因已解除；其余代码沿用先前审查，未重复运行检查。r1 报告与旧 pass 保留为历史。
- Auto 收尾：复核 FD Revision 8 的摘要与最新事件 FD-060-000008-verification-passed 一致，审查后仅有审查和流程记录变更；双方工作区干净，无未解决的 blocker 反馈文件。共两轮独立审查，r2 为最终通过依据。
- 实际执行 `aiw git wt local-merge FD-060` 与 `aiw fd close FD-060 Complete` 均成功。交付提交 05ba46a 的 FD-Source 为 e4b1167bbabecf8977f972c409343eeb0dc75e59；CLI 校验交付后清理 worktree 和分支，保留收据。
- 本轮未重复运行编译、测试、构建或网络操作；已安装 CLI、用户级技能和真实模型追问效果的剩余风险保持不变。

## Sources

- Issue: none

- 本轮用户确认将 grill-with-docs 作为需求讨论常用方法，按节点与复杂度调整。
- src/skills/issue-management/SKILL.md、src/skills/grill-with-docs/SKILL.md。
- src/internal/issue/context_methods.go、context.go、context_sources.go、conversation.go。
- openspec/specs/requirement/spec.md；FD-059 归档保持不变。

**Completed:** 2026-10-16
