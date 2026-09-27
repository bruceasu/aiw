## MODIFIED Requirements

### Requirement: 会话与 turn 持久化

系统 MUST 为 Session 保存状态、instructions、memory、prompt 与 turn 输出。
每次执行 MUST 使用递增的 turn 编号，将最终输出、事件输出及存在时的 stderr
保存为独立文件，并保存该次 Provider 调用的规范化 usage envelope 及其
Provider 返回证据。usage envelope MUST 保留 Token、货币成本和各字段的
可用性状态；缺失字段 MUST 表示为 unknown，不得估算为零。

#### Scenario: 执行一次 turn

- **WHEN** Session 可运行且输入非空
- **THEN** 系统使用 last_turn + 1 保存 prompt，调用后端，并更新最近 turn、
  阶段、退出码、完成时间、后端线程信息和该次调用的 usage 证据

#### Scenario: 已结束的 Session

- **WHEN** Session 处于 completed、archived 或 deleted
- **THEN** 执行入口拒绝运行新的 turn

#### Scenario: Provider usage 不完整

- **WHEN** Provider 未返回某个 Token 或货币成本字段
- **THEN** Session 保存实际返回的字段和 `usage_unknown` 状态，保留原始响应
  证据，不阻止该次结果记录
