# AIW Issue 使用手册

Issue 可以记录 Bug、功能或修改。新流程是 **Issue → 编号 FD → 角色交接**。
`docs/features/FD-XXX_SLUG.md` 保存工程决策、工作项与验证；
`.ai/fd/<fd-id>/` 保存交接回执。旧 Task 及 Core 数据继续可读。OpenSpec 负责
`openspec/specs/` 中的稳定规格，change 是可选关联。

## 当前命令与兼容性

`aiw issue new <slug> [title]` 创建 `ISSUE-00001` 格式的编号，数字至少五位，
超过 99999 自然扩展。slug 不作为 ID 后缀；省略 title 时用 slug 作为标题。
新记录保存在 `docs/issues/<id>/issue.toml`，Plan 为 `issue-plan.md`。
`aiw req` 继续作为同一程序的入口别名，其默认新建行为同样使用 ISSUE 编号。
旧 `REQ00001-slug` 记录保存在 `docs/requirements/<id>/`，既有 FD 引用和
Session 证据路径保留；需要时再单独迁移，不统一迁移。
审批与推广仍各自需要明确的人类决定。
临时草稿放在 `.ai/issues/drafts/`，旧 `.ai/requirements/drafts/` 仍可读取；
捕获后正式产物写入目标记录所在目录。草稿路径仅作为捕获来源，不作为通用上下文来源。

```text
aiw issue chat [id]
aiw issue new <slug> [title]
aiw issue capture <id> <artifact> --file <path>
aiw issue approve <id> APPROVED --by <actor> --reason <reason>
aiw issue promote <id>
aiw fd new "<title>" --issue <id>
aiw issue show <id> [--json]
aiw issue link-parent <child-id> <parent-id>
aiw issue children <parent-id>
```

大型 Issue 可按独立结果拆成子 Issue。在子 Issue 获批前使用 `link-parent`
记录结构化来源关系；`children` 查询直接子项。在各 Issue Plan 中仍应说明
拆分范围，不要把一个 Issue 的已批准内容静默转移到另一记录。

完整 ID 查找优先，大小写不敏感；例如 `issue-001` 可查找 `ISSUE-001`，
`req00008` 在唯一匹配时可查找 `REQ00008-fd-force-recovery`。缺失或歧义
返回错误，写操作不会选择任意匹配。列表覆盖新旧记录及对应终止目录。
`capture` 接受 `issue-plan` 和既有 `requirement-plan` 名称，新 Issue 写入
`issue-plan.md`，旧 REQ 写入原 `requirement-plan.md`；内部工件键保留
`requirement-plan`，以延续既有摘要与对话协议。

`show --json` 返回 `id`、`status`、`approval_status`，并校验已捕获工件的摘要。
FD 创建使用该接口获取规范 ID 和批准状态；人工阅读继续使用普通 `show`。

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
