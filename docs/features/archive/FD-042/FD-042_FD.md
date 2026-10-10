# FD-042: FD 工作流强制恢复与事件控制

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem

维护者/PM 在 FD 异常时需要取消当前 dispatched 收据、直接修改状态和重新 emit。现有恢复命令只支持受约束的正常恢复，无法覆盖批准需求中的人工强制恢复。取消不终止原 Agent，调用者负责处理仍运行的会话。

## Options and decision

比较扩展普通 emit 的隐式跳过开关与增加显式维护命令。选择 `cancel-event`、`set-status`、`force-emit` 三个命令，避免正常流程无意跳过校验。保留已定义事件/producer/target role 配对、仓库内存在的 artifact 和有效 FD 元数据；强制 emit 跳过状态、完成度、证据与前序 claim 校验。普通命令保持原契约。

## Solution

三个命令均要求单行且不超过 500 字的 `--reason` 和显式 `--operator`（声明身份，不代表认证）。仅操作活动 FD；归档 FD 继续使用现有 reopen/request-review，避免隐式搬移历史记录。

`cancel-event <id> --expected-event <event>` 只取消最新 dispatched 收据，精确 ID 防止取消已变化目标，保留 session、pid 和历史字段。它不改变 FD 状态或 revision。

`set-status <id> <status>` 支持 Planned、Design、Open、In Progress、Pending Test、Pending Test Acceptance、Pending Verification、Complete、Deferred、Closed；增加 revision 并更新索引，保留 Work Items、收据和文件位置。原收据可能失效，必须明确说明。

`force-emit <id> <event> --producer <role> --artifact <path>` 支持 ALLOWED 内的事件。design-requested 固定 pm，decision-recorded 固定 human，其余沿用 ROLE_PRODUCERS；needs-decision 支持已定义角色。decision-recorded 按当前状态映射恢复角色，终态映射 pm。保留 Independent 的 implementation-ready 路由。增加 revision；新收据为 pending，强制命令不自动启动 runner；旧未完成收据取消，已结束收据保持历史。事件编号取当前 revision 与历史最大事件 revision 的最大值再加一，防止覆盖历史。

在共享 `.ai/fd/<id>/operations/` 记录 UUID 审计 JSON，包含操作者、原因、时间、原状态/事件、结果及跳过校验；新收据保留强制标记和审计引用。使用现有 fd_lock 和原子写，失败回滚所有本次写入，回滚失败明确报告。强制 verification-passed 不代表独立审查通过：Complete 归档及 archived request-review 必须拒绝 forced 收据。

## Scope

范围：`plugins/aiw-fd.py`、`docs/usage/aiw-fd.md`、稳定 FD 规格及本 FD 证据。不添加依赖、进程管理、网络、任意事件/状态、自动重启、测试、迁移或强制归档。CLI 与审计持久化变更属于批准需求的显式范围；普通恢复行为保持兼容。

## Work items

- [x] 1.1 增加共用强制操作审计/事务和取消最新 dispatched 收据。规模：小（半天内）；难度：中；依赖：无；完成：精确 ID 检查、原因/操作者验证、历史保留与失败回滚可静态追踪。
- [x] 1.2 增加强制状态设置。规模：小（半天内）；难度：中；依赖：1.1；完成：全部定义状态可选、revision/索引更新、历史不伪造。
- [x] 1.3 增加强制 emit 和正常验收防伪边界。规模：小（半天内）；难度：中；依赖：1.1、1.2；完成：定义事件结构、pending 路由、历史取消留痕、forced 不满足 Complete 审查 Gate。
- [x] 1.4 同步使用文档与稳定规格，提交实现证据；独立审查由后续 Reviewer 阶段记录。规模：小（半天内）；难度：低；依赖：1.3；完成：命令参数、限制、实际检查与风险一致。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

1. cancel-event 仅对指定最新 dispatched 收据成功，保留原身份并明确 Agent 未停止；目标已变化、非 dispatched 或无事件时拒绝且不写数据。
2. set-status 可覆盖正常状态机及终态证据要求，只修改状态/revision/索引并留审计，不生成 Reviewer 证据或搬移归档。
3. force-emit 可从任意已定义状态生成任意 ALLOWED 事件，跳过阶段前置条件；无效 producer 配对、无效 artifact、非法状态/事件、无原因或操作者仍拒绝。
4. 三种操作可追踪操作者、原因、FD、旧状态/事件和结果；失败恢复原 FD、索引、收据和审计，无可领取孤儿。
5. forced verification-passed 和强制 Complete 均不能通过正常 Complete 归档的独立审查条件；正常命令保持校验。

## Verification

- Compile-only check and static review by default. Optional tests use `$fd-test`
  when requested and do not gate FD acceptance.
- 执行一次仓库 `scripts/compile.py`（Go，不保留分发产物；禁止依赖下载），并在同一编译命令中用 Python `compile()` 编译改动插件源码，不执行插件业务代码。
- 静态追踪 CLI 参数、角色路由、事务回滚、审计内容、普通归档 Gate 和文档一致性；不运行测试、运行时故障注入、格式化、lint、构建、网络或发布。
- 残余风险：静态证据不能证明文件系统故障下回滚、仍运行 Agent 的并发写入或各状态/事件组合的运行结果；独立 Reviewer 核对实现与验收。

## 实现证据

- 1.1–1.4 已实现；静态审查完成，离线 Python 源码编译及 `scripts/compile.py` 成功（exit 0）。
- 实现报告：`docs/features/reports/FD-042-implementation-r1.md` 及同名 JSON。
- 未运行测试、故障注入、lint/formatter/vet、最终构建或网络；剩余风险见报告。
- 独立 Reviewer：`fd042-reviewer-20261009-c95a17` 已领取 `FD-042-000004-implementation-ready`，审查 `0df4d1bf` → `5a1773cf`；五项验收均获得静态支持，未发现阻塞问题。报告：`docs/features/reviews/FD-042-review-r1.md` 及同名 JSON；未重跑编译、测试或运行时验证，剩余风险保留。

## TODO

- 1.1–1.4 已分别完成并提交；后续独立审查通过后交付父分支并归档。
- 当前没有 material NEEDS_INPUT。中文 CLI 输出需要 `PYTHONIOENCODING=utf-8`；原始 show 编码失败已通过该环境变量恢复，不扩展本 FD 的编码修复范围。

## Sources

- Issue: REQ00008-fd-force-recovery
- `docs/requirements/REQ00008-fd-force-recovery/requirement.toml`（APPROVED，revision 5）及 requirement-plan.md、decision-log.md。
- `openspec/specs/fd-workflow/spec.md`、`docs/usage/aiw-fd.md`、`plugins/aiw-fd.py` 的 recover_worker/prepare_event/close_locked。
- Planner/Worker 会话：`fd042-host-20261009-a73c92`；设计 source event：`FD-042-000002-design-requested`。

**Completed:** 2026-10-09
