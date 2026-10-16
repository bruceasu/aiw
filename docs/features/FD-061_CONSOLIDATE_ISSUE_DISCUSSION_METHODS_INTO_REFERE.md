# FD-061: 将 Issue 讨论方法收拢为内部 references

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户确认保留 issue-management 作为唯一需求流程入口，将五个仅用于 Issue 的讨论方法整合为内部 references，减少独立 Skill 分散维护和上下文选择负担。grill-with-docs 跨需求与工程设计使用，发布评估属于实现之后，二者保持独立。

## Options and decision

选择 issue-management 不改名、不并入 FD 生命周期；五个通用方法移入 references，移除仓库内旧独立入口，保留其分析正文与金融条件分支。重命名为 fd-issue 容易暗示每个 Issue 必须产生 FD；复制原文并保留独立入口则形成两个内容所有者，均不采用。

## Solution

新增 references/intake.md、value-assessment.md、metric-brief.md、engineering-options.md、synthesis.md。主 Skill 按当前缺口选择一个 reference，CLI 同样按固定映射加载对应正文。历史方法 ID 保留在方法选择元数据与日志中以兼容既有 Session；其对应 Skill 入口和 UI 元数据退出当前仓库目录。方法正文只保留讨论约束、输入输出、金融条件规则，删除独立 Skill frontmatter 与旧调用引用。

grill-with-docs、domain-modeling 和 finance-release-gate 不迁移或复制。保留当前 JSON 字段、模型调用数、证据摘要/引用、上下文预算、固定路径、必需方法缺失阻塞、可选追问方法失败不阻塞、审批和持久化边界。安装器现有整目录复制可包含 references，无需改代码。用户级安装与旧 CLI 不自动更新。

## Scope

范围：五个 canonical/项目独立技能入口和 UI 文件，issue-management/references、主 Skill 与项目副本、README、Issue 方法加载器和方法选择提示、requirement 稳定规格。不修改已归档 FD、不改依赖、公共 JSON/CLI 契约、安装器、grill-with-docs 或发布评估。

## Work items

- [x] 1.1 迁移五个分析方法，更新主 Skill 与说明，同步项目副本并移除旧入口。规模小、难度中，无依赖；一个内容所有者、按需链接、无悬空旧调用。
- [x] 1.2 更新 CLI 固定路径与稳定规格，保留历史方法标识和全部加载边界。规模小、难度中，依赖 1.1；新路径正文实际进入 Methods，JSON 与旧 Session 兼容。
- [x] 1.3 一次静态检查和离线库包编译，双格式报告并交接独立 Reviewer。规模小、难度低，依赖 1.2；记录真实证据和未执行检查。

## Acceptance

- 当前目录只保留 issue-management 入口；五个内部方法不再被识别为独立 Skill。
- 一次按 gap 装载一个 reference，保留通用领域适用、金融条件规则及原输入输出语义。
- Methods 的 Path 指向新固定路径，方法选择历史标识及 JSON 结构兼容。
- 无建议、无效建议、术语补充、必需来源缺失和可选追问失败的行为不变。
- 安装目录包含五个 references；grill-with-docs 与发布评估继续独立，无额外审批或写入权限。

## TODO

- 三个 Work Items 与双格式证据已完成，独立审查通过；等待 PM 后续交付决定，尚未合并或归档。

## Verification

- 一次静态 diff、内容迁移与引用/固定路径检查完成；主入口及五个 references 的源/项目副本哈希比较均为 True。检查后仅同步修正内部方法措辞，无 Go 变化。
- 在 src 执行 `$env:GOPROXY='off'; $env:GOSUMDB='off'; $env:GOTOOLCHAIN='local'; go build ./internal/issue`，退出码 0，仅编译库包。
- 未创建或运行测试、最终构建、formatter/linter/vet、安装同步命令、网络或下载。
- 实施报告：reports/FD-061-implementation-r1.md 与同名 JSON。
- 独立 Reviewer fd061-reviewer-20261017-6c4e 对 2d62ef9..58ff118 静态审查通过，无阻塞发现；报告 reviews/FD-061-review-r1.md 与同名 JSON，来源事件 FD-061-000004-implementation-ready。六组源/项目哈希一致、二十个旧入口/UI 文件均缺失；未重复编译或运行测试及安装同步。
- 剩余风险：旧 CLI/用户级技能需单独更新，实际安装和模型行为未验证；诊断保留历史方法 ID。

## Sources

- Issue: none

- 本轮用户确认整合 Issue 内部方法，保留 issue-management 名称，独立共享追问和发布评估。
- src/skills/issue-management/SKILL.md 及五个 finance-* 讨论技能。
- src/internal/issue/context_methods.go、conversation.go。
- src/plugins/aiw-skills/aiw-skills.py 整目录 materialize/copytree 行为。
- openspec/specs/requirement/spec.md；FD-059/FD-060 保持历史不变。
