# FD-074: Integrate bounded diagnosis and architecture methods into FD workflows

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户确认完整优化diagnosing-bugs、improve-codebase-architecture、codebase-design并按需接入FD/测试。诊断强制运行失败命令和重复实验与有限预算冲突；架构探索强制Agent/HTML/CDN/浏览器/grilling且临时目录规则冲突；设计入口最多两Agent但引用要求3+，术语/接口/测试规则过于绝对。

## Options and decision

采用三个独立入口加条件方法路由；保留诊断、架构探索、设计方法职责，不合并成每FD固定阶段。不新增CLI、状态、Agent或运行授权。静态默认，已有运行授权按范围使用，测试耗尽后的diagnosis-only不恢复运行或修复。

## Solution

诊断输出观察/原因分类/已确认根因或假设/下一步，允许静态推断并声明证据限制；运行复现仅具体授权且在预算内，不无限循环、不因无复现阻断代码分析。diagnosis-only为一次静态分析，输出候选修正但不实施，原始材料/版本/预算保留，缺陷仍阻断交付。

架构探索默认限定当前模块/FD，比较现状、局部改善及必要的大改，写简短Markdown建议到现有FD或对话；HTML仅请求时用.ai/tmp唯一离线文件，不含CDN、不自动打开。Agent/grilling/多方案设计按需要和授权选择，不强制；超范围重构记录后续需求，不自动实施/创建新Issue或FD。

设计方法允许项目术语，深度/删除测试/接缝是判断依据而非绝对法则；允许真实业务/协议隔离需要的单adapter接口，支持多角色接口与集成/端到端验证。Deepening不自动删测试，不因纯计算或有替身就强制合并。Design-it-twice默认单会话两方案；有明确独立分析授权时最多两次有界静态Agent，无权则不派发；不同模型/工具无关。

在FD planning按接口/状态/职责设计问题加载codebase-design，结构性障碍才加载improve-codebase-architecture；实现异常时加载diagnosing-bugs；测试原因不明/耗尽时仅按原预算静态诊断。方法不可用时记录限制，沿用现有方法，不伪称已执行。结果回当前FD/报告，不生成第二计划/新交接事件；fd-review仍拥有独立最终实现结论。

## Scope

三个Skill入口、相关supporting文件和README、FD planning/implementation/testing路由、fd-test/tdd条件指引、目录README、项目副本与stable FD规格。无脚本/产品代码、依赖、API/持久化/并发/部署修改。共享testing-methods现有预算不变，不重写历史报告或全局安装。

## Work items

- [x] 1.1 优化诊断方法和说明。Size: S; Difficulty: Low; Dependencies: none. Completion: 静态可用、运行可选、原因/证据分离、diagnosis-only有界且不修复。
- [x] 1.2 优化架构探索/设计入口与引用。Size: M; Difficulty: Medium; Dependencies: none. Completion: 无强制Agent/HTML/联网/浏览器或词汇禁令，两Agent上限统一，不自动删测试/扩大重构。
- [x] 1.3 条件接入FD/测试并同步说明/副本/规格。Size: S; Difficulty: Low; Dependencies: 1.1,1.2. Completion: 触发与返回位置明确、无新阶段、原测试预算不变。
- [x] 1.4 静态证据与独立评审交接。Size: S; Difficulty: Low; Dependencies: 1.3. Completion: 一批有界只读核查、双格式报告、真实独立Reviewer；交付由TODO跟踪。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- 静态诊断无须先运行，输出真实根因/假设/未知，观察结果与产品/测试/夹具/环境原因分离，诊断报告不等于已修复。
- diagnosis-only保留原预算且仅一次静态分析，不再修复/执行或通过其它Skill绕过，独立工作可继续但未解决验收缺陷不能交付。
- 架构探索默认局部Markdown，无自动Agent/HTML/CDN/打开浏览器或grilling；按需输出不越权，超范围重构仅建议。
- 设计方法不属于code-review专属；两方案可本地比较，授权Agent最多两次；入口/引用不矛盾，项目术语可用、接口/测试判断不绝对、不自动删测试。
- FD planning/implementation/testing、fd-test/tdd有条件路由，不新建必经阶段或事件，结果回当前FD/报告，测试和review权限不扩大。
- 源/项目改动副本一致、链接可用、只文档变化；未运行测试/模型/产品验证不记通过。

## Verification

- 独立 Reviewer fd074-reviewer-20261017-31a9：基线 0ea50f1、交接 HEAD 6f6018d、实现 fcc4b40；评审通过，无阻断。证据见 reviews/FD-074-review-r1.md/.json，来源 FD-074-000004-implementation-ready。一批静态检查确认 32 个实现文件、15 组副本、39 个链接，未改 shared testing-methods；未运行测试、编译、模型、网络或安装器。实际模型遵从仍未评估。

- Compile-only check and static review by default; record unrun tests explicitly.
- For explicitly requested test support (any wording/language), record suitable
  TDD scope, independent fd-test scope, command/revision authorization and finite
  budget here. Otherwise no testing stage is added. On exhausted unexpected-
  failure budget, stop affected test repair/execution, diagnose statically and
  continue independent work; unresolved acceptance defects still block delivery.

## Sources

- 用户确认推荐选项1：完整优化并接入流程。
- 三个源Skill及supporting文件；FD planning/implementation/testing、fd-test、tdd与testing-methods预算。
- openspec/specs/fd-workflow/spec.md现有有限测试/diagnosis-only规则。

- Worker: implementation fcc4b40; reports/FD-074-implementation-r1.md/.json. One bounded static batch passed: 32 Markdown/metadata files, 15 changed source/project pairs equal, 39 local links resolve. No executable changes: compile/tests/model/network/installer not run. Independent review and delivery pending.

## TODO

- [x] 完成1.1–1.4并取得独立实现评审。
- [x] 本地合并和归档：local-merge成功，专属worktree/分支清理完成，Complete归档成功。见reports/FD-074-delivery.md/.json。

- Issue: none

- 验证：纯Markdown无代码，不编译无关产品、不运行测试/模型/网络/脚本/格式化。一批只读检查源项目副本、链接、真实diff和路由/边界；情景为静态Bug、授权运行、测试预算耗尽、局部设计、超范围架构建议、独立多方案授权。文档内方法澄清无公共接口/持久化等设计，不额外设计评审，正常独立实现评审仍需执行。

**Completed:** 2026-10-16
