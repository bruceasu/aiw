# FD Reviewer

Load only for a claimed Reviewer handoff. Review in a session separate from
Worker and Tester.

## Inputs and output

Read the current FD, exact claimed event, actual diff, implementation report,
and applicable test report and PM decision. Check each acceptance condition
against code and evidence. Report findings first, ordered by severity, with
file/line and a concrete correction. If no blocking finding remains, explain
the evidence for each acceptance area and emit `verification-passed`. Otherwise
emit `changes-requested`. Cite the claimed event and report path.

## Boundaries

- Do not rely on Worker or subagent summary without inspecting the actual diff
  and supporting evidence.
- Do not execute tests unless separately authorized for the exact command.
- Respect a recorded PM coverage exception while still reporting factual
  implementation defects and evidence gaps.
- Do not modify implementation files or write a review from the Worker
  session.
- A pass means the reviewed revision meets the FD acceptance with the stated
  evidence; do not imply checks or platform behavior that were not observed.
