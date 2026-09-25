# Agent Session

## Purpose

固定 AIW 会话的持久化输入、模型调用和结果记录边界，提供可追溯的 LLM prompt 与 turn 证据，并说明会话执行成功与工作项需求验收之间的区别。
## Requirements
### Requirement: 会话与 turn 持久化

系统 MUST 为 Session 保存状态、instructions、memory、prompt 与 turn 输出。每次执行 MUST 使用递增的 turn 编号，将最终输出、事件输出及存在时的 stderr 保存为独立文件。

#### Scenario: 执行一次 turn

- **WHEN** Session 可运行且输入非空
- **THEN** 系统使用 last_turn + 1 保存 prompt，调用后端，并更新最近 turn、阶段、退出码、完成时间和后端线程信息。

#### Scenario: 已结束的 Session

- **WHEN** Session 处于 completed、archived 或 deleted
- **THEN** 执行入口拒绝运行新的 turn。

### Requirement: 可追踪的 prompt 组合

执行入口 MUST 保留 instructions、memory、phase、task 的实际组合，并装载角色所需需求、验收、Task 来源和依赖正文。派发前保存最终 prompt、来源身份/版本/读取状态、选择和降级理由。必需输入缺失或预算不足停止相关派发；摘要不可用时可使用完整原始工件。

#### Scenario: 摘要不可用
- **WHEN** 必需原始来源完整而摘要不可用
- **THEN** 记录降级、加载正文继续；历史 prompt 不被后来摘要覆盖。

### Requirement: Task handoff 与监督指令

Task Agent 入口 MUST 解析 handoff 精确来源并实际装载必需内容，按 Coder/Tester/Verifier 角色组织监督边界和输入；只传路径不满足装载要求。handoff 不自行增加验收约束，候选知识不能替代主需求。

#### Scenario: 必需精确版本缺失
- **WHEN** 主需求精确纳入的来源版本无法取得
- **THEN** 阻止受影响派发；普通可选知识不可读则登记降级，不伪装零命中。

### Requirement: 单次后端覆盖不改写会话配置

执行入口 MUST 允许本次 provider / model 覆盖，优先于保存的 Session 值和全局默认值；这类覆盖 MUST NOT 改写 Session 保存的 provider / model。进程环境覆盖也仅应用于本次执行。

#### Scenario: 临时选择另一个模型

- **WHEN** 本次执行指定 model override
- **THEN** 后端使用覆盖值，原会话的模型配置保持不变。

### Requirement: 执行结果不等于需求验收

Session MUST 区分后端执行成功与失败，保存可读取的输出。Workflow 对业务 outcome 的解析 MUST 由监督层单独执行，不能把 Session completed 本身解释为 Work Item 或需求完成。

#### Scenario: 后端成功但输出无效

- **WHEN** 后端退出成功，但最终文本没有合法监督 outcome
- **THEN** Session 仍保留真实输出；监督层按无效 outcome 处理，不因退出码为零直接完成工作项。

### Requirement: SW25 实际输入与缺口

系统 MUST 满足以下规则：每轮提供需求、任务上下文、handoff 和相关知识的实际内容，按角色组织；保存实际 prompt、来源身份/版本/状态、读取错误、选择/跳过和降级理由。核心主来源、角色边界、必需依赖缺失或版本无法解释则阻塞受影响派发；完整原始工件可替代不可用摘要。普通项目知识可选，读取失败不冒充零命中；只有被已确认主来源精确纳入的知识版本成为必需约束，handoff 不自行加约束。历史输入不被 latest 回写。

追踪：SW25 / AC25。

#### Scenario: AC25 验收行为

- **WHEN** 必需正文缺失或超预算
- **THEN** 不派发；普通知识不可读 → 记录降级而非零命中；主来源精确引用版本缺失 → 阻塞；历史 prompt 不被 latest 改写

## 实现依据

- [存储](../../../internal/session/store.go)、[生命周期](../../../internal/session/lifecycle.go)。
- [prompt 与执行](../../../internal/session/execute.go)、[输出保存](../../../internal/session/backend.go)。
- [Task prompt](../../../internal/commands/task/agent.go)、[监督结果校验](../../../internal/workflow/execution/session.go)。
- [监督 Session 回归材料](../../../internal/workflow/execution/session_test.go)；本次未执行。

%% 当前 turn 读取 memory 不等于 supervise 已实现自动记忆总结与更新；本基线不承诺跨模型线程的隐式记忆。
