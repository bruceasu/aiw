# FD-073: Publish local Issue and FD projections without requiring OpenSpec changes

**Status:** Complete
**Revision:** 7
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户确认发布应以本地Issue或FD为来源，OpenSpec change可选。现Skill和helper只处理change级单块，映射依赖change目录；没有change无法发布，多FD更新同一远程Issue时可能互相覆盖。

## Options and decision

采用来源级受管块和稳定外部映射：默认Issue拥有远程映射，关联FD复用并更新各自块；无Issue的FD独立映射。保留旧OpenSpec helper及映射兼容；不新增远程同步引擎/CLI，不要求生成change。当前只实现定义和本地纯helper，不执行发布。

## Solution

发布显式且单向。先解析真实Issue/REQ/FD与来源关系/目标仓库，Issue与FD的已存在映射都必须检查；默认不同目标则阻断。用户明确要求分开跟踪时FD映射publication_mode=independent，允许与父Issue指向不同目标，后续保留该策略不重新聚合；未标识的旧不同目标不可猜测。Issue关联多个FD默认一个远程Issue，每来源独立受管块保护其他FD和人工内容。

新映射置于docs/external/github/issue/<canonical-id>.json或fd/<canonical-id>.json，独立于Issue/FD归档。映射沿用version1基本字段，增加source_kind/source_id、published_at/content_hash等可选溯源；source表示映射所有者，投影来源另行明确。关联FD默认只读写Issue拥有的映射，不重复建FD指针；首次由FD发起时先建Issue所有者的根摘要，再加入该FD块。ID来自实际解析记录，不猜ID/路径。旧change映射继续原位可读，只有明确来源关联且无冲突时才经用户授权迁移，不静默删除或替换。映射与远程目标必须校验；映射失效不得创建替代Issue。

新helper提供render_source_projection和来源专属markers；旧render_projection及两参数replace_managed_block行为可继续使用，旧路径保留原空白归一化，新来源路径保证块外字节不变。新替换按显式source_kind/source_id定位，拒绝重复/残缺/逆序或不匹配块；保护其他来源块及远程人工标题，旧openspec块仅明确指定迁移时转换。load/save加强基本映射检查并原子写入，避免部分映射。

所有远程写入前在映射旁原子登记<id>.pending.json，包含版本、operation、repository、映射所有者、投影来源、请求时间和生成内容hash，update记录远程Issue号。未完成凭据阻断新写入；create结果未知时先按仓库/来源标记和时间核查，不能因映射缺失再create。成功响应先回填目标到凭据，再保存映射，最后清除本操作凭据；任一步失败保留现有证据，不盲目重试。关联FD同时检查所有者与FD位置的未完成凭据。该凭据是单次发布恢复资料，不是本地Issue/FD状态机。远程整body更新不具备CAS保证，读取后再核对变化，无法确认则停止，明确并发写窗口限制。

不新增测试或改变产品CLI/API；helper内部接口以可选参数/新增函数向后兼容。新的映射约定和兼容行为先做独立设计反馈，解决明确问题后design-ready。只读授权及网络/写入沿用用户实际范围，不额外要求固定confirm文字；当前confirm仅授权本地实现，不授权实际发布。

## Scope

包含publish-github-issue的SKILL/README/UI metadata/纯helper及项目副本、目录说明、稳定发布规格和双格式证据。现有脚本测试保留不运行，不新增测试。无GitHub请求/远程Issue操作、安装、自动FD同步/关闭、双向导入、课程或其它Skill改造。

## Work items

- [x] 1.1 定义来源/映射/兼容合同并独立设计反馈。Size: S; Difficulty: Medium; Dependencies: none. Completion: 来源聚合、路径、冲突和恢复语义明确，设计发现解决。
- [x] 1.2 更新Skill与纯helper。Size: M; Difficulty: Medium; Dependencies: 1.1. Completion: Issue/FD主入口、来源块隔离、旧接口兼容、严格映射和不重放恢复。
- [x] 1.3 文档/metadata/项目副本与稳定规格。Size: S; Difficulty: Low; Dependencies: 1.2. Completion: 调用示例、边界、来源及链接一致，未声称全局安装。
- [x] 1.4 编译/静态证据并提交独立评审。Size: S; Difficulty: Low; Dependencies: 1.3. Completion: Python内存compile不产生pyc，一批静态核查和双格式报告，后续独立评审/交付见TODO。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- 不存在OpenSpec change时Issue/FD仍可发布；关联FD默认复用Issue远程目标，无Issue FD可独立，多个FD来源块独立保护。
- 先检查所有相关映射，冲突/非法/远程无效则阻断，不能悄悄新建；新路径跨归档稳定。
- 映射基本字段严格有效、写入原子，来源字段可追溯；旧version1/OpenSpec接口和有效旧映射保留兼容，不静默迁移。
- 更新只覆盖选中来源的完整唯一受管块，保留其他来源及人工文本，重复/残缺/逆序或未授权旧块迁移拒绝。
- 明确用户授权后才写远程，写后保存失败/结果未知先核查，不能重复create；本次不实际发布。
- 不自动批准/归档本地记录或关闭远程Issue；不把远程变成计划权威；静态结果与未执行测试清楚。

## Verification

- Reviewer r2: `reviews/FD-073-review-r2.md/.json`, verification-passed, source `FD-073-000006-implementation-ready`, HEAD `96caf9e`, repair `f01b62d`. R1 independently resolved via unified legacy/source boundaries; blocker feedback resolved. Static review only; no tests, helper execution, models or network. Local delivery/archive remains PM-owned.
- Reviewer r1: `reviews/FD-073-review-r1.md/.json`, changes-requested, source `FD-073-000004-implementation-ready`, HEAD `6b34ce3`. R1: legacy/source overlapping spans can delete another FD block during migration; repair and fresh independent review required. No runtime validation performed by Reviewer.
- Compile-only check and static review by default; record unrun tests explicitly.
- For explicitly requested test support (any wording/language), record suitable
  TDD scope, independent fd-test scope, command/revision authorization and finite
  budget here. Otherwise no testing stage is added. On exhausted unexpected-
  failure budget, stop affected test repair/execution, diagnose statically and
  continue independent work; unresolved acceptance defects still block delivery.

## Sources

- 用户confirm本会话Issue/FD关联与可选OpenSpec发布方案。
- src/skills/publish-github-issue，github-issues-management，issue-management；src/internal/issue/location.go。
- 稳定规格：openspec/specs/fd-workflow；仅扩展可选外部投影边界。

- Design: reviews/FD-073-design-r1.md/.json changes-needed D1/D2, reconciled in Solution; r2 ready. No implementation verdict implied.
- Worker: reports/FD-073-implementation-r1.md/.json, implementation c569141; in-memory compile passed for source/project helper and one bounded static batch passed (5 pairs, 31 links, legacy signatures and unchanged historical tests). No test/model/network/remote publication or installer executed. Independent implementation review pending.

- Worker repair r2: reports/FD-073-implementation-r2.md/.json addresses review R1 with unified legacy/source marker boundary validation; repair in-memory compile passed for both copies, no helper/runtime tests. Blocker reports/FD-073-blocker-legacy-boundary.md/.json awaits independent review.

## TODO

- [x] 设计和实现已完成；独立实现评审 r2 通过，R1 已修复并独立复核。
- [ ] 本地合并和归档。

- Issue: none

- 验证：仅Python helper改动，采用内存compile源/项目副本，不使用无关Go构建，不运行现有测试、模型、网络或验证脚本。一次有界静态核对调用、来源块边界、映射/错误路径、字节副本及链接；独立Reviewer按实际diff检查。
