---
name: issue-management
description: Discover, refine, split, decide, or promote one AIW Issue covering a bug, feature, or modification. Use before Feature Design or Task implementation.
disable-model-invocation: true
---

# Issue Management

Use `aiw issue` for the Issue conversation. It currently delegates to the
compatible `aiw req` implementation, so new records still receive `REQ` IDs
and live under `docs/requirements/<id>/`. Do not migrate existing files or
rewrite Session evidence during a conversation.

An Issue records a problem or desired change, including bugs, features, and
modifications. Capture the actor, observed behavior, goal, scope, constraints,
rules, exceptions, acceptance examples, sources, and remaining decisions. A
large Issue may be split into smaller Issues with independent outcomes. Use
`aiw issue link-parent <child-id> <parent-id>` before child approval, and record
the relationship in both plans. Do not lose the original
scope or create duplicate Task work.

## Conversation

Use the host's current source snapshot, revisions, digests, confirmed facts,
and loaded methods. New requests have no invented ID. The host makes a method
selection call and a coverage call; return only the JSON shape requested for
each. The generic discovery baseline applies to every domain. Load only a
relevant support method; finance intake, value, metric, engineering options,
and synthesis skills are for finance Issues. Use `domain-modeling` when terms
or boundaries are unstable. An unread method was not applied.

Separate a missing business fact from an engineering strategy choice. For a
strategy choice, use evidence and the user's small preferences to select a
clearly better option and record the rationale. A bounded sub-agent may compare
options when delegation is authorized. Ask the human when options are close,
key information is absent, or the choice changes scope or risk. Ask at most
three high-impact questions per turn; do not repeat settled questions. Show
both sources for a real conflict.

Prepare drafts and durable actions through the supported CLI. In chat mode,
the host still requires a confirmation checkpoint for each durable action.
Do not treat model output, a captured draft, or Session memory as human
approval. Source or revision drift requires a fresh assessment. An invalid
model response is retained with diagnostics; it is not accepted by retrying
indefinitely.

## Readiness And Promotion

An Issue Plan states facts, assumptions, goals, scope, non-goals, rules,
acceptance examples, sources, and remaining decisions. Empty headings do not
prove readiness. Record proposed engineering postponements with reasons; a
missing business rule or conflict cannot be waived by the agent. Approval and
promotion remain separate human decisions.

Promotion creates or reuses one AIW Task and a source-backed handoff. Its
managed FD at `docs/features/<task-id>.md` then provides engineering decisions
and ordered work items. The Task lives at `.ai/tasks/<task-id>/`. Complete the
FD from the approved Issue sources, map its work items with `aiw wf plan`, and
update stable OpenSpec specs when behavior changes. A linked OpenSpec change
is optional. Do not start implementation or declare Design Readiness from
discovery alone.

For a post-promotion handoff, preserve approved sources, approval and Session
records, Task state, and authored FD content. Replace only known generated
placeholders. Record unsupported details as `%% NEEDS_INPUT`; ask only for a
material conflict or missing decision. Report repaired files and remaining
Gates without rerunning promotion or tests.
