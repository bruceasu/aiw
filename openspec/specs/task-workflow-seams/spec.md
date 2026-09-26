# task-workflow-seams Specification

## Purpose
Defines the ownership and dependency boundaries between Task lifecycle operations and Workflow orchestration, including the Task-owned adapter seam and the named Workflow Core module.
## Requirements
### Requirement: Task owns Task-facing Workflow operations

The system MUST keep Task lifecycle, metadata, workspace, Session, Agent, delivery, and Task-to-Workflow projection behavior in a Task-owned implementation module. Workflow orchestration MUST NOT depend on the root Task command module for these implementations.

#### Scenario: Workflow invokes a Task operation

- **WHEN** standalone Workflow prepares a workspace, creates a Session, runs an Agent, or performs delivery
- **THEN** it invokes the Workflow CLI's TaskAdapter interface and the concrete implementation is supplied by the Task-owned module

#### Scenario: Task behavior changes

- **WHEN** a Task-owned lifecycle or delivery rule changes
- **THEN** the implementation change is localized to the Task module or its Task-owned Workflow adapter without requiring Workflow Core to reimplement the rule

### Requirement: One Task-owned Workflow adapter seam

The system MUST expose one named Task-owned adapter module for Task-to-Workflow projection, handoff, artifact, and operational adapter behavior. The Workflow CLI MUST retain a narrow TaskAdapter interface, and Workflow Core MUST NOT import the concrete Task adapter module.

#### Scenario: Standalone Workflow is assembled

- **WHEN** `cmd/aiw-wf` assembles the Workflow program
- **THEN** it supplies the Task-owned concrete adapter to the Workflow CLI without importing the root command implementation as the adapter seam

#### Scenario: Workflow Core is tested

- **WHEN** Workflow Core tests exercise Store, state, Attempt, Gate, Session, verification, or delivery behavior
- **THEN** those tests do not require the concrete Task adapter implementation

### Requirement: Workflow Core has a named module

The system MUST expose Workflow Core through the `internal/workflow` module path. The generic `internal/workflow` path MUST NOT remain as a compatibility package after migration. The execution implementation MAY remain a subordinate `internal/workflow/execution` module.

#### Scenario: A Workflow caller imports Core behavior

- **WHEN** a Go package uses Workflow Store, state, Attempt, Gate, or delivery types
- **THEN** it imports the named `internal/workflow` module rather than a generic `core` path

### Requirement: Task command code remains CLI orchestration

The system MUST keep `internal/commands/task` focused on CLI dispatch and user-facing Task command orchestration. Workflow implementation and Task-owned side effects MUST NOT be added there as a second adapter implementation.

#### Scenario: Task CLI calls Workflow-related behavior

- **WHEN** a root Task command needs a Workflow projection or status operation
- **THEN** it calls the Task-owned module or the established Workflow seam rather than duplicating Workflow implementation in the command package

### Requirement: Migration preserves runtime contracts

The package migration MUST preserve persisted formats, plugin entrypoints, and runtime state transitions. The old internal import paths and the removed root `aiw turn`/`aiw chat` entrypoints do not require compatibility shims.

#### Scenario: Existing Workflow command runs

- **WHEN** a user invokes `aiw wf` or the `aiw-wf` plugin after migration
- **THEN** the command surface and Workflow state behavior remain unchanged

#### Scenario: Existing durable data is read

- **WHEN** Workflow or Task opens existing Task, Session, Attempt, Gate, or delivery records
- **THEN** the records are read using their existing formats without migration or reinterpretation caused by the package move
