# FD-024: Refresh stale Tester handoff

**Status:** Complete
**Revision:** 11
**Priority:** Medium
**Test policy:** Independent
**Evidence policy:** Dual

## Problem

FD-020 的 Pending Test 交接仍绑定 revision 5，FD 已修订至 6。CLI 只有 Worker 刷新入口，无法正式恢复独立测试交接。

## Options and decision

新增 refresh-tester，沿用 refresh-worker 的锁、修订和取消/替代机制。手写回执会破坏流程证据；伪造测试拒绝会冒认 Tester/PM 结果，因此不采用。

## Solution

命令为 `aiw fd refresh-tester <id> --reason <text> --artifact <implementation-report>`。仅允许 active Pending Test、Independent 策略及最新 pending Tester 交接，且修订或规范化摘要已过期。要求保留已知 Worker session；报告必须是仓库内现存文件，Dual 策略检查 worker-report sidecar。递增 FD 修订，状态和 Work Items 不变；新建 PM 产生的 test-requested，保存当前实施报告、worker_session_ref、reason、supersedes 和原 implementation_event；旧事件 cancelled 并记录 superseded_by。正常 claim/dispatch、报告校验和独立身份约束继续有效。preparing 回执在全部写入完成前不可认领；失败恢复 FD、旧回执及索引，删除新回执。不允许替换 current、claimed、launching、dispatched、非 Tester 或归档交接。测试报告引用新 test-requested 作为 source_event/implementation_event，旧证据和授权不复用。

## Scope

修改 plugins/aiw-fd.py、CLI 帮助和补全、稳定规格、共享工作契约及 fd-workflow 技能说明。用户本轮明确授权新增 CLI 能力和回执恢复行为。保留既有命令；不修改网关实现、不代写 Tester/PM 结果、不执行测试、下载或 Git 写操作。FD-020 的恢复作为该命令后续使用，当前先交付能力。

## Work items

- [x] 1.1 实现刷新和后续报告兼容【小 / 中；无依赖】。完成标准：守卫、报告验证、Worker 身份、新旧回执关联、事务回退及 test-report-ready 接收新事件。
- [x] 1.2 更新命令发现和流程契约【小 / 低；依赖 1.1】。完成标准：帮助、补全、规格、技能和共享契约说明命令及授权边界。
- [x] 1.3 编译与实施证据【小 / 低；依赖 1.1、1.2】。完成标准：一次离线无产物编译、一次静态检查、中文 Markdown/JSON 报告及正式交接。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- 过期 pending Tester 事件可恢复到当前 FD，新事件指向当前报告且保留 Worker 身份；旧事件取消并双向关联。
- 状态及工作项保持，修订递增；current、错误状态/角色、已认领或在途、缺失报告/Worker 身份均拒绝。
- 写入失败不留下可认领孤儿，恢复原 FD、回执和索引。
- 独立 Tester 可认领新事件并按当前修订提交报告；不得复用旧测试授权或冒认 Worker 完成。

## TODO

- 三项工作项已完成；第 1 轮独立测试和 PM 决策已记录。第 2 轮仅修复证据可读性，须重新交接独立 Tester 和 Reviewer；任何测试命令仍需当前修订的精确授权。

## Verification

- 静态追踪刷新、claim、test-report-ready 和报告身份/修订约束。运行一次离线 compile-only，输出到空设备；Python 使用内存 compile，不生成 pyc。
- 测试、服务、发布构建、网络及 Git 写操作未授权，不运行；失败回退和正常测试闭环需后续独立验证。
- 第 1 轮独立 Reviewer 审查：`docs/features/reviews/FD-024-review-r1.md`，结论为 `changes-requested`。主要实现与授权链静态吻合；FD 尾部和 Worker 报告的字面问号须恢复为可读中文。Tester 9/14 场景覆盖，PM 已批准 64.3% 与分支未测例外；五项场景仍无运行证据。
- 第 2 轮独立 Reviewer 审查：`docs/features/reviews/FD-024-review-r2.md`，结论为 `verification-passed`。乱码已修复；当前授权命令的四项行为测试全部通过。Tester 场景覆盖仍为 9/14，PM 对 64.3% 和分支未测再次记录例外；五项未运行行为经静态追踪，保留运行风险。

## Sources

- Issue: none

- 用户要求直接增加正式 Tester 交接刷新能力。
- openspec/specs/fd-workflow/spec.md
- docs/features/FD-020_AGENT_GATEWAY_REQUEST_OBSERVABILITY.md

## 实施记录

- 首轮 Worker session：`fd024-host-20261004-c39a71`；来源事件：`FD-024-000003-design-ready`。
- 已静态追踪 `refresh_tester`、`claim`、`test-report-ready` 的状态、身份和修订约束；正式交接由 `implementation-ready` 事件记录。
- 首轮报告记录 Python 内存编译及 `python scripts/compile.py` 返回码为 0；没有可核实的 Go 编译结果。未运行测试或发布构建。
- 已做一次静态差异检查；未运行格式化、lint、vet、网络命令或 Git 写操作。
- 首轮实施报告为 `docs/features/reports/FD-024-implementation-r1.md` 及同名 JSON；独立 Tester 和 PM 的后续结果见本 FD Verification。
- 第 1 轮 Reviewer 指出首轮文档含字面问号；第 2 轮 Worker 修复 FD 和实施报告的人类可读性，来源事件为 `FD-024-000007-changes-requested`。行为实现未因该修复改变。

**Completed:** 2026-10-04
