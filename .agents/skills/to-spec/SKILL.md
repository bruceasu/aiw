---
name: to-spec
description: Turn the current conversation into an OpenSpec change — no interview, just synthesis of what you've already discussed.
disable-model-invocation: true
---

This skill takes the current conversation context and codebase understanding and produces a spec (you may know this document as a PRD). Do NOT interview the user — just synthesize what you already know.

Follow `skills/reviewed-skill-contract.md`. Read `skills/work-management.md`
and the repository's
`docs/agents/work-management.md` when present. Prefer AIW for Task lifecycle
and OpenSpec for specification artifacts. If AIW is unavailable, use a
standalone OpenSpec change. Do not create a parallel `.scratch` specification
or publish externally unless the user explicitly asks.

## Process

1. Resolve the active AIW Task using the shared work-management contract when
   AIW is available. If the work needs a new managed lifecycle, create it
   through AIW; use the automatic backend
   so an installed OpenSpec CLI can create the matching change artifacts. If
   several Tasks match, ask for the Task ID before writing.

   In managed mode, the AIW Task and OpenSpec change are one managed unit.
   Before continuing,
   establish and record the same normalized Task ID in both locations:
   `openspec/changes/<task-id>/task.toml` and the matching OpenSpec change
   directory. The `task.toml` must be the AIW lifecycle record, not an
   OpenSpec-only substitute, and must retain the Task's status, branch,
   worktree, parent branch, and Session fields when those fields exist. Do not proceed with
   an untracked OpenSpec directory or an AIW Task that has no matching change.

   In standalone mode, resolve or create the matching OpenSpec change directly;
   do not fabricate AIW lifecycle fields.

   Then resolve the matching OpenSpec change, explore only the relevant repo
   area, use the project's domain glossary, and respect applicable ADRs.

2. Sketch out the seams at which you're going to test the feature. Existing seams should be preferred to new ones. Use the highest seam possible. If new seams are needed, propose them at the highest point you can. The fewer seams across the codebase, the better - the ideal number is one.

Check with the user that these seams match their expectations.

3. Write or update the applicable OpenSpec artifacts using the template below.
   A successful managed `to-spec` run must leave this minimum artifact set in the
   matching change directory:

   - `task.toml` — AIW Task identity and lifecycle mapping;
   - `proposal.md` — motivation, scope, and user-facing solution;
   - `design.md` — durable implementation and architectural decisions;
   - `specs/<capability>/spec.md` — normative requirements and scenarios for
     each changed capability;
   - `tasks.md` — ordered implementation checklist, including TODO and
     Verification sections or equivalent records.

   Put motivation and scope in `proposal.md`, decisions in `design.md`,
   normative requirements in capability specs, and follow-up work in
   `tasks.md`. Do not publish to GitHub or GitLab as part of this Skill.

   Before reporting completion, verify statically that all required files
   exist, that the change directory name, `task.toml.id`, and AIW Task ID are
   identical in managed mode, and that every checklist item is actionable by
   `/implement`. In standalone mode, verify artifact consistency without
   requiring an AIW Task ID.
   If AIW or the automatic OpenSpec backend cannot create or link these
   records, continue with standalone OpenSpec artifacts when safe and report
   the missing lifecycle capability; do not create a parallel `.scratch` or
   ad-hoc task record.

4. In managed mode, keep the AIW Task linked to the change and synchronize only its coarse title,
   goal, and planning status. Do not create the implementation worktree during
   specification work. The resulting `tasks.md` is the checklist source that
   `/implement` will resolve; do not leave implementation work only in the
   proposal or design.

   Once the required artifacts and ID checks pass, commit the specification
   artifacts on the current branch before handing the Task to `/implement`.
   The later AIW worktree must be created from that commit so it inherits the
   artifacts directly; never ask the implementer to copy them manually.

<spec-template>

## Problem Statement

The problem that the user is facing, from the user's perspective.

## Solution

The solution to the problem, from the user's perspective.

## User Stories

A LONG, numbered list of user stories. Each user story should be in the format of:

1. As an <actor>, I want a <feature>, so that <benefit>

<user-story-example>
1. As a mobile bank customer, I want to see balance on my accounts, so that I can make better informed decisions about my spending
</user-story-example>

This list of user stories should cover the important user-visible aspects of the
feature without padding the document with implementation details.

## Implementation Decisions

A list of implementation decisions that were made. This can include:

- The modules that will be built/modified
- The interfaces of those modules that will be modified
- Technical clarifications from the developer
- Architectural decisions
- Schema changes
- API contracts
- Specific interactions

Do NOT include specific implementation file paths or code snippets in the
user-facing spec. Record durable architectural decisions in `design.md`.

Exception: if a prototype produced a snippet that encodes a decision more precisely than prose can (state machine, reducer, schema, type shape), inline it within the relevant decision and note briefly that it came from a prototype. Trim to the decision-rich parts — not a working demo, just the important bits.

## Testing Decisions

A list of testing decisions that were made. Include:

- A description of what makes a good test (only test external behavior, not implementation details)
- Which modules will be tested
- Prior art for the tests (i.e. similar types of tests in the codebase)

Testing decisions are plans only. This Skill does not run tests.

## Out of Scope

A description of the things that are out of scope for this spec.

## Further Notes

Any further notes about the feature. Use `%%` notes for unresolved risks or
questions rather than guessing.

</spec-template>
