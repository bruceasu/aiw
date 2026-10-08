# FD-035: Adaptive risk assessment for FD test reports

**Status:** Complete
**Revision:** 14
**Priority:** Medium
**Test policy:** Independent
**Evidence policy:** Dual

## Problem

当前 FD 测试报告的 PM 决策固定要求三位独立 Agent 阅读同一报告、填写同一套字段，并以二票多数决定接纳风险或退回。三份意见缺少预设的互补视角；在证据清楚的常规报告中，重复评估增加时间和命令成本，未带来相应信息。用户要求常规报告用一份独立评估，只在失败、明显证据缺口或 PM 与首份意见分歧时追加评估；追加评估应有不同侧重点。

## Options and decision

- 保留所有报告固定三份：维持多数投票，但常规报告继续承担重复成本。
- 所有报告只用一份：成本最低，但争议风险只剩单一独立意见。
- 先取得一份完整独立评估，争议情况升级为三份并按多数决定；选择此项。单份路径的 PM 结果必须与该票一致。三份路径为评估者指定用户/验收影响、工程修复与证据、交付/运行风险三个侧重点，仍要求每人完整评估并独立投票。

## Solution

Tester 的事实报告和 Reviewer 的独立代码审查保持不变。PM 先派发评估者 A；存在失败的行为测试时直接派发 A、B、C。首份评估与 PM 拟定处置不一致时补派 B、C；PM 判定未覆盖或不可测量的证据缺口重大时，写明具体场景和影响并补派 B、C。覆盖率不作为自动升级阈值。追加评估者不得看到其他人的草稿或票，必须与 PM、Worker、Tester 和彼此使用不同 session。

新 PM 决策采用 `adaptive-v1` 证据策略，明确 `single` 或 `escalated` 模式、升级原因和覆盖缺口判断。单份模式仅在失败测试数为零、PM 认为剩余缺口可控且处置与唯一一票一致时有效；三份模式仍按至少两票 `accept-with-risk` 接纳。评估者 A 侧重用户和验收影响，B 侧重工程证据和修复，C 侧重交付和运行风险；每人仍填写完整风险字段并独立投票。CLI 核对模式、触发条件、来源/修订/摘要、各 session 与票数；Reviewer 与全部评估 session 隔离。既有三份式决策缺少新策略字段时仍沿旧校验读取，旧归档不改写。

## Scope

- 修改 FD PM 风险决策的评估数量、升级规则、互补视角、证据模板及 CLI 校验。
- 更新 `skills/work-management.md`、FD workflow/角色说明、使用文档和稳定 FD 规格。
- 保留 Tester 授权、覆盖率与失败事实、既有生命周期事件、Reviewer 独立性及历史归档证据；不改实际测试或自动发布行为。
- 补齐 Tester 报告中 S20–S25 六个未覆盖场景，并收集 FD CLI 的分支覆盖率证据。

## Work items

- [x] 1.1 定义单份/三份决策、升级与互补视角的证据契约，更新稳定规格和模板。Size: S; difficulty: Medium; dependencies: none; completion: 每种模式的条件、票数及历史证据边界可独立审查。
- [x] 1.2 调整 FD CLI 的 PM 决策校验和 Reviewer session 约束。Size: M; difficulty: Medium; dependencies: 1.1; completion: 合法单份和三份可路由，缺失/重复/过期证据、模式与票数不符被拒。
- [x] 1.3 更新共享工作契约、FD Auto、PM/Tester/Reviewer 指引与使用文档。Size: S; difficulty: Low; dependencies: 1.1, 1.2; completion: 常规单份与升级三份的操作、身份和风险保留规则一致。
- [x] 1.4 更新聚焦的 CLI 黑盒测试夹具，覆盖单份、多数、升级和历史三份兼容。Size: S; difficulty: Medium; dependencies: 1.2; completion: 测试代码置于根 `tests/`，用例能区分有效与无效决策证据。
- [x] 1.5 为 S20–S25 增加独立黑盒断言。Size: M; difficulty: Medium; dependencies: 1.2; completion: 六个场景各有可观察的 CLI 结果及拒绝路径无事件断言，S25 验证低场景覆盖率不自动触发升级；Tester 的执行结果另记在测试报告。
- [x] 1.6 增加隔离的 CLI 子进程分支覆盖率 runner。Size: S; difficulty: Medium; dependencies: 1.5; completion: runner 对被测 CLI 子进程插桩，临时数据目录结束后清理；Tester 的实际分支数、命令和限制另记在测试报告。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## TODO

- [x] 完成自适应决策契约、CLI 校验、流程说明和黑盒测试夹具。
- [x] 记录最终静态检查与无产物编译结果，提交 Worker 双份证据。
- [x] 接收独立 Tester 报告与后续 PM、Reviewer 决策。
- [x] 完成 S20–S25 的独立黑盒用例并补充测试报告。
- [x] 通过获批命令测量 CLI 子进程分支覆盖率并记录数据文件范围和结果。
- [x] 为 S20–S25 增加黑盒场景，并加入临时目录子进程分支覆盖率收集器。

## Acceptance

- 无失败且 PM 记录剩余证据缺口可控的 Tester 报告可凭一份有效独立评估作 PM 决策；处置必须与该票相同。
- 失败的行为测试自动要求三份；PM 与首份意见分歧或 PM 已记录明显证据缺口时也升级为三份。三份必须有效且来自不同 session；至少两票接纳才可交 Reviewer。
- 三份模式分别指定互补侧重点，同时每份都评估严重性、影响范围、修复与交付时间、理由、未知项及剩余风险；评估者不能相互读取草稿和票。
- 新编写的自适应决策显式记录 `adaptive-v1` 策略、模式、升级原因和覆盖缺口判断；无新策略字段的三份式决策保留兼容校验，以支持进行中的旧流程。
- 缺失、重复、过期、来源/修订/摘要不符、身份冲突或处置与有效票数不符时，CLI 不发出 PM 事件。
- Tester 的失败与未覆盖结果不得被改写为通过；Reviewer 与所有参与的评估者使用不同 session，并保留对未披露缺陷的审查权。
- 旧归档报告保持原样；已有三份评估决策仍可读取，且不需要回填新字段。
- S20–S25 均有独立黑盒测试证据；S25 的低覆盖率单份路径不会仅因覆盖率低而自动升级。
- 分支覆盖率来自实际被测 CLI 进程，报告文件范围、分支数、命中数和百分比；不得以测试夹具自身覆盖率替代。

## Verification

- 静态追踪 Tester 报告 → 风险评估 → PM 决策 → Reviewer claim 的来源、身份和风险字段；对照模板、Skill、使用文档与稳定规格。
- 1.1：已将单份/升级三份、PM 证据缺口判断、三种侧重点及旧三份证据边界写入稳定规格和双证据模板；尚未运行验证命令。
- 1.2：已静态追踪 PM 校验到事件 `assessor_session_refs` 和 Reviewer claim；现有 session 列表传递机制适用于一份或三份，CLI 增加模式、触发、侧重点和票数校验。编译检查留到实现结束。
- 1.3：已更新共享契约、两份 FD workflow Skill、PM/Tester/Reviewer 角色提示、两份 fd-review Skill 和用法文档，描述常规单份、争议三份及 Reviewer 对所有评估者的身份隔离；尚未运行命令验证。
- 1.4：已扩展根 `tests/test_fd032_risk_decision_blackbox.py` 的夹具与场景，覆盖单份接纳/退回、失败测试强制升级、互补视角、重大缺口、PM 分歧和旧三份兼容；测试尚未运行，不能视为通过。
- `git diff --check develop...HEAD` 通过；静态追踪了 PM 校验的模式/触发/票数、风险评估的来源和身份、Reviewer claim 中的评估者 session 列表，以及稳定规格、模板和 Skill 的一致性。
- `python -B -c "from pathlib import Path; files = ('plugins/aiw-fd.py', 'tests/test_fd032_risk_decision_blackbox.py'); [compile(Path(p).read_bytes(), p, 'exec') for p in files]; print('compile-only OK')"` 通过，未保留产物。
- 聚焦黑盒测试代码已准备在根 `tests/`；尚未运行测试、覆盖率、最终构建、lint 或网络命令，实际执行仍需独立 Tester 的精确授权。
- 独立 Tester 的两次聚焦执行分别有事前 Planner 授权：首轮 7 个方法中 1 个夹具准备失败；隔离夹具后唯一获批重跑 8/8 方法通过。25 个场景中 19 个有通过证据；S20–S25 未覆盖，业务代码分支覆盖率未测量。
- PM 在 `docs/features/reports/FD-035-test-decision-r1.md` 判定 S20–S25 的证据缺口重大，取得三份互补评估后以 3:0 票带风险接纳；未将缺口视为测试通过。独立 Reviewer `fd035-reviewer-20261008-01` 在 `docs/features/reviews/FD-035-review-r1.md` 静态审查当前实现与证据，结论 `verification-passed`，上述已披露限制保留。
- 修订 8：用户将原先已接纳的 S20–S25 与未测分支覆盖率指定为本轮补测范围。现有风险接纳仍是历史决策；六个场景不再作为本轮豁免项。先审查本次范围修订，再实施测试和覆盖率收集。
- 1.5：已为 S20–S25 增加独立 CLI 场景，其中拒绝路径核对决策事件未变化，S25 使用 76% 场景覆盖率验证单份路径。Tester 报告 r3 记录 25/25 场景有通过证据，14/14 个测试方法在获批修复重跑中通过；首轮 S20 夹具 claim 失败保留为历史事实。
- 1.6：已加入仅在测试子进程环境启用的 coverage.py runner，配置和原始数据写入系统临时目录，并从被测 CLI 收集分支计数。用户批准一次性临时安装后，Tester 获批运行 coverage 命令并报告 `plugins/aiw-fd.py` 210/548 分支（38.3%）；其余 338 个分支仍属未覆盖风险。
- 独立 Tester 的聚焦黑盒命令为 `python -B -m unittest tests.test_fd032_risk_decision_blackbox -v`；覆盖率命令由 `python -B -m tests.fd035_coverage` 运行同一用例并统计 CLI 子进程。两者均须版本化 Planner 授权后执行。
- Reviewer 第 2 轮：静态核对 `docs/features/reports/FD-035-test-report-r3.md`、三份风险评估、PM 决策及三份精确 Planner 授权；S20–S25、25/25 场景结果和 210/548 分支数据与记录一致。已更正上述 TODO 和 Verification 的过期未运行状态；结论及剩余覆盖风险见 `docs/features/reviews/FD-035-review-r2.md`。
%% RISK: “明显证据缺口”的严重性由 PM 记录判断，CLI 只能校验决策字段和模式一致性；独立 Reviewer 需检查该判断的证据依据。旧三份式决策保持可接受以兼容进行中的旧交接。

## Sources

- Issue: none
- 用户对常规单份、争议三份及互补视角的确认。
- 用户将 FD-035 未覆盖的六个场景及分支覆盖率测量加入本轮验收范围。
- `docs/features/archive/FD-032/FD-032_THREE_AGENT_RISK_DECISION_FOR_TESTER_REPORTS.md`
- `openspec/specs/fd-workflow/spec.md`
- `plugins/aiw-fd.py`、`tests/test_fd032_risk_decision_blackbox.py`
- `skills/work-management.md`、`skills/fd-workflow/SKILL.md`、`docs/usage/aiw-fd.md`

**Completed:** 2026-10-08
