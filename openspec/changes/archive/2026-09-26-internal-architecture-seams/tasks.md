## 1. Task-owned module

- [x] 1.1 Move `internal/taskx` implementation and tests to `internal/task`, preserving Task metadata, checklist, discovery, workspace, and verification behavior.
- [x] 1.2 Update all Go imports and package references from `internal/taskx` to `internal/task`.

## 2. Task Workflow adapter

- [x] 2.1 Create `internal/task/workflowadapter` and move Task-to-Workflow projection, handoff, artifact, and operational adapter implementation into it.
- [x] 2.2 Update `cmd/aiw-wf` assembly and the Workflow CLI seam so the concrete Task adapter is supplied by the Task-owned module.
- [x] 2.3 Remove duplicate Workflow implementation seams from `internal/commands/task`; retain Task lifecycle CLI behavior and keep Agent execution Workflow-owned.

## 3. Named Workflow Core

- [x] 3.1 Promote Workflow Core implementation from the generic `workflow/core` path to `internal/workflow`, retaining the `execution` submodule.
- [x] 3.2 Update Workflow imports, tests, focused command paths, build configuration, and documentation references.

## 4. Verification and handoff

- [x] 4.1 Perform a static import/package review confirming Workflow Core does not depend on `internal/commands/task` and the Task adapter is the single implementation seam.
- [x] 4.2 Run the repository-authorized narrow compile-only command for the changed Go entrypoint without retaining a final artifact.
- [x] 4.3 Update TODO, Verification evidence, and unresolved-risk notes; record that runtime tests and final builds were not run.

## 5. Build metadata

- [x] 5.1 Add a shared build version module, expose it through `aiw --help` and `aiw --version`, and inject the configured value from `build.bat` into the main executable and plugins.

## 6. Workflow domain ownership refinement

- [x] 6.1 Keep durable Notification, Auxiliary protocol, and Task-local Knowledge state in Workflow Core; keep Auxiliary process/HTTP/host implementations in `internal/workflow/execution`.
- [x] 6.2 Move the interactive `workflow knowledge` CLI command from `internal/workflow/execution` to `internal/workflow/cli` without changing its command contract.

## TODO

All implementation Work Items and verification evidence are complete; the coarse AIW Task lifecycle remains managed by the Workflow adapter.

## Verification

- Compile-only check: `go build ./cmd/...` succeeded with repository-local `GOCACHE`; no final artifact retained.
- Static review: no current `internal/workflow/core` references remain; `internal/workflow` and `internal/task/workflowadapter` do not import `internal/commands/task`; `cmd/aiw-wf` uses `DefaultOperations` from the Task adapter; the duplicate `internal/commands/task/workflow_seams.go` bridge was removed and its projection/runtime helpers now live in the Task adapter; Workflow CLI regression tests and the interactive knowledge command now live under `internal/workflow/cli`; the remaining Task CLI orchestration is split into `lifecycle.go`, `archive.go`, and `context.go`.
- Agent ownership: Agent execution and frozen context are under `internal/task/workflowadapter`; Task creation and metadata repair primitives are under `internal/task`.
- Build metadata: `internal/version` defaults to `dev`; `build.bat` injects `AIW_VERSION` through linker flags for the main executable and plugins.
- Tests: not authorized and not run by this specification step.

## Unresolved Risks

%% Root Task CLI still contains ordinary Task lifecycle orchestration, now separated by responsibility; Workflow Agent execution, delivery, focused validation, and metadata repair no longer depend on `internal/commands/task`.
