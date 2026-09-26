## Why

The recent `cmd` and plugin extraction moved Workflow entrypoints out of the root CLI, but Task-owned Workflow operations are still split across command code, Task helpers, and Workflow adapters. This makes ownership unclear and forces maintainers to follow a conceptual dependency loop when changing Task lifecycle, Session, workspace, delivery, or Workflow execution.

## What Changes

- Make Task the owner of Task lifecycle, metadata, workspace, Session, Agent, and delivery operations used by Workflow.
- Rename the Task-owned implementation module from `taskx` to `task`.
- Consolidate Task-to-Workflow projection, handoff, artifact, and operational adapter implementation under one Task-owned adapter module.
- Promote Workflow Core from the generic `workflow/core` path to the named `workflow` module while retaining `execution` as an internal submodule.
- Reduce `commands/task` to CLI orchestration and preserve the Workflow CLI's `TaskAdapter` interface as its caller-facing seam.
- Keep runtime behavior, persisted formats, CLI names, and plugin entrypoints unchanged.

## Capabilities

### New Capabilities

- `task-workflow-seams`: Defines ownership and dependency rules between Task lifecycle operations and Workflow orchestration.

### Modified Capabilities

- None. This change reorganizes implementation seams without changing user-visible Workflow or Task behavior.

## Impact

The change affects Go package paths, imports, test locations, Workflow CLI wiring, Task command adapters, and implementation references in documentation and OpenSpec evidence. It does not add dependencies, change persisted schemas, or change the `aiw wf`, `aiw req`, or `aiw cz` command contracts.
