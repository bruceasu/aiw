# Reviewed Skill Contract

All reviewed Skills MUST document their trigger and non-trigger conditions,
required inputs, outputs, completion criteria, unresolved-input behavior, and
verification scope.

Use `%% NEEDS_INPUT: <question or missing evidence>` for unknowns. Use an
explicit `BLOCKED` or `INCOMPLETE` status when an unknown prevents a valid
result. Never guess missing evidence or claim skipped sibling work completed.

Unless explicitly authorized by the Skill contract, a Skill is read-only: it
does not modify files, create or mutate AIW Tasks or OpenSpec changes, create
branches or worktrees, commit, publish externally, or run runtime validation.

## Managed Execution Vocabulary

| Term | Owner | Meaning |
| --- | --- | --- |
| Task | AIW durable metadata | Identity, workspace, branch, delivery summary, and Session mapping. |
| Work Item | Workflow Core | A typed execution unit mapped to a human checklist item. |
| Attempt | Workflow Core | One bounded execution run for a Work Item; it may reference one Session. |
| Gate | Workflow Core | An unresolved dependency, decision, authorization, validation, or delivery precondition. |
| Evidence | Workflow Core | A static review, authorized command result, approval, or manual record. |
| Specification | OpenSpec | Proposal, design, capability requirements, and human-authored checklist prose. |

Skills report changed artifacts, proposed Evidence, unresolved Gates, and a
recommended next action. They MUST NOT independently claim or release a Task
write lease, fabricate an Attempt outcome, set a Task display status, or alter
generated Workflow regions in OpenSpec artifacts.

AIW Task, worktree, branch, Session, commit, synchronization, archive, merge,
and cleanup rules belong to `skills/work-management.md`; reviewed Skills must
reference that contract instead of redefining lifecycle rules. External
publication requires an explicit user request.

Static evidence and commands actually run are reportable. Runtime tests,
builds, formatters, linters, validators, broad reviews, and sibling Skills are
opt-in; skipped work must be stated as skipped.
