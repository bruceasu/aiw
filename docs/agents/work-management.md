# Work Management

New engineering work uses a numbered FD in `docs/features/FD-XXX_SLUG.md`.
The FD is the source of truth for decisions, numbered Work Items, status, and
Verification. `FEATURE_INDEX.md` is a lookup index. Issue records may link an
FD, but Task creation is optional.

PM, Planner, Worker, and independent Reviewer/Verifier exchange explicit FD
events. `aiw fd` writes small dispatch receipts under `.ai/fd/<id>/` and may
start the next role through a configured runner. An unknown in-flight result
must be reconciled before another writer starts. Use the primary workspace
for sequential work and an FD worktree for conflicting or parallel writes.

Existing `.ai/tasks/<task-id>/` records, legacy managed FDs, and Workflow Core
state remain readable. Do not rewrite their history during migration. New FDs
do not require `aiw wf` or Supervisor. OpenSpec owns stable specs; create a
change only when explicitly requested.

Read `skills/work-management.md` for the full contract and repository
instructions for validation, Git, and authorization limits.
