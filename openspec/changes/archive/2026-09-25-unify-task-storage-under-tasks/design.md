## Context

Task 元数据和 Workflow Core 已切换为 `.ai/tasks/<id>`；旧 `.ai/<id>` 只用于发现并拒绝运行。当前仓库唯一仍活动的 Task 是本 Task，其旧运行目录需要在开发完成后人工调整。用户确认其它 Task 均已归档，且归档路径保持 `.ai/archive/<date>-<id>`。

## Design Readiness

FD_APPLIED：此前 managed Feature Design 已完成存储归属、兼容边界和调用方影响分析。用户随后缩小本次迁移工作：取消通用迁移 CLI、事务日志、rollback 与迁移入口级回归，只在本 Task 开发完成后人工调整唯一活动 Task。

## Goals / Non-Goals

**Goals:** 单一活动 Task 路径，统一解析接口，旧布局 fail-closed 诊断，保持 Task 身份、历史事件和归档位置。

**Non-Goals:** 通用或批量迁移工具、自动迁移、迁移 journal/rollback、已归档 Task 移动、Session/OpenSpec/归档位置变更。

## Decisions

### 1. 最终布局与职责

活动 Task 为 `.ai/tasks/<id>/`，包含 task.toml、state.json、events.jsonl、routing-plan、reports 和 artifacts。Task 归档仍为 `.ai/archive/<date>-<id>/`；Session 仍为 `.ai/sessions/<session-id>/`，其配对归档布局不变。OpenSpec 位置不变。

### 2. 统一路径与旧布局处理

taskx、Workflow Store 和发现模块共用低依赖的活动 Task 路径解析。正常读写只解析 `.ai/tasks/<id>`；发现旧目录或迁移标记时停止读写并显示手工调整提示，不回退读取、不自动移动、不创建同 ID 的第二份记录。保留 `task.toml` / `tasks.toml` 文件名兼容。

### 3. 当前 Task 的一次性人工路径调整

仅在代码工作完成、Supervisor 与所有 AIW 写入者停止后调整当前 Task。先备份 `.ai/tasks/<id>/migrated-to` 标记，再把 `.ai/<id>` 的完整目录移至 `.ai/tasks/<id>`；核对 Task ID、state.json、events.jsonl 及目录内容后再移走标记备份。若源/目标身份冲突或写入者未停止，则保持原状并报告。已归档 Task 不动。

此变更不提供 `aiw task storage migrate`、批量扫描、迁移日志、自动恢复或 rollback。手工调整保留完整目录，不重建 Task，不修改事件序号或历史正文。

## Testing Decisions

保留 Task 创建、列表、Workflow、归档入口的既有测试范围。用户取消 2.1、2.2 的通用迁移实现与 3.1 的迁移入口级回归；本 Task 仍需完成 3.2 的文档、静态调用方核对和 compile-only。测试命令未获授权，不运行。

## Risks / Trade-offs

- 人工调整只覆盖当前唯一活动 Task；后续旧布局 Task 需要运维人员停写并按诊断提示完整调整。
- 必须先停止本 Task Supervisor；活动 Attempt 或写租约未清除时不得移动运行目录。
- 调整时先备份目标迁移标记，并以 state/events 身份核对避免覆盖已有记录。

## Migration Plan

1. 完成剩余代码和文档工作，更新 checklist/Core 状态。
2. 停止 Supervisor 与所有 AIW 写入者，确认 `.ai/<id>` 是唯一活动数据目录，目标只有迁移标记。
3. 备份标记，将完整目录移到 `.ai/tasks/<id>`，核验状态、事件和 Task 身份，再保留/移走标记备份。
4. 使用新路径版本做只读状态核对；归档仍位于 `.ai/archive/<date>-<id>`。

## Open Questions

2026-09-25：用户确认没有其他活动 Task；通用迁移 Work Item 2.1、2.2 和迁移回归 3.1 取消。剩余仅为 3.2 和开发完成后的单次人工路径调整。

## 1.1 启动基线与路径影响盘点

启动条件依据本 Task handoff/Session Memory 中用户确认：不存在待完成或待归档的前置 Task；supervised-dependency Gate 已 waived。重新核对归档后的 `openspec/specs/task-lifecycle/spec.md` 及当前代码后，稳定规格仍描述旧正式目录 `.ai/<id>`、旧目录兼容读取 `.ai/tasks/<id>`；本 change 将更新此存储归属要求。

| 调用方/证据 | 当前路径行为 | 迁移影响与目标 |
| --- | --- | --- |
| `internal/taskx/meta.go`: `RuntimeTaskDir`, `TaskMetaPath`, `RuntimeTasksPath` | 元数据正式写入 `.ai/<id>`；目标 `.ai/tasks/<id>` 存在时回退读取 | 用共享活动路径解析器返回 `.ai/tasks/<id>`；旧根仅探测并给出 migration-required，保留 `task.toml` / `tasks.toml` 文件名兼容 |
| `internal/workflow/store.go`: `Store.taskDir`, `legacyTaskDir`, `Load`, 迁移/恢复路径 | `Root` 是 `.ai`；Core 位于 `.ai/<id>`，`Load` 回退 `.ai/tasks/<id>` 并可迁移旧状态 | `Root` 继续表示 `.ai`、锁继续位于 `.ai/locks`；Store 的活动目录交由共享解析器，新旧 Core 不再回退混读 |
| `internal/taskx/discovery.go`: `DiscoverTaskLocations` / `TaskLocation` | 扫描 `.ai`、`.ai/tasks`、运行归档及 OpenSpec 活动/归档目录；旧路径参与重复检测和补建选址 | 新活动发现根统一由解析器提供；旧活动记录保留为迁移诊断，归档扫描及身份冲突检查继续有效 |
| Task 创建/补建/执行/恢复/归档调用方 | 创建和 Workflow 写入使用各自根路径；发现结果向元数据修复、归档、Session 配对等入口提供 `RuntimeDir` | 这些入口必须消费统一的活动目录结果；旧布局诊断时禁止创建、补建或继续执行；归档目的地保持 `.ai/archive/<date>-<id>/` |
| 维护文档与技能 | `README.md` 已描述 `.ai/tasks` 和归档发现；`internal/commands/task/init.go` 帮助仍写 `.ai/<task>/task.toml`；`.agents/skills/metrics-review/skills/work-management.md` 将正式元数据路径写为 `.ai/tasks/<task-id>/task.toml` | 同步正式路径说明及迁移操作；OpenSpec、Session、归档位置保持既有约定 |
| 其它路径引用 | workflow 迁移标记 `migrated-to`、事件 `runtime.migrated` 和历史目录探测散布于 `taskx` / `workflow`；README 列出归档根 | 标记只由显式迁移流程核验/备份；不要删除历史事件或重写历史正文；工件路径映射限定到已记录 Task 本地引用 |

最终路径接口约定：低依赖公共模块提供 `ActiveTaskDir(root, id)`（唯一新活动目录）、`LegacyTaskDir(root, id)`（只读诊断探测）及供调用方区分“新路径不存在”与“旧记录待迁移”的解析结果；`taskx`、Workflow Store、发现/创建/修复/归档调用方不得各自拼接活动根。`Store.Root` 仍为 `.ai`。解析接口不执行迁移、不读取旧 Core、不创建目录；显式迁移命令是唯一移动数据的入口。实现时须保持归档和 Session 路径不变，并避免引入 `taskx` 与 `workflow` 的循环依赖。
