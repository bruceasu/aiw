
# CODEX.md
Always respond in Chinese.

## Role
You are operating as a local coding agent for a Go codebase.
Follow AGENTS.md first, then apply these Go-specific execution preferences.
---
## Default Mode
For medium or high complexity work, begin in planning mode.
Before editing files, provide:
- Understanding
- Relevant packages / files
- Assumptions
- Missing information
- Step-by-step plan
- Risks
- Validation commands
Prefer `/plan` first for:
- new features
- refactors
- concurrency changes
- public API changes
- dependency changes
- package structure changes
---
## Execution Preferences
### Preferred sequence
1. inspect only relevant files
2. identify affected packages, exported symbols, tests, and configs
3. produce a plan
4. apply minimal code changes
5. add or update tests
6. run validation
7. summarize results and remaining risks
### Editing rules
- keep changes localized
- preserve package boundaries
- avoid speculative abstractions
- keep interfaces minimal
- prefer explicit, idiomatic Go
---
## Go-Specific Guidance
### Package and API Design
- Be careful with exported names and public types
- Preserve API stability unless the task explicitly requires change
- Do not introduce unnecessary interfaces
### Context and Concurrency
Treat changes involving:
- `context.Context`
- goroutines
- channels
- locks
- retries
- background workers
as high-risk.
Explicitly call out:
- cancellation behavior
- timeout behavior
- shutdown behavior
- race-condition risks
- validation strategy
### Dependency Management
Treat `go.mod` and `go.sum` changes as high-risk.
Do not casually add libraries when the standard library or existing patterns are sufficient.
---
## Approval Required Before
- adding or changing dependencies
- changing package structure broadly
- changing exported APIs
- changing concurrency or context behavior
- changing auth/security logic
- changing persistence/migrations
- changing deployment or CI config
---
## Validation Preferences
Prefer:
```bash
./scripts/verify.sh
```
Otherwise use repository-standard commands only.
Common examples:
```bash
go test ./...
go build ./...
go vet ./...
```
Always report:
  - exact commands run
  - pass/fail results
  - what remains unverified

## Output Format
Use this structure for non-trivial tasks:
- Understanding
- Relevant Packages
- Assumptions
- Plan
- Implementation
- Validation
- Risks
- Harness improvements suggested

## Definition of Done
A task is not done until:
  - requested behavior is implemented
  - relevant tests are updated or added
  - public API and concurrency impacts are called out
  - validation is run or limitations are explicitly stated
  - docs are updated if needed

## Forbidden Shortcuts
Do not:
  - skip tests silently
  - introduce interfaces without need
  - change concurrency behavior without explanation
  - change public APIs casually
  - add dependencies casually
  - claim correctness without evidence


