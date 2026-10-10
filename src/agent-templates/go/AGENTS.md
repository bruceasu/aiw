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
  - also load `.agents/prompts/domains/go-service.md`
- CLI markers:
  `cobra`, `urfave/cli`, command trees under `cmd/`, single-binary tools
  - also load `.agents/prompts/domains/go-cli.md`

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
Use static review by default. After implementation, run one compile-only check:
prefer `scripts/compile*` or root `compile*`, otherwise run the narrowest
applicable `go build` command without retaining a final artifact. Do not run
`scripts/verify.sh`, tests, vet, `build*` scripts, or final-artifact builds.

For Go work, use current compile evidence and an independent static review.
Do not create or run Go tests.

Ask before repository-wide commands. Rerun only after a relevant change.

## Go Cache Isolation

Use the repository compile script, which owns its cache settings. Do not
change global Go settings or `GOMODCACHE`.

## Escalation
If a deeper subtree has its own `AGENTS.md`, prefer that local file.
