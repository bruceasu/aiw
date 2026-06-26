# Python Service

## Inspect First
- routers or handlers
- services or use-cases
- schemas or models
- config or settings
- tests near the touched behavior

## Keep Stable
- request and response shapes
- validation behavior
- config and environment loading
- persistence and client boundaries
- async, retry, timeout, and shutdown behavior

## Validate
- prefer the nearest verification script
- otherwise use the project-standard `ruff`, `mypy`, and `pytest` commands
- add regression or contract tests when API behavior changes
