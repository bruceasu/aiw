---
name: fd-workflow
description: Create or refine a numbered FD; use when an implementation request asks for parallel work, isolation, a worktree, or wt, or asks to run the FD lifecycle through review and archive.
---

# Feature Design Workflow

Read `skills/work-management.md` and the repository instructions. Use this
Skill when the user wants to create, refine, review, or close a Feature Design.
New engineering work is FD-first. OpenSpec supplements the FD with stable
capability requirements when needed; it is not a second work plan. Use the
automatic lifecycle below when the user invokes `$fd-workflow auto`, asks to
run an FD through implementation and review, or makes an implementation request
that asks for parallel work, isolation, a worktree, or `wt`. A mere question
about those terms, or a design-only request, does not start implementation.

## Resolve the FD

Use an explicit FD ID first, then a unique FD linked from the current Issue or
conversation. Ask when several FDs match. For a new FD, use `aiw fd new
"<title>" [--issue <id>]`; this allocates an unused `FD-XXX` number, creates
the file from `docs/features/TEMPLATE.md`, updates the index, and records a
`design-requested` event. Do not create an AIW Task as a prerequisite or
convert the FD into a Task workflow.

## Design

Read the FD, approved Issue evidence when present, relevant code, prior FDs,
and stable specs. Write the problem, credible options, decision and reason,
scope, compatibility effects, numbered Work Items, acceptance, and a realistic
Verification plan. Use `%% NEEDS_INPUT: ...` for material unknowns. Do not
turn a routine technical choice with clear evidence into a human question.
Keep Work Item IDs stable after implementation starts.

### Work Item size and splitting

Before marking a design ready, assess every Work Item's size and difficulty.
Each item must be small, with low or medium difficulty, and deliver one result
that can be reviewed on its own. Aim for half a day or less per item; this is
a planning estimate, not a time guarantee.

Record the size, difficulty, dependencies, and completion criteria for each
item. Split items that combine several outcomes, mechanisms, or independent
decisions. Separate unresolved design work from implementation, and keep
shared logic in one owner item rather than duplicating it across the split.

Splitting does not remove technical uncertainty or security risk. If an item
depends on an unknown external contract, platform strategy, or security
design, record a `%% NEEDS_INPUT` Gate and resolve it before treating the
implementation as low or medium difficulty. Split again if the resolved work
still exceeds the size or difficulty target.

Preserve scope and acceptance coverage when splitting. Record how the old
items map to the new ones, update dependencies, TODO, and Verification, and
keep IDs stable after implementation starts. After implementation starts, add
new child IDs and explicitly cancel superseded unfinished items with a reason;
do not reuse IDs or erase completed work. Do not aim for a fixed item count.

If the human later changes an acceptance or verification decision, reconcile
the FD before requesting another review: record the decision and its scope,
update the affected Work Items and Verification plan, and remove superseded
`%% NEEDS_INPUT` gates. A waived check is recorded as not run with its residual
risk; it is never described as passed. Do not infer that waiving a check also
waives the required behavior. A previous review finding remains historical
evidence, not the current acceptance policy. Request a fresh independent review
against the revised FD and report.

`Design` becomes `Open` only when the FD contains all required sections,
numbered Work Items, and no material `%% NEEDS_INPUT` note. The Planner then
emits `design-ready` with `--producer planner --artifact <fd-path>` and the
current source event ID when one is dispatched. This records the handoff to
Worker. Do not emit an event merely because a heading or template exists.

Use `aiw fd list`, `show`, and `resume` for status and recovery. A pending event
can be handled by the named role in the current host after `aiw fd claim <id>
<event-id> --session <host-session-id>`. A dispatched event with unknown outcome
must be reconciled against its original session or log.

The `--session` value is an opaque ownership reference for this host session;
it need not be a platform-provided ID. Prefer a stable ID supplied by the host.
If none is available, the role handling the event must generate its own unique
reference (for example, `fd011-reviewer-20261001-8f3c2a`, with a freshly
generated suffix), keep it for the whole session, and reuse it for an
idempotent claim. Do not ask the human to
provide an agent session ID or borrow another role's reference. Check the
latest receipt immediately before claiming: only claim a pending event for
the current role, and never replace a launching or dispatched owner. Record
the chosen reference in the role's report with the source event ID.

For an active `Pending Verification` FD blocked by a stale, unclaimed handoff,
`aiw fd request-review <id> --reason "..."` creates a fresh Reviewer request
for the current FD. It does not fabricate Worker completion. An independent
Reviewer claims that event and reports findings. Do not use this recovery path
for an in-flight event or to bypass unresolved `%% NEEDS_INPUT` gates.

For an active `Open` or `In Progress` FD whose latest pending Worker handoff
no longer matches the FD revision or digest, PM uses `aiw fd refresh-worker
<id> --reason "..."`. It cancels the stale receipt and creates a new
`work-requested` Worker handoff bound to the current FD. Claim the new event
and cite it on `implementation-ready`. Never use this operation to replace
a claimed or in-flight handoff, or edit the old receipt by hand.

To continue an archived `Closed` or `Deferred` FD, use `aiw fd reopen <id>
--reason "..."`. It preserves the earlier disposition and evidence, returns
the FD as `In Progress`, and creates a new Worker handoff. Claim that exact
event before editing and cite it on `implementation-ready`. Keep previous
Work Item checkboxes as historical progress; revise them only when the renewed
scope warrants it. Use distinct filenames for new reports and reviews (for
example, a revision suffix), since `close` preserves earlier archived evidence
and rejects collisions. An archived `Complete` FD uses `request-review` instead.
If a reopen reason is unreadable or mistaken, correct it with `aiw fd reopen
<id> --reason "..." --correct-reason` before claiming the pending Worker
event. This managed correction keeps the event ID, records the previous value,
and updates the FD and receipt digest together. Never hand-edit a receipt.

## Auto: one FD through review, merge, and archive

Apply this procedure to one numbered FD when the user invokes
`$fd-workflow auto`, asks for the whole FD lifecycle, or requests implementation
with parallelism/isolation/worktree/`wt`. The host agent is PM, Planner, and
Worker; independent Tester and Reviewer stages use separate subagents. An
isolation request uses
the recorded FD branch and worktree. Under the shared work-management contract,
this request authorizes local commits, merge to the recorded parent after a
passed review, and FD archive after a successful merge. It does not authorize
tests, final builds, network access, permission escalation, push, deployment,
publishing, or worktree removal. Follow narrower repository rules.

1. **Preflight and resolve.** Read the repository instructions and this FD's
   Issue, stable specs, and current receipt. If `AIW_FD_ROLE_RUNNER` is set,
   stop before creating or claiming a handoff: the configured runner owns
   dispatch. For a new feature request, use `aiw fd new`, then claim its exact
   pending `design-requested` event as this host Session. For an existing FD,
   use its unique ID and inspect the latest receipt and recorded Session.
   Before claiming any existing Worker handoff, count prior Reviewer outcomes
   in the active implementation cycle. If the latest result is
   `changes-requested`, read its report and assess actionable repair. At
   three failed outcomes or with no actionable repair, leave the Worker
   handoff pending and report the human-directed recovery Gate. Apply this
   check again after interruption; do not reset the count on a new invocation.
   Claim only a pending event addressed to the current role. Never take over
   a foreign claim, restart a launching/dispatched role, synthesize missing
   receipts, or treat an old direct review as a current Reviewer event. If a
   handoff is stale or ambiguous, stop and report the recovery command and
   state. Route an existing `Design` FD to Planner, `Open` or `In Progress` to
   Worker, `Pending Test` to an independent Tester, `Pending Test Acceptance`
   to PM, `Pending Verification` to an independent Reviewer, and an active
   `Complete` FD with a current pass to close. Skip already completed stages;
   an archived Complete FD is already done. Before any implementation, capture
   the current branch, commit the ready FD plan on the parent branch, and make
   sure the parent workspace is clean. Do not mix unrelated changes into the FD
   plan commit; if other parent changes are uncommitted, resolve and commit
   them separately first. Ensure `.wt/` and `.ai/` are ignored by Git;
   `wt add` rejects missing rules before changing Git state. Then create the branch/worktree with
   `aiw wt add <id>` (`feature/<id>` and `.wt/<id>`). This is the
   default for every numbered FD, not only requests that mention isolation or
   parallel work. Immediately read
   `.ai/fd/<id>/workspace.json` and verify `fd_id`, `parent_branch`, `branch`,
   and `worktree` against the created worktree. This file records the exact
   parent for the later merge; stop if it is missing or inconsistent. Keep all
   Worker writes in that worktree.
2. **Plan and split when in Design.** As Planner, resolve routine choices from evidence and
   write Problem, options and decision, scope, ordered numbered Work Items,
   acceptance, TODO, Verification, and relevant spec updates. Split items by
   independently reviewable outcome and dependency order; keep IDs stable.
   Stop for a material `%% NEEDS_INPUT` decision. When ready, emit
   `design-ready --producer planner --artifact <fd-path> --source-event
   <claimed-design-event>`. Claim the resulting Worker handoff as this host.
3. **Implement when assigned Worker.** Complete all ready Work Items in order, making
   minimal code/doc changes. Update checkboxes, TODO, Verification, and
   remaining `%%` notes with actual evidence and skipped checks. Apply the
   repo's validation budget. For a Dual evidence FD, write a Chinese Markdown
   report and same-basename JSON using `docs/features/REPORT_DATA_TEMPLATE.json`;
   point `--artifact` to Markdown. If isolated, inspect the worktree status and
   commit only FD-scoped changes so the Reviewer has a stable diff. Use
   `aiw wt commit <id>` for focused local commits. If authorization or
   design is missing, stop with the FD active. Emit `implementation-ready
   --producer worker --artifact <implementation-report> --source-event
   <claimed-worker-event>` only after all scoped items and required evidence
   are resolved. New FDs have `**Test policy:** Independent`, so this event
   routes to Tester; older FDs without the marker retain the direct Reviewer
   route.
3a. **Independent Tester for Pending Test.** Spawn a separate Tester subagent
   with the pending event, FD acceptance, public contract, report template at
   `docs/features/TEST_REPORT_TEMPLATE.md`, and authorization limits. It must
   use a different session from Worker and Reviewer, claim the exact Tester
   event, derive black-box cases without reading implementation source, and
   write only assigned root `tests/` paths/report. Before any test or coverage command,
   Tester proposes the exact command, working directory, scope, expected
   duration, and side effects. The host acts as Planner: inspect the invoked
   test code and decide whether it is focused, offline, inspectable, and
   confined to assigned or temporary paths. If so, write a versioned
   authorization using `docs/features/TEST_AUTHORIZATION_TEMPLATE.md`, bound
   to the implementation event, FD revision/digest, and Tester session. For
   Dual evidence, add same-basename JSON from
   `docs/features/TEST_AUTHORIZATION_DATA_TEMPLATE.json`, keep the Markdown
   in Chinese, then
   allow only that exact command. If it may affect unrelated data, secrets,
   network/external services, dependencies, privileges, release artifacts, or
   has unknown effects, request explicit human approval first and record its
   affirmative reference using the authorization template's
   `approved:<source>:<id>` form. `denied` and `pending` never authorize a
   command. Do not infer approval from the auto request or Tester handoff.
   An unapproved Tester may report unrun tests honestly and
   must distinguish prepared from executed cases. The report inventories
   each distinct observable acceptance scenario rather than treating a broad
   numbered item as one case. It maps executed, partial, and uncovered
   behavior separately, and reports both coverage measures, raw evidence or unavailable reasons,
   exact commands, and residual risk. For Dual evidence, use a Chinese Markdown
   report and `docs/features/TEST_REPORT_DATA_TEMPLATE.json`; put every
   distinct scenario in JSON `data.scenarios` and keep counts consistent.
   Then emit `test-report-ready --producer
   tester --artifact <report> --source-event <claimed-test-event>`.
3b. **PM Test Report Decision.** Read the Tester report and actual evidence.
   Write a versioned Chinese Markdown decision with a same-basename JSON using
   `docs/features/TEST_DECISION_TEMPLATE.md` and
   `docs/features/TEST_DECISION_DATA_TEMPLATE.json` for Dual evidence.
   Cite the Tester report, FD revision/digest, both coverage results, rationale,
   exceptions, residual risk, PM identity, and decision time. A failed
   executed behavior test cannot be accepted. Below 70% or unavailable
   coverage needs an explicit exception and risk. If acceptance is materially
   uncertain, stop for human input. Emit `test-accepted` to Reviewer or
   `test-rejected` to Worker with `--producer pm`, the decision artifact, and
   `--source-event <test-report-ready-event>`.
4. **Independent review when Pending Verification.** Count Reviewer outcome events for this FD's active
   implementation cycle, including earlier auto invocations; the limit is
   three, not three new attempts on every resume. Spawn one separate Reviewer
   subagent through the host's subagent capability. Give it the FD ID, exact
   pending Reviewer event ID, current FD path, Worker report, applicable
   `fd-review` Skill, and repository authorization limits. It must claim the
   event in its own Session, inspect the actual diff and evidence, write a
   review report, and emit `verification-passed` or `changes-requested` with
   `--source-event` pointing to that claim. The host must not write the
   review report, emit a Reviewer event, or substitute a same-session review.
   For an independent-policy FD, Reviewer inspects the Tester report, PM
   decision, and raw evidence. It respects PM's recorded coverage exception
   while still reporting factual implementation or evidence defects.
   If subagents are unavailable, stop and leave the handoff pending.
5. **Repair, merge, and close.** Inspect the subagent's report and latest receipt;
   subagent prose alone is not a pass. On `changes-requested`, first count
   Reviewer outcomes and assess whether the findings have an actionable fix.
   If this is the third failed review or no fix is actionable, stop before
   claiming the new Worker handoff; leave it pending, preserve the report, and
   state the Gate. Otherwise claim that handoff, fix the concrete findings,
   update the FD/report, and emit `implementation-ready` again. On
   `verification-passed`, verify the latest receipt matches the current FD
   revision and content. Read `.ai/fd/<id>/workspace.json` and take its
   `parent_branch`, `branch`, and `worktree` as the merge coordinates. Confirm
   the parent branch is clean and still matches the recorded parent. Run
   `aiw wt local-merge <id>` for delivery. On a parent-side content conflict,
   the command aborts the failed parent merge and merges the parent into the FD
   worktree for resolution. Resolve and commit there, then rerun `local-merge`
   explicitly. If recovery fails, leave the FD active and worktree intact.
   After a successful merge, close with
   `aiw fd close <id> Complete` from the parent workspace and commit only the
   archive/index changes. A close rejection is a Gate, never a reason to edit
   a receipt or archive manually. Do not remove the worktree automatically.
   Report the merge result, archive path, review count, commands actually run,
   unrun checks, and residual risks.

Treat an interruption or a new user instruction as a checkpoint: inspect the
latest FD and receipt before continuing. Do not reset the three-review limit
on resume. Never collapse a failed review into a passed one to meet the cap.

## Close and archive

A passed Reviewer event is required before `aiw fd close <id> Complete`.
Deferred and Closed need `--reason "..."`. Native archives use
`docs/features/archive/<FD-ID>/<FD-ID>_SLUG.md`; matching reports and reviews
go into that FD directory. Archive changes the FD path and index; it does not
merge or publish code. Update a changelog only if the repository uses one. Do
not commit, push, or merge as a side effect of design or archive unless the
user authorized that action. The isolated implementation trigger authorizes
merge and archive only as described in the Auto procedure.

Read `references/portable-operations.md` for numbered FD conventions and
`references/templates.md` for layout examples. When they conflict with the
FD-first contract above, use this Skill and the current CLI behavior.
