# FD-072: Rename release gate and clarify release assessment evidence

**Status:** Open
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户要求将已跨领域使用的 finance-release-gate 改名 fd-release-gate，并询问质量改进。旧名误导领域定位，现有定义缺少统一合同、证据版本及 GO_WITH_RISK 的清晰界限。

## Options and decision

采用一个正式入口 fd-release-gate，更新活跃引用、项目副本和安装清单；不保留重复薄入口。保留历史FD原文与外部全局安装，明确旧名称迁移。仅强化只读评估，不新增部署审批或自动运行。

## Solution

Skill仍跨领域、按真实发布触发，FD可选，非每个FD的必经环节。输入增加目标环境、变更/候选版本、证据来源时间、适用性和实际验证；证据缺失不伪造。关键缺口NO_GO；GO_WITH_RISK只能用于有明确责任人和明确接受证据的非关键剩余风险，不能由模型代为接受。给出具体下一步、未知owner明确缺失。保持现有只读建议边界，补合同引用、未决/完成/静态验证范围。

改名是用户明确要求的Skill调用接口变化，更新本仓库所有活跃引用并移除旧项目入口/manifest键；不更改产品CLI/API、依赖、持久化或部署机制，不需要额外独立设计评审。历史引用不重写，发行及全局用户副本不安装，可能需用户后续同步并移除旧副本。

## Scope

范围：源/项目Skill目录改名、README、活跃SKILL/共享合同/目录/manifest/稳定FD规格引用、双格式证据与独立评审。不调整resolving-merge-conflicts或github-issues-management，仅在最终答复评估。无远程请求、发布、测试、安装、CLI状态改动。

## Work items

- [ ] 1.1 改名并强化发布评估指令和README。Size: S; Difficulty: Low; Dependencies: none. Completion: 新name/目录一致，GO决策有证据与适用性，不授部署权限。
- [ ] 1.2 更新活跃调用方、项目副本和安装记录。Size: S; Difficulty: Low; Dependencies: 1.1. Completion: 活跃资料无旧标识调用，副本一致，旧入口删除且历史不变。
- [ ] 1.3 静态检查、独立评审和交付归档。Size: S; Difficulty: Low; Dependencies: 1.2. Completion: 双格式证据、真实独立评审和本地合并归档。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- fd-release-gate为唯一活跃入口，名称/引用/清单一致；旧Skill移除，历史保留，未声称已安装发行/全局副本。
- 跨领域/可无FD/只在真实发布使用；本地FD评审合并归档不自动触发，GO不允许发布。
- 候选版本/环境与证据绑定，过期/未执行/不适用分别记录，关键缺失NO_GO，不伪造owner或验证结果。
- GO_WITH_RISK有非关键剩余风险、责任人和实际接受证据，模型不能自行接受；检查不适用必须说明原因，未知阻断清楚。
- 输出有范围、证据、判断和下一步，遵循只读/现有授权预算，无新增网络/模型/部署或测试。

## Verification

- Compile-only check and static review by default; record unrun tests explicitly.
- For explicitly requested test support (any wording/language), record suitable
  TDD scope, independent fd-test scope, command/revision authorization and finite
  budget here. Otherwise no testing stage is added. On exhausted unexpected-
  failure budget, stop affected test repair/execution, diagnose statically and
  continue independent work; unresolved acceptance defects still block delivery.

## Sources

## TODO

- [ ] 完成1.1–1.3及独立评审。
- [ ] 本地交付和归档。

## Sources

- Issue: none

Link relevant specs and evidence used for decisions.
