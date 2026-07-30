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

AIW Task, worktree, branch, Session, commit, synchronization, archive, merge,
and cleanup rules belong to `skills/work-management.md`; reviewed Skills must
reference that contract instead of redefining lifecycle rules. External
publication requires an explicit user request.

Static evidence and commands actually run are reportable. Runtime tests,
builds, formatters, linters, validators, broad reviews, and sibling Skills are
opt-in; skipped work must be stated as skipped.
