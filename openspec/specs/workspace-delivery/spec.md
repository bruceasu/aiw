# 工作区与本地交付

## Purpose

固定 Task 工作区归属、监督执行的隔离要求和完成后的本地 Git 交付边界，覆盖主工作区绑定、受管理分支、交付前置条件、合并失败及成功后的清理顺序。
## Requirements
### Requirement: 显式工作区绑定

系统 MUST 在 Task 元数据中记录 workspace_kind、worktree、branch 和 parent_branch。`workspace bind <id> --primary` MUST 在主工作区执行，并校验已有 parent_branch 与当前分支一致。

#### Scenario: 绑定主工作区

- **WHEN** 显式主工作区绑定成功
- **THEN** Task 指向 `.` 和当前分支，workspace_kind 为 primary，delivery 为 unmanaged。

### Requirement: 隔离工作树

`wt add` MUST 使用受管理 Task 信息创建 `.wt/<task-id>` 和 `feature/<task-id>`，记录其父分支及隔离关系。supervise MUST 使用隔离 Task 工作区；显式主工作区执行属于单步 run 的独立选项。

#### Scenario: 启动监督执行

- **WHEN** supervise 准备执行 Task
- **THEN** 系统解析或准备隔离工作区并检查运行绑定，不把单步 run 的 primary 选项当作 supervise 的工作区策略。

### Requirement: Git 预检与阻塞证据

监督执行 MUST 在派发前执行其范围内的 Git 预检。实际预检失败 MUST 记录可诊断阻塞，而不是只依据 Agent 对工作区的描述继续执行。初始 Git 预检失败 MUST 不消耗普通 no-progress 重试。

#### Scenario: Git 预检失败

- **WHEN** Supervisor 无法通过当前工作区的 Git 预检
- **THEN** 在派发前报告阻塞并保留原因，不用连续 Agent 重试消耗实现预算。

### Requirement: 本地交付资格

Supervisor MUST 仅在当前开发接受成立、无在途写请求/阻塞、工作区为 isolated 且有适用 AI 批准计划及 grant 时尝试自动本地交付。必需测试的旧 waived 不满足资格，Verifier 不另加 Gate。merged/discarded 不重复交付，primary Task 不走隔离自动合并入口。

#### Scenario: 缺少交付授权
- **WHEN** 开发完成但具体计划未获 AI 批准或其范围已变化
- **THEN** 暂停相关交付并评估新计划，保留开发完成事实。

### Requirement: 合并、祖先验证与清理顺序

本地交付 MUST 核对隔离绑定、Task/父分支、可提交路径和 grant，只提交相关改动，再本地合并、验证祖先关系和记录 merged。清理须先保全后台必需来源，再通过 AIW 受管入口对获授权目标执行。禁止 Agent 直接 Git 删除及 push；该链路不创建 PR 或 archive。

#### Scenario: 合并后清理
- **WHEN** 祖先验证成功、后台来源已保全且清理目标获授权
- **THEN** AIW 可清理受管工作树/分支，Agent 不执行直接删除或 push。

#### Scenario: 冲突或结果未知
- **WHEN** 合并冲突或结果无法确认
- **THEN** 保留工作区并对账/等待处理，不盲重放、不冒充成功或清理。

### Requirement: SW33 Git 具体交付授权

系统 MUST 满足以下规则：具体 Git 交付计划由 AI 按用户政策批准或拒绝，记录于 .ai/<task-id>/grant.md；明确 Task 分支、父分支、可提交路径、动作和 AIW 清理目标。相同计划、授权有效且状态可证明安全时自动推进，无需逐步人工批准。允许本地 commit、创建分支和本地 merge；禁止 push 和直接 Git 删除，AI 不得绕过政策。worktree/branch cleanup 只能经 AIW 受管操作，须有对应授权及目标校验；AIW 内部的清理实现不构成允许 Agent 直接删除。范围变化、冲突、未知或危险变化需重新评估适用计划，不能沿用旧 grant；未知结果先对账。不提交无关改动，清理前保全辅助工作来源。PR、archive 不纳入自动交付计划。

追踪：SW33 / AC33。

#### Scenario: AC33 验收行为

- **WHEN** 具体交付计划已由 AI 批准且 grant 与范围一致
- **THEN** 自动执行允许的本地 commit、分支创建、merge；push/直接 Git 删除被拒绝，清理只经 AIW 且保存辅助来源

### Requirement: R2 交付与清理的逐步恢复契约

系统 MUST 将本地交付固定为有稳定动作身份、精确输入和预期前后状态的计划，仅提交计划内路径。每个副作用前核对 grant、真实目标、当前提交、索引、Stop 及在途状态，先持久登记意图再执行；未知结果先按具体提交和目标事实对账，不按消息或缺失路径猜成功。cleanup MUST 使用独立且绑定精确目标的受管授权，先证明合并与来源封存，再移除受管工作树、解除当前绑定、清理分支，逐步保存观察并保留历史。无法证明安全或来源清单完整时等待；禁止强删和扩大目标。此要求细化 SW33，关联 SW06/SW19、AC33 和 AX02/AX03。

#### Scenario: 无关索引内容或父分支变化
- **WHEN** 具体交付计划生成后发现范围外已暂存内容，或父分支提交发生计划外变化
- **THEN** 停止相关动作并重新评估，不全量暂存或沿用旧计划合并。

#### Scenario: 合并结果未知
- **WHEN** 合并可能已经发生但观察记录未提交
- **THEN** 按冻结的动作身份、提交和祖先关系对账；确认已合并则补记事实，不重做 commit/merge，不能确认则保留工作区等待。

#### Scenario: 清理中断后恢复
- **WHEN** 工作树已被受管移除但解除绑定或分支清理尚未完成
- **THEN** 复核原请求、目标和历史绑定后只续做未完成步骤；分支移动、被其他工作树引用或目标无法匹配时停止。

#### Scenario: 辅助来源未封存
- **WHEN** 已合并但必需来源仍引用待清理工作树，或无法证明来源清单完整
- **THEN** 保留工作树并显示 cleanup 等待；不撤销已记录的合并，不靠缺省空清单批准删除。

## 实现依据

- [Task 绑定](../../../internal/commands/task/workflow.go)、[工作区协调](../../../internal/commands/task/workspace_coordinator.go)。
- [工作树插件](../../../plugins/aiw-wt.py)、[监督 Git 预检](../../../internal/commands/task/supervised_git.go)。
- [自动交付资格](../../../internal/workflow/execution/delivery.go)、[本地合并与清理](../../../internal/commands/task/local_delivery.go)。
- [Git 预检回归材料](../../../internal/commands/task/supervised_git_test.go)；本次未执行。
