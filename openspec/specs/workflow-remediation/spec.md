# workflow-remediation Specification

## Purpose
TBD - created by archiving change workflow-remediation-experience. Update Purpose after archive.
## Requirements
### Requirement: 结构化 Remediation 问题

Workflow MUST 为可自动处理的失败建立稳定的 Remediation 问题，绑定 Task、WorkItem、Attempt、事件序号和现有证据。问题报告 MUST 保留原始失败事实，不能用 AI 分析覆盖原报告。

#### Scenario: 同一问题重复发现

- **WHEN** Supervisor 重启或重复扫描同一个失败事件
- **THEN** 系统复用同一个 `problem_id` 和报告，不重复调用 AI 或重复执行已经完成的动作

#### Scenario: 证据不足

- **WHEN** 失败没有可验证的 Attempt、事件或证据绑定
- **THEN** 系统不自动修复，生成说明缺失证据的人类选择题文件

### Requirement: 只读 AI 原因分析

系统 MUST 将冻结的问题上下文交给只读 AI Analyzer，并要求结构化返回分类、原因、置信度、证据引用、建议动作和风险。AI Analyzer MUST NOT 直接写入 Workflow Store、修改授权或返回任意命令。

#### Scenario: AI 返回合法分析

- **WHEN** Analyzer 返回与问题 digest、证据和允许动作一致的结构化结果
- **THEN** Workflow 保存分析并进入 Core 动作校验

#### Scenario: AI 返回无效或低置信度分析

- **WHEN** 输出无法解析、引用不存在的证据、建议未知动作或置信度低于策略阈值
- **THEN** 系统保留分析失败信息，不执行动作，转为人工选择题

### Requirement: 有界安全自动修复

Workflow MUST 只执行版本化的安全动作白名单。每次动作 MUST 先由 Core 校验当前状态、Attempt/租约、原始证据和动作前置条件；动作完成后 MUST 重新读取状态并验证结果。自动 Remediation 最多三轮，并且不得绕过现有 RetryPolicy、Gate、Stop 或未知结果保护。

#### Scenario: 缺失 Session 且满足既有恢复条件

- **WHEN** 问题明确是符合既有条件的缺失 Session Attempt
- **THEN** 系统调用现有缺失 Session repair，保留原 Attempt 历史并重新选择 WorkItem

#### Scenario: 投影修复

- **WHEN** 已提交事件存在幂等的 projection repair
- **THEN** 系统仅重试投影并记录结果，不重放原始业务转换

#### Scenario: 未知执行状态

- **WHEN** 无法证明原 AI/Agent 请求未启动或已经终止
- **THEN** 系统不得自动重派发、释放租约或创建第二个 Attempt，而是转为人工处理

#### Scenario: 自动轮次耗尽

- **WHEN** 同一问题连续三轮自动动作仍未解决
- **THEN** 系统停止自动处理，生成带风险和下一步选项的人类答复文件

### Requirement: 文件化人工选择题

需要人工决定时，系统 MUST 在 Task 的 `reports/remediation/` 下生成问题报告和答复模板。每个选项 MUST 包含选项 ID、影响范围、风险、资源影响、外部副作用、回滚方式和需要的授权。答复模板 MUST 不能通过自由文本直接改变 Workflow 状态。

#### Scenario: 无前台人工

- **WHEN** Supervisor 在无人值守运行中无法安全继续
- **THEN** 系统写入答复模板，更新 Automation cursor 为 awaiting-human，并以稳定文件路径和问题 ID 作为下一步提示

#### Scenario: 人工答复缺失

- **WHEN** `continue` 或 `resume` 找不到当前问题的答复文件
- **THEN** 系统只显示待答复选项和模板路径，不改变 Attempt、Gate 或 WorkItem 状态

#### Scenario: 人工答复过期或被篡改

- **WHEN** 答复中的问题 ID、报告 digest、选项 ID 或结构化字段不匹配
- **THEN** 系统拒绝答复，保留原文件和原问题，输出明确的修复方法，不执行任何动作

### Requirement: continue 和 resume 继续 Workflow

`continue` 和 `resume` MUST 使用同一答复读取和校验逻辑。成功读取后，系统 MUST 只执行答复中对应的已列出动作，并在执行前显示或记录其风险确认；执行完成后 MUST 重新进入正常 Workflow 选择和状态投影流程。

#### Scenario: 选择低风险继续

- **WHEN** 人类提交了匹配问题 digest 且明确确认风险的低风险选项
- **THEN** 系统执行对应安全动作，记录答复引用和结果，然后继续 Supervisor

#### Scenario: 选择需要额外操作的选项

- **WHEN** 人类选择会增加重试、改变模型或产生外部副作用的选项
- **THEN** 系统在执行前要求明确授权，并显示预计影响和风险；未确认时保持 awaiting-human

#### Scenario: 重复提交同一答复

- **WHEN** 同一个答复已经成功消费
- **THEN** 系统返回已消费结果，不重复执行动作、不新建 Attempt、不重复调用 AI

### Requirement: 人类输出可操作且可追溯

Remediation 输出 MUST 优先显示当前问题、最可能原因、证据路径、系统已尝试的动作、当前限制和可选动作，而不是只输出原始异常文本。每次自动分析、动作和人工答复 MUST 可通过问题 ID 追溯。

#### Scenario: 自动修复成功

- **WHEN** 白名单动作完成且状态验证通过
- **THEN** 输出简短结果和继续执行的下一阶段，不要求人工介入

#### Scenario: 无法自动处理

- **WHEN** 没有安全动作、证据未知或自动轮次已耗尽
- **THEN** 输出选择题、风险、文件路径和可执行的 `continue`/`resume` 方法

