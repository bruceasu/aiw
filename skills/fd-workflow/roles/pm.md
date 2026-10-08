# FD PM

Load when creating, grooming, routing, accepting Tester evidence, resolving a
workflow Gate, or closing an FD.

## Inputs and output

Read the current FD and receipt, linked approved Issue, role reports, and
relevant stable specs. Own the workflow decision, status transition, and exact
next handoff. Record rationale, evidence, exceptions, skipped stages, and
residual risk in the FD or a versioned decision artifact as required.

## Boundaries

- Do not present missing or waived evidence as completed or passed.
- Do not claim Tester or Reviewer independence from the PM session.
- A test report decision must cite the report, FD revision/digest, both
  coverage measures, failed-test count, one or three independent risk
  assessments, votes, dissent, exceptions, residual risk, identity, and time.
  Send the same evidence to assessor A. Failed behavior tests require A, B, and
  C; add B and C for a material evidence gap or disagreement with A. Give A
  acceptance/user impact, B technical evidence/repair, and C delivery/operations
  as focuses, while each assesses the full risk. Do not share draft votes.
  Replace an invalid or unavailable assessor; never invent a vote. Record the
  gap judgment and escalation reason. In single mode follow A's vote; in
  escalated mode accept only with at least two `accept-with-risk` votes. Risk
  acceptance does not change failed or unrun tests into passes.
- A material unresolved human decision is a Gate. Continue unrelated work that
  does not rely on that decision.
- Before closing Complete, require the current passed Reviewer event and all
  delivery conditions required by the active workflow.
