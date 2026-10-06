# FD-029: Squash FD local delivery without rebase

**Status:** Complete
**Revision:** 18
**Priority:** Medium
**Test policy:** Independent
**Evidence policy:** Dual

## Problem

FD worktree 中逐项提交便于回退和审查，但父分支不需要保留这些提交历史。现有流程又要求实现结束后 rebase，并在 `local-merge` 中直接合并 FD 分支，导致父分支带入 Work Item 提交或合并节点。用户要求取消 rebase，并让本地交付只形成一笔 squash 提交。

## Options and decision

| 方案 | 取舍 |
| --- | --- |
| 保留 rebase 和普通 merge | 与用户要求相反。 |
| 在父分支 cherry-pick 后再 squash | 仍需逐项重放提交，增加冲突面。 |
| `local-merge` 在父分支执行 squash，冲突时仍在 FD worktree 解决 | 保留逐项本地提交；父分支只接收一笔交付提交。 |

选择第三项。交付提交写入 FD ID 和被交付的 FD HEAD，供归档后安全核对并删除不属于父分支祖先的 FD 分支。

## Solution

保留 `aiw wt local-merge <fd-id>` 命令和已记录的父分支目标。命令要求父、FD worktree 均干净并位于记录的分支；在父分支 squash FD HEAD，成功后创建单父提交并记录源 SHA。若 squash 产生内容冲突，先安全恢复父工作区，再把父分支合入 FD worktree；解决并提交冲突后显式重试 `local-merge`。不得 rebase FD 或父分支，不推送远端。归档清理先核对交付提交的源 SHA 与 FD HEAD，再删除 worktree 和分支。

## Scope

- 修改 FD 原生 `aiw wt local-merge` 的交付与内容冲突恢复路径。
- 撤销自动 rebase 规则，保留逐 Work Item 的局部提交。
- 同步工作流技能、共享约定、稳定规格和使用文档的交付与清理说明。
- 不修改 legacy Task 命令，不推送或部署。

## Work items

- [x] 1.1 `local-merge` 在记录的父分支创建一笔 squash 交付提交，包含 FD ID 与源 HEAD；父分支不包含 Work Item 的提交历史。Size: M; difficulty: Medium; dependencies: none.
- [x] 1.2 squash 内容冲突时恢复干净的父分支，并将父分支合入 FD worktree 供解决；显式重试才能交付。Size: M; difficulty: Medium; dependencies: 1.1.
- [x] 1.3 移除自动 rebase 的技能与规格要求，并调整归档后分支清理的安全核对和使用文档。Size: M; difficulty: Medium; dependencies: 1.1.
- [x] 1.4 静态审查最终 diff、编译 Python 模块、记录未执行测试与残余风险。Size: S; difficulty: Low; dependencies: 1.1–1.3.

## Acceptance

- 同一 FD 的多个 Work Item 提交在 FD 分支保留；父分支交付后只增加一笔带 FD ID 与源 HEAD 的单父提交。
- 命令只交付到 `workspace.json` 记录的父分支；脏工作区、错误分支或不安全状态在改动父分支前拒绝。
- squash 内容冲突后父分支保持原 HEAD 且干净，冲突转到 FD worktree；解决提交后第二次调用完成 squash。
- 归档后仅在交付提交的源 HEAD 等于当前 FD HEAD 时删除 worktree 和分支；自动流程不运行 rebase。

## Verification

- 静态检查：`git diff --check develop...HEAD` 无空白错误；比较 `plugins/aiw-wt.py` 的调用路径，并搜索技能、规格、README 和使用文档中的 rebase 与 squash 说明。
- 编译检查：使用 Python `compile()` 在内存中编译 `plugins/aiw-wt.py`，退出 0，无产物。
- R1 独立 Tester 未运行测试，场景 0/12；PM 退回并保留该历史记录。
- 用户授权的 R2 聚焦命令 `python -B -m unittest tests.test_fd029_wt_squash -v` 由独立 Tester 只运行一次：5/5 通过，12 个场景中的 10 个有运行证据，需求覆盖率 83.33%。PM 有边界接受并交独立 Reviewer；业务代码分支覆盖率未测量。
- R1 独立 Reviewer：`docs/features/reviews/FD-029-review-r1.md`；发现 `local-merge` 重复调用会因 `--allow-empty` 产生第二笔空交付提交，结论为 changes-requested。审查没有运行新测试或构建。
- R3 Worker 已提交 `56a401a`：在 squash 前查询当前源 HEAD 的历史交付标记，重复交付提前拒绝，并移除 `--allow-empty`；增加重复调用黑盒用例。静态检查改动路径后，使用 Python `compile()` 在内存中编译插件与测试模块，退出 0。新增用例尚未运行，等待独立 Tester 的新证据。
- R3 独立 Tester：`docs/features/reports/FD-029-test-report-r3.md`；用户授权的同一聚焦命令本轮仅运行一次，6/6 通过；13 个场景中 11 个有运行证据，需求覆盖率 84.62%，业务代码分支覆盖率未测量。PM 在 `docs/features/reports/FD-029-test-decision-r3.md` 接受有边界的证据；S11–S12 未运行。
- R2 独立 Reviewer：`docs/features/reviews/FD-029-review-r2.md`；对照修复提交 `56a401a`、稳定规格、R3 原始测试和授权记录，结论为 verification-passed；未运行新测试或构建，保留归档清理未运行风险。
- 未运行最终产物构建、格式化、lint、网络或部署。
%% RISK: 归档后来源 SHA 不匹配时保留分支、匹配时清理 worktree/分支（S11–S12）尚无本轮运行证据；不得描述为通过。

## Sources

- Issue: none
- 用户决策：逐 Work Item 提交；取消 rebase；`local-merge` 使用 squash，不在父分支保留 worktree 提交历史。
- `openspec/specs/fd-workflow/spec.md`、`openspec/specs/cli-and-plugins/spec.md`、`openspec/specs/command-help-consistency/spec.md`。
- `plugins/aiw-wt.py`、`skills/fd-workflow/SKILL.md`、`skills/work-management.md`。

**Completed:** 2026-10-06
