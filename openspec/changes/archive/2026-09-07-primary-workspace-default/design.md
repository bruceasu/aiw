## Context

AIW currently pre-populates every Task with a future feature branch and
`.wt/<task-id>` path. Engineering Skills then require that worktree and couple
Task completion to commit, merge, cleanup, and archive. This makes a simple
sequential change pay the same setup cost as parallel or disposable work.

Task metadata is tracked in Git and is written by both Go code and the Python
`aiw-wt` plugin. Existing Tasks may omit the new fields or contain stale paths,
so destructive behavior must not rely on path shape alone.

## Goals / Non-Goals

**Goals:**

- Bind new Tasks to the primary checkout and current branch by default.
- Make linked worktree isolation explicit and centrally managed by `aiw wt`.
- Represent workspace and delivery state explicitly and compatibly.
- Separate Task completion from optional Git delivery.
- Make handoff, cleanup, discard, repair, and archive fail safely.
- Keep native and delegated Task creation behavior consistent.

**Non-Goals:**

- Multiple worktrees for one Task.
- Automatic stash, commit, checkout, merge, push, or fetch.
- A repository-external registry or global current-Task pointer.
- Automatic source-change ownership inference or bulk metadata migration.
- Parent/child Task lifecycle aggregation.

## Decisions

### Use explicit workspace and delivery states

`workspace_kind` is `primary`, `isolated`, `unassigned`, or a derived
`unknown` compatibility state. `delivery` is `unmanaged`, `pending`, `merged`,
or `discarded`. Commit state remains a Git fact and is not duplicated.

New Tasks store `worktree = "."`, resolved relative to the AIW project root,
and bind both `branch` and `parent_branch` to the current branch. Absolute
machine-specific paths are not persisted.

Alternative: infer state from empty, dot, and `.wt/` paths. Rejected because
path inference is unsafe for cleanup and cannot distinguish stale metadata.

### Treat isolation as an explicit transition

`aiw wt add` is the single low-level transition from primary or unassigned to
isolated. It verifies that the Task artifacts are committed on the recorded
parent branch and that no branch or path conflicts exist. It does not fetch,
commit, stash, or overwrite. `task agent next --isolated` delegates to this
operation; `aiw new --isolated` is not introduced in this phase.

### Keep Task completion separate from Git delivery

Primary Tasks use unmanaged delivery and may archive after DONE without a Git
delivery claim. Isolated Tasks use pending delivery and may archive only after
Git ancestry proves their branch was merged and managed resources were cleaned
up. Checklist interpretation remains a Skill responsibility; the CLI gates on
Task status rather than parsing arbitrary Markdown.

### Preserve recoverability during cleanup

Ordinary `wt rm` removes only a verified linked worktree, preserves the branch,
and leaves the Task unassigned with pending delivery. `wt discard` is the only
command that deletes an unmerged experimental branch; it requires confirmation
and records CANCELLED/discarded. Repair is explicit and non-destructive.

### Infer legacy state without bulk migration

Missing `workspace_kind` is inferred at read time: empty is unassigned, the
project root is primary, a Git-registered linked worktree is isolated, and
anything else is unknown. Normal later writes may persist the inferred state.
Unknown state remains readable but blocks mutation and destructive operations.

## Risks / Trade-offs

- [Several Tasks share one primary workspace] -> Require explicit Task or
  Session binding when workspace mapping is not unique.
- [Primary branch changes after Task creation] -> Stop implementation on branch
  mismatch; never checkout automatically.
- [Task artifacts are not present in an isolation baseline] -> Make `wt add`
  validate the recorded parent branch and fail with corrective guidance.
- [Legacy scripts use destructive flags] -> Retain parsing with deprecation
  diagnostics and reject unsafe unmerged, primary, unknown, or unassigned
  targets.
- [Primary archive is mistaken for Git delivery] -> Record unmanaged delivery
  and emit an uncommitted-work warning without blocking archive.
- [Python metadata writes lose fields] -> Use a lossless writer for every AIW
  lifecycle field and preserve list metadata.

## Migration Plan

No bulk migration is performed. New Tasks receive the new fields. Existing
Tasks are inferred on read and normalized only on a later legitimate metadata
write. Existing isolated Tasks keep their worktree-based behavior. Legacy
cleanup and finalize flags remain temporarily available behind stricter safety
checks and deprecation messages.

Rollback restores the old creation defaults and Skill wording; metadata readers
continue ignoring unknown fields, so Tasks already containing the new fields
remain readable.

## Open Questions

None. The workflow decisions were confirmed through the grilling session.
