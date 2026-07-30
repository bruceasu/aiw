# Design: Repository-wide Skill Review Remediation

## Workstreams

1. Core routing and lifecycle: `ask-matt`, `to-spec`, `to-tickets`, `implement`, `handoff`, `triage`, `wayfinder`.
2. Planning and finance governance: `autoplan-finance`, `office-hours-finance`, `business-review`, `metrics-review`, `eng-review-finance`, `release-review`.
3. Quality and engineering: `tdd`, `code-review`, `codebase-design`, `improve-codebase-architecture`, `diagnosing-bugs`, `resolving-merge-conflicts`.
4. Session, tools, and publishing: `resume-ext`, `prototype`, `publish-github-issue`, `setup-matt-pocock-skills`.
5. General-purpose Skills: `domain-modeling`, `edit-article`, `grill-me`, `grill-with-docs`, `grilling`, `research`, `teach`, `writing-great-skills`.

## Shared Contract

Every updated Skill should state trigger and non-trigger conditions, required inputs, output shape, completion criteria, unresolved-input convention, and whether it may mutate files or lifecycle state. Shared AIW/OpenSpec rules remain referenced from `skills/work-management.md` rather than copied into every Skill.

## Routing and Lifecycle Boundary

Consultation Skills do not create Tasks or modify files. Work-starting Skills use one AIW Task linked to one OpenSpec change. `implement` operates only in the matching Task worktree and does not automatically invoke TDD or code review. `handoff` preserves resumable context; `compact` remains session-local.

## Verification Boundary

Static review is the default. Runtime tests, builds, validators, external publication, and broad review scopes require explicit authorization. Each workstream records evidence and blockers; no Skill may claim a sibling Skill ran when it was unavailable or skipped.
