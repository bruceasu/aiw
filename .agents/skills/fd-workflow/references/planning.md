# FD planning and readiness

Load for the Planner role, not for implementation or routine status questions.
Read only Splitting principles in `../skills/workflow-methods.md` when shaping
Work Items; load Specification writing when a stable rule actually changes.
The entry declares this installed shared resource. The FD is the sole plan.

## Prepare the design

Read confirmed intent/approved Issue, relevant existing behavior and stable
specs. State the problem, affected actors, scope/non-goals, credible options,
chosen approach and reasons, boundaries, compatibility, failure behavior,
risks and acceptance examples. Resolve important decisions before calling work
ready; routine choices with clear evidence do not need another confirmation.

Trace each acceptance condition to a Work Item and expected evidence. Split
into small low/medium-difficulty items under the entry's sizing rules. Record
dependencies and readiness; unresolved external/platform/security decisions
remain gated even if an item is small. Keep IDs stable after implementation
starts. Do not invent a second OpenSpec proposal/design/tasks set by default.

Borrow propose/apply principles: prepare coherent decisions and acceptance,
check prerequisites before implementation, and keep planning artifacts aligned
when facts change. This is a method, not an invocation of OpenSpec commands.
An explicitly requested change is optional supporting material, not the plan.

## Test support

For an explicit request for test support (any wording/language), load
`testing.md` and record the selected scope, suitable TDD slices, independent
acceptance round, authority and finite budget in FD Verification. Cases-only
or no-TDD limits prevail. No test request means no default testing stage.

## Specification collaboration

When confirmed capability behavior changes, load only Specification writing
from `../skills/workflow-methods.md`; record the affected stable requirements,
proposed wording/status and corresponding FD items. A standalone to-spec
request uses that same method; Planner need not invoke a second Skill or ask
for a second approval within already authorized FD scope. A design-only limit
still controls which files may change. No stable rule change means no forced
spec rewrite. Identify gaps rather than inventing behavior.

## Optional independent design review

Use an independent design review when the human requests it or the design
introduces material public-interface/compatibility, persistence/migration,
auth/permission or concurrency decisions. For simple internal or documentation
changes, record why no extra design review is needed. An unresolved policy or
high-risk authorization gate still needs the human; review does not waive it.

When authorized to progress through design, give one bounded independent
subagent the current FD, relevant specs/Issue, proposed diff and exact design
questions. It checks scope/acceptance coverage, option reasoning, boundaries,
compatibility, failure behavior, dependencies and unresolved risks. It reads
local evidence only, does not implement, execute tests or change workflow
state. Write a Chinese Markdown design-review report with a same-name JSON
sidecar using schema aiw.fd.evidence.v1, kind design-review, FD ID, actual
source event (or null with a reason), findings and a ready/changes-needed
recommendation. Store distinct filenames under docs/features/reviews/.

Planner incorporates findings and updates the FD; material unresolved findings
block design-ready. If no independent agent is available, report that gate for
a required design review rather than self-review. Use a focused recheck only
when a material fix requires it; do not create an automatic review loop.
The report is design evidence, never an implementation verification-passed
receipt. Do not dispatch fd-review against an unimplemented design. No new
role event type or runtime permission is introduced.

## Reconcile and hand off

Before design-ready, check required sections, complete outcome-to-item mapping,
clear acceptance/evidence, resolved material decisions and any required design
review findings. Emit only the entry's real Planner handoff. If intent or
acceptance changes later, reconcile affected FD/specs/items and the scoped
implementation before requesting fresh review; retain superseded decisions
as history and do not reuse IDs or treat skipped checks as passed.
