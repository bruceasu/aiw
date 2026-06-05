# AGENTS.md
This repository uses OpenSpec-lite workflow.
Before coding, ALWAYS read:
1. relevant files under:
   - openspec/changes/<task>/
   - openspec/specs/
2. especially:
   - task.md
   - design.md (if exists)
   - spec.md
Rules:
- Work on ONE task at a time.
- Keep changes scoped.
- Do not refactor unrelated modules.
- Preserve backward compatibility unless explicitly required.
- Prefer small reviewable diffs.
- Update TODO and Verification before finishing.
- Add risks/questions using `%%` notes instead of guessing.
- If stable requirements changed, update openspec/specs/.
- If architecture decisions changed, update design.md.
Git worktree convention:
- one task = one branch = one worktree
- branch: feature/<task-id>
- worktree: .wt/<task-id>

<!-- aiw-prompts:go:agents begin -->
# AGENTS.md
This repository uses OpenSpec-lite workflow.
Before coding, ALWAYS read:
1. relevant files under:
   - openspec/changes/<task>/
   - openspec/specs/
2. especially:
   - task.md
   - design.md (if exists)
   - spec.md
Rules:
- Work on ONE task at a time.
- Keep changes scoped.
- Do not refactor unrelated modules.
- Preserve backward compatibility unless explicitly required.
- Prefer small reviewable diffs.
- Update TODO and Verification before finishing.
- Add risks/questions using `%%` notes instead of guessing.
- If stable requirements changed, update openspec/specs/.
- If architecture decisions changed, update design.md.
Git worktree convention:
- one task = one branch = one worktree
- branch: feature/<task-id>
- worktree: .wt/<task-id>

---

# AGENTS.md
默认假设�?- 主要�?**Go 服务 / CLI / 微服�?/ 基础设施工具**
- 强调包边界、接口最小化、context 传播、错误处理、测试、构建可重复�?
Always respond in Chinese.

## Purpose
This Go repository is maintained with human oversight and AI coding agents.
Optimize for correctness, clarity, small diffs, and predictable behavior.
---
## Core Working Rules
### 1. Plan first for non-trivial work
Before modifying files, first provide:
1. Task understanding
2. Relevant packages / files
3. Assumptions
4. Missing information
5. Proposed implementation plan
6. Risks
7. Validation steps
Do not jump directly into implementation for medium or large tasks.
### 2. Respect package boundaries
Follow the existing package structure.
Keep changes localized and avoid introducing unnecessary coupling between packages.
### 3. Keep interfaces minimal
Do not introduce interfaces unless there is a clear need consistent with the codebase.
Prefer concrete types unless abstraction is already justified.
### 4. Make behavior explicit
Prefer straightforward control flow and explicit error handling.
Do not hide important behavior in magic helpers or overly generic abstractions.
### 5. Make every change verifiable
Every meaningful change should include:
- relevant tests
- formatting
- vet/static checks if configured
- build verification
---
## Go-Specific Engineering Rules
### Package Design
- Respect existing package ownership and responsibilities.
- Avoid cyclic dependencies.
- Do not move logic across packages casually.
- Keep internal package boundaries intact.
### Context Usage
- Preserve existing `context.Context` patterns.
- Do not drop context propagation.
- Be careful when changing cancellation, timeout, or request-scoped behavior.
### Errors
- Follow existing error handling conventions.
- Do not swallow errors.
- Wrap errors only when it adds useful context and matches existing style.
### Concurrency
- Treat goroutines, channels, locks, worker pools, and shared state as high-risk.
- Be explicit about shutdown, cancellation, retries, and race-condition implications.
### Dependencies
Treat changes to:
- `go.mod`
- `go.sum`
as high-risk.
Request approval before adding or upgrading dependencies.
---
## Testing Expectations
Prefer:
1. format
2. vet / static checks
3. unit tests
4. integration tests
5. build
Add or update tests for:
- package behavior
- edge cases
- error handling
- concurrency-sensitive logic when relevant
- handler/service behavior when relevant
---
## High-Risk Changes
Pause and explain before:
- dependency changes
- concurrency model changes
- context propagation changes
- public API changes
- persistence or migration changes
- auth/security changes
- deployment or CI changes
Explain:
- what changes
- why it is needed
- what could break
- how it will be validated
---
## Preferred Workflow
### Phase 1: Understand
Identify:
- affected packages
- call paths
- interfaces or exported types affected
- tests affected
### Phase 2: Plan
Provide a concise, reviewable implementation plan.
### Phase 3: Implement
Apply the smallest correct change.
Preserve package and naming conventions.
### Phase 4: Verify
Use repository-standard commands first.
### Phase 5: Report
Summarize:
- files changed
- behavior changed
- assumptions
- risks
- validation
- harness improvements suggested
---
## Validation
Prefer:
```bash
./scripts/verify.sh
```

If unavailable, use repository-standard Go commands, for example:
```bash
go test ./...
go build ./...
go vet ./...
```

Run only commands appropriate for the repository.
Do not claim success without reporting what actually ran.

## Documentation Expectations
Update docs when changing:
  - CLI behavior
  - config/env usage
  - service behavior
  - developer workflow
  - build or run instructions

## Communication Style
For non-trivial tasks, structure responses as:
  - Understanding
  - Relevant Packages
  - Assumptions
  - Plan
  - Implementation
  - Validation
  - Risks
  - Harness improvements suggested

---
<!-- aiw-prompts:go:agents end -->
