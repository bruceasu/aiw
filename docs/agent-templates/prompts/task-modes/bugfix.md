# Bugfix

## First
- identify the broken behavior
- say how the problem was observed or reproduced
- locate the contract that is failing

## Change
- prefer a small fix close to the root cause
- avoid unrelated cleanup
- avoid speculative rewrites

## Validate
- add or update a regression test when practical
- run the nearest checks that prove the bug is fixed
- note nearby behavior that still was not verified

## Report
- explain the cause
- explain the fix
- explain what was verified
