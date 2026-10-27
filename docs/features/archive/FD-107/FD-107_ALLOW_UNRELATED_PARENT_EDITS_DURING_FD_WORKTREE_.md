# FD-107: Allow unrelated parent edits in FD worktree operations

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Review policy:** Independent
**Evidence policy:** Markdown

## Problem

用户要求完整工作均留在 FD worktree 后，不应再因主工作区其他 FD 的未提交修改阻止创建或无冲突交付。现有技能规则和 git-wt.py 都要求整个 parent 干净。

## Solution

创建仅检查当前 FD 文档及宿主根据实际 diff 确定的相关路径；发现相关修改时要求选择先提交，或明确从 HEAD 创建并保留、不携带 parent 修改。无关修改保留。交付时保持 FD worktree 干净；允许不与交付路径冲突的 parent 已暂存/未暂存/未跟踪修改。使用独立 index 形成仅含 FD 的 squash，绝不混入用户原有 index。分支内容冲突且 parent 不干净时，在修改 parent 前停止，要求先整理干净再恢复冲突。Git 判定会覆盖本地修改的路径也先要求整理，不自动 stash/reset。

## Scope

aiw git wt add/local-merge/cleanup、共享工作合同、fd-workflow preflight/delivery、相关 AGENTS 和使用文档。保留既有 FD-Source/单父提交、归档、审查和坐标校验。不涉及 customer-profile FD-027 内容、FD-011 实现、发布安装或测试执行。

## Work items

- [x] 1.1 创建按 FD 相关路径判定；明确 from-head 选择，不遗漏或擅自提交其他修改。
- [x] 1.2 允许无冲突 dirty-parent squash，隔离原 index；冲突与失败保留用户修改，清理仍验证交付。
- [x] 1.3 同步 canonical 技能/仓库规则及使用说明，编写未执行的定点回归用例并记录静态/编译证据。

## Acceptance

- 无关 parent 修改不阻止 add；相关修改无显式选择即拒绝；from-head 不携带修改且主工作区原状保留。
- 无冲突交付不混入原 staged 内容，原修改和 staging 保持；有分支冲突且 dirty 时未修改 parent 即停止。
- FD worktree 仍须干净；FD-Source、单父 squash 与坐标检查保留，cleanup 不再仅因无关 parent 修改失败。

## Verification

已静态阅读创建、交付、清理、独立 index 和冲突恢复路径；一次 scoped git diff --check 通过。
已运行 GOPROXY=off GOSUMDB=off python scripts/compile.py（已结束且无错误输出，独立退出码未捕获，不声称 Go 编译已通过）；其后同一编译命令块对修改的 Python plugin/新增测试内存 compile()，不 import、不留产物。
新增 tests/test_fd107_parent_edits.py 覆盖无关修改创建、当前 FD 显式选择、staged 排除/保留、分支冲突拒绝；同步旧用例的预期。测试未执行。
测试、实际 Git 临时仓库回归、技能校验器、安装、打包、发布、联网均未运行。源码改变尚未同步到 /home/victor/apps/aiw 分发安装或 customer-profile 已安装技能，不能声称生效。
Independent 原因：Git index/工作区持久状态及CLI契约改变，交付前需要独立审查。

## TODO

实现、静态检查和独立复核已完成。现有无关 .agents 修改不提交。customer-profile FD-011 待规则实现后的可用工具恢复。

## Sources

用户本轮要求；src/plugins/aiw-git/git-wt.py；src/skills/work-management.md。

### 独立审查 r1

Reviewer fd107-reviewer-20261027-73a9c4 实际认领 FD-107-000002-review-requested，
对 HEAD 61a9040008b47763b1513069bb5dbfcf75ffe300 的限定未提交 diff 完成静态审查。
结论 changes-requested，见 [review r1](reviews/FD-107-review-r1.md)：R1/P2 稳定 FD workflow spec 仍要求 clean parent/workspaces，需与本次规则同步。
累计失败 1/3；未测试、未执行业务/Git 场景、未提交。Host 的 Python 内存编译通过与 Go 脚本独立退出码未捕获分别记录，不混称通过。

### Worker repair r2

实际 claim FD-107-000003-changes-requested，session fd107-worker-20261027-621f08。
R1 修复：同步 openspec/specs/fd-workflow/spec.md 的创建、交付、dirty-parent冲突恢复和清理边界，新增相关/无关修改及暂存保留的可观察场景。没有创建 OpenSpec change。
保留独立审查失败 1/3 和 r1 原报告。本轮仅规格/证据修订，未重复执行编译或测试。

### 独立聚焦复核 r2

Reviewer fd107-reviewer-20261027-73a9c4 实际认领 FD-107-000004-implementation-ready；
R1 稳定规格已解决，结论 pass，见 [review r2](reviews/FD-107-review-r2.md)。
共 2 轮，历史失败累计 1/3 保留。修复仅涉及规格/证据，本轮没有实现漂移；未执行测试或 Git 场景、未提交/安装/发布。

**Completed:** 2026-10-27
