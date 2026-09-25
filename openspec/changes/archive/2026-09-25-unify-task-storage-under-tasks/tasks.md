## 1. 已完成：路径与兼容边界

- [x] 1.1 启动前确认没有待完成或待归档的前置 Task，重新读取相关归档后的稳定规格及现有实现；盘点 Task/Workflow/脚本/工件引用路径，记录迁移影响清单和最终路径接口。
- [x] 1.2 实现统一活动 Task 路径解析并接入元数据、Core、发现、创建/补建、执行/恢复及归档；新布局只用 .ai/tasks，旧布局返回迁移诊断，保留旧文件名和现有归档位置。

## 2. 已取消：通用迁移与迁移回归

用户决定取消 2.1、2.2 和 3.1。当前仓库只有本 Task 仍活动；开发完成后仅人工调整它的路径，不实现通用迁移 CLI、journal、rollback 或其入口级回归。已归档 Task 不移动。

## 3. 已完成：交付说明

- [x] 3.2 更新文档、技能中的正式路径及单次手工路径调整说明，静态核对所有调用方并按预算执行 compile-only；记录停止旧程序、停止本 Task supervisor 后再执行路径调整的顺序，完成 Verification。

## Verification

- Delivery conflict review: `develop` and this Task both updated the unchanged-checklist-fingerprint reconciliation in `internal/workflow/commands.go`. The Task implementation includes `develop`'s running-Attempt cleanup and its own pending-item reconciliation, so that implementation was retained. Conflict markers were removed and compile-only passed; no tests were run.

- Follow-up delivery fix: resolve a relative Task worktree against the primary worktree before checking Git registration. This allows `local-merge` invoked from inside the Task worktree to verify the same isolated workspace binding. Compile-only passed with `go -C .wt/unify-task-storage-under-tasks build -o NUL .`; no tests or merge were run.

- 3.2 更新了 README、运行时存储说明、自动化/Supervisor 文档、multi-actor handoff 文档、AIW work-management/to-spec 技能及 worktree 插件；活动 Task 正式路径统一为 `.ai/tasks/<task-id>/`，归档保持 `.ai/archive/<date>-<task-id>/`。
- 一次性调整步骤已记录：停止旧 AIW 程序 → 停止本 Task Supervisor → 确认无活动 Attempt/写租约/其他写入者 → 核实唯一源和纯标记目标 → 备份标记并移动完整目录 → 核验身份、状态、事件、reports 和 artifacts → 新版只读核对。冲突时停止；不迁移已归档 Task。
- 静态调用方核对范围：`internal/taskpath`、`internal/taskx/meta.go`、`internal/taskx/discovery.go`、`internal/workflow/store.go`、Task 创建/补建/归档调用方和 `plugins/aiw-wt.py`；旧 `.ai/<id>` 仅作 fail-closed 诊断，不再作为活动写入路径。
- Core 状态：wi-0001、wi-0002、wi-0006 completed；wi-0003、wi-0004、wi-0005 cancelled。3.2 由 worktree 版本运行 `workflow sync` 受管同步完成；受管同步后的 Task `task.toml`/summary 为 DONE，execution=completed，validation=not-required，delivery=pending；Git delivery 未执行。状态和事件历史随完整目录保留。
- Compile-only：cancelled Work Item 完成推导更新后再次执行 `GOPROXY=off GOCACHE=.ai/compile-cache/unify-task-storage-under-tasks go -C .wt/unify-task-storage-under-tasks build -o NUL .`，exit code 0；没有生成可分发二进制。未运行测试或 Git 交付。
- 2026-09-25 单次路径调整已完成：完整目录从 `.ai/unify-task-storage-under-tasks/` 移到 `.ai/tasks/unify-task-storage-under-tasks/`；迁移标记备份留在 `.ai/unify-task-storage-under-tasks.migrated-to.pre-move.bak`。移动前核对 Task ID、state/events 序号、无活动 Attempt/写租约及 Supervisor 已停止；移动后用受管 `workflow sync` 完成 Work Item 对账。已归档 Task 和 `.ai/archive/` 未变。
- %% Windows 拒绝了宿主进程枚举；路径调整依据 Core 无写租约、无活动 Attempt、Supervisor stopped，以及本会话中旧 CLI 已退出完成。若曾有本会话之外的旧写入者，需在新版本下只读核对状态与事件。
- 2026-09-25 follow-up：按用户确认，Work Item 的 `cancelled` 是已结清状态，不再阻止 Task execution completed / DONE；`internal/workflow/state.go` 和 workflow-supervision 稳定规格已同步此语义。活动 Attempt、未结清项、阻塞项或阻塞 Gate 仍会阻止完成。
