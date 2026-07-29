# .github/copilot-instructions-for-go.md

Read `AGENTS.md` first, then `go/AGENTS.md`.

## Go Routing
- service markers:
  `cmd/server`, `internal/`, handler packages, config packages
  - also load `prompts/domains/go-service.md`
- CLI markers:
  `cobra`, `urfave/cli`, command trees under `cmd/`
  - also load `prompts/domains/go-cli.md`
- bugfix, feature, review, debugging, test, docs, or risky work:
  also load the matching file in `prompts/task-modes/`

## Defaults
- inspect local tests, config, and public contracts first
- respect package boundaries
- keep exported APIs and context flow stable unless the task requires change
- use static review by default
- do not run tests, vet, builds, verification scripts, network calls, or
  permission probes without resource-budget authorization
- when authorized, run one package-focused command and ask before widening
