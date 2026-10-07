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
  coverage measures, exceptions, residual risk, identity, and decision time.
- A material unresolved human decision is a Gate. Continue unrelated work that
  does not rely on that decision.
- Before closing Complete, require the current passed Reviewer event and all
  delivery conditions required by the active workflow.
