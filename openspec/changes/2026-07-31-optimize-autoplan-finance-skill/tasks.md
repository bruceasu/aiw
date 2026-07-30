## 01 — 建立统一 Skill 合同与验证规则

**What to build:** 建立适用于全部已评审 Skill 的统一触发、输入输出、完成标准、未决事项、验证和 AIW/OpenSpec 边界。

**Prerequisites:** None — can start immediately

- [ ] 明确 Skill 的触发条件与非触发条件。
- [ ] 明确输入、输出、完成标准和 `%%` 未决事项格式。
- [ ] 明确文件、Task、OpenSpec、worktree、commit 和外部发布的授权边界。
- [ ] 复用 `skills/work-management.md`，不复制第二套生命周期规则。

## 02 — 修正核心路由与生命周期

**What to build:** 让 `ask-matt`、`to-spec`、`to-tickets`、`implement`、`handoff`、`triage` 和 `wayfinder` 形成可预测且不绕过 AIW/OpenSpec 的主流程。

**Prerequisites:** 01

- [ ] 咨询、探索、规格、任务拆分、实现和交接的入口与出口清晰。
- [ ] 咨询阶段不创建 Task、OpenSpec change 或修改文件。
- [ ] `/implement` 不自动触发 TDD、code review、测试或构建。
- [ ] handoff、Task、OpenSpec change、branch 和 worktree 的关系可追踪。

## 03 — 修正金融规划与治理 Skills

**What to build:** 统一 `autoplan-finance`、`office-hours-finance`、`business-review`、`metrics-review`、`eng-review-finance` 和 `release-review` 的规划、审查与发布边界。

**Prerequisites:** 01, 02

- [ ] 保留五阶段 review 顺序，并支持 full、focused 和 resume 模式。
- [ ] 分离 Plan Status 与 Release Readiness。
- [ ] 缺少证据时使用 `%%` 和阻塞状态，不补写业务规则或数据来源。
- [ ] release 默认是 `NOT YET REVIEWED`，只有满足触发条件和证据要求才允许 `GO`。
- [ ] 正确区分 proposal、design、capability spec、tasks 和 AIW Task metadata。

## 04 — 修正质量与工程 Skills

**What to build:** 规范 `tdd`、`code-review`、`codebase-design`、`improve-codebase-architecture`、`diagnosing-bugs` 和 `resolving-merge-conflicts` 的触发条件、范围和验证成本。

**Prerequisites:** 01, 02

- [ ] TDD 和 code review 均为显式 opt-in。
- [ ] 默认不运行测试、构建、格式化、完整 review 或扩大检查范围。
- [ ] 诊断、架构改进和实现保持清晰边界。
- [ ] 合并冲突处理保留现场，并遵循可恢复的生命周期规则。

## 05 — 修正 Session、原型和外部发布 Skills

**What to build:** 规范 `resume-ext`、`prototype`、`publish-github-issue` 和 `setup-matt-pocock-skills` 的恢复、原型交接、安装和外部发布行为。

**Prerequisites:** 01, 02

- [ ] Session 恢复结果可预测，并保留必要上下文。
- [ ] prototype 结果可通过 handoff 回流主流程。
- [ ] GitHub 或其他外部发布必须由用户显式请求。
- [ ] 安装、发布或恢复失败时不伪造成功状态。

## 06 — 修正通用工作 Skills

**What to build:** 统一 `domain-modeling`、`edit-article`、`grill-me`、`grill-with-docs`、`grilling`、`research`、`teach` 和 `writing-great-skills` 的触发、输出和上下文规范。

**Prerequisites:** 01

- [ ] 通用 Skill 不隐式创建 AIW Task 或 OpenSpec 生命周期资源。
- [ ] 外部资料、推断和未验证结论有明确区分。
- [ ] 每个 Skill 都有可检查的输出和完成标准。
- [ ] 保留用户明确要求的交互、教学或压力测试风格。

## 07 — 跨 Skill 一致性检查

**What to build:** 对照 `docs/skills-review.md` 检查所有 31 个 Skill，消除冲突、重复、死路由和错误完成声明。

**Prerequisites:** 02, 03, 04, 05, 06

- [ ] 31 个已评审 Skill 均有对应处理记录。
- [ ] 推荐路由的 Skill 均存在于当前 `skills` 目录。
- [ ] 没有 Skill 声称执行了实际未执行的 sibling Skill。
- [ ] 同一类生命周期动作在不同 Skill 中使用一致术语和边界。

## 08 — 完成文档、规格和验证记录

**What to build:** 完成 OpenSpec 规格、任务清单、Verification 和剩余风险记录，使后续实现可以从当前 change 直接接续。

**Prerequisites:** 07

- [ ] 更新受影响的稳定规格或设计说明。
- [ ] `tasks.md` 中所有实现与验证项目均可追踪。
- [ ] 未运行的测试、构建和验证命令明确记录。
- [ ] 剩余不确定性使用 `%%` 记录，不使用 `TODO` 掩盖缺失证据。
- [ ] 明确下一步是 `/implement`，且不暗示规划阶段已经完成实现。

## Verification

- [x] Task 06 general-purpose Skills now state lifecycle and evidence boundaries; interactive and teaching styles are preserved.

- [x] Task 05 Session, prototype, setup, and external-publication boundaries added; publication remains explicit opt-in.

- [x] Task 04 quality boundaries added to TDD, design, and diagnosis Skills; runtime checks remain opt-in.

- [x] Task 03 finance governance boundaries added to the five review Skills; Plan Status and Release Readiness remain separate.

- [x] Task 02 core routing boundaries updated for `ask-matt`, `to-tickets`, and `triage`; `to-spec`, `implement`, `handoff`, and `wayfinder` retain their managed-flow boundaries.

- [x] Task 01 contract defined in `skills/reviewed-skill-contract.md`; `autoplan-finance` now references it and reports completion and authorization boundaries.

- [ ] 8 个 ticket 按前置关系执行，未引入第二套任务追踪系统。
- [ ] 31 个 Skill 均被覆盖。
- [ ] 路由、生命周期、证据和验证边界跨 Skill 一致。
- [ ] 未执行未经授权的测试、构建、提交、worktree 创建或外部发布。
