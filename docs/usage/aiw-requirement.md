# AIW Requirement 兼容入口

`aiw req`、既有 `REQ` ID 和 `docs/requirements/<id>/` 仍受支持。
新工作请使用 [AIW Issue 使用手册](aiw-issue.md)；`aiw issue` 是同一程序的首选入口。
旧记录不自动迁移，原有 Session、工件摘要和批准历史保持有效。

## 兼容命令

```text
aiw req new <slug> [title]
aiw req new --id <id> [title]
aiw req chat [id]
aiw req show <id>
aiw req capture <id> <artifact> --file <path>
aiw req approve <id> APPROVED --by <actor> --reason <reason>
aiw req promote <id> --task <task-id>
aiw req link-parent <child-id> <parent-id>
aiw req children <parent-id>
aiw req archive <id> --reason <reason> [--by <actor>]
aiw req cancel <id> --reason <reason> [--by <actor>]
```

推广要求正式批准，创建或复用 AIW Task、来源 handoff 和 FD。FD 的设计
就绪状态与工作项可映射后，推广状态成为 `FD_READY`。OpenSpec change 可选；
旧 `SPEC_DRAFTED` 记录仍可读取。批准与推广各自需要原有授权，推广不开始实现。

正式讨论工件继续保存在 `docs/requirements/<id>/`。`parent_id` 为可选
元数据字段；缺少它的旧记录按根 Issue 读取。子 Issue 必须在批准前建立
父项关系。独立的 ISSUE 编号和 `docs/issues/` 存储留待显式迁移设计。
