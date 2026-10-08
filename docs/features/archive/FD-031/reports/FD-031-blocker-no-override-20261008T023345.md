# FD-031 自动流程阻塞反馈：dispatched Worker handoff 无安全接管命令

<!-- aiw-data: FD-031-blocker-no-override-20261008T023345.json -->

## 定位

- FD：`FD-031`
- 阶段与角色：Worker 恢复／PM
- 关联交接事件：`FD-031-000006-test-rejected`
- 记录时间：2026-10-08 11:33:45 Asia/Tokyo
- 记录者／会话：PM／`fd031-auto-20261008-023345`

## 阻塞事实

- 用户已确认 Tester 停止、原流程中断并要求恢复。事件回执仍显示 Worker session `fd031-worker-repair-20261008-01` 已领取，状态为 `dispatched`；无对应事件日志，也没有后续 Worker 结果事件。
- `plugins/aiw-fd.py` 的 `refresh_worker` 只接受最新 `pending` Worker handoff；`resume` 对 `dispatched` 状态要求检查原 session 或事件日志。当前 `aiw fd` 命令列表没有取消或 PM override 接管 dispatched handoff 的操作。
- 已确认的 Worker 修复发现仍有效：Issue TOML 标题转义没有还原；Tester 的模板缺失注入不能触发 FD 创建失败。当前 worktree 干净，FD 仍为 `In Progress`。
- 已尝试的恢复：静态读取命令集合、`refresh_worker`、`resume` 和 event validation 实现；没有执行会失败的恢复命令，没有改写事件回执或 FD 正文。
- 当前状态：未解决；不能安全创建新的 Worker handoff，也不能把当前 session 冒充为已领取事件的 Worker。
- 需要人工决定：要么等待原 Worker handoff 正常产出结果，要么批准新增公开的 FD CLI PM recovery 命令及其契约设计。仓库规则要求在新增公开 CLI/API 前暂停确认。

## 结果与改进

- 实际解决方案：用户授权新增 `recover-worker`；FD 分支提交 `f14f0c61` 实现精确事件和 session 匹配。PM 执行恢复后，旧事件 `FD-031-000006-test-rejected` 已取消并记录原因，新事件 `FD-031-000008-work-requested` 已创建为 pending。
- 解决时间：2026-10-08 02:45:46 UTC。
- 下一步：由新 Worker session 认领 `FD-031-000008-work-requested`，提交当前实现报告，再进入独立 Tester。
- 可复用的流程改进建议：已在 FD-031 实现经过审计的 dispatched Worker 恢复命令；验收仍由独立 Tester 和 Reviewer 按 FD 流程完成。
- 建议处理状态：已解决。
