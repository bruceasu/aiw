# FD-077 父工作区门槛反馈

<!-- aiw-data: FD-077-blocker-20261018-parent-dirty.json -->

## 定位

- 阶段与角色：preflight，PM / Planner。
- 事件：FD-077-000002-design-requested；由 fd077-host-20261018-53b7ac claim。
- 日期：2026-10-18（Asia/Tokyo）。

## 阻塞事实

`git status --short` 显示未跟踪的 `docs/issues/ISSUE-00008/`，它属于此前文件转换插件需求，不属于 FD-077。当前分支为 develop。仓库 AGENTS.md 要求先提交当前 FD 计划，并在父工作区干净后创建 worktree；不能把其他工作混进计划提交。

已批准 ISSUE-00009，promote 为 FD-077，claim Planner 事件，并记录范围及门槛；未创建 worktree、未修改实现。已通过异步问题请求独立提交 ISSUE-00008 文档的授权，尚未收到答复；未暂存、删除或提交该目录。

当前状态：未解决。需要用户授权独立提交此前已确认的 ISSUE-00008 文档，或由用户自行处理该目录使父工作区干净。ISSUE-00009 与 FD-077 本身可以按本次生命周期授权进行聚焦提交，但不能借此包含 ISSUE-00008。

## 后续

收到授权或父工作区恢复干净后，从 Planner 继续完成设计；未发 design-ready，不存在 Worker 实现或 Reviewer 通过结果。测试、编译、构建、网络均未执行。

可复用改进建议：在需求内容 capture 时明确区分正式文件保存与 Git 提交授权；跨 Issue 切换前提示残留的正式文档。该建议尚未实现，不授权修改工作流规则。

## 已解除

用户已明确授权单独提交 ISSUE-00008 文档并继续。该目录已独立提交到 develop，提交 a732b98；原事实和等待记录作为历史保留。当前阻塞已解除；剩余未提交内容属于 ISSUE-00009/FD-077，将按 ready plan 范围聚焦提交后再创建 worktree。解除时间见同名 JSON。
