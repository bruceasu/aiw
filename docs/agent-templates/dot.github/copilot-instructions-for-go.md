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
- after implementation, run one compile-only check using a `compile*` script or
  the narrowest language-level compiler command; do not retain a final artifact
- do not run tests, vet, `build*` scripts, final-artifact builds, verification
  scripts, network calls, or permission probes without authorization
- when authorized, run one package-focused command and ask before widening
