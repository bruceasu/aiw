# FD-059: 通用讨论技能与 Issue 方法路由

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户要求 finance-* 技能适用于通用需求，在 Issue 流程中适时加载。当前描述、正文、Issue 指引和 Go 加载器仍限制金融领域。

## Options and decision

保留技能 ID 与目录名称，通用化内容并按已验证的 gap 选择方法。只修改描述不能解除代码门槛；重命名会影响既有引用。金融规则保留为条件要求。

## Solution

五种讨论方法按 intake/value/metric/feasibility/synthesis 缺口加载，不受 finance/generic/unknown 标签限制。保留领域枚举、摘要与引用校验、上下文预算、固定路径、缺失技能阻塞和术语补充。六个技能通用化；release-gate 仍只用于实现后的发布阶段。同步仓库内副本，不改用户级安装。

## Scope

包括六个 src/skills/finance-* 技能、issue-management、README、项目副本、src/internal/issue 方法路由和提示、openspec/specs/requirement/spec.md。无依赖、公共结构、CLI、持久化格式或权限变更。

## Work items

- [x] 1.1 通用化技能与说明并同步仓库副本。规模小、难度低，无依赖；描述和正文按缺口触发，金融规则条件化，发布评估独立。
- [x] 1.2 修正加载器、方法选择提示与稳定规格。规模小、难度中，依赖 1.1；有效的 generic/unknown/finance 建议均可选对应方法，保留证据及预算边界。
- [x] 1.3 记录静态与编译证据并交接独立 Reviewer。规模小、难度低，依赖 1.2；双格式报告与真实验证记录齐全。

## Acceptance

- 非金融 Issue 按缺口使用五种方法，不自动执行全链。
- 无建议仍使用基线；无效建议仍降级；选中技能缺失仍阻塞。
- 金额相关的币种、精度、结算规则只在适用场景要求。
- 技能 ID 和引用兼容；发布评估不进入需求发现链。

## TODO

- 实施与双格式证据已完成，独立审查通过。交付合并与归档尚未执行，保留专用 worktree。

## Verification

- 已执行一次静态 diff 与调用路径检查：技能描述、正文、界面元数据、Issue 指引、固定 gap 映射、证据校验与错误分支、模型提示及稳定规格保持一致。
- 已在 src 执行 `$env:GOPROXY='off'; $env:GOSUMDB='off'; $env:GOTOOLCHAIN='local'; go build ./internal/issue`，退出码 0。仅编译库包，无最终分发产物。
- 未创建或运行测试，未运行 formatter/linter/vet、全量构建、网络或下载。
- 剩余风险：尚无运行时行为验证；旧版已安装 CLI 与用户级技能需另行更新，本轮只同步仓库内项目副本。
- 实施报告：reports/FD-059-implementation-r1.md 与同名 JSON。
- 独立审查通过：reviews/FD-059-review-r1.md 与同名 JSON；Reviewer `fd059-reviewer-20261016`，来源事件 `FD-059-000004-implementation-ready`，审查 `db8b7fd..e48d42a`。静态确认五种方法跨领域路由、证据与预算边界、七个技能目录副本一致及发布阶段分离；Reviewer 未运行测试或编译。

## Sources

- Issue: none

- 本轮用户确认创建 FD、必要本地提交和 worktree 操作。
- src/skills/issue-management/SKILL.md 与六个 finance-* 技能。
- src/internal/issue/context_methods.go 与 conversation.go。
- openspec/specs/requirement/spec.md。
