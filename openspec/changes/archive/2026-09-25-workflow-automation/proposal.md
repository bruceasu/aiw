# 自动化开发工作流

## Why

已批准的 [REQ00001-workflow-automation](../../../docs/requirements/REQ00001-workflow-automation/requirement-plan.md) 要把现有 Coder—编译链扩展为可恢复的 Coder—编译—Tester—测试—接受闭环，并保留报告、授权、预算及输入版本证据。现有稳定规格仍允许编译后靠清单同步完成，Tester/Verifier 未接通完整监督链；记忆、知识和通知也不能只靠保存类型来表示已执行。

本 change 处理这些实际功能差距。Requirement 推广命令本身的模板生成缺陷不是这项业务功能的实现范围。

## What Changes

- E01：每轮报告及一次补交、来源/验证输入快照、内容适用性与旧 Task 复用。
- E02：阶段状态、串行写入权、未知结果对账、持久 Stop 和可恢复预算；交付只消费已接受成果。
- E03：独立 Tester、受控测试 Runner、AI 批准的具体计划和 `.ai/<task-id>/grant.md`，构成最终接受链。
- E04：按生成请求归因的模型选择、两次有效失败/两次升级及 Coder/Tester 共享六次修复边界。
- E05：有界辅助宿主与 Task memory；原始事实优先，后台任务按固定输入有界恢复。
- E06：知识提取、部分草稿、五态版本审阅、失效复核、确定性注入和相关历史限额。
- E07：`aiw-notify` 文本通知，首期 console 和 Microsoft Teams（`send_teams_msg.py`）；仅明确网络失败时 60 秒后重发一次。
- E08：封存输入并实际异步派发只读 Verifier；报告不撤销接受或触发返工。
- 具体 Git 计划由 AI 审批，允许本地 commit/分支创建/merge；禁止 push 与 Agent 直接 Git 删除，清理只能经 AIW 和对应 grant。

## Capabilities

### New Capabilities

- `task-memory`：Task 来源、Session 投影、有界辅助宿主和固定输入恢复账（SW09/SW18/SW19）。
- `project-knowledge`：异步生成、版本审阅、选择和资源边界（SW20–SW24/SW26/SW27）。
- `workflow-notifications`：完成/阻塞文本通知、有限网络重试与插件通道（SW28–SW30）。

### Modified Capabilities

- `workflow-supervision`：报告、阶段、写入权、恢复与接受（SW01/SW02/SW05–SW08）。
- `verification`：内容有效性、旧证据复用、AI 计划授权、独立测试及 report-only Verifier（SW03/SW04/SW10–SW14/SW31/SW32）。
- `ai-routing`：请求快照、有效失败和升级预算（SW15–SW17）。
- `agent-session`：真实来源正文和精确版本的角色输入（SW25）。
- `task-lifecycle`：开发/交付/辅助分维度状态、一个整体生命周期和设计交接（SW34）。
- `workspace-delivery`：基于具体 AI grant 的本地交付及受管清理（SW33）。

## Impact

| 工程组 | 保持的职责 / 拟改动入口 | 验收入口 |
| --- | --- | --- |
| E01 | `internal/workflow/actor_contracts.go`、Task handoff、Session prompt 的事实和来源绑定 | AC01–AC04/AC25 |
| E02 | `internal/workflow` runner/attempts/store 与 `workflow/execution` supervisor/session | AC05–AC08/AC14/AC33 |
| E03 | verification plan、test authoring、受控测试执行服务；CLI 保留展示/参数职责 | AC10–AC14、AX01/AX03 |
| E04 | 现有 Profile/AISelection/routing plan 和预算事实 | AC08/AC15–AC17 |
| E05 | Task 来源投影、辅助登记/队列/固定输入/恢复 | AC09/AC18/AC19/AC24/AC25/AC27 |
| E06 | 项目知识条目与版本、覆盖清单、审阅/失效索引 | AC20–AC27、AX05 |
| E07 | `internal/workflow/notification.go`、`plugins/aiw-notify.py`、`plugins/send_teams_msg.py` | 经修订 AC28–AC30、AX04 |
| E08 | `internal/workflow/verifier.go`、只读派发与封存来源 | AC19/AC27/AC31/AC32 |

保留 AIW 对 Task/Session/工作区和运行事实的所有权、OpenSpec 对需求/设计/清单的所有权。当前稳定规格保持原样，本目录的 delta 表达目标变更；实现时按组推进，不另建平行 Task 或状态权威。

## Scope and Source Precedence

一个 Requirement、一个 `workflow-automation` Task/change，内部 E01–E08。初始测试与 Git 计划的 AI 授权、通知两次总尝试和插件兼容边界，以 Plan 后续确认章节为准；旧来源的人工逐计划批准、三次通知/30 秒及两分钟间隔、不允许无幂等网络重试等条款被显式替换。

[原正文](../../../docs/requirements/REQ00001-workflow-automation/software-requirements.md) 和 [验收追踪](../../../docs/requirements/REQ00001-workflow-automation/acceptance-traceability.md) 保留为历史来源。本目录 specs 已将修订写入对应规范和场景，包括 AX04 的网络失败例外。所有场景仍待执行。

不包含集成/E2E 业务测试、向量检索、全项目历史扫描、自动知识/需求批准、Verifier 返工、永久服务或开机自启。通知能力不扩大开发测试的网络权限。工件补全由工程承担，不增加一次用户确认；只有已确认行为、授权或成本边界必须改变时才重新提问。
