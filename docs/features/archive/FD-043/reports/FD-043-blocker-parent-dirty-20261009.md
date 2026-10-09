# FD-043 自动流程阻塞反馈

<!-- aiw-data: FD-043-blocker-parent-dirty-20261009.json -->

## 定位

- FD：`FD-043`
- 阶段与角色：Preflight / PM
- 关联交接事件：`FD-043-000003-design-ready`，仍为 pending Worker，未认领。
- 记录日期：2026-10-09，Asia/Tokyo。
- 记录者：当前 fd-workflow auto 主会话。

## 阻塞事实

- `git status --short` 显示父工作区分支 `office-dev` 存在修改的 `docs/features/FEATURE_INDEX.md`，以及未跟踪的 FD-042、FD-043 设计文件。
- 索引的现存改动同时包含 FD-042 和 FD-043；FD-042 来源于本会话之前已完成的 REQ00008 promote，不属于 FD-043 的交付范围。
- 已确认根因：父工作区包含另一个 FD 的未提交改动；本次 auto 只明确授权 FD-043 的本地 Git 生命周期。
- 已尝试恢复：读取状态、FD-043 计划和待领取事件；未运行 worktree 创建、stash、恢复/删除其他 FD 改动或提交 FD-042。
- 当前状态：未解决。单独提交 FD-043 计划和本反馈不会消除 FD-042 与索引的未提交改动。
- 需要人工决策：是否允许把 FD-042 计划及其索引行单独提交，随后再单独提交 FD-043 索引行并继续 auto；或由用户自行处理剩余父工作区改动。

## 结果与改进

- 实际解决方案：尚未解决；保留 FD-042 文件与待领取 Worker 事件。
- 下一步：父工作区干净后，重新核对事件、认领 Worker，创建 `feature/FD-043` / `.wt/FD-043`。
- 未执行：实现、compile-only、测试、构建、Reviewer、合并与归档。
- 可复用改进：多个 FD 共用索引时，计划提交应按 FD 分开暂存，避免把无关索引行带入另一 FD；本反馈不授权提交其他 FD。

依据：`.agents/skills/fd-workflow/SKILL.md` 的 Preflight 要求父工作区干净且禁止混入无关改动；`AGENTS.md` 要求 Git 写操作具有用户授权。

## 恢复记录

2026-10-09 再次预检时，office-dev 父工作区已干净；已创建 feature/FD-043 与 .wt/FD-043，并核对 workspace.json 四项坐标。阻塞已解决，无需进一步人工决定。旧阻塞事实保留。
