## Design Readiness

READY。实现范围、schema 9 约束、人工答复文件格式和安全动作集合已确定；未知外部执行状态仍按现有 Workflow supervision 规则进入人工处理。

## Context

Workflow Core 已有 `FailureReport`、`Automation.Cursor`、`ProjectionRepair`、`RepairMissingSessionAttempt` 和有界 RetryPolicy。Execution 层已有只读 AI 诊断与模型路由适配。问题是这些 seam 没有被统一成一个“分析—修复—验证—人工选择”的深模块，前台退出后也没有稳定的答复协议。

本 change 必须运行在 schema 9。schema 10 的精确派发、预算和真实宿主接线不在本 change 中，也不能因为新增 Remediation 就切换 schema 10。

## Goals / Non-Goals

**Goals:**

- 把内部失败转换为可解释、可追溯、可继续处理的 Remediation 问题。
- 在只读 AI 分析后，仅执行预先列出的安全动作。
- 让 Supervisor 在无人值守时生成答复文件，并在 `continue`/`resume` 时安全读取。
- 重启、重复调用和重复答复保持幂等。
- 每个人工选项明确范围、风险、资源影响和回滚方式。

**Non-Goals:**

- 不实现 schema 10 的真实执行、成本计量或完整未知状态对账。
- 不允许 AI 直接编辑 RuntimeState、修改授权、扩大预算或运行任意命令。
- 不保证所有基础设施故障都能自动修复。
- 不把人工答复文件当作新的 Task 或第二套状态机。

## Decisions

### 1. 使用独立的 Remediation 文件协议

每个问题使用稳定的 `problem_id`，由 Task、WorkItem、Attempt、事件序号和证据摘要派生。报告写入受管 Task 目录：

```text
reports/remediation/<problem-id>.json
reports/remediation/<problem-id>.response.json   # 仅在需要人工选择时使用
```

报告保存问题快照、AI 诊断、允许动作、已经执行的动作、结果、风险和下一步。答复文件只允许引用报告中已列出的选项，并且必须带有报告 digest；过期或篡改答复拒绝执行。

使用文件报告而不是把完整诊断塞进 RuntimeState，是为了保持 schema 9 的兼容性和报告可读性。`Automation.Cursor` 只保存当前阶段和 `problem_id`，Core 状态仍是唯一调度事实来源。

### 2. AI 是只读分析 Adapter，不是状态机

定义小的 `RemediationAnalyzer` seam。Adapter 只接收冻结的问题上下文和允许的证据摘要，返回结构化诊断：分类、原因、置信度、建议动作和风险。Core 校验分类、问题绑定、证据引用和动作白名单；AI 不获得 Store 写接口，也不能返回任意命令。

### 3. 自动动作采用有限白名单

第一版允许：复用原 Attempt、恢复缺失 Session Attempt、重试现有投影修复、补交一次报告、以及在已有 Workflow 策略明确允许时继续下一步。未知派发结果、写租约仍在途、授权缺失、路径/身份冲突和会产生外部副作用的动作一律进入人工 Gate。

自动处理最多进行三轮；每轮仍受现有 RetryPolicy 和专项恢复额度约束。三轮不是新的预算，而是防止 Supervisor 无限调用 AI 的控制上限。

### 4. `continue` 与 `resume` 共享同一读取路径

两个命令都读取当前 `Automation.Cursor` 指向的问题报告和对应 `.response.json`。没有答复时只打印模板路径和选择说明，不改变状态；答复存在时先严格解析、校验问题 digest、选项 ID、风险确认和操作者，再执行对应的 Core 操作。二者的差别只保留为用户表达习惯，不产生两套语义。

### 5. 先自动处理，后输出选择题

Supervisor 按以下顺序工作：冻结问题 → 只读 AI 分析 → Core 校验 → 执行一个安全动作 → 读取新状态并验证。成功则继续；需要更多信息、动作不在白名单、结果未知或自动轮次耗尽时，才生成选择题文件并暂停当前问题。原始失败报告永不覆盖。

## Risks / Trade-offs

- [AI 诊断错误] → AI 只能提出白名单动作，Core 重新验证事实，低置信度直接人工处理。
- [Supervisor 无限循环] → 固定三轮 Remediation 上限，并复用 WorkItem RetryPolicy 和现有专项额度。
- [答复文件被篡改或过期] → 绑定 `problem_id`、报告 digest 和允许选项；严格拒绝未知字段。
- [外部调用状态未知] → 不自动重派发；保留原 Attempt、lease 和证据，要求人工对账。
- [schema 9 旧数据缺少问题报告] → 从现有 `FailureReport` 派生问题；不能补造不存在的证据。

## Migration Plan

1. 增加 Remediation 报告和答复文件读写，不迁移既有 Task。
2. 把 Supervisor 的已知失败路径接到分析器和白名单动作执行器。
3. 增加 `continue`/`resume`，没有答复时保持只读。
4. 对新问题生成新报告；既有 `latest-failure` 文件继续可读。
5. 回滚时停止自动 Remediation，保留报告和答复文件，继续使用原有 `diagnose`/`repair` 路径。

## Open Questions

%% 真实模型调用、token/金额计量和 schema 10 联合启用仍属于另一个 activation change；本 change 不以测试替身证明生产上线。
