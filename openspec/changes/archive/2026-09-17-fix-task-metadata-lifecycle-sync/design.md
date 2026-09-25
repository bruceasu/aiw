## Context

reportWorkflowState 明确只输出 Core 摘要；localMergeDelivery 记录 merged 后清理 worktree/branch，却未解除元数据绑定。归档根据 Core DONE 进入同步，再由 WorkflowChecklistPath 根据 task.toml 的 isolated 路径读取，构成跨阶段不一致。Core task.workspace 的初始 primary 值也不能直接作为交付后的重新绑定证据。

## Goals / Non-Goals

**Goals:** 完成状态、交付结果与当前绑定一致；清理后可归档；失败可恢复；历史记录可安全修复。

**Non-Goals:** 本轮不实现生产代码，不自动执行 Git 合并/删除/归档，不伪造验证通过，不用清单勾选或目录消失推断 DONE/merged。

## Decisions

### 启动时运行投影缺失

`supervise start` 先读取已有 Core；只有读取结果为不存在、且活动 Task 元数据的 ID 与请求一致时，才复用现有 `EnsureCompatible` 创建不含 Attempt 或写租约的初始投影。不得覆盖损坏状态，不得仅凭 OpenSpec change 或 `migrated-to` 推测执行历史。元数据也丢失时拒绝启动，并明确要求先恢复有效元数据。`status`、`stop` 仍只读取已有状态；非法 action 在访问运行状态前拒绝。此修复不修改 Task 目录布局，也不自动恢复用户删除的 Task 数据。

1. Core 是唯一执行状态来源。task.toml.status 使用 DeriveSummary 的完整状态枚举，delivery 使用 Core 的真实值；两者仅是兼容快照。只在生命周期写操作及显式 repair 同步，不在 list/show 等读取时写入。已有 Core 不接受这些缓存反向覆盖。修订现有禁止摘要投影的注释和文档，明确允许粗粒度缓存，仍禁止投影 Attempts、租约、重试等细节。
2. 工作区绑定不能从 Core 初始 TaskReference 机械复制。隔离 worktree 成功移除后设置 workspace_kind=unassigned、worktree=""；不自动设 primary，也不清空原 branch、parent_branch、session。历史 workspace_binding 与交付证据保留。主工作区任务保持 primary 与 unmanaged。
3. 合并、工作树清理、分支清理是独立步骤。记录 merged 后立即同步交付快照；工作树清理成功后同步解除绑定；分支删除失败仍可依历史分支重试，不能因已 merged 拒绝恢复。Core 与 TOML 无跨文件事务时，Core 先落盘，元数据采用原子替换；失败返回可恢复诊断，不谎报整体完成。
4. 归档单独解析最终工件。仍在执行的 isolated Task 继续只读其 worktree。仅有终止 Core 状态、交付证据、已清理资源及唯一主工作区 change 时，允许读取主工作区清单。unassigned 不代表交付已完成；归档仍核验遗留工作树注册、目录和任务分支，不跳过 Git 交付条件。native/openspec 使用同一资格逻辑，重复清理不要求删除不存在的资源。
5. 历史修复复用固定根身份发现及 Core 派生摘要。覆盖活动、旧路径及归档 Task，归档就地修正，不重建活动目录。只处理 Core 确认终止且无活动写入的记录；缺失/损坏/冲突/证据不足则报告并跳过。支持预览、差异清单、修复前备份和幂等重试；持有适当写锁，保留未知字段。不得由修复自动完成 Work Item、解除 Gate 或授权测试。

## Risks / Trade-offs

- 状态缓存可能再次过期 → 所有生命周期写入口统一调用投影，读取以 Core 为准，修复提供差异诊断。
- 清理后丢失交付依据 → 保留历史分支及 workspace_binding；未来记录交付提交与清理结果，缺证据时不猜测。
- 元数据修复不会独立修好归档全部资格逻辑 → 本轮修正数据后，不宣称 archive 已可成功；生产修复仍需完成清单。
- 多文件更新中断 → 原子写单文件、失败留证据、幂等重试，不回滚真实 Git 事实。

## Migration Plan

本轮按用户显式授权修正三项已确认 DONE 的现有记录，修复前全文及字段差异保存于本 Task 的 artifacts。两项 merged 且工作树/分支已消失的记录解除绑定；已归档 primary 任务仅同步 status/updated。不改 Core。回滚前比较当前文件与修复后快照，仅在未发生后续修改时恢复原文件。未来实现统一修复入口后，使用同一证据规则处理其他历史数据。

## Open Questions

%% 历史记录中的 merged 是已持久化的交付事实，本轮未重新运行 Git 祖先检查；生产归档对已删除分支应使用持久化交付证据，不尝试验证不存在的引用。
