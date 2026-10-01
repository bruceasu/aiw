# Numbered FD operations

The portable layout is `docs/features/TEMPLATE.md`,
`docs/features/FEATURE_INDEX.md`, `docs/features/FD-XXX_SLUG.md`, and
`docs/features/archive/<FD-ID>/<FD-ID>_SLUG.md`. The FD file owns status and
numbered Work Items. Active reports and reviews live under
`docs/features/reports/` and `docs/features/reviews/`; close moves the FD's
evidence into `archive/<FD-ID>/reports/` and `archive/<FD-ID>/reviews/`.
The index is rebuilt by `aiw fd` after stage changes. Do not create a second
status store or require a vendor-specific agent command.

## Create

Run `aiw fd new "title" [--issue ID]`. The command allocates the next number
across active and archived files, creates a Design FD from the template, and
records a Planner handoff. `--issue` requires an approved Issue record. Do not
invent an Issue ID or create an OpenSpec change unless requested.

## Explore and design

Read the FD, project guidance, relevant code, stable specs, and earlier FDs.
Compare meaningful options and record the chosen approach and reason. Use
`%% NEEDS_INPUT` for material gaps. Planner emits `design-ready` only after the
FD contains a solution, numbered Work Items, acceptance, and Verification.

`aiw fd list` shows the index and `aiw fd show FD-001` shows one FD and its
last handoff. If an event is pending, use the named role Skill or `aiw fd
resume`. If it is launching or dispatched, inspect the original process,
session, or log before trying to continue.

## Implement and review

Worker implements all ready items in order, updates the FD with real progress,
and emits `implementation-ready` with an implementation report. Independent
Reviewer emits `changes-requested` with findings or `verification-passed` with
its report. Pass `--source-event` when completing a dispatched handoff. A
business decision uses `needs-decision` and later `decision-recorded`.

Run only checks allowed by the repository. Local focused commits are allowed
under the project rules; commits do not authorize merge, push, or release.
An implementation request that asks for parallel work, isolation, a worktree,
or `wt` enters the isolated FD workflow; an informational or design-only
mention does not. Capture the current branch and commit the FD plan, then create
the worktree with `aiw fd worktree add FD-001`. The command records the FD ID,
parent branch, feature branch, and worktree path in
`.ai/fd/FD-001/workspace.json`. Read and verify that record immediately after
creation; use its parent branch as the merge target. Stop if the record is
missing or inconsistent. Use `aiw fd worktree status` to inspect Git worktrees.

Check `aiw help wt` before using its operations. Use them only if they accept
the FD ID; never create a Task to satisfy a Task-only command. If no FD-aware
merge command is available, use scoped local Git operations for the recorded
recorded feature branch and parent branch. Merge after a passed review; archive
the FD only after the merge succeeds. A conflict leaves the FD active and the
worktree intact.

## Auto

`$fd-workflow auto` is a host Skill operation, not an `aiw fd auto` CLI
command. Follow `../SKILL.md`: one numbered FD, exact handoff claims,
independent Reviewer subagents, and at most three Reviewer outcomes across
resumes of the same implementation cycle. With an isolation trigger, merge the
reviewed FD branch to its recorded parent before closing and archiving. Stop
without merge or archive when a review, parent state, or merge gate fails.

## Close

Use `aiw fd close FD-001 Complete` only after a passed review and, in isolated
mode, a successful merge to the recorded parent. Deferred and
Closed require `--reason "..."`. Close archives the file and updates the
index. It does not clean a worktree or deliver code. Update a project changelog
only when the repository uses one.
