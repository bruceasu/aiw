# Requirement Artifact

## Metadata

- Artifact Type: Requirement Plan
- Requirement ID: recommend-routing
- Stage: synthesis
- Status: READY_FOR_HUMAN_DECISION
- Based On: Problem Brief: Recommend workflow AI routing
- Human Approval Required: yes

## Facts

- The first release applies only to Managed Workflow `supervise`.
- A generated Requirement Plan MUST include a numbered `## Confirmed Workflow`
  section so promotion can deterministically render its Work Items.
- `analysis`, `coder`, `tester`, and `verifier` receive routing profiles;
  `compiler` is a deterministic, non-LLM Actor.
- A routing Profile identifies both the AI provider and model to use. The
  default stage mapping is `analysis=fast`, `coder=balanced`,
  `tester=balanced`, and `verifier=reasoning`.
- Default profiles are `analysis=fast`, `coder=balanced`,
  `tester=balanced`, and `verifier=reasoning`.
- An unavailable named Profile falls back to the existing global `[ai]`
  provider and model configuration.
- Requirement promotion generates a routing plan and Compile Plan after
  OpenSpec artifacts and Work Items are synchronized.
- Review is recommended but does not block `supervise`.
- Compile Plans may reference only repository-owned compile scripts and
  built-in adapters, never arbitrary commands.
- Multi-language repositories may define multiple path-owned compile targets.

## Assumptions

- A routing recommendation can be generated from the promoted Task's OpenSpec
  artifacts and synchronized Work Items.
- A failed routing recommendation can safely use stage-specific defaults.

## Evidence Index

- `requirements/recommend-routing/problem-brief.md`: confirmed routing,
  Compile Plan, compatibility, and scope decisions.

## Scope

- Generate and validate the `## Confirmed Workflow` section in every
  Requirement Plan used for promotion.
- Add a workflow command to recommend and persist Task routing and Compile
  Plans.
- Automatically invoke routing recommendation after successful Requirement
  promotion and Work Item synchronization.
- Persist a reviewable plan and use its per-Actor selections during
  `supervise` execution.
- Freeze the resolved Profile name, provider, model, and secret-free digest on
  each supervised request so retry and recovery use the original selection.
- Add a deterministic Compiler stage that executes the request-recorded
  Compile Plan, persists diagnostics, and creates bounded Coder repairs.
- Select affected compile targets from changed paths; compile all targets for
  shared or unmapped changes; aggregate failures into one repair handoff.

## Confirmed Workflow

1. Add an AI Profile catalog and stage defaults for `analysis`, `coder`,
  `tester`, and `verifier`, where every Profile selects a provider and model,
  with fallback to global `[ai]` configuration.
2. Add `aiw task workflow recommend-routing <task-id>` to generate a
  reviewable routing and Compile Plan, using stage defaults when
  recommendation generation fails.
3. Invoke routing recommendation after Requirement promotion has generated
  OpenSpec artifacts and synchronized Work Items.
4. Make Managed Workflow `supervise` resolve and snapshot the selected routing
  Profile, provider, model, and digest for each request, without changing
  `aiw turn`, `aiw chat`, or CZ behavior.
5. Add a deterministic Compiler stage that runs the request-recorded
  repository script or built-in adapter, selects affected multi-language
  targets, aggregates diagnostics, and creates bounded Coder repairs.

## Risks and Open Questions

- A Requirement Plan without `## Confirmed Workflow` blocks deterministic Task
  rendering and must report the missing section before promotion completes.
- Routing recommendations are advisory and may not be accurate. Output must
  identify its source and recommend user review.
- A compile target without a repository script or built-in adapter must open a
  Gate and recommend a repository script correction.
- Provider credentials must remain outside Task and workflow state.

## Earliest Blocking Decision

Approve the proposed Managed Workflow-only routing and deterministic Compile
Plan behavior for implementation planning.

## Human Decision Requested

Approve this Requirement for promotion into one AIW Task and OpenSpec change.

## Suggested Next Stage

Record `APPROVED`, then promote `recommend-routing` to the matching Task.

## Release

Status: NOT_STARTED
Trigger: after an AIW Task exists, implementation evidence is available, and a
release is planned.
