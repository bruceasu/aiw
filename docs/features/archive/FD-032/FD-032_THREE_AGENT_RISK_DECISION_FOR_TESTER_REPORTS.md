# FD-032: Three-agent risk decision for Tester reports

**Status:** Complete
**Revision:** 10
**Priority:** Medium
**Test policy:** Independent
**Evidence policy:** Dual

## Problem

当前 FD Tester 提交事实报告后，PM 接纳规则仍以“失败测试不得接纳”和 70% 覆盖率门槛为硬条件。即使缺口范围很小、修复会显著推迟交付，也会在 PM Gate 退回或等待人工判断。用户要求由三个独立子 Agent 先评估测试报告的严重性、影响范围、修复与交付时间，再由多数意见决定继续修改或带风险通过，减少流程中断。

## Options and decision

- 继续使用覆盖率与失败数硬门槛，仅让 PM 备注例外：无法满足“多数 Agent 同意即接纳”，仍需反复人工解除 Gate。
- 让 Tester 直接判定接纳：失去独立风险评估，且混淆测试事实与交付决策。
- Tester 保留完整事实报告；三个独立风险评估 Agent 各自形成可审计意见，PM 按至少两票的多数结果发出 `test-accepted` 或 `test-rejected`。选择此项。Reviewer 仍独立检查实现和证据，能够退回未评估的新缺陷或不实证据，不重新否决已明确接纳的同一风险。

## Solution

Tester 继续记录实际执行、失败、未覆盖场景、需求场景覆盖率与分支覆盖率，不以单一数值决定接纳。PM 在收到 `test-report-ready` 后，把同一 FD 修订、Tester 报告和原始证据分别交给三个独立子 Agent；彼此不共享草稿或投票。每个 Agent 写中文 Markdown 与同名 JSON 评估，列出严重性、影响范围、预计修复/交付时间、建议 `accept-with-risk` 或 `repair`、理由、剩余风险和未知项，绑定同一 Tester 事件与 FD 摘要。评估 session 互不相同，且不得等于 Worker、Tester 或 PM。若某 Agent 未完成或证据无效，PM 自动更换一个独立 Agent，直到有三份有效意见；不得补票或代写。

PM 决策引用三份评估与票数。至少两份建议 `accept-with-risk` 时必须发出 `test-accepted`，否则发出 `test-rejected`；失败测试或低/未知覆盖率本身不再构成自动拒绝。PM 必须保留原始测试事实、反对意见与明确风险，不能把带风险接纳写成测试通过。CLI 在发出 PM 事件前校验报告身份、来源、独立 session、三个有效投票及多数结果。新回执携带评估 session，后续 Reviewer 不得与评估者同 session。Reviewer 可退回投票证据不实、风险未披露或与已评估风险不同的缺陷，不能仅因多数票接纳的同一已知失败而重新要求修复。历史归档证据保持原样；新决策适用新规则。

## Scope

- 更新 `plugins/aiw-fd.py` 的 PM 决策证据校验与 Reviewer 身份约束，不增加新的生命周期事件类型。
- 新增风险评估双证据模板，更新 PM 决策模板、FD Skill、角色提示、用法与 `openspec/specs/fd-workflow/spec.md`。
- 保留 Tester 的执行授权、测试事实与场景统计校验；不自动运行额外测试、不取消独立 Reviewer 阶段、不改历史报告。

## Work items

- [x] 1.1 定义三个独立风险评估的证据格式与多数决策契约；更新模板和稳定规格。 Size: M; difficulty: Medium; dependencies: none; completion: 评估字段、身份、来源与票数规则可被独立审查。
- [x] 1.2 在 FD CLI 校验三份评估及 PM 多数结果，移除按失败数/70% 自动拒绝，保留 Tester 事实校验与 Reviewer 独立身份。 Size: M; difficulty: Medium; dependencies: 1.1; completion: 不足三份、重复 session、来源不符或票数不符被拒，合法多数意见可路由。
- [x] 1.3 更新 FD Auto/PM/Reviewer 指引和使用说明，使并行独立评估及自动替换无效评估成为常规操作。 Size: S; difficulty: Low; dependencies: 1.1, 1.2; completion: 文档不再引用 70% 接纳门槛或“失败测试必拒”。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- Tester 失败数、覆盖率和未运行项仍准确记录，不能把未通过测试写为通过；它们不自动决定 PM 结果。
- 三份评估分别引用同一当前 Tester 报告、事件、FD 修订和摘要，来自不同 session，并填写严重性、影响范围、修复/交付时间、理由与风险。
- 两票或三票 `accept-with-risk` 时 PM 接纳，即使有失败测试或覆盖率不足；不足两票时 PM 退回 Worker。PM 报告包含三份评估路径、票数、反对意见和剩余风险。
- 缺失、重复、过时、伪造来源或身份冲突的评估不能使 PM 发出决定；无效 Agent 可被自动替换，不要求用户仅为覆盖率门槛作决定。
- 独立 Reviewer 保持最终代码审查权，不得与 Worker、Tester 或三位评估者同 session；可对未评估缺陷和不实证据发出 `changes-requested`，但不重复否决已被多数票明确接纳的同一风险。
- 历史归档报告与既有 FD 读取兼容。

## Verification

- 默认静态审查与无产物 Python compile-only；独立 Tester 如需执行聚焦命令，须按当前 FD 修订获得精确授权。
- Work Item 1.1：静态核对风险评估模板、PM 决策模板与稳定规格的事件/摘要绑定、三票规则和历史证据保留语义；未执行命令验证。
- Work Item 1.2：沿 `test-report-ready` → PM 决策校验 → `test-accepted`/`test-rejected` → Reviewer claim 静态追踪事件、摘要、独立 session 和多数票；编译检查在实现结束后执行。
- Work Item 1.3：静态对照 Auto、PM、Tester、Reviewer 指引与用法中的三份评估、多数票、风险保留和原测试事实；未执行运行时检查。
- 审查边界复核：Reviewer 不会因已由有效多数票接纳的同一已知风险重复退回，仍核查未披露缺陷及证据真实性。
- `python -c "from pathlib import Path; p=Path('plugins/aiw-fd.py'); compile(p.read_bytes(), str(p), 'exec')"` 无产物编译通过；未运行测试、覆盖率、最终构建、lint 或格式化。
- 计划黑箱场景：多数接纳带失败/低覆盖率、少数接纳退回、缺失/重复/过时评估拒绝、Reviewer session 冲突拒绝、历史报告可读。
- 独立 Tester 第 1 轮夹具失败记录保留；第 2 轮经 Planner 授权的聚焦命令报告 3 个方法通过、13/13 可观察场景通过，分支覆盖率未测。三位独立评估者 3:0 同意带风险接纳；PM 发出 `FD-032-000009-test-accepted`。
- 独立 Reviewer 静态审查通过，报告 `docs/features/reviews/FD-032-review-r1.md`；未运行 Reviewer 侧测试，历史归档兼容性仅作静态检查。

## TODO

- [x] 独立 Tester 按当前修订准备黑箱证据，并在 Planner 精确授权后运行聚焦命令。
- [x] PM 按三份独立评估执行多数决策，再交独立 Reviewer。

## Sources

- Issue: none；用户直接要求更新 Tester 报告的接纳流程。
- `openspec/specs/fd-workflow/spec.md`
- `plugins/aiw-fd.py`
- `skills/work-management.md`、`skills/fd-workflow/SKILL.md`、`skills/fd-workflow/roles/pm.md`
- `docs/features/TEST_REPORT_DATA_TEMPLATE.json`、`docs/features/TEST_DECISION_DATA_TEMPLATE.json`

**Completed:** 2026-10-08
