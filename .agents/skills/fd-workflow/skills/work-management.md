# AIW Work Management Contract

Use this contract for AIW engineering Skills.

This file is the canonical shared contract. A Skill that needs these rules
should reference `skills/work-management.md` in its `SKILL.md` and keep its
own instructions focused on its domain. Do not copy lifecycle rules into each
Skill. `aiw skills install` and `sync` materialize this file beside the Skill
when that reference is present; treat the installed copy as generated output.
The same applies to `skills/reviewed-skill-contract.md`, so user-level
installations remain usable outside the AIW repository.

## New FD workflow

- FD-first is the normal path for new engineering work. OpenSpec supplements
  it with stable capability requirements; it does not replace the FD plan.
- A numbered file under `docs/features/FD-XXX_SLUG.md` is the source of truth
  for design, Work Items, status, Verification, and unresolved `%%` notes.
- `docs/features/FEATURE_INDEX.md` is an index. Rebuild it from FD files; do
  not treat it as a second state owner.
- An Issue may link to an FD. An AIW Task is not required for a new FD.
- `aiw fd` is the FD command surface. Read `aiw fd --help` before an unfamiliar
  or mutating operation. `aiw fd emit` records an explicit handoff; ordinary
  file saves and Git commits do not dispatch roles.
- `.ai/fd/<fd-id>/` contains dispatch receipts and logs. It does not own FD
  status or replace the Markdown plan. Never write a fake event or review.
- `docs/features/reports/` and `docs/features/reviews/` contain authored
  evidence while an FD is active. `aiw fd close` archives its design at
  `docs/features/archive/<FD-ID>/` and the FD's evidence in that directory's
  `reports/` and `reviews/` subdirectories.
  Under `**Evidence policy:** Dual`, each new report is Chinese Markdown for
  people plus same-basename JSON for CLI/AI. The Markdown contains one
  `<!-- aiw-data: <same-basename-json-file> -->` comment; JSON names its
  Markdown, FD, evidence kind, and source event. Emit the Markdown path as
  `--artifact`. Both files are archived together. Older Markdown-only
  evidence remains historical.
  Prior evidence stays archived if a completed FD is reopened for review.
- `aiw fd reopen <id> --reason <text>` returns an archived `Closed` or
  `Deferred` FD to active `In Progress` with a fresh Worker handoff. It keeps
  prior close metadata and evidence. Archived `Complete` uses `request-review`.
  New reports and reviews need filenames distinct from archived evidence.
- `aiw fd refresh-worker <id> --reason <text>` replaces a stale pending Worker
  handoff for an active `Open` or `In Progress` FD. It cancels the old receipt,
  records supersession, and creates a digest-bound `work-requested` event.
  Claimed or in-flight handoffs cannot be replaced.
- OpenSpec owns stable capability specs in `openspec/specs/`. Create an
  OpenSpec change only when the user explicitly asks for one.

## Roles and handoffs

PM creates and grooms an FD. Planner records options, decisions, acceptance,
and numbered Work Items. Worker implements all ready items in dependency order.
For FDs with `**Test policy:** Independent`, a separate Tester owns black-box
case authorship, execution when authorized, and the test report. PM records a
versioned acceptance or rejection with both coverage measures, exceptions,
and residual risk before Reviewer checks the diff and evidence in a third
session. Existing FDs without that policy retain the direct Reviewer route.
A failed review returns concrete findings to Worker. Human decisions and
authorization stop automatic dispatch until answered.
Planner reviews each Tester command and its invoked code before execution.
A Tester writes repository test code under the root `tests/` directory.
A focused, offline command confined to assigned or temporary paths may receive
a recorded, revision-bound low-risk approval without human review. Dangerous
or unclear effects require human approval and a recorded reference. Tester
must cite the authorization in any report of executed tests or measured
coverage; the handoff itself grants no permission.

Use an explicit stage event after the producing role has written its result.
The event must name an existing project-relative artifact. If the host has a
configured role runner, AIW starts the next role. Otherwise the event stays
pending for a Skill or later `aiw fd resume`. Before a host session writes for
a pending handoff, claim its exact event with `aiw fd claim <id> <event-id>
--session <host-session-id>`; include that event in `--source-event` at the
next handoff. Do not redispatch an unknown in-flight result; inspect the
recorded session or log first.

FD Work Item checkboxes record authored progress. Mark one complete only when
the required change and real evidence exist. Report checks that were not run.
Do not mark the FD Complete until all scoped items are resolved and the
Reviewer has recorded a passed result.

## Workspace and Git

Use the primary workspace for ordinary sequential work. When an implementation
request asks for parallel work or isolation, or says to use a worktree/`wt`, run
the FD in an isolated worktree. A question or design-only discussion that merely
mentions these terms does not start implementation.

Use `feature/<fd-id>` and `.wt/<fd-id>` for an isolated FD. Determine the
current branch before creation. Commit the FD plan before creating its worktree
so the plan is present there. `aiw fd worktree add` records `fd_id`,
`parent_branch`, `branch`, and `worktree` in `.ai/fd/<fd-id>/workspace.json`.
Read that file immediately after creation and verify all four values; use its
`parent_branch` as the sole merge target. Stop if the file is missing or does
not match the created worktree. Use `aiw fd worktree add/status` for FD
worktrees. Before using
`aiw wt` operations, check their current help and use them only if they accept
an FD ID; never invent or create a Task to satisfy a Task-only command. If no
FD-aware integration command is available, use scoped local Git operations for
the requested FD branch. Keep each FD's writing role in one workspace at a
time and do not mix unrelated files into its commits.

The user permits local Git commits for the new FD workflow. A Worker may make
a focused commit for an independently reviewable slice. A Reviewer checks a
specific commit or diff. When an isolation request asks for the complete
worktree lifecycle, it authorizes local commits, merging that FD branch into
its recorded parent after a passed review, and archiving the FD after the merge
succeeds. It does not authorize push, release, deployment, or worktree removal.
For other requests, commit does not authorize merge or archive. A merge conflict
or dirty parent workspace is a stop condition; preserve the worktree and active
FD while resolving it. Follow any narrower local rule.

## Validation

Static review is the default. After code edits, run one compile-only check
using a repository `compile*` script when available. Tests, final builds,
formatters, linters, vet, network calls, and deployment follow the repository's
authorization rules. Report actual commands, skipped checks, and risks.
