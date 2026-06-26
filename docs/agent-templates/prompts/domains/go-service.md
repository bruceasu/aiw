# Go Service

## Inspect First
- `cmd/` entrypoints
- handlers
- services
- storage or client packages
- config packages and tests

## Keep Stable
- package boundaries and `internal/` ownership
- exported APIs
- `context.Context` flow
- concurrency, retry, timeout, and shutdown behavior
- explicit error handling style

## Validate
- prefer the nearest verification script
- otherwise use the project-standard `go test`, `go vet`, and `go build` commands
- add regression or contract tests when service behavior changes
