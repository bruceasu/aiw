## MODIFIED Requirements

### Requirement: 状态由 Workflow Core 映射与派生

status 和 done MUST 通过 Workflow Core 的兼容状态映射更新，再投影 Task 摘要。摘要 MUST 分开表达 planning、execution、validation、delivery 和 workspace，不能用一个 DONE 字段代替全部维度。生命周期写操作 MUST 将 Core 派生 status 和 delivery 同步到 task.toml 作为兼容快照；已有 Core MUST NOT 被快照反向覆盖。读取命令 MUST NOT 为刷新快照写入文件。

#### Scenario: 执行完毕但验证未结束
- **WHEN** 所有 Work Item 完成但验证处于 pending 或 failed
- **THEN** 汇总显示 AWAITING_VERIFICATION；验证等待授权时显示 AWAITING_AUTHORIZATION，元数据不得投影 DONE。

#### Scenario: 执行与验证满足当前完成规则
- **WHEN** execution 为 completed 且没有等待授权、待验证或失败验证
- **THEN** 可派生 DONE，同时 delivery 仍可能为 pending 或 unmanaged；相应生命周期写操作同步这两个独立维度。

### Requirement: 归档前置条件与显式清理

archive MUST 检查 DONE / CANCELLED 的终止资格、工作区绑定和交付状态。主工作区 Task MUST 拒绝受管理工作树/分支的 finalize 选项；隔离 Task 的归档 MUST 核验交付和清理结果。已交付且资源已清理的 Task MUST 支持使用核验后的主工作区工件归档，不再读取失效的隔离路径，也不要求重复删除资源。push MUST 由显式选项触发。

#### Scenario: 工作区归属未知
- **WHEN** archive 无法确认 Task 的工作区绑定或历史交付及清理依据
- **THEN** 系统要求先修复绑定或补齐证据，不继续归档。

#### Scenario: 主工作区归档要求删除分支
- **WHEN** primary Task 使用 cleanup-wt、delete-branch 或 finalize
- **THEN** 系统拒绝，因为该 Task 没有对应受管理工作树或分支可清理。

#### Scenario: 完成并交付后工作树已删除
- **WHEN** Core 确认 DONE、交付为 merged、历史隔离资源均已清理且主工作区 change 唯一存在
- **THEN** native 和 openspec 归档均从该 change 同步清单并执行同样的资格检查，不访问已删除工作树。

## ADDED Requirements

### Requirement: 清理结果同步当前绑定

系统 MUST 在成功移除隔离工作树后将当前绑定设为 unassigned、worktree 置空，并保留历史分支、父分支、Session 和交付证据。系统 MUST NOT 自动将其绑定为 primary。清理失败 MUST 保留符合实际结果的状态并允许恢复。

#### Scenario: 分支清理失败
- **WHEN** 合并和工作树移除成功，但分支删除失败
- **THEN** delivery 保持 merged，当前工作树绑定解除，历史分支保留，并可重试未完成步骤。

### Requirement: 历史终止任务的幂等修复

显式修复 MUST 依据 Core 派生状态及实际资源检查处理活动和归档 Task，保留未知字段、原位置及修复前快照。无法确定身份、终止状态、占用或清理结果时 MUST 跳过并诊断。修复 MUST NOT 改写 Core 执行事实、凭清单推断完成或自动执行 Git 写操作。

#### Scenario: 已完成任务残留旧元数据
- **WHEN** Core 为 DONE 且 merged，工作树目录、注册和临时分支均不存在，但 task.toml 为 TODO/pending/isolated
- **THEN** 修复为 DONE/merged/unassigned，清空 worktree 并保留历史字段；再次修复不写入文件。

#### Scenario: 已归档主工作区任务
- **WHEN** 归档目录中的 Core 确认 DONE，元数据仍为 TODO，交付为 unmanaged
- **THEN** 就地同步完成快照，保留 primary/unmanaged，不复活活动目录。

#### Scenario: 尚未完成或证据不足
- **WHEN** 存在待验证、待授权、阻塞、活动执行或缺失损坏的 Core
- **THEN** 不将 Task 修正为 DONE，不解除绑定，报告跳过原因。
