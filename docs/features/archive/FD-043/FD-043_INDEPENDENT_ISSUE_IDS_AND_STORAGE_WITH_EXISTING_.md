# FD-043: Independent Issue IDs and storage with existing REQ references

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem

维护者需要让新 Issue 使用独立的稳定编号和目录。当前 `aiw issue` 仍自动生成 `REQ00001-slug`，读取时直接把输入拼进 `docs/requirements/<id>/`；`req00008` 无法找到 `REQ00008-fd-force-recovery`。FD 创建端也硬编码了 Requirement 路径。

用户已确认：新 Issue 可使用 `ISSUE-…` 编号和独立目录；现有 FD 中的 REQ 来源继续保留，需要时再迁移，无需统一迁移。该确认是本 FD 的需求来源，不代表旧数据已经迁移或新行为已经实现。

## Options and decision

- 继续为新记录生成 REQ 编号：改动较小，但不能满足用户确认的独立 Issue 命名方向。
- 批量改写旧 REQ、FD 引用和历史证据：涉及大量历史记录，用户明确要求按需迁移。
- 新记录使用独立 ISSUE 编号，通过统一记录定位读取新 Issue 和已有 REQ：选定。新编号、存储和旧记录读取分开处理，现有引用按原 ID 解析。

按需迁移是后续单条记录工作的政策；本 FD 不实现自动迁移或迁移命令。

## Solution

1. 新记录的完整 ID 为 `ISSUE-001`，至少三位数字，超过 999 自然扩展，不附加 slug；标题独立存储。沿用 `new <slug> [title]` 的命令形态，slug 不再参与完整 ID。活动记录位于 `docs/issues/<id>/`，终止记录位于对应 `archive/`、`cancelled/` 子目录，元数据为 `issue.toml`。
2. 独立编号分配使用 `.ai/issues/` 下的序列及现有分配锁机制，扫描活动和终止 ISSUE 记录，避免重用已分配编号。REQ 序列不参与 ISSUE 编号计算。
3. 在 `internal/issue` 中集中负责记录定位。完整 ID 优先匹配；支持大小写无关的 `ISSUE-001` 和旧编号 `REQ00008`，后者仅在唯一匹配时解析为完整 REQ ID。缺失和歧义必须明确报错，禁止选择第一个匹配项。定位覆盖活动、归档和取消目录，重复完整 ID 同样报错。
4. 新 Issue 的 Plan 使用 `issue-plan.md`；命令接受 `issue-plan` 工件名，已有 `requirement-plan` 名称继续可用。沿用现有状态、revision、来源摘要、审批和父子关系规则；旧记录写入仍采用其原格式和原目录。暂不重命名全部内部类型、JSON 字段或 Session 协议，避免把存储变更扩大成全量协议改造。
5. show、capture、approve、link-parent、children、chat、archive、cancel、promote 使用同一个定位规则，写操作先取得规范 ID 与实际路径。列表可见新旧两种记录，保留现有活动/终止过滤语义。跨新旧记录的父子关系保存规范 ID。
6. FD `new --issue` 和 promote 支持两种存储，审批校验与重复关联校验使用规范 ID；新 FD 写入规范 ID，已有 FD 来源行和归档不重写。Python FD 入口通过受支持的 CLI 读取记录身份和审批信息，避免再实现一套目录扫描逻辑；结构化读取接口只返回此流程所需字段，不暴露未使用的内容。
7. `aiw req` 继续作为现有入口别名，默认创建行为跟随 Issue 命令，不维护另一套新建 REQ 流程。已有完整 REQ ID 和精确 ID 创建语义仍可用；显式创建 REQ 记录时遵循旧格式。

风险集中在编号锁、双目录查找、审批摘要和 FD 来源解析。Worker 应逐项追踪读写路径，避免将归档记录可读误当作允许修改或重新推广。

## Scope

包含新 ISSUE 编号与目录、统一记录定位、旧 REQ 短编号解析、Issue 生命周期命令、FD 来源读取及对应帮助和稳定规格。

范围外：批量迁移、改写现有 FD/历史证据、单条迁移命令、全量 Session/内部字段重命名、状态机变更、审批豁免。FD-042 的强制恢复操作属于独立需求，本 FD 不交付该范围。

## Work items

- [x] 1.1 实现统一记录定位与规范 ID 返回，覆盖新旧根目录、终止目录、大小写、REQ 短编号及歧义诊断。大小：S；难度：Medium；依赖：无；完成证据：静态追踪完整匹配优先、重复拒绝和路径约束。
- [x] 1.2 实现 ISSUE 编号分配、`issue.toml` 与 Plan 工件存储，保留精确 REQ 创建语义。大小：M；难度：Medium；依赖：1.1；完成证据：编号回收、序列锁和新旧格式选择路径可审查。
- [x] 1.3 将读取、列表和上下文加载接入统一定位，保留旧审批及工件摘要校验。大小：M；难度：Medium；依赖：1.2；完成证据：新旧记录及终止过滤调用路径一致。
- [x] 1.4 将捕获、审批、父子关系、对话绑定和终止操作接入实际记录路径。大小：M；难度：Medium；依赖：1.3；完成证据：所有写入使用规范 ID，历史来源不自动迁移，终止记录写限制有效。
- [x] 1.5 接通 promote 与 FD 来源读取的结构化接口，使用规范 ID 检查审批和重复关联。大小：M；难度：Medium；依赖：1.4；完成证据：直接 `fd new --issue` 与 promote 共享来源身份语义，现有 FD 引用不改写。
- [x] 1.6 更新帮助、Issue/Requirement 使用说明、稳定规格和有关技能指引。大小：S；难度：Low；依赖：1.5；完成证据：文档明确新建 ISSUE、旧 REQ 读取、歧义报错和按需迁移政策，不宣称实现迁移。

各项预计不超过半天，属于计划估计。若 Worker 发现单项超过该边界，应在实施前进一步拆分，保留编号。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- 连续新建 Issue 获得递增的 `ISSUE-001` 形式 ID，标题/slug 变化不改变 ID，活动和终止编号不会重用。
- 使用完整 ID 或大小写不同的 `issue-001` 能定位新记录；使用 `req00008` 能唯一定位旧完整 ID。无匹配或多匹配时错误退出，写命令不改记录。
- 新旧 Issue 可分别捕获 Plan、审批、关联父项、查看子项和终止；旧格式缺少新字段时保持既有默认语义，既有摘要和审批证据继续有效。
- 已批准的两种来源均可创建 FD，未批准、终止或重复关联的来源按既有规则拒绝；通过短编号创建时 FD 记录完整规范 ID。
- 新功能不会改写现有 REQ 目录、已有 FD 来源或历史批准记录。列表和帮助明确展示实际 ID。
- 默认验证只包含静态检查及实施后的一次窄范围 compile-only；未执行的运行场景不得记为通过。

## TODO

- [x] 记录用户确认的新建与按需迁移政策，核对现有存储和 FD 来源代码。
- [x] Worker 在专用分支/worktree 实施上述工作项。
- [x] 记录 compile-only 与静态证据，准备独立 Reviewer 交接。

## Verification

- 独立 Reviewer r1 静态审查通过；报告：docs/features/reviews/FD-043-review-r1.md 及同名 JSON。审查提交 da22fe2c，来源 FD-043-000004-implementation-ready；无阻塞发现，未运行测试或重复编译。

- 设计阶段未运行验证。Worker 已完成工作项 1.1-1.6，并在离线环境执行一次 Python 内存编译与 `scripts/compile.py` Go compile-only，返回 0；未执行测试、运行时验收、最终构建、格式化、lint 或 vet。详见 `docs/features/reports/FD-043-implementation-r1.md` 及 JSON。
- 实施后只运行一次覆盖实际改动的 compile-only；同时涉及 Go 与 Python 时使用单个无最终产物的命令覆盖两者，具体命令由 Worker 在检查 compile 脚本后记录。
- 静态核对定位、编号、摘要、审批、终止过滤、FD 来源及重复关联路径。实现完成后只使用一次静态验证命令审阅最终差异。
- 独立 Reviewer 核对上述验收和实现证据；本计划不创建测试、不授权测试运行，也不把缺失运行时覆盖标记为通过。

## Sources

- Issue: none
- 本会话用户 2026-10-09 确认：现有 FD 中的 REQ 引用保留，需要时迁移，不统一迁移；随后 `confirm` 确认新 Issue 可独立编号和存储。
- `internal/issue/store.go`：当前根目录、路径拼接及元数据。
- `internal/issue/numbering.go`：REQ 编号与锁。
- `cmd/aiw-req/issue-create.go`：创建入口与精确 ID 语义。
- `plugins/aiw-fd.py` 的 `new_fd`：来源路径、批准与重复关联判断。
- `openspec/specs/requirement/spec.md`：原有编号、审批、来源摘要和推广规则；实施时更新受影响条款。
- `docs/usage/aiw-issue.md`：独立 ISSUE 存储的待定政策，本会话确认后已可设计。
- `docs/features/archive/FD-031/FD-031_RESTORE_ISSUE_PROMOTE_AS_FD_WORKFLOW_ENTRY.md`：已交付的 promote 契约，历史文件不改写。
- Planner session：`fd043-planner-233afb7954a0`；来源事件：`FD-043-000002-design-requested`。

**Completed:** 2026-10-09
