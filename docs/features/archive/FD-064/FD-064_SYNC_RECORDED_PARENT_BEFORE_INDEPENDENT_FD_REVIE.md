# FD-064: 按需恢复 FD 本地交付失败

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem

FD-063 因父分支并发删除导致交付冲突，用户手动恢复后成功交付。先前拟定每次审查前主动同步，用户随后纠正：保持单个 FD 的独立实现，直接尝试交付，失败后才按原因处理父分支同步。恢复后的实现变化需要真实审查依据，不能误认为 commit 自动解决冲突或 local-merge 自动归档。

## Options and decision

按用户最新决定，保留正常实现、独立审查、直接 local-merge 的顺序，不主动 sync，也不因父分支单纯前进而强制合入或重新审查。失败后先诊断；现有 CLI 已可能在冲突恢复时将父分支合入 FD，不应盲目重复 sync。仅文档澄清恢复及证据要求，不修改 CLI。

## Solution

正常完整隔离流程不额外同步父分支。local-merge 失败时保留完整输出，核对父/FD 状态、MERGE_HEAD 和冲突索引，区分已交付但清理失败、脏工作区、实际冲突及其他错误。用户进一步确认：实际冲突自动加载 resolving-merge-conflicts，进行一次有明确证据的解决尝试；无法安全解决时保留合并现场、剩余冲突及已明确解决的修改，不 abort/reset 或提交部分合并。若 CLI 已开始父分支合入，不重复 sync。确需额外同步时使用登记父分支显式参数。提交前逐项确认解决内容，不将 git add -A 视为解决方案。恢复造成实现变化时，通过 managed set-status Pending Verification 与 request-review 请求新的独立审查，记录恢复提交和 diff，保留历史报告和累计审查上限；不得接管在途角色或沿用不覆盖变化的旧通过。再次满足 Gate 后交付、清理、close 归档。

## Scope

仅修改 fd-workflow、implement、fd-review、resolving-merge-conflicts、PM 角色与 portable operations 文档、root AGENTS、共享契约、稳定规格及项目副本。明确 local-merge 清理与 close 归档分离，收紧冲突技能中无条件运行检查与全部暂存的旧指令。不整合 implement 目录，不改 CLI、事件 schema、依赖、安装、网络或测试权限；主工作区模式保持原约定。继承显式阶段/部分范围限制。

## Work items

- [x] 1.1 更新共享契约及技能的按需恢复约定。规模小、难度中，无依赖；常规路径不主动 sync，失败先诊断，明确可解冲突加载技能尝试，无法解决则保留现场，恢复变化需要审查覆盖。
- [x] 1.2 更新稳定规格和项目生成副本。规模小、难度低，依赖 1.1；明确清理/归档边界、登记父分支参数、主工作区兼容和恢复审查路线。
- [x] 1.3 一次静态一致性核对、双格式证据及独立审查交接。规模小、难度低，依赖 1.2；报告实际命令和未运行检查。

## Acceptance

- 完整隔离生命周期顺序为实现提交、独立审查、直接 local-merge 清理、close 归档和归档提交；不主动 sync，不强制父分支先成为 FD 祖先。
- 失败先诊断，不盲目重复 sync；确需同步时使用登记父分支显式参数，记录恢复提交和 diff。
- 恢复后实现变化不得沿用不覆盖变化的旧通过；通过 managed review 恢复，并保留审查计数与真实证据。
- 实际冲突加载 resolving-merge-conflicts 进行一次有证据的解决尝试；无法安全解决时保留现场，不强制选边、abort/reset 或部分提交。脏工作区、在途角色和阶段限制保留原停止约定；不把 git add/commit 视为语义解决。
- canonical/项目副本一致；不修改 CLI，不新增运行验证权限。

## TODO

- 实现完成，独立审查 r1 通过；已 squash 到 develop（e9235a8），CLI 清理 feature/FD-064 与 .wt/FD-064，宿主 close 归档并提交归档记录。

## Verification

- 纯文档改动，无适用编译目标。一次静态核对 diff、失败诊断、登记父分支参数、审查恢复路线、显式阶段限制及生成副本一致性；不运行测试、构建、验证脚本或网络。

- Worker 证据：`reports/FD-064-implementation-r1.md` 及同名 JSON；最后策略修改后的最终副本一致性由独立 Reviewer 核查。
- 独立静态审查通过：`reviews/FD-064-review-r1.md` 及同名 JSON；审查范围 `8f3f432..ce5a5ea`，31 个共享副本和六组技能文件一致。未运行测试、编译、构建或网络；未运行实际冲突恢复。

## Delivery record

- 2026-10-17，独立审查事件 FD-064-000005-verification-passed，FD revision/digest 与交付前文件一致。审查报告位于 reviews/FD-064-review-r1.md 及同名 JSON。
- 源 HEAD：941ba25299e0c7cceef9b29ad5c538af247f8fe5；父分支 develop 的 squash：e9235a8。审查后仅添加审查证据、FD 和索引，无实现漂移。
- 实际执行 aiw git wt local-merge FD-064，成功交付并校验清理；随后执行 aiw fd close FD-064 Complete，归档由宿主单独完成。本次未执行 sync，也未发生实际冲突。
- 最终独立静态核对确认 31 个共享副本和六组技能文件一致。未运行编译（无适用目标）、测试、构建、格式化、lint、验证脚本或网络；实际冲突恢复仍未运行验证。

## Sources

- Issue: none

- 用户先确认工作流优化，随后明确纠正常规路径不主动 sync，仅交付失败后按需处理。最新决定优先。
- 用户随后明确选择“自动尝试明确可解的冲突，无法解决则保留现场”。此决定替代本 FD 初稿中冲突先等待恢复指令的策略。
- FD-063 归档及冲突反馈；src/plugins/aiw-git/git-wt.py 的 sync/local_merge/commit 静态实现。
- src/skills/work-management.md、fd-workflow、implement、fd-review；openspec/specs/fd-workflow/spec.md。

**Completed:** 2026-10-16
