# AIW Requirement 兼容入口

`aiw req`、既有 `REQ` ID 和 `docs/requirements/<id>/` 仍受支持。
新工作请使用 [AIW Issue 使用手册](aiw-issue.md)；`aiw issue` 是同一程序的首选入口。
旧记录不自动迁移，原有 Session、工件摘要和批准历史保持有效。
默认 `new` 使用独立 ISSUE 编号和 `docs/issues/`；显式 `new --id REQ…`
仍可创建旧格式记录。REQ 短编号在唯一匹配时可用，大小写不敏感。

## 兼容命令

```text
aiw req new <slug> [title]
aiw req new --id <id> [title]
aiw req chat [id]
aiw req show <id>
aiw req capture <id> <artifact> --file <path>
aiw req approve <id> APPROVED --by <actor> --reason <reason>
aiw req promote <id>
aiw req link-parent <child-id> <parent-id>
aiw req children <parent-id>
aiw req archive <id> --reason <reason> [--by <actor>]
aiw req cancel <id> --reason <reason> [--by <actor>]
```

推广要求正式批准，并使用 Requirement 标题创建编号 FD、关联来源 Issue。
命令等价于 `aiw fd new "<标题>" --issue <id>`，由 FD workflow 创建 Planner
handoff。推广不会创建 Task、开始实现或修改 `[promotion]` 的 `status` 和
`task_id`；旧 `SPEC_DRAFTED` 记录仍可读取。OpenSpec change 可选。

旧 REQ 的正式讨论工件继续保存在 `docs/requirements/<id>/`。`parent_id` 为可选
元数据字段；缺少它的旧记录按根 Issue 读取。子 Issue 必须在批准前建立
父项关系。新 Issue 使用 `docs/issues/<ISSUE-ID>/`；已有 FD 来源按原 ID
读取，按需迁移政策不要求统一迁移，本次不提供迁移命令。
