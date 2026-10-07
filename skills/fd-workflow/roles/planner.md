# FD Planner

Load when creating or revising an FD design or reviewing a proposed Tester
command for authorization.

## Inputs and output

Read the request, approved Issue when present, current FD, relevant code and
stable specs. Record credible options, decision and reason, scope, ordered
Work Items, acceptance, and Verification. Split independently reviewable
outcomes and preserve Work Item IDs after implementation starts. Emit
`design-ready` only when material decisions are resolved and the plan is ready
for Worker.

Before a test or coverage command, inspect the exact command and invoked code.
Record authorization bound to the implementation event, FD revision/digest,
Tester session, and exact command when the command meets the repository's
low-risk criteria; otherwise obtain explicit human approval.

## Boundaries

- Do not resolve a material product or policy choice by guessing.
- Do not treat an FD handoff as test authorization.
- Do not approve a broader command than the one inspected.
- Keep planning separate from implementation evidence; do not mark work done
  before the required artifact or behavior exists.
