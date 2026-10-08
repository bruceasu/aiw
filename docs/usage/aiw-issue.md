# AIW Issue 使用手册

Issue 可以记录 Bug、功能或修改。新流程是 **Issue → 编号 FD → 角色交接**。
`docs/features/FD-XXX_SLUG.md` 保存工程决策、工作项与验证；
`.ai/fd/<fd-id>/` 保存交接回执。旧 Task 及 Core 数据继续可读。OpenSpec 负责
`openspec/specs/` 中的稳定规格，change 是可选关联。

## 当前命令与兼容性

`aiw issue` 是现有 `aiw req` 程序的入口别名。当前版本继续使用 `REQ` 编号、
`docs/requirements/<id>/` 和原 Session 证据路径。旧记录无需迁移。
审批与推广仍各自需要明确的人类决定。
临时草稿放在 `.ai/requirements/drafts/`；捕获后，正式产物存入
`docs/requirements/<id>/`。草稿路径仅作为捕获来源，不作为通用上下文来源。

```text
aiw issue chat [id]
aiw issue new <slug> [title]
aiw issue capture <id> <artifact> --file <path>
aiw issue approve <id> APPROVED --by <actor> --reason <reason>
aiw issue promote <id>
aiw fd new "<title>" --issue <id>
aiw issue show <id>
aiw issue link-parent <child-id> <parent-id>
aiw issue children <parent-id>
```

大型 Issue 可按独立结果拆成子 Issue。在子 Issue 获批前使用 `link-parent`
记录结构化来源关系；`children` 查询直接子项。在各 Issue Plan 中仍应说明
拆分范围，不要把一个 Issue 的已批准内容静默转移到另一记录。

%% NEEDS_INPUT: 独立的 ISSUE 编号和 `docs/issues/` 存储需要单独确定迁移策略；本阶段保留 REQ 数据格式以兼容旧记录。

## FD 交接与推广

已批准的 Issue 可以直接运行 `aiw issue promote <id>`，由 AIW 使用 Issue
标题创建编号 FD，并把来源 Issue 关联到 FD。该命令等价于
`aiw fd new "<Issue 标题>" --issue <id>`；两个入口都会校验审批并由 FD
workflow 负责 Planner handoff。已存在关联时命令会报告错误，不会重复创建。

`[promotion]` 是历史元数据字段。promote 到 FD 不写入 Task ID，也不修改
该字段。FD 设计、角色交接和验证步骤见 [FD 工作流](aiw-fd.md)。

如稳定行为改变，更新相关 `openspec/specs/`。只有用户明确要求使用
OpenSpec change 时才创建或绑定 change；更新稳定规格本身不创建 change。
FD 的实现和完成不依赖它。

## 历史 Task 记录

以下交付与归档说明只适用于已有 Task 记录。新工作使用编号 FD；FD
worktree 交付命令为 `aiw git wt local-merge <fd-id>`。

旧 Task 的本地交付要求隔离工作树干净、相关改动已经提交，并检查分支、
在途写入和 Git 合并安全条件。旧 Task 合并成功后在 primary 继续推进，
保留原工作项、证据和 Session 关联。

%% RISK: 手动合并不会自动核对分支中已提交文件的路径范围；执行前仍须人工审阅分支提交。

旧 Task 归档时，受管 FD 移至
`docs/features/archive/<YYYY-MM-DD>-<task-id>.md`，与 Task 运行记录及存在的
OpenSpec change 使用同一日期和 Task ID。
