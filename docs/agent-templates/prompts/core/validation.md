# Validation

## Scope
- Prefer the nearest `scripts/verify.sh`.
- Otherwise validate at the smallest scope that proves the change.
- Use broader checks when the change crosses module or project boundaries.

## Evidence
- Report the exact commands you ran.
- Report pass, fail, or not run.
- Say what remains unverified and why.

## Tests
- Add or update tests when behavior changes.
- Prefer regression tests for bugfixes.
- Prefer contract or boundary tests for public behavior changes.
