## Context

The implementation Task `schema-10-codex-pilot` now contains the opt-in Schema 10 Codex path. This separate trial Task must have exactly one OpenSpec WorkItem and an isolated worktree based on the implementation branch, so it can exercise that code without editing the primary workspace.

## Goals / Non-Goals

**Goals:**
- Verify one real Codex CLI turn, exact Session/Task binding, Provider usage persistence, report validation, and the frozen compile-only plan.
- Restrict intended edits to a new `docs/codex-pilot-trial.md` file in the trial worktree.
- Stop at Tester with the Task and worktree available for inspection.

**Non-Goals:**
- Do not run tests, accept or deliver the WorkItem, merge branches, or remove the trial worktree.
- Do not treat compile success as proof that the real Codex process or accounting path is correct.

## Decisions

- Use one checklist entry so activation sees exactly one mapped WorkItem.
- Use the existing `aiw wf pilot <task-id> status|activate|run` flow; status is read-only, activation is explicit, and run performs one bounded Codex invocation followed by report validation and the frozen compile stage.
- Bind the trial Task to `.wt/schema-10-codex-pilot-trial`, based on `feature/schema-10-codex-pilot` so the worktree includes the implemented pilot code.
- Require an explicit positive input/output Token budget from the existing configuration and stop if preflight does not verify the profile, plan, Task, or workspace. Do not infer missing limits.
- Preserve the trial file and all runtime evidence. Cleanup and delivery require separate decisions.

## Risks / Trade-offs

- The Codex CLI uses the operator's authenticated account and consumes its resources → run only after reviewing status and confirming the configured profile and budget.
- The local process has ordinary workspace and network access → use only the isolated disposable worktree; no OS sandbox is claimed.
- Provider usage or process completion can remain unknown → preserve the original request and do not redispatch it.
- The root primary worktree currently has unrelated dirty paths → do not stage or commit them; task creation is scoped to the new Task/change paths.

## Migration Plan

No production migration. Create the Task/change, create its worktree from the implementation branch, review read-only pilot status, then invoke activation/run only under the already granted pilot authorization. Leave the Task at Tester and retain the worktree for human review.

## Open Questions

- The exact usable Codex profile and configured input/output Token limits remain subject to the read-only pilot preflight; execution is blocked if they are absent or invalid.
