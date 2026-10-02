# FD-007 独立审查

- 来源事件：`FD-007-000004-implementation-ready`
- 审查基准：`HEAD c01fe2c` 至当前工作区中 FD-007 相关未提交差异，以及新增的 FD 和 Worker 报告
- 结果：`changes-requested`

## 发现

1. **第三次失败会占用本应保持待处理的 Worker handoff。** `skills/fd-workflow/SKILL.md` 的 Auto 第 5 步要求收到 `changes-requested` 后先 claim 新的 Worker handoff，再检查是否已达到三次失败或无可执行修复。第三次失败应立即停止，但先 claim 会把事件设为 `dispatched`。根据 `plugins/aiw-fd.py` 的 `claim`、`resume` 和 `prepare_event`，其他 Session 此后无法领取，`resume` 也不会重新派发；这与 FD 验收条件 2、3 所要求的安全恢复和第三次失败停止相冲突。请在 claim 之前先检查本轮累计 Review 结果与可修复性；达到上限时保留新 Worker 事件为 pending，并报告停止原因。对前两次失败，才 claim、修复并再次提交。

## 已核查

- 对照 FD-007、Worker 报告、源和安装版 `fd-workflow` Skill、两份 portable operations、稳定 spec、用法文档、`fd-review` Skill，以及 `plugins/aiw-fd.py` 的 claim、emit、resume、close 状态与 digest 约束。
- 新入口作为 host Skill 操作有明确说明；安装版引用仓库内源 Skill。Reviewer 分工、来源事件、当前通过事件的归档门槛、无隐式测试／构建／Git 授权均有文字约束。
- FD-007 本身已有 `new`、Planner claim、`design-ready`、Worker claim 的记录。完整 auto 循环尚未实际运行，不能据此声称端到端验证通过。

## 本次命令与限制

- 执行 `go run cmd/aiw/main.go fd --help`、精确 Reviewer claim、`fd show FD-007`，以及定向 `Get-Content`、`git status`、`git diff`、`git rev-parse`、`rg` 静态读取。
- 未运行测试、构建、网络请求或完整 auto 流程；未做 Git 写操作。
- 剩余风险：三轮计数由 Skill 所在 host 执行，CLI 暂无强制计数；本次只对其静态规则作审查。
