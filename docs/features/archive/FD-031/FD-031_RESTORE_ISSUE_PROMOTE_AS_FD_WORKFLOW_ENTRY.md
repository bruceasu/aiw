# FD-031: Restore issue promote as FD workflow entry

**Status:** Complete
**Revision:** 16
**Priority:** Medium
**Test policy:** Independent
**Evidence policy:** Dual

## Problem

`aiw issue promote` 和 `aiw req promote` 当前没有命令实现，但 Issue 文档和 Requirement 稳定规格仍描述旧 Task 推广流程。用户无法用熟悉的 promote 入口把已批准 Issue 交给新的 FD workflow。

## Options and decision

- 移除 promote 相关的命令文档与规格：这会丢失已批准 Issue 转入工程工作的便捷入口。
- 恢复旧 Task 创建、handoff 和 `promotion` 状态迁移：与 FD 替代 Task workflow 的现行约定不符。
- 恢复 promote 命令并委托 `aiw fd new <Issue 标题> --issue <Issue ID>`：复用 FD 的编号、Issue 校验、重复关联保护和 Planner handoff；选择此项。

## Solution

在 Issue/Requirement 命令中提供 `promote <id>`。命令验证来源 Issue 已批准，然后通过当前 AIW CLI 执行 `fd new`，标题取自 Issue 记录，Issue ID 原样传递。FD CLI 继续负责创建文件与 Planner handoff。命令不得创建 Task、写旧 handoff，或修改 Requirement 元数据中的 `[promotion]` 状态和 `task_id`。重复关联由 FD 创建流程拒绝并返回其诊断。

恢复执行时，旧 Worker session 已停止而其交接仍为 `dispatched`。经用户明确授权，增加受限的 PM `recover-worker` 入口：必须精确指定旧事件、旧 session 和原因；记录取消与替代关系，再创建新的待认领 Worker 事件。普通 `resume` 与 `refresh-worker` 的既有限制保持不变。

## Scope

- 恢复 `aiw issue promote <id>` 和 `aiw req promote <id>` 兼容入口。
- 更新命令帮助、Issue/Requirement 使用说明和 Requirement 稳定规格。
- 保留既有 `[promotion]` 字段及其读写兼容性。
- 为本 FD 的中断恢复增加可审计的通用 Worker 交接入口。
- 不迁移历史记录，不创建 Task 或 OpenSpec change，不改变审批流程。

## Work items

- [x] 1.1 恢复 promote 命令：拒绝未批准 Issue，将已批准 Issue 标题与 ID 交给 `aiw fd new`，保留其错误与 handoff 输出；不更改 `[promotion]` 元数据。 Size: M; difficulty: Medium; dependencies: none.
- [x] 1.2 更新 Issue/Requirement 帮助、使用文档和稳定规格，使 promote 表示转入 FD workflow，并说明旧 `[promotion]` 字段不由此流程更新。 Size: S; difficulty: Low; dependencies: 1.1.
- [x] 1.3 增加精确匹配事件与 session 的 PM Worker 恢复命令，保留旧回执和替代记录，并更新 FD workflow 规格与用法。 Size: M; difficulty: Medium; dependencies: 1.1; completion: 新 pending Worker 事件可由独立 session 认领，旧事件可追溯且无法被普通刷新覆盖。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- `aiw issue promote <id>` 与 `aiw req promote <id>` 均支持同一入口。
- 未批准或不存在的 Issue 不会创建 FD。
- 已批准 Issue 使用其标题和 ID 调用 FD 创建流程；成功时由 FD workflow 创建编号 FD 并留下 Planner handoff。
- 已有关联、CLI 不可用或 FD 创建失败时返回错误，不创建 Task、不伪造成功，也不更新 `[promotion]` 状态。
- 帮助、使用文档和 Requirement 规格描述一致，且不再把 promote 指向 Task workflow。
- `recover-worker` 只接受活动 FD 最新、已派发的 Worker 事件和准确的旧 session；记录原因与双向替代关系，生成绑定当前 FD 内容的新 pending Worker 事件，错误时不留下可认领的孤儿事件。

## TODO

- [x] 修复 Issue 标题中引号等 `%q` 转义字符的读取，使 FD 获得原始标题。
- [x] 将 FD 创建失败的黑箱夹具改为真正阻止模板写入，并检查无 FD 和 handoff。
- [x] 独立 Tester 对修复版本重新获得精确授权并运行聚焦测试；PM 已接纳第 2 轮报告，未覆盖场景保持风险记录。
- [x] 新 Worker 认领 `FD-031-000008-work-requested`，提交本轮实现报告并交给独立 Tester。
- [x] 修复 Reviewer r1 指出的 `recover-worker` 索引解码异常回滚缺口；回滚恢复索引原始字节并明确报告各步骤失败。
- [x] 独立 Tester 对本轮修复重新取得精确授权，验证索引解码失败后的状态恢复；PM 已接纳第 3 轮报告。

## Verification

- 历史 Tester 第 1 轮执行 `python -B -m unittest tests.test_fd031_promote_blackbox -v`，8 例中 6 例通过、2 例失败；S06 为标题反转义缺失，S12 为无效的模板缺失注入。
- 本轮静态复核 `%q` 写入与 `strconv.Unquote` 读取闭环，并沿 `promoteIssue` → `aiw fd new` 检查标题传递；模板路径为目录时，FD 插件在生成 FD 和 handoff 之前写入失败。
- 本轮 compile-only 结果及未执行检查见 `docs/features/reports/FD-031-implementation-r2.md`。新版本测试尚未运行，旧 Tester 的 6/8 不能视为当前结果。
- `recover-worker` 的事件、session、状态、FD revision/digest 和原子回滚路径经静态检查；本轮未执行其失败注入测试。
- 恢复事件 `FD-031-000008-work-requested` 的 revision 8 与 FD SHA-256 一致，旧 `FD-031-000006-test-rejected` 已审计取消；新 Worker session `fd031-worker-repair-20261008-4e367cb0` 已认领。
- 静态复核 Issue 标题 `%q` 读写、promote 参数传递、模板写入失败夹具，以及 `recover-worker` 的匹配条件和失败回滚；`plugins/aiw-fd.py` 内存编译通过。证据见 `docs/features/reports/FD-031-implementation-r3.md`，新版黑箱测试仍待独立 Tester 授权执行。
- 独立 Tester 第 2 轮在修订 9 上执行获批聚焦命令，12/12 通过，场景覆盖 18/21（85.7%），分支覆盖率未测；PM 接纳 S13/S20/S21 未覆盖和分支覆盖率例外。见 `docs/features/reports/FD-031-test-report-r2.md` 与 `docs/features/reports/FD-031-test-decision-r2.md`。
- 独立 Reviewer 第 1 轮审查当前实现与证据，发现 `recover-worker` 索引读取的 `UnicodeDecodeError` 会绕过失败回滚，结论为需要修改。见 `docs/features/reviews/FD-031-review-r1.md`。
- 新 Worker 已认领 `FD-031-000012-changes-requested`，其 revision 12 与 FD SHA-256 一致。修复后，索引的原始字节在写入前保存；写入后的常规异常进入回滚，逐项恢复新事件、FD、旧回执和原索引，回滚失败带步骤名与异常类型报告。`plugins/aiw-fd.py` 内存编译通过；本轮没有运行测试，见 `docs/features/reports/FD-031-implementation-r4.md`。
- 独立 Tester 第 3 轮在修订 13 上执行获批聚焦命令，13/13 通过，场景覆盖 19/21（90.5%）；S21 索引解码失败后目标 FD、旧回执、索引与事件集合逐字节保持原值。PM 接纳 S13/S20 未覆盖与分支覆盖率未测例外。见 `docs/features/reports/FD-031-test-report-r3.md` 与 `docs/features/reports/FD-031-test-decision-r3.md`。
- 独立 Reviewer 第 2 轮复核回滚修复与当前证据，结论为通过验证。见 `docs/features/reviews/FD-031-review-r2.md`。Reviewer outcome 累计：1 次需要修改、1 次通过。

%% RISK: S13 和 S20 缺运行时覆盖；S21 只注入索引解码失败，其他磁盘故障与分支覆盖率仍未验证。

## Sources

- Issue: none
- `openspec/specs/requirement/spec.md`
- `docs/usage/aiw-issue.md`
- `docs/usage/aiw-requirement.md`
- `cmd/aiw-req/`
- `plugins/aiw-fd.py`

**Completed:** 2026-10-08
