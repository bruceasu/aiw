# FD-067: Improve test evidence and orchestrate optional with test workflow

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem

当前 TDD 缺少有效红灯和预期结果依据的操作标准；fd-test 报告偏向数量与结果，且禁止 Reviewer 读取，难以把已确认缺陷反馈到同一实现链。用户希望保留默认无测试流程，只用自然语言 `with test` 选择核心规则 TDD 与独立验收测试，并在反复失败后自动降级为静态诊断，继续不依赖该问题的工作。

## Options and decision

| 方案 | 决定 | 理由 |
| --- | --- | --- |
| 强制所有 FD 测试并新增 Tester 状态 | 不采用 | 增加默认成本，破坏现有路由。 |
| 只改善两个入口的描述 | 不采用 | 不能解决证据质量、失败预算与反馈隔离。 |
| 共享测试方法，按需 with test 编排 | 采用 | 一个用户选项，复用现有 FD 角色和报告，不增加 CLI 参数或状态。 |

测试支持按用户语义识别：`with test`、`带测试`、`需要测试支持` 或其它语言同义表达均可，不做固定关键词匹配；否定、只生成用例、不运行或不用 TDD 等更具体限制优先。明确测试子流程请求仍遵守仓库的精确命令、版本、风险和次数授权；不是任意测试、网络或 Go 测试限制的豁免。TDD 正常 Red/Green 阶段与非预期失败修复预算分开：每个预先限定的行为循环需要授权，预期 Red 不是缺陷重试，不得以新场景或新会话重置失败预算。默认首次执行、相关修改后一次复测；更严格的本地预算优先。

## Solution

新增可安装共享 testing-methods.md，定义规则来源、oracle、关键断言、可区分错误、风险选例、替身边界、版本与失败分类。TDD 使用有效 Red/Green 小步循环，fd-test 先独立推导场景再适配已有测试。优先重用现有 tests.md/mocking.md，避免无意义的引用层。

fd-workflow 新增按需 testing reference。Planner 在 FD Verification 中记录范围、授权、预算和证据；Worker 内执行合适的 TDD，独立测试子 Agent 在实现完成且 implementation-ready 发出前处理 fd-test，报告回到 Worker/PM，独立 Reviewer 可读取相关事实。子流程不认领 FD Tester 事件，不篡改现有 CLI 路由，也不复活旧 Tester 状态。

非预期失败达到预算或无安全修复时，停止本次测试相关修复/执行，进行有界静态诊断并记录 diagnosis-only。重启、Reviewer 修复或换命令不自动重置预算；继续其余独立工作。已确认验收缺陷、关键未知或未完成交付项仍阻止整 FD 就绪/合并；环境故障不能自动当作通过。只允许按现有规则的部分静态反馈，不伪造 full-FD implementation-ready；不满足就绪时最终保留现场并报告阻塞。显式终止优先，绝不自动清场。

无公共 CLI/API、数据、依赖、权限或并发实现变化，仅共享资源声明增加一项；无需额外独立设计评审，正常独立实现评审必做。

## Scope

包含：tdd/fd-test、fd-workflow 入口与角色/引用、fd-review、共享 work-management、报告模板、相关资源预算说明与 FD 模板、稳定规格、README、项目安装副本；安装器只增加共享文件白名单，FD 插件只同步内置默认模板文案，不改执行逻辑。

不包含：运行本次测试或宿主行为评估、引入测试框架/变异测试、改 CLI 状态/参数/事件、部署、用户级技能安装、历史 FD/报告迁移。三个手工静态案例检查规则可操作性，不宣称运行结果。

## Work items

- [x] 1.1 共享测试质量方法与 TDD 指令。Size: M; Difficulty: Medium; Dependencies: none. Completion: oracle/现状刻画/风险选例/有效红灯/替身限制有明确操作规则。
- [x] 1.2 fd-test 场景过程与双报告模板。Size: M; Difficulty: Medium; Dependencies: 1.1. Completion: 场景来源、输入/预期、关键断言、错误辨别力、版本、命令、失败分类及未执行项可审查。
- [x] 1.3 with test 编排与诊断降级。Size: M; Difficulty: Medium; Dependencies: 1.1,1.2. Completion: 默认不测试；显式选用可委派独立测试；预算耗尽静态诊断并继续独立工作；不绕过缺陷/就绪/交付门槛。
- [x] 1.4 安装资源、项目副本和规格一致性。Size: M; Difficulty: Medium; Dependencies: 1.3. Completion: 安装器最小声明、源与副本匹配、无旧报告禁读冲突；一次 compile-only 与有界静态检查、独立评审证据齐备。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- 未选择 with test 时沿用默认编译/静态检查与独立评审，不自动调用测试。
- `auto with test` 或同义中文只需一个选项，明确选择合适核心规则 TDD 与实现后独立 fd-test；不能被解析成新 CLI flag 或无限执行授权。
- 关键场景有独立规则依据和具体 oracle；不得从生产输出复制答案；现状刻画与需求正确性证据分开。
- TDD 区分有效目标 Red、测试/环境故障和未执行；没有实际 Red/Green 不得声称完成 TDD。
- fd-test 先固定需求场景再读必要适配资料，记录复用和独立性限制，不以会话分离代替 oracle 独立。
- 每个执行结果绑定实际实现与测试版本、脏工作区情况、精确命令、授权、退出码/原始证据；观察状态与原因分类分开。
- 持续失败预算跨恢复保持；耗尽后只诊断，不再自动修复或执行测试，继续不依赖失败的工作；不自动削弱断言/更改需求/标记测试通过。
- Reviewer 可利用相关可选测试证据；测试数量/覆盖率不自动成为门槛；确认验收缺陷或关键未知仍阻止正常交付。
- 新增资源可以独立安装，项目副本一致，旧 CLI Tester 与历史报告保留。

## Verification

- 本 FD 不选择 with test，不创建/运行产品测试、coverage、安装、网络、最终构建或宿主运行评估。
- 安装器声明和 FD 插件内置模板修改后，使用一次 Python 内存 compile() 检查两个文件，不执行插件、不保留产物。
- 一批静态检查源/副本/共享声明和 JSON 模板；静态推演计算边界、权限拒绝、环境失败三个案例，检查预算、证据与交付边界，不冒充测试通过。
- 独立 Reviewer 检查真实 diff、FD 验收与证据后正常本地合并归档。
- 实施报告：docs/features/reports/FD-067-implementation-r1.md 与同名 JSON。
- Python 两个文件一次内存 compile-only 通过；未运行测试、安装、网络或宿主行为评估。
- 静态脚本因范围映射和整模板比较假设中止，完成部分字节/JSON检查；没有声称整批通过，详见报告。
- 独立 Reviewer 静态评审通过：docs/features/reviews/FD-067-review-r1.md 与同名 JSON；评审版本 2fbcf6d，来源 FD-067-000004-implementation-ready。独立比较源/副本、资源声明和模板变更文案，未运行测试或重复编译。便携概述一项非阻塞措辞建议见报告。

## TODO

- [x] 完成 1.1–1.4、报告和独立评审。
- [x] 正常本地交付与归档。

## Sources

- Issue: none

- 用户本会话对测试质量、配合、with test 和反复失败降级的决定。
- src/skills/tdd/、src/skills/fd-test/、src/skills/fd-workflow/、src/skills/fd-review/。
- src/skills/work-management.md、docs/templates/FD_TEST_REPORT*、openspec/specs/fd-workflow/spec.md。
- FD-039 历史隔离决定由本 FD 在显式测试证据反馈范围内调整，历史文件不改。

## Delivery record

- 独立评审：FD-067-000005-verification-passed，Reviewer session fd067-reviewer-20261017-c47e；评审实现 2fbcf6d，证据提交 41440248639e0a0f14abfad7f853c9a68c5911f5。
- 交付前 FD revision 5，SHA256 174d24949cbfd9e43809b837f165e715ca6ab15be4ed13b8863971c4582a5fb6，与通过事件一致；父与 FD 工作区干净，评审后只有 FD/index/评审证据变化。
- 直接 local-merge 成功，无主动 sync、无合并冲突；develop squash 提交 5c37728，FD-Source 为上述证据提交。
- CLI 校验后删除 feature/FD-067 和 .wt/FD-067，保留 receipts；完整输出位于 .ai/fd/FD-067/local-merge.log。
- aiw fd close FD-067 Complete 成功。归档后才补写本段与交付 TODO，未提前改变通过事件绑定的正文。
- compile-only 通过及独立静态审查通过；未执行测试、安装/sync、覆盖率、网络、最终构建或宿主行为评估。便携操作概述的一处旧用语为非阻塞维护建议，见归档评审报告。

**Completed:** 2026-10-16
