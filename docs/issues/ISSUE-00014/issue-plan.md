# Issue Plan: FD Reviewer 通过后的 PM 自动接续

Issue ID: `ISSUE-00014`

Status: Captured; awaiting Issue approval. Capture does not authorize implementation.

## 用户、问题与目标

用户希望 AIW 的 FD 工作流在 Reviewer 通过后由本地宿主 PM 继续，不要求用户登录 Plane 或手工接收 PM 事件。目标是让 AI 助手和 CLI 能区分“宿主 PM 应继续处理”与“确实需要用户作决定”，并在当前 Reviewer 结果失效时说明具体原因和下一步。

本次 FD-129 处理中，`verification-passed` 事件的 `target_role` 为 `pm`、状态为 `pending`。AIW 拒绝 Agent 会话 `claim` 该事件；助手误将此限制理解为需要用户接收。检查 AIW 源码后确认，Agent 不应 claim PM 事件；正常 `close Complete` 会在归档成功时自动确认待处理的 PM 回执。真正阻止旧归档的是 Reviewer 事件摘要与当时 FD 文件摘要不一致，需要对当前 FD 重新审查，而不是用户接收事件。

## 已确认事实与来源

- AIW 源码 `src/plugins/aiw-fd.py` 的 `claim()` 禁止 Agent claim `human` 或 `pm` 角色事件，这是角色边界。
- 同文件 `close_locked()` 要求 Reviewer 通过事件与当前 FD 的修订号和摘要一致；成功归档时会将待处理的 PM 回执标记为 `acknowledged`。
- `src/skills/work-management.md` 规定宿主 PM 负责接手并继续角色交接，阶段门槛通过后必须继续正常本地交付。
- 本次复核记录：旧通过事件 `FD-129-000003-verification-passed` 的摘要为 `608f6cd6…`，当时工作树摘要为 `eae38c2e…`；新复审事件 `FD-129-000005-verification-passed` 的摘要为 `ab87a5aa…`，与当前 FD 文件一致。相关事件和报告在 `D:\03_projects\AI-tools\aiw-de\.ai\fd\FD-129\events\` 与 `D:\03_projects\AI-tools\aiw-de\.wt\FD-129\docs\features\FD-129\reviews\`。
- 本 Issue 基于 AIW 源码 revision `607a2327a2d179434bcdde322d3296f5bd336b1c` 及本次 FD-129 操作记录。AIW 源码仓库中尚无同一问题的活动 Issue。

## 范围

- 让 `aiw fd claim` 对 PM 事件给出明确提示：该事件由宿主 PM 通过正常工作流命令处理，不是要求用户在 Plane 接收。
- 让 `aiw fd close <id> Complete` 在缺少当前 Reviewer 通过事件、修订号不匹配或内容摘要不匹配时分别说明实际阻塞项，并指出需要更新审查或继续归档。
- 在 AIW 的工作流说明或相关角色指引中明确：只有显式 `human` 交接或真实业务决策缺失时才请求用户介入。
- 保留当前的 Reviewer 角色隔离、修订号和摘要校验，以及归档时自动确认 PM 回执的行为。

## 非目标与约束

- 不允许 Agent claim `pm` 或 `human` 事件。
- 不绕过当前 Reviewer 通过、修订号或摘要门槛，不使用 force 代替正常交付。
- 不依赖 Plane，不新增远程人工接收步骤。
- 不调整其他事件路由、角色权限或 FD 生命周期。
- Issue 捕获不代表批准、工程设计或实现授权。

## 验收示例

- Reviewer 对当前 FD 通过后，PM 事件可保持 `pending`，宿主执行普通 `close Complete` 即完成归档并自动确认回执；助手不要求用户认领或接收该事件。
- Agent 尝试 claim PM 事件时，CLI 明确说明这是宿主 PM 的工作流步骤，并给出下一步方向；Agent 仍不能 claim。
- 若 `close Complete` 因事件缺失、修订号变化或内容摘要变化而拒绝，错误说明对应的真实原因和应采取的复审/归档动作。
- 目标角色明确为 `human` 或存在未决用户业务选择时，系统仍会请求用户作决定。

## 规模与拆分

这是一个边界明确的 CLI 与工作流指引改进。错误提示、宿主规则和验收场景共同服务于“Reviewer 通过后由宿主 PM 正确继续”这一结果；当前不拆分子 Issue。工程改动细节由后续 FD 设计确定。

