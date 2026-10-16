# FD-069: Document canonical skills and explain packaging and installation

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户需要明确 src/skills 顶层 Markdown、build.py 和 aiw skills install 的关系，并了解 teach、edit-article、research、prototype 的用途。总 README 虽有阶段性更新，仍缺完整可导航目录和准确的打包安装说明；各 Skill 缺独立面向使用者的说明。

## Options and decision

| 方案 | 决定 | 理由 |
| --- | --- | --- |
| 只补总 README | 不采用 | 用户仍无法就地了解各技能的输入、产物和限制。 |
| 修改打包安装逻辑并新增文档系统 | 不采用 | 当前请求是核实与文档更新，不应扩大到工具行为改造。 |
| 总览加 27 个独立 README，并同步项目副本 | 采用 | 覆盖全部 canonical Skills，区分人类说明、Agent 入口和共享方法，复用现有目录复制机制。 |

## Solution

基于实际源码说明：build.py skills 将整个 src/skills 复制到 AIW_INSTALL_DIR/skills，包含顶层 MD、各 README 和资源，合并复制不清理已删除文件；默认 AIW_INSTALL_DIR 是 C:\green\aiw，可由环境变量覆盖。此步骤不是安装到宿主项目的 .agents/skills。

aiw skills 默认从插件发行树的 skills 获取 canonical source，可由 AIW_SKILLS_SOURCE_ROOT 覆盖，不自动使用当前项目 src/skills。canonical list/--all 只发现直接子目录且含 SKILL.md 的条目。安装整体复制 Skill 目录，包括 README 与引用资源；只额外处理四种白名单共享 Markdown，且只扫描入口 SKILL.md 中的字面路径，不递归扫描 README 或 references 来发现共享依赖。顶层 README 不会作为 Skill 安装或自动注入 Agent。

重写 src/skills/README.md，提供目录导航、阶段选用、四个辅助技能的用途、源/发行/项目安装层次和真实限制。每个实际 canonical Skill 编写中文 README，描述用途、适用场景、输入与产物、调用例子、边界及相关本地材料。调用例子用 Easy English，明确是宿主 Skill 调用而非新 CLI 命令。README 是使用说明，SKILL.md 是行为合同，不改变技能授权、流程或输出定义。

仅同步已有项目安装中本次新增的 README；不运行 build、install 或 sync，不改 manifest 哈希、不清理发行/用户级目录。不为 references/shared Markdown 单独创造 Skill/README 入口。

纯文档变更，无 API、数据、依赖、部署、权限实现或并发变化；无需额外独立设计评审，正常独立实现评审必做。

## Scope

包含：src/skills/README.md、27 个包含 SKILL.md 的直接子目录中的独立 README.md、已有同名项目技能的 README 副本、FD 和双格式实施/独立评审证据。

不包含：改变 SKILL.md/安装器/build.py 行为、实际安装/打包/联网、增加测试、用户级副本、全局发行目录清理、已退役技能恢复。发现的覆盖复制不清旧文件和共享依赖白名单限制如实记录，不宣称已修复。

## Work items

- [x] 1.1 总览与打包安装说明。Size: M; Difficulty: Low; Dependencies: none. Completion: 全部 canonical Skills 有可用导航，顶层 MD/build/install/声明机制与源码一致，四个辅助技能用途明确。
- [x] 1.2 核心流程和工程技能独立说明。Size: M; Difficulty: Low; Dependencies: 1.1. Completion: 路由、Issue、领域建模、FD、测试、设计、诊断、规格与评审 README 描述真实输入/输出、调用与边界。
- [x] 1.3 辅助与集成技能独立说明。Size: M; Difficulty: Low; Dependencies: 1.1. Completion: 教学、写作、研究、原型、交接、初始化、发布和技能维护等其余 README 齐全，不添加原技能未有的默认权限。
- [x] 1.4 项目副本、一致性与交付证据。Size: M; Difficulty: Low; Dependencies: 1.2,1.3. Completion: 27 个 README 与总目录匹配，本地文档链接和项目副本一致，通过独立静态评审后交付归档。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- 总览覆盖且仅索引当前 27 个 canonical Skills，不把共享 MD、references、旧 implement/wayfinder/resume-ext 视为当前源目录技能。
- 每个技能具独立中文 README，含实际用途、适用场景、输入/输出、可理解的调用例子和重要边界；本地材料链接有效。
- 四个辅助技能的 README 区分教学、文章修改、资料调查和一次性验证，不误称它们为 FD 必经阶段或生产交付。
- build.py 覆盖复制、发行源位置、环境变量、安装目标、共享白名单与仅入口扫描限制准确；不夸大自动注入或安装效果。
- 安装器 whole-directory copy 会携带独立 README，但总 README/任意顶层 MD 不会因此成为独立可调用技能；README 不新增行为合同。
- 不修改产品源码、SKILL.md、manifest 或用户级技能；不运行构建/安装/测试，实际检查和剩余风险如实记录。

## Verification

- Compile-only check and static review by default; record unrun tests explicitly.
- For explicitly requested test support (any wording/language), record suitable
  TDD scope, independent fd-test scope, command/revision authorization and finite
  budget here. Otherwise no testing stage is added. On exhausted unexpected-
  failure budget, stop affected test repair/execution, diagnose statically and
  continue independent work; unresolved acceptance defects still block delivery.

- 本 FD 无可编译源码变更，编译检查不适用；不运行测试、build.py、安装/sync、lint、网络或宿主效果评估。
- 一批有界只读检查 canonical 覆盖数量、总览链接、各 README 的本地链接/必要信息、源与项目副本、变更范围；结合实际复制调用链静态检查安装结论。
- 独立 Reviewer 检查真实文档与对应技能合同，不用测试数量代替质量；正常通过后本地合并归档。
- 实施报告：docs/features/reports/FD-069-implementation-r1.md 与同名 JSON；27 个独立说明和总览已完成。
- 一批静态检查通过：27 技能目录覆盖一致，27 源／项目 README 字节一致，总览及各说明本地链接有效，实现范围仅 README。
- 核对发现 AIW_SKILLS_SOURCE_ROOT 不会改变共享白名单文件的插件树定位，已补文档说明；工具行为不变。
- 无可编译源码，编译不适用；未运行测试、build.py、install/sync、网络或宿主效果评估。
- 独立静态评审通过：docs/features/reviews/FD-069-review-r1.md 与同名 JSON。Reviewer fd069-reviewer-20261017-b74e 核对基线 6daa683 至 57fe00f 的文档、实际技能合同及打包安装调用链；独立确认 27 项目录、27 对副本、本地链接与仅 README 实现范围，无阻塞发现。

## TODO

- [x] 完成 1.1–1.4 与独立评审。
- [ ] 正常本地合并及归档。

## Sources

- Issue: none

- 本会话用户对顶层 Markdown、四个辅助技能及逐技能 README 的请求。
- build.py 的 copy_tree/install_skills；src/plugins/aiw-skills/aiw-skills.py 的 source_root/discover_skills/shared_skill_references/materialize_skill_source。
- src/skills/*/SKILL.md、四种共享合同/方法和当前总 README。
