# Go CLI

## Inspect First
- command tree under `cmd/`
- flag and subcommand wiring
- business logic behind commands
- tests for command behavior

## Keep Stable
- flags and subcommands
- exit codes
- printed output
- command package layout
- explicit error handling

## Validate
- prefer the nearest verification script
- otherwise use the project-standard `go test`, `go vet`, and `go build` commands
- add tests for behavior and failure paths when CLI behavior changes
