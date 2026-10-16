# FD-064: Sync recorded parent before independent FD review

**Status:** Open
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

FD-063 在独立审查后才发现父分支并发删除引起的冲突。当前 auto 直接尝试 squash，失败后才在 worktree 合并父分支，最终交付内容可能没有被当前审查覆盖。用户确认采用先同步、再审查、再交付归档的顺序。

## Options and decision

选择调整宿主技能流程及共享契约，不改 CLI 的独立命令行为。单纯增加 sync 命令但放在审查后仍会留下证据缺口，因此要求审查前完成同步并记录父分支 OID；交付前父分支若变化，旧通过结果不能直接交付。

## Solution

完整隔离实现完成并提交后，在发出审查交接前读取登记坐标，以显式父分支参数执行 aiw git wt sync，确认父分支提交包含于 FD HEAD。记录同步后的实现提交与父 OID，由独立 Reviewer 核对。交付前再次比较父 OID；有变化时停止交付，不在旧审查后静默同步。由受支持的 managed request-review 路径重新请求审查，宿主先同步并更新独立补充证据，Reviewer 对同步后的结果审查；保留历史证据及累计审查上限。若正在派发则不接管。实际冲突仍停止并等待明确恢复指令；commit 不等于解决冲突。

## Scope

仅修改 fd-workflow、implement、fd-review、角色与 portable operations 文档、共享契约、稳定规格及项目副本。明确 local-merge 清理与 close 归档分离。不整合 implement 目录，不改 CLI、事件 schema、依赖、安装、网络或测试权限；主工作区模式跳过同步。继承显式阶段/部分范围限制。

## Work items

- [ ] 1.1 更新共享契约及技能、角色的同步和交付约定。规模小、难度中，无依赖；保证阶段限制先判断、同步先于审查，父分支漂移不沿用旧通过结果。
- [ ] 1.2 更新稳定规格和项目生成副本。规模小、难度低，依赖 1.1；明确清理/归档边界、同步冲突停止、主工作区兼容和恢复审查路线。
- [ ] 1.3 一次静态一致性核对、双格式证据及独立审查交接。规模小、难度低，依赖 1.2；报告实际命令和未运行检查。

## Acceptance

- 完整隔离生命周期顺序为实现提交、同步登记父分支、独立审查、父 OID 复核、local-merge 清理、close 归档和归档提交。
- 同步使用登记父分支显式参数，记录 parent OID 和 reviewed implementation HEAD；独立审查覆盖同步结果。
- 审查后父分支变化时不得直接交付或静默同步后沿用旧通过；通过 managed review 恢复，并保留审查计数与真实证据。
- 实际冲突、脏工作区、在途角色和阶段限制保留原停止约定；不把 git add/commit 视为语义解决。
- canonical/项目副本一致；不修改 CLI，不新增运行验证权限。

## TODO

- 按上述工作项实施、独立审查，通过后自动本地交付并归档。

## Verification

- 纯文档改动，无适用编译目标。一次静态核对 diff、同步参数与 OID、审查恢复路线、显式阶段限制及生成副本一致性；不运行测试、构建、验证脚本或网络。

## Sources

- Issue: none

- 用户对同步、独立审查、交付、归档顺序的确认。
- FD-063 归档及冲突反馈；src/plugins/aiw-git/git-wt.py 的 sync/local_merge/commit 静态实现。
- src/skills/work-management.md、fd-workflow、implement、fd-review；openspec/specs/fd-workflow/spec.md。
