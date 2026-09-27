## Why

`aiw wf supervise` 当前能够保留失败报告，但报告仍然偏向内部诊断，无法在没有前台人工值守时继续处理。需要让 Workflow 先用受控 AI 分析原因，再执行有限且可验证的安全修复；只有无法安全处理时，才生成一份人类可以编辑的选择题答复文件。

本 change 只针对当前 schema 9 的监督体验和有限恢复，不启用 schema 10，也不把未知的外部执行结果当成可重试失败。

## What Changes

- 增加 schema 9 下的结构化 Remediation 问题、诊断、动作和结果记录。
- 让 Supervisor 使用只读 AI 分析失败原因，并由 Workflow Core 校验可执行动作。
- 增加有界的自动修复循环，复用现有 Attempt、RetryPolicy、Session repair 和 projection repair seam。
- 无法安全自动处理时，生成带风险、范围、资源影响和回滚说明的人类选择题文件。
- 增加 `continue` 与 `resume`，读取经过校验的人类答复文件后继续 Workflow。
- 保留未知执行状态、授权边界和原始失败证据；禁止 AI 直接修改状态或执行任意命令。

## Capabilities

### New Capabilities

- `workflow-remediation`: schema 9 下的自动诊断、有限修复和文件化人工答复协议。

### Modified Capabilities

无。现有监督、Attempt、Gate 和未知结果规则继续有效，本 change 只增加其上的受控恢复能力。

## Impact

- 影响 `internal/workflow` 的 Remediation 领域模型、报告持久化和安全动作校验。
- 影响 `internal/workflow/execution` 的只读 AI 分析适配和 Supervisor 循环。
- 影响 `internal/workflow/cli` 的 `continue`、`resume` 和人工选择题输出。
- 增加 Task 目录下的 remediation 报告和答复文件；不改变 schema 9 的启用边界，不引入外部依赖。
