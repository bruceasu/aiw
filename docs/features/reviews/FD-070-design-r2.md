# FD-070 独立设计聚焦复核 r2

<!-- aiw-data: FD-070-design-r2.json -->

- 建议：**ready**。
- 来源：`FD-070-000002-design-requested`。
- 对象：当前 FD Revision 2 的 D1/D2 补充与可选 SystemPrompt 适配；只复核这三项设计修订。
- 性质：设计反馈，不是实现 verification-passed；未修改 FD、源码或角色事件。

## 发现处理

**D1 已在设计层解决。** 调用前记录 pending 轮次、输入和 phase；辅助 LastTurn 与 Session.LastTurn 不一致或残留 pending 时拒绝新模型调用。显式 recover 仅使用已有冻结输出补本地提交，成果按相同内容核对、不覆盖冲突，未知运行状态或锁保留现场。该方案能避免把 sidecar 原子写误当成跨文件事务，也避免自动重放模型。实现评审需追踪 Session 已完成而辅助提交未完成、部分成果落盘两个案例，并核对每一步失败后的拒绝与恢复行为。

**D2 已在设计层解决。** 完整历史逐轮保存，模型上下文仅包含固定目标、当前 brief、最近 6 轮和最近成果索引；总记忆 32KiB，Skill 和资料另外有界。预算不足明确报错而非静默截断规则，/recall N 显式引入旧轮次，不增加隐藏摘要调用。此策略作为首版允许实现；它保证完整记录可回顾，不承诺模型每轮已读全部历史。帮助和教学提示需明确这个区别。

**SystemPrompt 适配可接受。** 可选内部 FrozenTurn 字段仅辅助调用显式填写，原调用默认空、不改变 Session 公共 JSON，阶段指令与 Prompt 数据明确组装，且不传 InvocationObserver。与 r1 已读取的 ExecuteFrozenTurn / Request 能力相容，兼容性与 provider 实际透传仍交后续实现评审。

## 边界与后续

没有新增设计阻塞。r1 的 Windows 成果文件名、独立外层锁和 provider 只读限制提醒仍适用。该结论允许 Planner 在修订合同下推进设计就绪，不证明任何恢复、取消、隔离或教学行为已运行验证。

本轮只读取当前 FD 与 r1 JSON，未扩大源码探索；写入本报告及同名 JSON。未运行测试、编译、模型、网络或应用命令。
