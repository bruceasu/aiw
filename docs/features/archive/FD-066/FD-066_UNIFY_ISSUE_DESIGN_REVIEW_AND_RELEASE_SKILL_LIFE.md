# FD-066: Unify Issue design review and release skill lifecycle

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户要求整体统一 Issue、FD 设计、实现审查和发布检查。当前 Issue 已拆需求、Planner 已拆工程工作项，但 to-tickets 仍指向不存在的旧 Task；代码审查方法重复、规格编写缺少阶段协作，设计与发布检查边界不够清晰。用户明确允许删除旧 Task 内容，并要求生成 FD 后执行。

## Options and decision

采用 FD 为唯一工程计划，Issue 为业务范围和批准来源，OpenSpec 为稳定能力规则。吸收 propose/apply 的准备、依赖、就绪与修订一致性方法，不额外维护 proposal/design/tasks。把拆分、规格、设计审查和实现审查方法按阶段加载；不将全部技能串行调用，也不默认增加审查 Agent 或轮次。退役 to-tickets，不保留旧 Task 兼容入口。

## Solution

Issue 按可独立决策/验收的业务结果拆子 Issue；Planner 按可实现/审查的工程结果拆 Work Items，分别维护唯一清单。Planner 主导设计，按需加载内部规格方法与独立设计审查参考；重要接口、数据、权限、并发等设计或用户明确要求时，受限独立设计 Reviewer 只反馈设计发现，不伪造实现 verification 事件。无未解决设计问题方可 design-ready。to-spec 保留独立规格入口并与 Planner 共享同一规格方法；变化后重新协调 FD、工作项和验收，规格不因讨论未定而提前冒充已实现事实。

fd-review 统一当前 FD 的实现验收、真实代码、证据与状态结论，读取共享 Standards/Spec/正确性方法；code-review 保留独立分支/PR审查入口，不隐式再派两名 Reviewer。通过新增一个安装器共享参考名，让两者独立安装也可读取同一 canonical 方法；不调用另一个技能形成循环。finance-release-gate 只在实际发布前评估运维证据，不作为普通本地归档 Gate，也不授权部署。

清理当前技能、共享契约和根规则中的旧 Task 指令，删除 to-tickets 源/项目文件及索引项，同步项目副本。少量当前技能引用按 FD 语义替换；一般英文 task/人工待办及宿主工具名 Task 不属旧 AIW Task，保留。历史 FD/报告、旧 CLI 实现和描述现存 CLI 的历史能力规格不作删除。

## Scope

范围是技能和 Agent 当前流程 Markdown/YAML/JSON、稳定规格、项目生成副本，以及安装器 SHARED_SKILL_REFERENCES 的单一新增常量项。用户级安装、CLI Task 删除/API迁移、数据库、历史资料、外部发布、网络和测试不在本 FD 范围。无依赖或公共 CLI 参数变化；不运行安装器做验证。已批准的能力变化和原有 FD 本地交付、冲突恢复、角色独立和显式阶段限制保持不变。

## Work items

- [x] 1.1 统一 Issue/FD 拆分及规格/设计审查方法。规模小、难度中，无依赖；Planner 按需加载，Issue 与 Work Items 无双重归属，重要设计独立反馈且不伪造实现审查事件。
- [x] 1.2 统一实现代码审查与发布边界。规模小、难度中，依赖 1.1；fd-review/code-review 共享自包含可安装方法，正常 FD 不重复派双审查 Agent，release gate 不阻塞本地交付。
- [x] 1.3 退役旧 Task 技能和当前协议引用。规模小、难度中，依赖 1.1/1.2；删除 to-tickets，清理当前文档并同步安装索引/源项目副本，不动历史和 CLI。
- [x] 1.4 编译安装器源并作一次静态一致性核对、双格式证据和独立审查交接。规模小、难度低，依赖 1.3；编译不运行安装器、不落最终产物，检查单一方法所有者、引用与共享物化、范围和副本。

## Acceptance

- 当前链路明确 Issue、Planner、Worker、独立 fd-review、宿主本地交付与发布 gate 的职责；普通 FD 不新增自动发布检查。
- Issue 与 FD 分层拆分，只有 FD Work Items 为工程执行计划；不新增 OpenSpec change 或 tasks.md。
- Planner 按需加载规格/设计审查方法；重要设计或用户要求时独立设计反馈，未决问题阻止就绪；简单改动不固定增加审查。
- fd-review 阅读真实实现、规范、需求和证据；code-review 独立使用同一方法。共享参考在入口声明并能由现有安装流程物化，FD 审查不递归调用独立 code-review。
- to-spec 保留独立请求能力；to-tickets 源/项目入口及索引项删除，当前技能无旧 Task/Attempt/租约操作指令，不删除历史或现有 CLI。
- 显式阶段、累计审查限制、冲突现场保护、零测试/网络等预算不扩大；纯静态证据不冒称运行验证。

## TODO

- 全部实现、编译和独立静态审查完成；已交付 develop（1a97a18），CLI 清理分支/worktree，宿主 close 归档并提交归档记录。

## Verification

- 独立 Reviewer fd066-reviewer-20261017-a918 审查 cf6163f..99e8ef4，通过，无实质发现；证据 reviews/FD-066-review-r1.md 与同名 JSON。检查真实契约、安装器调用路径、规范、验收和证据，不重复编译或运行测试/安装。
- 安装器只新增共享参考常量，窄编译命令：python -c "from pathlib import Path; p=Path('src/plugins/aiw-skills/aiw-skills.py'); compile(p.read_bytes(), str(p), 'exec')"。仅编译不执行、不生成 pyc。
- 一次静态核对变更 diff、当前 Task 引用、源项目副本、共享参考声明和安装器读取路径、索引单项删除、设计/实现/发布边界。无测试、构建、格式化、安装、验证脚本或网络。
- 本 FD 不改重要系统设计或公共接口，按需设计审查条件不触发；实现仍交独立 Reviewer。

## Delivery record

- 2026-10-17，独立审查 r1 通过，事件 FD-066-000005-verification-passed；FD revision/digest 与交付前文件一致。审查证据在 reviews/FD-066-review-r1.md 及同名 JSON。
- 源 HEAD：5f8b0775c041511bca982dab6e0fcb709cd0b533；父分支 develop 的 squash：1a97a18。审查后仅增加审查报告、FD 和索引，无实现漂移。
- 实际执行 aiw git wt local-merge FD-066，成功交付并清理 feature/FD-066、.wt/FD-066 及 workspace 记录；随后 aiw fd close FD-066 Complete 归档。完整交付输出保存在 .ai/fd/FD-066/local-merge.log。本次无同步、冲突或发布操作。
- 安装器 compile-only 通过；独立静态审查确认共享方法物化调用链、五个入口依赖及源项目一致性。未运行测试、最终构建、安装、验证脚本、网络或宿主行为评估；用户级技能和旧 CLI/历史资料保持本 FD 明确范围。

## Sources

- 实现证据：reports/FD-066-implementation-r1.md 与同名 JSON；安装器 compile-only 通过，一次静态核对确认 25 组技能副本、29/22/5 份共享副本一致。未运行测试、安装或宿主行为验证。

- Issue: none

- 用户对整体方案及删除旧 Task 内容的明确确认；FD-063 至 FD-065 已完成的本地生命周期约定。
- src/skills/work-management.md、issue-management、fd-workflow、fd-review、code-review、to-spec、to-tickets、finance-release-gate。
- src/plugins/aiw-skills/aiw-skills.py 的共享参考物化实现；openspec/specs/fd-workflow/spec.md 与适用 Issue/skills 稳定规格。

**Completed:** 2026-10-16
