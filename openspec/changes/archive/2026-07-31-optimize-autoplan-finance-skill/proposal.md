## Problem Statement

`docs/skills-review.md` 汇总了 31 个 Skill 的评审结果。问题不仅集中在 `autoplan-finance`，还涉及路由、任务生命周期、OpenSpec 映射、测试与 review 的 opt-in 边界、handoff、外部发布、文档技能和通用 Skill 质量规范。若只修一个 Skill，仓库仍会保留相互矛盾的行为约定。

## Solution

以整份评审文档为输入，分批优化全部已评审 Skill，并建立一致的 Skill 合同：明确触发条件、输入输出、完成标准、未决事项、路由关系、AIW/OpenSpec 边界和验证方式。按领域拆分实施清单，但由一个 AIW Task/OpenSpec change 统一跟踪，避免重新建立第二套任务系统。

## Scope

覆盖评审文档列出的全部 Skills：`autoplan-finance`、`tdd`、`business-review`、`code-review`、`codebase-design`、`diagnosing-bugs`、`domain-modeling`、`edit-article`、`eng-review-finance`、`grill-me`、`grill-with-docs`、`grilling`、`handoff`、`implement`、`improve-codebase-architecture`、`metrics-review`、`office-hours-finance`、`prototype`、`publish-github-issue`、`release-review`、`research`、`resolving-merge-conflicts`、`resume-ext`、`setup-matt-pocock-skills`、`teach`、`to-spec`、`to-tickets`、`triage`、`wayfinder`、`writing-great-skills`、`ask-matt`。

## User Stories

1. As a Skill user, I want every Skill to state its trigger, inputs, outputs, and completion criteria, so that I can predict what it will do.
2. As an engineer, I want related Skills to route consistently, so that planning, implementation, testing, review, and handoff do not duplicate or bypass lifecycle boundaries.
3. As a maintainer, I want all unresolved information represented consistently, so that missing evidence is visible instead of silently guessed.
4. As an AIW user, I want Task, OpenSpec, worktree, commit, and external publication responsibilities separated, so that a Skill does not perform unauthorized lifecycle actions.
5. As a reviewer, I want each Skill's verification and static acceptance criteria recorded, so that improvements can be checked without assuming runtime success.

## Implementation Decisions

- Group implementation by shared contracts and dependency order rather than editing 31 files ad hoc.
- Prioritize routing/lifecycle Skills (`ask-matt`, `to-spec`, `to-tickets`, `implement`, `handoff`, `triage`, `wayfinder`) before dependent domain/review Skills.
- Keep focused improvements scoped to the findings recorded in `docs/skills-review.md`; do not redesign unrelated Skill behavior.
- Use `%%` for unresolved decisions and preserve explicit opt-in for tests, builds, code review, TDD, external publication, and destructive lifecycle actions.
- Maintain compatibility aliases or migration notes where existing command names or artifact formats are retained.

## Testing Decisions

- Use static contract review as the default validation for each batch.
- Where a Skill already has a validator, align its scenarios with the revised contract; do not run validators unless explicitly authorized.
- Reuse existing Skill fixtures and repository conventions; add runtime tests only as explicitly scoped implementation tasks.

## Out of Scope

- Rewriting every Skill from scratch.
- Implementing product features described by finance review Skills.
- Automatically creating worktrees, commits, PRs, or external tracker records during specification or planning.

## Further Notes

%% NEEDS_INPUT: Confirm whether all 31 reviewed Skills should be delivered in one change lifecycle or split into independently archived batches after the first implementation slice.
%% NEEDS_INPUT: Preserve any user changes already present in the working tree; do not infer ownership from the review document alone.
