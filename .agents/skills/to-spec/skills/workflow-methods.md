# Shared workflow methods

Canonical methods for stage-specific use. The installer materializes this file
for entries that declare `skills/workflow-methods.md`. Read only the section
needed by the current role. This file grants no role, state, Git, runtime or
external publication authority; the active entry and shared contract own those.

## Splitting principles

Assess outcome size, difficulty and uncertainty before approval or handoff.
Split by one reviewable result, with acceptance, dependencies and non-goals;
do not split solely by files or technical layers. Preserve IDs and lineage.
Map each original outcome once, including retained/deferred scope and shared
safety or release dependencies. Splitting does not remove uncertainty or risk.

Issue owns business scope: use child Issues for independently decidable and
acceptable user outcomes. A child needs its own approval; parent approval is
not inherited. Planner owns engineering scope: use the FD's Work Items for
implementation steps within an approved outcome, not another execution plan.
Keep inseparable outcomes together and explain the choice. Do not create a
second Task or OpenSpec tasks checklist. Unresolved scope/acceptance decisions
are explicit gates, not ready work. Use the active stage's write authority.

## Specification writing

Use when settled decisions change a stable capability rule or observable
behavior, or for an explicitly requested spec maintenance action. Read only
relevant stable specs, confirmed Issue evidence, the FD and an explicitly
linked change when present. Write requirements and observable success,
failure and boundary scenarios; do not prescribe incidental implementation.

The FD owns options, engineering decisions and Work Items; stable specs own
capability behavior. Identify which requirements change and why. Keep current
and proposed behavior distinguishable: link a proposed rule to its pending FD
and reconcile wording at completion. Do not represent unfinished behavior as
implemented. Do not update stable specs for a routine internal refactor.

Do not reopen settled decisions or create missing policy. Record material
unknowns as `%% NEEDS_INPUT` and report BLOCKED/INCOMPLETE. Return affected
requirements and FD items to Planner for reconciliation; keep existing IDs and
completed evidence. Create an OpenSpec change only on explicit user request.
If one exists, keep its relevant artifacts coherent without making tasks.md a
second plan. Missing OpenSpec tooling blocks only that requested operation.
Completion requires rules/scenarios consistent with confirmed scope, a clear
proposed/implemented status and no unresolved acceptance ambiguity. Report
changed files, evidence and gates. Runtime checks follow the repository budget.

## Implementation review

Review actual changed source and relevant surrounding call paths, not only a
report or checkbox. Pin the implementation revision and comparison base; read
the originating FD/Issue, applicable stable specs and documented repo rules.
For documentation/configuration-only work, review those implementations and
contracts directly; do not fabricate code findings or demand irrelevant tests.

Assess these dimensions while keeping findings distinct:

- Correctness: trace control/data flow, edge and failure paths, compatibility,
  affected interfaces, permissions and data integrity where relevant. Report
  concrete defects and consequences rather than hypothetical checklists.
- Standards: cite actual repository rules. Apply the smell heuristics below
  only where useful; tooling-enforced style is not a manual review target.
- Spec: identify missing/partial requirements, wrong behavior or scope creep
  against current approved acceptance. A code style pass does not prove the
  requested behavior, and a spec pass does not prove sound implementation.
- Evidence: check commands actually run, revision coverage, skipped checks and
  remaining risks. A waived check is not passed and does not waive behavior.

Smell heuristics (judgment calls, never automatic blockers; repo rules prevail):
unclear names; duplicated logic; feature envy; data clumps; primitive obsession;
repeated type switches; shotgun surgery; divergent change; speculative
generality; message chains; middle-man delegation; refused inheritance.
Suggest a local correction only when it improves the changed scope; do not
turn review into unrelated refactoring. Severity follows actual impact.

Each actionable finding names the file/location, current requirement or rule,
observed defect, consequence and minimal correction. Separate hard violations
from heuristics; label uncertainty instead of inventing evidence. State what
was inspected and what remains unverified. The active entry owns report
format, verdict and events. Loading this method never invokes another Skill,
spawns extra reviewers, runs tests or expands review rounds by itself.
