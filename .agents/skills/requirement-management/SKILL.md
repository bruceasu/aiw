---
name: requirement-management
description: Start or continue one AIW Requirement conversation.
disable-model-invocation: true
---

# Requirement Management Conversation

Start or resume one Requirement with `aiw requirement chat [requirement-id]`.
Use for requirement discovery and human decisions, not implementation or release.
The Requirement domain builds context, selects methods, checks coverage, and
derives discussion advice. Session stores evidence; the CLI displays questions
and executes explicitly confirmed actions. Supporting Skills provide discussion
methods, not permission to write or approve.

## Supporting Skill Routing

Start with the built-in generic discovery baseline for every domain. Cover
roles, problem, current workflow, goals, scope, rules, exceptions, data,
permissions, dependencies, and acceptance. Unknown or non-financial domains
stay generic; do not force a finance workflow.

The host validates a source-backed domain/gap suggestion, then loads one primary
method from the allowlist below only for a finance domain. A terms_unclear
suggestion may add domain-modeling in Requirement mode. The human need not name
Skills, artifacts, paths, or CLI arguments.

| Requirement gap or phase | Support Skill | Expected artifact or result |
|---|---|---|
| Finance intake gap | `finance-requirement-intake` | Problem Brief |
| Finance value gap | `finance-value-assessment` | Business Case |
| Finance metric gap | `finance-metric-brief` | Metric Brief |
| Finance feasibility gap | `finance-engineering-options` | Engineering Options |
| Finance synthesis gap | `finance-requirement-synthesis` | Requirement Plan |
| Business terms, actors, entities, relationships, or domain boundaries are unclear or inconsistent | `domain-modeling` in Requirement mode | Confirmed vocabulary and domain boundaries returned to the Requirement Artifact |

The runtime reads the actual project
`.agents/skills/<allowed-name>/SKILL.md` and records its digest and load status.
It does not fall back to personal directories or canonical source files, and
does not recursively load linked resources. Never claim an unread method was
applied. Invalid suggestions explicitly degrade to the generic baseline;
an unavailable selected method blocks the turn.

Conflicts use the current method's deep-discovery questions, not an automatic
extra Skill call. Do not run every phase or invoke sibling Skills. Release
review and engineering design are separate workflows, not discovery methods.

Use `domain-modeling` before or alongside a phase Skill when the requirement's
business language is unstable. In Requirement mode, return the agreed
vocabulary, relationships, and boundaries to the Requirement Artifact; do not
create an ADR or write engineering-owned files directly.

After confirmed promotion, engineering planning owns OpenSpec Design Readiness
under the managed work contract. A discovery method does not write design.md
or declare implementation ready. The standalone agent may repair initial
artifacts as described below; this does not grant Design Readiness.

## Conversation contract

1. Use the current host snapshot: identity/revision, captured source bodies,
   source digests and load results, confirmed fragments, unconfirmed candidate,
   loaded methods, and the current answer. New requests have no invented ID.
   Only use background actually supplied; do not scan the repository or fetch
   linked resources. Missing required input blocks readiness.
2. The host makes two bounded calls: method-selection, then coverage-assessment.
   Follow that call's JSON contract, not a support Skill's Markdown template.
   Method selection must not use tools or prepare actions. Coverage must cite
   current sources and use resolved, needs_input, conflict, or not_applicable.
   Current answers and captured text are not automatically confirmed facts.
3. Ask up to three high-impact questions: conflicts first, then irreversible
   choices, correctness, scope, and goals. State the known evidence, missing
   decision, and impact. Give alternatives only with evidence and trade-offs.
   Do not repeat settled questions without changed evidence or a conflict.
4. Prepare only requested draft actions through `aiw requirement chat prepare`.
   The host displays the action, target, summary, and scope, then accepts
   `confirm` or `确认`. Never execute the durable operation yourself.
   Capture without --facts-json saves a draft only. With --facts-json, propose
   exact draft excerpts for separate human confirmation; the host verifies
   revision and draft digest. Never write confirmation or Session records.
5. After answers and confirmed actions, the host reloads sources and refreshes
   the phase. Empty questions or synthesis do not authorize approval. Invalid
   output is retained with diagnostics, not accepted or retried indefinitely.

For each turn, present only confirmed facts, source-backed conflicts, and up to
three questions that affect the next decision or action. Keep a short pointer
to earlier settled context instead of repeating the full history. Do not turn
an unconfirmed candidate into a confirmed fact while shortening the summary.

## Recovery and readiness

Formal Requirement files live in `docs/requirements/<id>/`, with `archive`
and `cancelled` folders under that root. Do not read or write the old root-level
`requirements` folder. Local counters, backups, and temporary files live in
`.ai/requirements`; the counter is not disposable cache. Session evidence keeps
its existing location. Use the host's actual source paths; moved sources require
fresh candidate review, not edits to historical evidence.

Session evidence binds actual inputs, methods, candidate output, and turn
numbers to the Requirement revision. Recovery reopens sources and checks
versions; stale candidates require reassessment. Failed or missing latest
records do not fall back to older success. Legacy sessions rebuild from formal
artifacts; explain unrecoverable discussion. Memory is not confirmation.

A Requirement Plan must state facts, assumptions, goals, scope, non-goals,
rules, acceptance examples, sources, and remaining decisions. Use actual body
evidence for plan_review; empty headings do not count. State explicitly when
there are no assumptions or remaining decisions. Facts, goals, scope, rules,
and acceptance statements need human-confirmed fragments for approval advice.

List proposed engineering postponements with the issue and reason in the Plan.
Only a matching human-confirmed excerpt accepts a postponement; never waive an
open business rule or conflict. The readiness report separates business gaps,
Plan gaps, proposed postponements, and confirmed postponements.

Incomplete drafts can still be captured. Chat approval advice is checked at
display and confirmation against current evidence. Direct approve CLI keeps
its existing contract; do not use it to bypass a blocked chat checkpoint.
Existing approvals are not revoked by new advice. Approval and promotion each
still need explicit human confirmation.

## Authorization boundary

The conversation may prepare artifact content and command parameters. Creating
a Requirement, capturing an artifact, recording a decision, and promotion each
require their own explicit human confirmation. Promotion creates or reuses one
Task, its handoff, and any initial OpenSpec planning artifacts; implementation
and release remain subsequent workflows.

## Post-promotion artifact repair

Run this step after user-confirmed promotion, before reporting the artifact
handoff complete, or when the user resumes that handoff. It is a standalone
agent step, never part of the host's method-selection or coverage JSON calls.
It completes the approved promotion; it does not require another confirmation.

If the shared generator reports `awaiting-agent` or `validating`, the approved
scope and Task remain reusable but the handoff is incomplete. Do not claim
`SPEC_DRAFTED`, rewrite formal artifacts directly, or rerun promotion. Read the
generated handoff and submit the Agent candidate through `aiw requirement
prepare-spec <requirement-id> --candidate <path>`; it rechecks the approved
sources, frozen targets, and ownership before writing. `--regenerate` starts a
new candidate request only when the user explicitly asks. Only an `accepted`
candidate can finish the promotion handoff.

Read `skills/work-management.md`. Use the linked Task/change, the current
approved Plan, its captured source bodies and confirmed amendments, and only
the relevant stable specs. Check source identity and revision before editing.
Missing sources, changed approval context, or unclear ownership mean
`INCOMPLETE` with a `%% NEEDS_INPUT` note, not permission to guess.
Carry confirmed goals, scope, rules, and acceptance examples from that Plan
into the OpenSpec artifacts. Do not ask the user to confirm them again; ask
only when new evidence creates a material conflict or a required decision is
absent.

- Check meaning, not only file names or headings: proposal must describe the
  actual goal and scope; design must reflect known decisions and real open
  engineering items; specs must state the actual rules and acceptance scenarios;
  tasks must map to that scope. Choose capabilities from the requirement and
  existing specs, not a fixed `requirement-management` name.
- If generated content is missing, generic, or about the generator instead of
  the approved feature, fill it in within the same change automatically.
  Replace or remove only positively identified generated placeholders; preserve
  authored content, checklist IDs, completion marks, and Core-owned regions.
  If a safe merge is unclear, report the conflict instead of overwriting.
- Keep edits limited to proposal, design, capability specs, and human-owned
  checklist prose. Do not alter approved sources, approval/Session records,
  handoff lineage, Task state, or gates. Do not rerun promote, start
  implementation, run tests, or perform Git operations as part of this repair.
- Do one static coverage and consistency check after repair. Record remaining
  engineering details in TODO/Verification and `%%` notes; do not invent
  decisions or declare Design Readiness. If content remains unsupported, report
  `INCOMPLETE`. Ask only for a real missing decision, scope/permission change,
  or conflict, not permission to replace a known template.
- Report the repaired files, static evidence, and skipped runtime checks.
  Distinguish artifact repair from an AIW generator fix; keep any generator
  issue open until its implementation is actually fixed.

## Output

Inside a host model call, return only the requested JSON object. The host
renders the questions, evidence, readiness report, and pending action.
For a standalone human-facing summary outside those calls, show only fields
that affect the next step. Keep CLI confirmation requirements explicit:

```markdown
## Requirement Conversation

- Phase and Requirement state:
- Relevant confirmed facts or conflict:
- Decision needed or next question (up to three):
- Next action, target, and write scope:
- CLI confirmation required: yes | no

%% NEEDS_INPUT: <blocking fact, when applicable>
```

## Completion

Finish the requested discussion slice with evidence-backed questions, a draft,
or a human decision checkpoint. Stop for missing required evidence or declined
actions; do not force capture or promotion. Report only completed actions.
Structural checks do not prove semantic quality. Human review assesses question
quality and Plan sufficiency; real-model calls and tests need separate approval.
