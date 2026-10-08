# FD-030: 将 FD worktree 管理并入 aiw-git 并在交付后清理

**Status:** Complete
**Revision:** 17
**Priority:** High
**Test policy:** Independent
**Evidence policy:** Dual

## Problem

FD worktree 命令目前由独立的 `aiw-wt` 插件提供；用户希望把它合并进 `aiw-git`，并使用 `aiw git wt`。成功执行 `local-merge` 后，当前实现保留 `.wt/<FD-ID>` worktree 和 `feature/<FD-ID>` 分支，产生过期资源。`.ai` 是独立于 Git 管理、供 FD 流程共享的目录；但 `aiw-fd` 当前把每个 worktree 根目录当作收据根目录，导致在 FD worktree 内无法访问主工作区的 handoff 收据。用户后续明确不需要旧 `aiw wt` 命令兼容入口。

## Options and decision

| 方案 | 取舍 |
| --- | --- |
| 保留独立 `aiw-wt` 插件 | 不满足合并要求，且清理行为仍散落在专用入口。 |
| 把 FD worktree 实现放进 `aiw-git` 的 `wt` 子命令，并保留旧入口兼容 | 避免破坏既有调用，但用户明确无需兼容。 |
| 删除旧入口并只支持 `aiw git wt` | 命令归入 Git 插件，避免与原生 `git worktree` 冲突，接口唯一。 |

最初实现时选择了保留兼容入口；用户后续明确选择第三方案，因此最终只支持 `aiw git wt` 并删除独立 `aiw-wt` 插件。`local-merge` 只有在 squash 提交成功并再次核对交付源 SHA 后，才执行 FD worktree 与分支清理。

## Solution

将 FD worktree 命令实现放在 `plugins/aiw-git/git-wt.py`，由 `aiw-git` 子命令发现机制提供唯一入口 `aiw git wt`；删除 `plugins/aiw-wt.py`，不保留 `aiw wt` 命令。`aiw-fd` 从当前 Git worktree 列表解析主 worktree，并将所有 handoff 收据和锁写入主工作区的共享 `.ai/fd/<FD-ID>/`；FD 文档仍从当前项目树读取。成功交付后先确认 FD 与父工作区均符合清理前置条件、`.wt/<FD-ID>` 与 `feature/<FD-ID>` 和 workspace metadata 完全匹配、父分支历史中存在单父 squash 提交且其 `FD-Source` 等于当前 FD HEAD。然后安全移除 worktree，再删除分支。若 worktree 内 `.ai` 是 junction/reparse point，只能在确认其目标是本仓库主工作区 `.ai` 后删除链接本体；未知类型或目标时停止并保留 worktree/分支。清理部分失败时报告已完成步骤与可执行恢复方式，不回滚已成功的交付提交。

## Scope

- 把 FD 命令实现并入 `aiw-git`，只公开 `aiw git wt` 并删除独立的 `aiw-wt` 插件。
- 成功 squash 交付后校验来源，再自动删除该 FD 的 worktree 和分支。
- 明确并实现 Windows `.ai` junction 的安全处理与部分清理失败提示。
- 让 `aiw-fd` 从任一 worktree 访问主工作区的共享 `.ai/fd` 收据。
- 更新稳定规格、帮助一致性和 README/工作流文档。
- 不修改 legacy Task worktree 生命周期，不推送远端，不清理 `.ai` 下的 FD 收据。

## Work items

- [x] 1.1 将 FD worktree 子命令接入 `aiw-git`，最初保留旧入口兼容。该决定被后续用户决策 supersede，见 1.6。Size: M; difficulty: Medium; dependencies: none.
- [x] 1.2 在 `local-merge` 成功后核验单父 squash 提交、FD-Source、记录分支和 worktree 干净状态，再串联清理并处理部分失败。Size: M; difficulty: Medium; dependencies: 1.1.
- [x] 1.3 实现 Windows `.ai` junction/reparse point 的安全识别与链接本体删除；目标不匹配或类型未知时停止清理。Size: M; difficulty: Medium; dependencies: 1.2.
- [x] 1.4 更新 FD workflow、CLI 与 help consistency 稳定规格，以及用户文档的命令与生命周期说明。Size: S; difficulty: Low; dependencies: 1.1–1.3.
- [x] 1.5 静态审查调用路径，执行允许的 Python 内存编译检查，并记录未运行的测试与平台风险。Size: S; difficulty: Low; dependencies: 1.1–1.4.
- [x] 1.6 按用户后续决定删除 `plugins/aiw-wt.py`，清除活跃文档和 Skills 中的旧命令及兼容承诺。Size: S; difficulty: Low; dependencies: 1.1, 1.4.
- [x] 1.7 从 Git worktree 列表解析共享 `.ai/fd` 根目录，并更新 FD 文档与实施证据。Size: S; difficulty: Low; dependencies: none.
- [x] 1.8 明确 FD 角色边界与独立性，并增加按当前阶段加载的 Planner、Worker、Tester、Reviewer、PM 短提示。Size: S; difficulty: Low; dependencies: 1.4.
- [x] 1.9 按 Reviewer R1 迁移旧 aiw-wt 测试调用到 aiw-git dispatcher，重写重复交付终态并把 Task 专属冲突用例转换为 FD 冲突恢复用例。Size: S; difficulty: Low; dependencies: 1.1–1.3.
- [x] 1.10 按 Reviewer R2 删除未记录的 `aiw git wt ls` 别名，仅保留已声明的 `list` 命令。Size: XS; difficulty: Low; dependencies: 1.1, 1.4.

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- `aiw git wt add|status|commit|local-merge|list` 执行现有 FD 管理行为；不存在独立 `aiw wt` 命令或转发插件。
- 未完成 squash、冲突恢复或交付来源校验失败时，worktree 与分支均保留。
- 成功 squash 后仅当记录路径、分支、clean 状态及 `FD-Source` 与当前 FD HEAD 全部匹配，才自动移除对应 worktree 和分支。
- 清理前安全摘除已验证指向主工作区 `.ai` 的 junction 本体；不删除 junction 目标、不清理 `.ai/fd/<FD-ID>/` 收据。遇到未知 reparse point 或非预期目标时，拒绝清理并说明原因。
- 清理失败不得回滚 squash 提交；必须报告已完成/未完成的步骤与恢复命令，保留可用于恢复的 metadata。
- README、顶层帮助、规则和稳定规格一致说明唯一入口、成功自动清理与失败保留行为。
- 从任一 linked worktree 执行 FD 命令时，事件与锁均使用主 worktree 下同一 `.ai/fd/<FD-ID>`；FD Markdown 仍从当前项目树解析。
- 共享契约说明职责和独立性；自动流程只加载当前阶段的角色提示，Tester/Reviewer 证据来自与 Worker 不同且彼此分离的会话。
- 活跃测试不引用已删除的 `plugins/aiw-wt.py`；worktree 生命周期场景通过 `plugins/aiw-git/aiw-git.py` 的 `wt` dispatcher 调用；成功清理后的重复操作从仍存在的父 worktree 验证未注册终态。
- `aiw git wt` 仅接受声明的 `add|status|commit|local-merge|list` 命令；`ls` 返回未知命令错误。

## Verification

- 静态检查新旧插件分发、metadata 校验、squash 后清理路径和错误返回；检查 README、稳定规格和工作流文档的一致性。
- 编译检查将在本次所有代码编辑完成后，对 `plugins/aiw-git/git-wt.py`、`plugins/aiw-git/aiw-git.py` 和 `plugins/aiw-fd.py` 执行 Python 内存编译；不产生编译产物。
- 未运行测试、最终构建、格式化、lint、网络或部署；清理分支、Windows reparse 与共享 handoff 的运行行为没有运行时证据。
- Worker handoff 曾因 `aiw-fd` 把当前 worktree 根目录当作 `.ai` 根目录而未登记；本次修复将运行收据定位到主 worktree 的共享 `.ai/fd`。
- 角色提示改动属于文档静态审查；尚未通过运行新的 FD handoff 流程验证主 Skill 与各阶段提示的实际组合效果。
- 实施报告：`docs/features/reports/FD-030-implementation-r1.md` 与同名 JSON。
- 独立 Reviewer R1：changes-requested；source event FD-030-000008-test-accepted，报告 docs/features/reviews/FD-030-review-r1.md。三项遗留测试入口引用需迁移；Tester 0/20 覆盖及运行时风险仍未解除。
- Worker R2 静态修复 Reviewer R1 的三项 finding：FD-027/FD-029 测试改用 aiw-git dispatcher；FD-029 重复交付检查自动清理后的终态；旧 Task 专属冲突 fixture 移除并转换为根 `tests/` 下的 FD 冲突恢复场景。测试未运行；Tester R1 的 0/20 覆盖仍是历史事实。
- Tester R2：handoff FD-030-000010-implementation-ready；授权命令因五个 unittest 模块导入失败而未执行行为断言，覆盖 0/20、branch coverage 未测量。报告 docs/features/reports/FD-030-test-report-r2.md。
- PM R2 按用户 2026-10-08 指示豁免本 FD 后续测试，仅接受报告以进入独立 Reviewer；用户将在实际使用中验证。决策 docs/features/reports/FD-030-test-decision-r2.md。该例外不代表测试通过。
- Reviewer R2：docs/features/reviews/FD-030-review-r2.md，changes-requested；发现源码接受但 metadata/help/README/spec 未列出的 `aiw git wt ls` 别名。
- Worker R3 移除 `ls` dispatch 别名，保留 `list` 为唯一列出命令；未运行测试。
- Python 内存编译 `plugins/aiw-git/git-wt.py` 通过；静态确认 dispatch 与 usage/help 命令面一致。
- PM 按用户已记录的测试豁免取消本轮 pending Tester handoff，直接请求独立 Reviewer；R3 未运行测试，不代表测试通过。
- PM 根据用户 2026-10-08 明确指示“Verification Pass”直接将状态设为 Complete。此为 PM gate override，不代表 Reviewer R3 已完成或独立审查通过；决策见 `docs/features/reports/FD-030-verification-override-r1.md` 与同名 JSON。
- `FD-030-000016-review-requested` 收据未修改，仍记录为 pending，且绑定本次覆盖前修订版 16。状态记录更新至修订版 17 后，该 handoff 不再绑定当前 FD；没有伪造或补写 Reviewer 结果。

%% RISK: 用户豁免 FD-030 后续自动测试并计划在实际使用中验证。Tester R2 覆盖为 0/20（模块导入错误，未运行行为断言）；自动 worktree/分支清理、Windows junction、冲突恢复、共享 handoff 和部分清理失败仍无运行时证据，不得描述为已验证。

## Sources

- Issue: none
- 用户决策：合并进 `aiw-git` 并只使用 `wt` 子命令，不保留旧命令；`local-merge` 成功后自动清理 FD worktree 和分支。
- `docs/features/archive/FD-027/FD-027_WT_USES_FD_WORKTREES.md`
- `docs/features/archive/FD-028/reports/FD-028-blocker-20261006T160953Z-cleanup.md`
- `docs/features/archive/FD-029/FD-029_SQUASH_FD_LOCAL_DELIVERY_WITHOUT_REBASE.md`
- `openspec/specs/fd-workflow/spec.md`、`openspec/specs/cli-and-plugins/spec.md`、`openspec/specs/command-help-consistency/spec.md`
- `plugins/aiw-wt.py`、`plugins/aiw-git/aiw-git.py`、`skills/work-management.md`、`skills/fd-workflow/SKILL.md`
