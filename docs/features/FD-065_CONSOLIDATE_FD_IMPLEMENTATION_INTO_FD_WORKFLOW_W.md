# FD-065: Consolidate FD implementation into fd-workflow without alias

**Status:** Complete
**Revision:** 7
**Priority:** Medium
**Evidence policy:** Dual

## Problem

implement 是编号 FD 的 Worker 阶段，但目前有独立技能入口，并与 fd-workflow 重复描述实现和后续交付。用户确认整合到 fd-workflow，且不保留薄入口；需要单独实现时直接调用 fd-workflow。

## Options and decision

选择把实现方法收进 fd-workflow/references/implementation.md，由入口和 Worker 角色按需读取，移除 implement 的源和项目安装入口。保留薄入口被用户明确否决；把全部实现细节继续复制到多个入口会增加维护成本，故只保留阶段路由与必要边界。

## Solution

fd-workflow 统一设计、实现、审查和交付的宿主入口，实现阶段加载 references/implementation.md 与 roles/worker.md。明确支持自然语言“只实现 FD-XXX”“只实现 1.1”“暂不合并”，不新增或假称 CLI 阶段参数。共享契约仍拥有生命周期、Git 和授权约定，参考文档只保存实现准备、逐项变更、证据与交接方法。更新 ask-asu、Issue、项目配置、TDD、技能目录和安装示例等当前引用；更新稳定规格和项目副本，移除安装索引中 implement 一项。

## Scope

仅仓库内技能 Markdown/YAML/安装索引 JSON 和稳定规格，不改 CLI、事件/schema、测试、依赖或用户级全局安装；不改历史 FD/报告或清理其它退役技能。旧 implement 调用不再提供兼容入口，改用 fd-workflow 是用户明确要求的入口变更。旧 Task/Core 兼容说明保留。

## Work items

- [x] 1.1 整合实现方法并移除独立源入口。规模小、难度中，无依赖；Worker 自动加载唯一实现参考，显式阶段和部分范围限制先于交接判断，角色/独立审查约定不变。
- [x] 1.2 更新路由、安装说明、稳定规格与项目副本。规模小、难度低，依赖 1.1；当前维护文档无失效 implement 调用，项目索引和文件一致，不触碰全局安装和历史资料。
- [x] 1.3 一次静态核对并写双格式实现证据。规模小、难度低，依赖 1.2；核对引用、文件删除、项目副本、单一所有者和阶段边界，然后独立审查。

## Acceptance

- source 和项目内不再存在可发现的 implement/SKILL.md 或其 UI/共享文件；不保留兼容薄入口。
- fd-workflow 入口与 Worker 角色按需读取 implementation.md；正常全 FD 实现继续独立审查及自动交付归档，明确只实现/部分项/不合并则保持原限制。
- 设计或提问不触发实现，参考文档不反向调用旧入口或循环调用 fd-workflow。
- 当前路由/说明/索引更新；历史 FD、报告和全局用户技能不修改。
- 项目副本一致；不新增测试、构建、网络、push 或部署权限，不改 CLI。

## TODO

- 实现及 r1 共享依赖修复完成，等待独立复核；通过后宿主自动本地交付并归档。

## Verification

- 独立审查 r2：verification-passed；见 reviews/FD-065-review-r2.md 和同名 JSON。R1 的入口依赖声明和项目共享副本已补齐，引用路径、安装器识别和 canonical 哈希一致。聚焦审查 cef9b78，沿用 r1 其它验收证据；未运行测试、编译、构建、安装或网络。

- 独立审查 r1：changes-requested；见 reviews/FD-065-review-r1.md 和同名 JSON。发现实现参考要求的 reviewed-skill-contract 未在入口声明，安装器不会物化且项目副本缺失；须补齐后重新审查。审查范围 a88b62d..d9650be；未运行测试、编译、构建、安装或网络。

- 纯 Markdown/YAML/JSON 元数据变更，无适用编译目标。一次静态检查最终 diff、当前引用、六个旧文件删除、项目副本与索引、范围限制；不运行测试、安装、构建、验证脚本或网络。

## Sources

- r1 发现实现参考的共享契约未在入口声明，已补齐入口声明和项目物化副本；修复证据 reports/FD-065-implementation-r2.md 及同名 JSON，等待独立复核。未重复运行 Worker 验证命令。

- 实现证据：reports/FD-065-implementation-r1.md 及同名 JSON；一次静态检查确认八组副本一致、六个旧文件已删除、当前引用与索引无旧入口。尚未运行实际安装或宿主行为评估。

- Issue: none

- 用户明确同意整合 implement，且不保留薄入口。
- src/skills/implement、fd-workflow、ask-asu、issue-management、setup-project、tdd 与 README。
- openspec/specs/fd-workflow/spec.md、src/skills/work-management.md；FD-063/064 已完成的默认交付和冲突恢复约定。
