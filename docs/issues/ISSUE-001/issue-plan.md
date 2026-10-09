# Issue Plan：强制归档 FD 并设为 Complete

Issue ID：`ISSUE-001`。

状态：实现请求已确认，需作为独立 Issue 建立新的编号 FD。此 Issue 不修改已批准并完成的 REQ00008 / FD-042 范围。

## 使用者、问题与目标

FD 操作者在已完成或需要强制结束的 FD 上，可能没有当前 Reviewer 的 `verification-passed` 事件。现有 `aiw fd close <id> Complete` 因此拒绝归档；`set-status` 虽可强制设为 Complete，却不执行归档。目标是提供显式、可审计的强制归档操作。

## 已确认事实与来源

- 用户要求 `aiw fd close` 增加 `--force` / `-f` 和 `--reason <text>`，强行归档。
- 用户明确决定：强制归档不要求调用前状态为 Complete；操作可在归档过程中把状态设为 Complete。
- 用户确认 FD-044 功能已完成并实际验证；随后明确要求在主工作区实现。
- `REQ00008-fd-force-recovery` 已批准并关联到已归档完成的 FD-042。FD-042 明确规定常规 Complete 归档要求 Reviewer 验证事件、强制状态变更不生成 Reviewer 证据；不改写该已完成 FD 的范围或记录。
- 当前 `plugins/aiw-fd.py` 的 `close_locked` 要求状态和最新 Reviewer 事件均满足 Complete 门槛，并拒绝归档时仍有 launching/dispatched 的最新收据。

## 目标范围

- `aiw fd close <id> Complete --force --reason <text>` 和 `-f` 别名可将活动 FD 强制归档为 Complete，不要求调用前状态为 Complete，也不要求 Reviewer `verification-passed` 事件。
- 强制操作在一次锁定事务中将状态设为 Complete、递增 revision、更新索引、迁移计划和匹配的 Markdown/JSON 报告与审查证据，并记录 Completed 日期及原因。
- 审计记录操作者、操作前后状态、理由、跳过的状态/Reviewer 检查，并明确 `review_verified: false`；不得创建或伪称 Reviewer 通过事件。
- 即使使用强制模式，归档目标冲突、路径安全、迁移失败回滚，以及最新 launching/dispatched 收据的未知执行结果保护仍然有效。保留未知执行保护可避免仍运行的 Worker 在归档后写入。
- 未带 `--force` 时，现有 close 行为保持不变；`--force` 必须要求非空、单行原因。

## 非目标

- 不停止 Agent 或取消 dispatched 收据。
- 不更改 `set-status`、`force-emit` 或 Reviewer 事件的语义。
- 不将强制归档表示为 Reviewer 验证通过，也不跳过归档事务和路径安全检查。

## 验收示例

- FD 处于任一已定义活动状态且没有当前 Reviewer 通过事件时，调用 `aiw fd close FD-XXX Complete -f --reason "..."` 后，计划及匹配证据位于归档目录，状态为 Complete，索引同步更新，审计显示跳过门槛且 `review_verified` 为 false。
- `-f` 未提供有效原因、FD 已归档、目标已存在、归档路径不安全或有 launching/dispatched 收据时，操作失败并保持原状态、索引、计划和证据不变。
- 不带 `--force` 时，正常 Complete 归档继续要求当前修订的 Reviewer `verification-passed` 事件。

## 规模与拆分评估

单一 CLI 生命周期结果，涉及 `plugins/aiw-fd.py`、FD 工作流稳定规格和使用文档；规模小到中等，状态、审计、事务与现有证据迁移需保持一致。不拆分。

## 风险与设计决定

- 强制 Complete 会绕过 Reviewer 完成门槛，调用者必须提供审计理由，记录必须明确没有 Reviewer 验证。
- 选择保留最新 launching/dispatched 收据保护；只要角色执行结果未知，强制归档仍拒绝，以免归档后发生并发写入。

## 推荐下一步

创建并实现一个与本 Issue 链接的编号 FD；在主工作区按用户指示工作。完成后执行一次窄范围 compile-only 检查，并交由独立 Reviewer 审查。不要运行测试。
