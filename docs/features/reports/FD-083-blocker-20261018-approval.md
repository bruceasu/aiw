# FD-083 自动流程阻塞反馈

<!-- aiw-data: FD-083-blocker-20261018-approval.json -->

## 定位

- FD：FD-083；Issue：ISSUE-00011。
- 阶段：Planner，尚未 design-ready。
- 交接事件：FD-083-000002-design-requested。
- 会话：fd083-planner-20261018-7a9b2f。
- 日期：2026-10-18，Asia/Tokyo；未记录精确时间。

## 阻塞事实

- 用户已明确授权 approve、promote、auto，并确认 core 可显式依赖 aiw-git。
- 首次通过 PowerShell 管道向 Python 写入中文时发生编码损坏，替换未生效，追加文字成为问号。
- 首次 capture、approve 和 promote 成功，创建 FD-083；审批摘要绑定当时的计划版本。
- 随后修复中文、过时的 aiw-git 待确认项及规模说明，再次 capture 成功。
- 再次 approve 被 CLI 拒绝：`requirement must be DECIDED before approval: APPROVED`。
- 当前 Issue revision=13，approval.source_digest 仍绑定旧来源；当前计划摘要已变化。
- 静态源码依据：`src/internal/issue/store.go` 的 Capture 更新工件摘要但保留 APPROVED；Approve 仅允许 DECIDED 执行 APPROVED。
- 当前状态：未解决。未修改审批元数据、伪造审批结果或重复晋级。

## 结果与下一步

- 尚未完成 FD 工程设计，未进入 Worker，未创建工作树、提交或修改产品源码。
- %% NEEDS_INPUT: 需要受支持的已批准 Issue 来源更新及重新审批恢复路径；现有 CLI 无该命令。用户授权决定已经明确，无需重复询问 aiw-git 业务规则。
- 恢复必须保留既有审批历史和 FD-083，不能通过再建 Issue/FD 或手改摘要掩盖版本变化。
- 建议另行授权修复 Issue 重新审批流程，或提供现有受支持的恢复操作；本报告不授权产品修改。
- 可复用改进：中文文件使用直接补丁或显式 UTF-8；在审批前核对实际 capture 内容；capture 已批准来源时应明确拒绝或进入可重新审批状态。
- 未执行测试、编译、构建、网络、独立审查、交付或归档。审批恢复前，编译不能解决此记录门禁。

## 2026-10-18 恢复结果

用户明确 confirm，并再次要求将状态改回后重新 approve。已备份原记录及决策日志到
`.ai/fd/FD-083/approval-recovery-20261018/`，恢复顶层 DECIDED 后，通过正式
`aiw issue approve issue-00011 APPROVED --by user --reason ...` 成功绑定当前捕获计划。
审批历史、当前计划和 FD-083 均保留，未重复晋级。上文为原始阻塞历史，当前审批阻塞已解决。
无需修改产品审批逻辑；继续现有 Planner 事件，设计完成后进入正常实施流程。
