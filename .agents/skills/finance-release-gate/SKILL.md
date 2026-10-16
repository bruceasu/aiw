---
name: finance-release-gate
description: Assess whether an implemented change in any domain is ready to release. Use only near launch with implementation and relevant operational evidence; not for Issue discovery or design discussion.
---

# Release Gate

Use this method across domains. The `finance-` skill ID is retained for compatibility; it does not restrict activation to finance. Apply domain-specific rules only when relevant to the supplied evidence.

Evaluate release evidence for an implemented change (linked FD when present) and return a `Release Review`. This is a post-implementation release gate; it is outside the requirement-discussion chain.

Local FD merge/archive is not a release. Do not require this method for every
FD, call it from ordinary fd-review, or interpret GO as deployment permission.
Use it only when a real release is planned and its operational evidence is
available for assessment. Missing evidence is reported, not fabricated.

## Use when

- A release is planned and the implemented change needs an evidence-based readiness assessment, regardless of domain.
- The user supplies implementation and operational evidence sufficient to assess launch risk.

## Do not use when

- The requirement is still being discovered, valued, or explored technically.
- The user asks to create rollback scripts, migrations, monitoring, or implementation code.

## Inputs

- Implemented change scope and affected systems.
- Target launch window.
- Evidence for rollback, monitoring, and named ownership, plus migration/data impact, permissions, and audit where relevant. Mark inapplicable areas with a reason.

## Produce

Return `Release Review` with `GO`, `GO_WITH_RISK`, or `NO_GO`, clearly separating supplied evidence from missing evidence.

## Rules

- Missing critical evidence yields `NO_GO`; do not infer procedures or owners.
- Do not create or modify FD, OpenSpec, deployment, migration, rollback, or monitoring artifacts.
- Refer missing requirement intent or metric semantics back to the appropriate Requirement Artifact; refer implementation gaps to the managed FD workflow.
- Use `%%` for missing evidence and name the required next action and owner.

## Output

```markdown
# Release Review

## Metadata
- FD ID or implemented change reference: %% REQUIRED
- Release Window:
- Decision: GO / GO_WITH_RISK / NO_GO

## Evidence Reviewed
## Scope
## Migration and Data Impact
## Permissions and Audit
## Rollback and Operations
## Open Risks
| Risk | Severity | Evidence Gap | Owner | Release Blocker |
|---|---|---|---|---|

## Final Recommendation
## Required Next Action
```
