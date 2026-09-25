## 1. TODO：统一生命周期投影

- [x] 1.1 实现统一 Core 摘要到 task.toml 的派生快照写入，覆盖完整显示状态和 delivery，原子更新并保留未知字段；接入完成及交付入口，禁止从缓存反向覆盖已有 Core。
- [x] 1.2 在工作树清理成功后解除当前绑定并保留历史，支持 merged 后部分清理失败及幂等恢复；同步相关注释和工作流契约。
- [x] 1.3 修复归档工件定位和资格检查，支持已交付且已清理的 unassigned/历史 isolated Task，双后端保持一致，不为未交付任务静默回退主目录。

## 2. TODO：历史修复与回归

- [x] 2.1 提供统一显式历史修复入口，覆盖活动/旧版/归档记录、预览、备份、锁定及幂等性，证据不足或占用时跳过并诊断。
- [x] 2.2 编写生命周期集成回归：DONE+merged+已清理、清理各阶段失败、快照写入失败、待授权/待验证不误判、归档就地修复、未知字段保留及重复操作；覆盖 native/openspec。
- [x] 2.3 聚焦静态核对、更新文档和 Verification；实现后按预算执行 compile-only，测试运行需另行授权。

## Verification

- 2026-09-17 恢复记录：第 6 轮 Session 已持久化 completed、exit_code=0 和结构化最终输出；后续 status.json rename 失败造成 Core 将该轮记为 no-progress。经用户授权，保留失败 Attempt 与原始错误，新增独立恢复事件及证据，使用 Core 完成接口接受 2.3。实际在本工作树执行 `python scripts/compile.py`，禁用依赖下载，退出码 0，未运行测试或保留发行物。Core 及 task.toml 已恢复为 DONE，工作区 isolated，delivery=pending，supervisor 已停止。
- %% 历史事件 6–9 在现有日志和此前备份中均缺失；未补造、重编号或删除历史记录，event-gap 诊断仍存在。详见主工作区 `.ai/fix-task-metadata-lifecycle-sync/artifacts/session-result-recovery/README.md` 及恢复前备份。Windows rename 错误的永久代码修复未在本次状态恢复中实现；未执行 commit、merge、归档或工作树清理。

- 1.3: Archive eligibility rejects an unassigned Task before checklist synchronization unless it records merged or discarded delivery. A cleaned unassigned Task synchronizes the discovered change directory and rechecks its historical worktree and branch after archive. Native and OpenSpec backends share this preflight and recheck path; the supervisor owns tests and compile-only validation.

- 1.2：`localMergeDelivery` 在 Core 记录 merged 后将工作树移除、原子解除 task.toml 当前绑定、分支删除分为可恢复步骤；`UnassignWorkspace` 仅更新 `worktree` 与 `workspace_kind` 并保留历史字段及未知 TOML 内容。静态追踪首次合并、元数据写入失败、工作树已清理和分支已删除的重试路径；测试和 compile-only 由 supervisor 按预算执行。

- 1.1：`taskx.WriteWorkflowSummary` 复用纯 Core 摘要映射，仅原子更新 `status` 与 `delivery`，保留其他 TOML 内容；`projectWorkflowState`、交互式 Attempt 结果与归档同步均通过该入口。已静态追踪完成、delivery 与读取/展示调用链；测试未运行，compile-only 由 supervisor 执行。

- [x] 启动阻塞修复：`supervise start` 在 Core 运行投影缺失且 Task 元数据身份有效时，复用 `EnsureCompatible` 初始化；不凭 change 或迁移标记重建 Task。`status`、`stop` 和非法 action 不初始化，已有损坏状态不覆盖。
- 已增加 `workflow_supervisor_initialization_test.go` 回归源码，覆盖启动初始化、只读/非法 action、仅迁移标记、身份冲突和损坏状态。测试未运行。
- 2026-09-17 现场：目标 `.ai/<id>`、Session 和 worktree 均不存在，旧 `.ai/tasks/<id>` 仅含 `migrated-to`。不能从当前证据确定删除原因或恢复 Attempt 历史。用户明确选择保留现场、只修复代码；没有重建 Task、同步 Core 或启动 supervisor。
- 编译：禁用依赖下载后执行 `python scripts/compile.py`，退出码 0，临时可执行文件由脚本清理。静态检查了最终 diff、初始化调用链和回归夹具；未运行测试、formatter、lint、网络或 Git 写操作。compile-only 不编译 `_test.go`。
- 本次仅修复启动阻塞与恢复提示，不代表上述生命周期任务完成。首次修复按用户要求保留现场，未执行 Workflow sync。
- 后续用户明确授权重建初始运行信息：恢复 `.ai/fix-task-metadata-lifecycle-sync/task.toml` 为普通新建 Task 的 primary / `.`、develop、同名 Session 映射及 TODO 初始值；执行 `go run . workflow sync fix-task-metadata-lifecycle-sync` 成功创建 state.json、events.jsonl 和六项 ready Work Item。Core 为 DRAFT / queued，delivery=unmanaged；Attempts、Gates 为空，无写租约或 prepared request。旧执行历史没有恢复，未创建 Session/worktree，未启动 supervisor。

- 本轮仅创建修复提案并执行用户明确授权的历史数据修正，没有实现上述生产代码任务。
- 通过 AIW 创建 Task/change：`aiw new fix-task-metadata-lifecycle-sync --backend auto --allow-unrelated-dirty`。主工作区创建，没有新建分支/worktree。
- 静态核对：Core DeriveSummary、reportWorkflowState、localMergeDelivery、WorkflowChecklistPath、archive 资格检查、task-lifecycle 稳定规格及六项原有 Task 元数据。
- 已修正 fix-task-list-discovery、fix-paired-task-archive：status=DONE、delivery=merged、workspace_kind=unassigned、worktree 为空；保留 branch/parent_branch/session。
- 已就地修正归档 improve-requirement-management：status=DONE，updated=2026-09-17；保留 primary/unmanaged 及归档路径。
- 依据：三项 Core 均明确记录 DONE/completed/not-required；两项隔离 Task 已记录 merged，目录和工作树注册不存在，loose/packed 分支引用均无对应记录。未重新执行 Git 合并或祖先检查。
- 修复前后全文和 Core 摘要保存在 `.ai/fix-task-metadata-lifecycle-sync/artifacts/metadata-repair/<task-id>.json`。未修改 Core、Gate、Attempts、清单完成状态或其他未完成 Task。
- 实际命令：Get-Content、Get-ChildItem、Get-Command、Test-Path、rg、Select-String；aiw help/new；openspec.cmd status/instructions；PowerShell 定向元数据写入及 apply_patch 文档编辑。
- 未运行测试、编译、最终构建、lint、格式化、结构验证脚本、网络或 Git 写操作；本轮没有生产代码实现，无需编译。
- %% 生产归档修复尚未实现，不能把本次元数据修正视为 archive 已通过。历史交付依赖已有 Core 记录；本轮未重新验证 Git 祖先关系。

- 2.1：`workflow repair-metadata [task-id] [--dry-run]` 以 `DiscoverTaskLocations` 枚举活动、旧版与归档运行记录；在 Core 任务锁下只从合法终止状态派生 `task.toml` 快照，并仅在已交付隔离资源均不存在时解除当前绑定。预览不写入，真实变更在原子替换前备份原 TOML，未知字段保留；运行中、占用、身份冲突、Core 缺失/损坏或清理证据不足的记录被跳过并输出诊断。测试与 compile-only 由 supervisor 按预算执行。

- 2.3：静态核对 `WriteWorkflowSummary`/`DeriveSummary` 的单向投影、`localMergeDelivery` 的可恢复清理、`WorkflowChecklistPath` 与归档资格预检、`repair-metadata` 的历史定位和锁定路径，以及 `supervise start` 的受限初始化条件；变更规格的场景与清单保持一致。已更新本清单和 Verification。未运行测试、编译、最终构建、lint、格式化、网络或 Git 写操作；compile-only 由 supervisor 按冻结计划执行。
