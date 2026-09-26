## Design Readiness

FD_NOT_REQUIRED: the ownership, seam placement, naming, compatibility, and migration decisions were resolved during architecture review. The change is a mechanical in-repository package migration with no new external interface or persisted-data design.

## Context

AIW now builds its executable entrypoints under `cmd/`, while reusable implementation packages live under `internal/`. Workflow Core is already isolated from the root executable, but its Task integration is not: `commands/task`, `taskx/workflow`, and `workflow/taskadapter` jointly implement Task-facing Workflow behavior.

The current Workflow Core is deep and owns Store, state, Attempt, Gate, Session execution, verification, and delivery. The problem is not missing behavior; it is that generic names and split ownership make the seam difficult to navigate and preserve.

## Goals / Non-Goals

**Goals:**

- Make `internal/task` the Task-owned module for Task lifecycle and side effects.
- Make `internal/task/workflowadapter` the single Task-owned implementation seam used by standalone Workflow.
- Keep the `TaskAdapter` interface in the Workflow CLI module, where it is consumed.
- Make `internal/workflow` the named Workflow Core module and keep `execution` as a subordinate implementation module.
- Leave `internal/commands/task` responsible for CLI orchestration only.
- Preserve Task lifecycle behavior, durable file formats, and plugin contracts; the previously exposed `aiw turn`/`aiw chat` commands are intentionally removed because Agent execution is now Workflow-owned.

**Non-Goals:**

- No new Workflow features or state transitions.
- No persisted schema migration.
- No compatibility package for the old internal import paths.
- No redesign of Requirement or Conventional Commit modules.
- No test execution, final artifact build, formatter, linter, or broad verification as part of specification work.

## Decisions

### Task owns Task behavior

Task owns metadata, checklist, workspace, Session creation, Agent execution, delivery, and Task-to-Workflow projection. Workflow owns its Store, state machine, Attempt/Gate rules, and orchestration. This prevents Workflow Core from depending on the whole CLI command package.

### One Task-owned Workflow adapter module

The implementation currently spread across Task workflow seams, Task workflow projection helpers, and the Workflow task adapter will be consolidated into `internal/task/workflowadapter`. This module may depend on Task internals and Workflow Core types, but Workflow Core will not depend on the adapter.

The `TaskAdapter` interface remains in `internal/workflow/cli` because that is the module consuming it. The concrete adapter is assembled by `cmd/aiw-wf`.

### Named Workflow Core module

The implementation currently under `internal/workflow` will become the package at `internal/workflow`. Its `execution` submodule remains separate. `cli`, `facade`, and the Task adapter seam remain explicit sibling modules. The generic `core` path is removed rather than preserved as a compatibility alias.

### Thin Task command module

`internal/commands/task` remains the root CLI-facing module. Workflow implementation and Task-owned side effects move out; command dispatch and user-facing Task command orchestration remain. The old `turn`/`chat` entrypoints are not retained; Workflow invokes the Task-owned Agent runner directly.

### Workflow domain ownership

Durable Notification state, Auxiliary protocol state, and Task-local Knowledge
state remain owned by Workflow Core because they participate in RuntimeState,
Store events, Work Item/Gate decisions, retry accounting, and review
transitions. Auxiliary process, HTTP, configuration, and host-resource
implementations remain under `internal/workflow/execution`. Knowledge search
or indexing infrastructure would be a separate module only if it becomes an
independent corpus/index capability; the current Task-local extraction and
review lifecycle is not that capability. The interactive `workflow knowledge`
command belongs to `internal/workflow/cli` and uses execution only for
production Auxiliary configuration.

### Build metadata

The user-facing version is owned by the small `internal/version` module.
Source builds default to `dev`; `build.bat` injects the release value through
Go linker flags so the main executable and plugins use the same version
without generated source files.

### Mechanical migration

Tests move with their implementation packages. Import paths, documentation links, focused command paths, and compile configuration are updated together. No runtime behavior or persisted data is changed, so no compatibility shim or rollback format is required; rollback is a source-tree revert before delivery.

## Risks / Trade-offs

- [Import blast radius] → Update all Go imports and path-based test fixtures in one ordered migration, then run one compile-only check.
- [Hidden Task/Workflow coupling] → Keep the TaskAdapter interface stable and statically inspect imports after the move.
- [Test package relocation errors] → Move tests with their package and preserve package declarations explicitly.
- [Documentation drift] → Update stable spec evidence and user-facing implementation references in the same change.

## Migration Plan

1. Move and rename the Task-owned package from `internal/task` to `internal/task`.
2. Consolidate Task-to-Workflow implementation files into `internal/task/workflowadapter` and update the standalone Workflow wiring.
3. Promote Workflow Core files from `internal/workflow` to `internal/workflow`, retaining `execution`.
4. Remove Workflow implementation seams from `internal/commands/task` while preserving CLI behavior.
5. Update imports, docs, stable evidence links, and focused command paths.
6. Perform static review and one compile-only check; tests remain unrun unless separately authorized.

## Open Questions

None for this change.
