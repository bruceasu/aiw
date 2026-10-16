# FD-060: 在 Issue 讨论中使用 grill-with-docs 提高需求质量

**Status:** Open
**Revision:** 3
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

- [ ] 1.1 定义讨论质量方法及 Issue 路由节点，同步项目副本和界面说明。规模小、难度低，无依赖；触发、深度、结束和记录边界清晰。
- [ ] 1.2 接入可选方法装载、模型提示和稳定规格。规模小、难度中，依赖 1.1；有效建议才尝试装载，缺失/空/预算不足不阻塞，既有边界保留。
- [ ] 1.3 完成一次离线库包编译及静态检查，记录双格式证据并交接独立审查。规模小、难度低，依赖 1.2；如实记录执行和未执行检查。

## Acceptance

- 有效的五种 gap 建议均可同时获得 grill-with-docs 正文，跨领域一致。
- 方法文本要求按需选择讨论节点和追问深度，不为每个回合强制完整访谈。
- 无效建议或无建议不新增方法；主方法及术语补充优先于可选质量方法。
- 可选方法缺失、空或超预算只记录诊断；不改变主方法已选状态，不假称使用。
- 不自动调用 grill-me，不新增模型调用、文件写入或审批权限。

## TODO

- 完成三个 Work Items 并取得独立审查；交付合并与归档须按已授权范围执行。

## Verification

- 计划一次静态 diff 与调用路径检查，并在 src 离线执行 go build ./internal/issue；只编译库包，不保留最终分发产物。
- 不创建或运行测试，不运行 formatter/linter/vet、全量构建、网络或下载。
- 尚未实施或验证。

## Sources

- Issue: none

- 本轮用户确认将 grill-with-docs 作为需求讨论常用方法，按节点与复杂度调整。
- src/skills/issue-management/SKILL.md、src/skills/grill-with-docs/SKILL.md。
- src/internal/issue/context_methods.go、context.go、context_sources.go、conversation.go。
- openspec/specs/requirement/spec.md；FD-059 归档保持不变。
