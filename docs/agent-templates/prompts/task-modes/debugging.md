# Debugging

## First
- gather evidence before editing code
- reproduce the issue or state why reproduction is not possible
- compare expected behavior with actual behavior

## Investigation
- inspect the nearest failing path first
- prefer small confirming experiments over broad rewrites
- add temporary logging or instrumentation only when needed

## Change
- fix the confirmed cause, not only the visible symptom
- remove temporary debug-only edits unless they are part of the final solution

## Validate
- rerun the reproduction path
- add a regression test when practical
- report the evidence chain from symptom to fix
