# AGENTS.md

## Purpose
This is the default local rule file for Go subtrees.
Use it when the task is mostly Go or the current directory contains Go markers.

## Inspect First
- `go.mod` and module layout
- `cmd/` entrypoints, handlers, services, and `internal/` packages
- config packages and tests near the touched code

## Detect The Go Domain
- Service markers:
  `cmd/server`, `internal/`, handler packages, config packages
  - also load `../prompts/domains/go-service.md`
- CLI markers:
  `cobra`, `urfave/cli`, command trees under `cmd/`, single-binary tools
  - also load `../prompts/domains/go-cli.md`

Use one domain prompt by default.
Load both only when the task truly spans both service and CLI code.

## Keep Stable
- package boundaries and `internal/` ownership
- exported APIs
- `context.Context` flow
- concurrency, retry, and shutdown behavior

## High-Risk Go Areas
- dependency changes
- exported API changes
- context or concurrency changes
- auth, schema, or deployment changes

## Validation Options
Use static review by default. Do not automatically run `scripts/verify.sh`,
tests, vet, or builds.

When the shared resource budget authorizes runtime validation, choose one
smallest relevant command:
- `go test ./path/to/package`
- `go vet ./path/to/package`
- `go build ./path/to/package`

Ask before repository-wide commands. Rerun only after a relevant change.

## Escalation
If a deeper subtree has its own `AGENTS.md` or `CODEX.md`, prefer that local file.
