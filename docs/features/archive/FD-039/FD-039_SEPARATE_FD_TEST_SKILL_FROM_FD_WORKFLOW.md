# FD-039: Separate FD Test Skill from FD Workflow

**Status:** Complete
**Revision:** 7
**Priority:** Medium
**Evidence policy:** Dual

## Problem

FD workflow 中 Tester 能稳定地产生测试场景并执行测试，但把 Tester 作为每个新 FD 的必经阶段，会额外引入测试覆盖率指标、PM 风险决策和其他评审 Agent 对测试报告的复核。这些步骤增加了交接和验收成本，也让实现验收依赖测试报告。

用户希望保留 Tester 的测试用例与执行能力，将其拆为独立的 `fd-test` Skill；默认 FD 实施由 Coder 做编译验证和静态分析，再由 Reviewer 按 FD 验收审查。

## Options and decision

| 方案 | 决定 | 理由 |
| --- | --- | --- |
| 保留 Tester 为 FD Workflow 的必经阶段 | 不采用 | 继续让测试指标、PM 风险投票和测试报告复核影响默认验收。 |
| 将测试能力放入单独的可选 `fd-test` Skill，并从新 FD 默认流程移除 Tester | 采用 | 保留稳定的测试生成与执行能力，同时让 FD 默认验收聚焦 Coder 编译/静态证据和 Reviewer 对 FD 的审查。 |
| 删除 `aiw fd` 的 Tester 事件、状态与恢复命令 | 不采用 | 会破坏既有 FD 和 CLI 合约；本次保留旧记录与旧 CLI 能力，不让新 FD 默认使用它们。 |

用户确认保留 CLI Tester 状态、事件与刷新能力，作为兼容/可选能力。

## Solution

新增独立 `skills/fd-test/SKILL.md`，按 FD 验收生成可观察测试场景，在用户明确要求执行时运行授权范围内的命令，并产出独立测试报告。它不认领或发出 FD Workflow 事件、不改变 FD 状态，也不要求 PM 或 Reviewer 评价测试报告。

移除 `fd-workflow`、`implement` 和 `fd-review` 默认路径中的 Tester 阶段、PM 测试报告决策和 Tester 报告审查要求。新 FD 不再默认声明 `Test policy: Independent`；Coder 在 `implementation-ready` 前完成编译验证和静态分析，随后直接进入独立 Reviewer 阶段。保留旧 FD 的 Tester 事件、授权记录、报告和 CLI 命令以维持兼容。

稳定规格与用法文档同步上述默认流程。可选 `fd-test` 报告保留场景、命令和执行结果等测试事实；这些指标只属于独立测试报告，不作为 FD Workflow 验收门槛。

## Scope

包含：独立 `fd-test` Skill 与配套报告模板；更新 FD Workflow、实现、Review、工作管理和使用文档；调整新 FD 模板及 CLI 新 FD 默认正文，使新 FD 直接路由 Reviewer；更新 `openspec/specs/fd-workflow/`。

不包含：删除既有 Tester CLI 事件/命令、迁移或改写历史 FD/报告、运行测试或覆盖率命令、改变旧 FD 的兼容路由、让 `fd-test` 自动参与 FD 验收。

## Work items

- [x] 1.1 新建独立 `fd-test` Skill 及报告模板。Size: M; Difficulty: Medium; Dependencies: none. Completion: Skill 描述触发、输入、测试场景生成、执行授权、输出与完成条件；测试报告可独立于 handoff 记录；测试代码仍放在根目录 `tests/`；Skill 不改变 FD 状态或派发 Reviewer。
- [x] 1.2 从默认 FD Workflow、Implement、Reviewer、PM 和工作管理规则中移除 Tester 验收链。Size: M; Difficulty: Medium; Dependencies: 1.1. Completion: 新的默认路径由 Worker 编译/静态检查后直接交 Reviewer；Reviewer 不读取或评价 Tester 报告，且无 PM 测试报告决策或风险评估 Agent 阶段。
- [x] 1.3 调整新 FD 默认模板与稳定规格。Size: S; Difficulty: Medium; Dependencies: 1.2. Completion: 新 FD 不含 `Test policy: Independent`；CLI 为此类新 FD 将 `implementation-ready` 路由到 Reviewer；旧 FD/Tester CLI 事件仍兼容。
- [x] 1.4 同步使用说明并静态核对所有默认路径。Size: S; Difficulty: Low; Dependencies: 1.1, 1.2, 1.3. Completion: 相关 Skill、模板、使用说明与稳定规格对默认流程和兼容边界一致。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- 新 FD 默认生命周期在 Worker/Coder 完成编译验证与静态分析后直接进入 Reviewer，不产生 Tester handoff，也不要求测试报告、覆盖率指标、PM 测试决策或风险评估报告。
- `fd-test` 可单独接收 FD 验收条件并生成测试场景、在授权范围内执行测试、写出事实性测试报告；该 Skill 不修改 FD 状态、不发出工作流事件，报告不成为 Reviewer 的输入或验收门槛。
- `fd-review` 与 FD Reviewer 角色只按 FD、实现差异及 Coder 报告检查验收；不评价测试报告或测试覆盖率。
- 新 FD 模板和 `aiw fd new` 默认正文不声明 `Test policy: Independent`；带该标记的既有 FD 仍可使用原 Tester CLI 流程。
- 测试权限边界、历史证据与既有 CLI Tester 合约保持一致；没有实际执行的检查如实记录为未运行。

## Verification

- 已静态复核 FD-039 变更集，核对新 FD 默认 Worker → Reviewer 路径、可选 $fd-test 隔离边界及旧 Tester CLI 兼容说明。
- Python 内存式 compile-only 检查 plugins/aiw-fd.py 通过；未生成文件产物。
- 未运行测试、coverage、最终构建、格式化或 lint。
- 实施报告：docs/features/reports/FD-039-implementation-r1.md 及同名 JSON sidecar。
- Work Items 1.1–1.4 均已完成。
- Reviewer 结论：changes-requested；报告：docs/features/reviews/FD-039-review-r1.md。

- Implementation follow-up: docs/features/reports/FD-039-implementation-r2.md
- Reviewer R2 结论：verification-passed；报告：docs/features/reviews/FD-039-review-r2.md。

## TODO

- [x] 1.3 调整新 FD 默认模板与稳定规格，同时保留旧 CLI Tester 兼容。
- [x] 1.4 同步使用说明并静态核对所有默认路径。

## Sources

- Issue: none
- `skills/fd-workflow/SKILL.md` 与 `skills/work-management.md`：当前 Tester、PM 风险决策和 Reviewer 交接规则。
- `skills/fd-review/SKILL.md`、`skills/implement/SKILL.md`、`plugins/aiw-fd.py`：当前 Reviewer 输入、实施后路由与新 FD 默认模板。
- `docs/features/TEST_REPORT_TEMPLATE.md`、`TEST_REPORT_DATA_TEMPLATE.json`：现有场景与覆盖证据格式。
- `openspec/specs/fd-workflow/spec.md`：稳定的 FD handoff 与验收要求。

**Completed:** 2026-10-08
**Disposition reason:** Done
