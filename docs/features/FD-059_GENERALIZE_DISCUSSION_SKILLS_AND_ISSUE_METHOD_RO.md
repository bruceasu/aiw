# FD-059: 通用讨论技能与 Issue 方法路由

**Status:** Open
**Revision:** 3
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

- [ ] 1.1 通用化技能与说明并同步仓库副本。规模小、难度低，无依赖；描述和正文按缺口触发，金融规则条件化，发布评估独立。
- [ ] 1.2 修正加载器、方法选择提示与稳定规格。规模小、难度中，依赖 1.1；有效的 generic/unknown/finance 建议均可选对应方法，保留证据及预算边界。
- [ ] 1.3 记录静态与编译证据并交接独立 Reviewer。规模小、难度低，依赖 1.2；双格式报告与真实验证记录齐全。

## Acceptance

- 非金融 Issue 按缺口使用五种方法，不自动执行全链。
- 无建议仍使用基线；无效建议仍降级；选中技能缺失仍阻塞。
- 金额相关的币种、精度、结算规则只在适用场景要求。
- 技能 ID 和引用兼容；发布评估不进入需求发现链。

## TODO

- 完成 1.1 至 1.3，交接独立审查；审查通过前不声明 Complete。

## Verification

- 计划：一次静态 diff 与调用路径检查；在 src 对 ./internal/issue 执行一次离线 go build，不保留最终分发产物。
- 不创建或运行测试，不运行 formatter/linter/vet、全量构建、网络或下载。
- 尚未实施或验证。

## Sources

- Issue: none

- 本轮用户确认创建 FD、必要本地提交和 worktree 操作。
- src/skills/issue-management/SKILL.md 与六个 finance-* 技能。
- src/internal/issue/context_methods.go 与 conversation.go。
- openspec/specs/requirement/spec.md。
