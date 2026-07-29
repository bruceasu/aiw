---
name: ask-matt
description: Ask which engineering Skill or AIW/OpenSpec workflow fits the current situation.
disable-model-invocation: true
---

# Ask Matt

Use this router to choose a Skill flow. Read `skills/work-management.md` for
managed engineering work.

## Main Engineering Flow

1. Use `/grill-with-docs` when the idea still needs clarification.
2. Resolve or create the AIW Task before creating managed planning artifacts.
3. Use `/to-spec` to write the matching OpenSpec proposal, design, specs, and
   task outline.
4. Use `/to-tickets` for ordered implementation slices in `tasks.md`.
5. Use `/implement` for one selected item in the AIW-managed Task worktree.
6. After development, ask once whether the user wants one focused test command.
   Default to no test.
7. Use `/code-review` only when the user explicitly requests a review.
8. Mark the AIW Task done or archive it only when the user requests that
   lifecycle transition.

Keep checklist items under one AIW Task when they share a goal, branch,
worktree, delivery, and archive lifecycle. Create another AIW Task and OpenSpec
change only for an independent lifecycle, after user approval.

AIW owns Task, branch, worktree, Session, and handoff state. OpenSpec owns
proposal, design, specs, and the detailed checklist. AIW's automatic backend
may use an installed OpenSpec CLI.

## Execution Boundaries

- `/implement` does not automatically invoke `/tdd` or `/code-review`.
- Writing tests is allowed; running tests requires explicit user instruction.
- The main agent may use at most two bounded sub-agents.
- Sub-agents do not run tests, builds, network calls, permission escalation,
  commits, archive operations, or worktree operations.
- Worktrees are created and resolved through `aiw wt`, not raw Git.

## On-Ramps

- Incoming bugs or requests: `/triage`, then merge into the main flow.
- A difficult defect: diagnose statically first; use runtime reproduction only
  when the user explicitly authorizes it.
- A huge, uncertain effort: `/wayfinder` to resolve decisions, then `/to-spec`,
  `/to-tickets`, and `/implement`.
- Codebase health: `/improve-codebase-architecture`, then route an approved idea
  through the main flow.

## Crossing Sessions

Use `/handoff` when a fresh Thread needs the current context. In an AIW Session,
store the handoff in Session artifacts. When the user asks to continue in a new
Thread, use `aiw task agent next <task-id>` so AIW preserves Task, worktree,
Session, lease, and lineage.

Use `/compact` only to continue the same conversation at an intentional phase
boundary.

## Supporting Skills

- `/domain-modeling`: domain language and ADRs.
- `/codebase-design`: module boundaries, seams, and interfaces.
- `/tdd`: explicit test-first work where running tests has been requested.
- `/code-review`: explicit static review against a fixed point.
- `/publish-github-issue`: explicit external publication.
- `/teach`: multi-session learning.
- `/writing-great-skills`: Skill authoring guidance.

Run `/setup-matt-pocock-skills` before the first engineering flow when repository
work management has not been configured.
