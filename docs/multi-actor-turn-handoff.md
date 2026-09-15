# Multi-Actor Supervise Workflow

Status: IMPLEMENTATION MAP AND REQUIREMENT SOURCE

## Current Implementation

This section describes the current source tree. The sections starting at
**Purpose** retain the target architecture; their MUST/SHOULD statements are
requirements, not a claim that every stage is wired into supervise. Operational
instructions are in [Supervise](supervise.md).

| Capability | Current supervised path |
| --- | --- |
| Agent dispatch | Sequential `aiw turn --supervised`, bound to one Work Item, Attempt, workspace, expected Session turn, and dispatch marker. |
| Actor roles | Role contracts and helpers exist. The dispatcher does not yet execute the full Analysis → Coder → Tester → Verifier pipeline. |
| Model routing | `[ai.profiles.<name>]` defines provider/model pairs. Recommendation stores actor mappings in `routing-plan.json`; dispatch currently selects `coder`. |
| Compile | A completed structured Agent outcome runs the request's frozen Compile Plan before the Attempt closes. |
| Repair | Same Attempt and AI selection; a fresh Session turn receives compiler diagnostics. The third consecutive compile failure stops repair; success resets the count. |
| Ordinary retry | Only no-progress consumes its budget. Blocked outcomes and initial Git preflight failures do not. |
| Tests | Optional focused verification without a Plan is waived. The focused-test path has separate authorization; full Tester authoring/final execution is not automatically sequenced by this loop. |
| Reports | `workflow report` reads `reports/latest-failure.md`; no Session-output scan or runtime initialization is needed. |
| Delivery | Eligible completed isolated Tasks invoke local commit, merge, ancestry verification, and cleanup automatically. No automatic push, PR, or archive follows. |
| Conflict resolution | Local delivery preserves the original Task worktree as a candidate and recommends the Skill; it does not automatically invoke a conflict-resolution Agent or create another temporary worktree. |
| Notifications | Core notification contracts/helpers are not evidence that every supervise transition dispatches a notification Plugin. |

### Current Request And Repair Sequence

```text
requirement promote -> recommend-routing -> routing-plan.json
supervise -> Git preflight -> prepare/reuse Attempt-bound request
          -> snapshot coder Profile and Compile Plan
          -> dispatch and validate structured Session outcome
          -> completed: execute frozen compile targets
               -> pass: reset compile failures, close Attempt, sync checklist
               -> failure 1/2: prepare repair handoff, dispatch next Session turn
               -> failure 3: block and report
          -> blocked: Gate without ordinary retry increment
          -> no-progress: apply ordinary retry policy
          -> eligible isolated Task complete: local merge and cleanup
```

Default Profile names are `analysis=fast`, `coder/tester=balanced`, and
`verifier=reasoning`. They are labels, not an automatic capability ladder.
Missing/incomplete provider-model pairs fall back to `[ai]`. New requests apply
supervise CLI overrides before recording `AISelection`; an existing request
and its repairs keep the recorded choice. There is no automatic upgrade after
failure. Ordinary `turn`, `chat`, and CZ keep their existing resolution rules.

A prepared request freezes Compile Plan, selected compiler inputs/results, and
recovery state. Missing plans or unavailable targets stop through a Gate.
Checked checklist text cannot release active supervised compilation or bypass
a blocked Work Item. One initial failed compile permits at most two repair
turns before the third failed compile stops the loop.

The current local-delivery conflict path attempts to abort the failed parent
merge, merges the parent into the existing Task worktree, and preserves that
candidate for review. `resolving-merge-conflicts` is guidance at this boundary,
not an automatically dispatched turn. The separate `wt pull --conflict-handoff`
proposal flow is not used by supervise's local delivery.

Source entry points: `internal/commands/task/workflow_supervisor.go`,
`workflow_commands.go`, `workflow_compile.go`, and `local_delivery.go`;
configuration lives in `internal/ai/config.go`, durable state in
`internal/workflow/`. This documentation update used static inspection only.

## Purpose

The remaining sections define the target behavior of `aiw task workflow supervise`.
It is the requirement source for later OpenSpec proposals, designs, capability
specifications, and implementation tasks. It does not describe the current
implementation beyond the map above and does not authorize a running supervisor to assume that an
unimplemented capability already exists.

The target workflow coordinates an entire AIW Task through requirement
analysis, production-code implementation, compile validation, unit-test
authoring, final unit-test execution, optional project verification, Git
delivery, notification, recovery, and cleanup.

## Goals

The workflow MUST:

1. Run automated Task development in an isolated, durable Git task branch and
   worktree.
2. Give Analysis, Coder, Compiler, Tester, and Verifier distinct responsibilities
   and persisted handoffs.
3. Store all runtime state, Actor interaction, Evidence, Gates, Attempts,
   delivery records, and notifications for one Task under one task directory.
4. Compile production-code changes before unit tests are authored.
5. Derive an observable-interface inventory from the implementation so Tester
   can write tests against stable seams rather than internal details.
6. Author tests after each Work Item and execute the collected unit-test plan
   after all Work Items are implemented.
7. Return failures to Coder through bounded repair loops.
8. Support an optional, report-only project verification stage.
9. Commit completed work and merge it into the recorded local parent branch.
   Pull-request publication is a future `aiw-github` Plugin responsibility.
10. Persist notifications before dispatching them through the existing Python
    `aiw-plugins` mechanism.
11. Be recoverable after process termination, partial writes, Plugin failure,
    Agent failure, or Git conflict without creating a duplicate Attempt.

## Non-Goals

- The initial implementation does not push a branch or create a pull request
  unless the selected delivery policy explicitly enables external publication.
- Project-wide integration or system-test failures do not fail an otherwise
  accepted Task during the optional Verifier stage.
- Skills do not own Task lifecycle, write leases, Git delivery, or status
  transitions.
- Supervisor does not infer missing requirements, silently expand test scope,
  or treat Agent prose as authoritative runtime state.
- Read-only repository mounts are not required. Linked Git worktrees require
  the repository's `.git` metadata to be writable.

## Domain Vocabulary

| Term | Meaning |
| --- | --- |
| Task | Durable AIW lifecycle for one goal, workspace, delivery, and archive lifecycle. |
| Work Item | One typed execution unit mapped to a stable human requirement/checklist identifier. |
| Actor | A role-specific Agent invocation: Analysis, Coder, Compiler, Tester, or Verifier. |
| Handoff | Persisted structured input from one Actor or stage to the next. |
| Attempt | One bounded implementation-and-repair cycle for a Work Item. |
| Gate | An unresolved decision, authorization, dependency, validation, or delivery precondition. |
| Evidence | A persisted static review, command result, Agent result, approval, or manual record. |
| Parent branch | The branch that was current when supervision first created the Task workspace. |
| Task branch | The durable `ai/<task-id>` branch used for AI development and delivery. |
| Delivery | Commit plus verified local merge and conditional cleanup. Future PR delivery belongs to `aiw-github`. |

## Architectural Modules

The implementation SHOULD expose a small Supervisor interface and keep
orchestration details behind deep modules:

```text
Supervisor.Advance(taskID) -> Outcome
       |
       +-- WorkspaceCoordinator
       +-- TaskStateStore
       +-- SkillRegistry
       +-- TurnCoordinator
       +-- VerificationCoordinator
       +-- DeliveryCoordinator
       +-- NotificationDispatcher
```

### WorkspaceCoordinator

Owns parent-branch capture, task-branch creation or reuse, worktree binding,
workspace identity checks, and cleanup eligibility. Callers MUST NOT reconstruct
Git ancestry or workspace paths independently.

### TaskStateStore

Owns atomic state transitions, append-only events, write leases, Attempt
identity, pending-event recovery, Evidence references, and task-directory
layout. It is the only writer of authoritative Workflow state.

### SkillRegistry

Discovers allowed Skills, resolves routing, records provenance, and prepares a
stable Skill manifest for Actors. It does not execute Skills or grant lifecycle
authority to them.

### TurnCoordinator

Builds role-specific requests, invokes Actors, validates structured outputs,
grants role-scoped write leases, and persists handoffs. It MUST reject missing
required output instead of guessing.

### VerificationCoordinator

Selects compile and test commands under the persisted execution policy,
executes commands through deterministic runners, records Evidence, and maps
failures back to Work Items.

### DeliveryCoordinator

Owns commit, merge probing, local merge, conflict-state preservation and human
handoff, ancestry verification, and cleanup decisions. Core does not push or create pull requests.
Future push, pull-request creation, and pull-request status queries belong to
the existing Python `aiw-github` Plugin.

### NotificationDispatcher

Persists a notification event, discovers the configured Python AIW Plugin,
dispatches the event, and records delivery status. Notification retries are
independent of Work Item Attempts.

## Workspace And Branch Model

When supervision first starts, it MUST capture the current branch and commit as
the immutable Task baseline:

```text
parent_branch @ parent_commit
       |
       +-- ai/<task-id>            task branch
               |
               +-- .wt/<task-id>   managed worktree
```

Requirements:

- The current branch MAY be `develop`, `feature/*`, `bugfix/*`, `release/*`, or
  another valid development branch.
- Task origin, such as OpenSpec, GitHub, or a local request, does not determine
  Git ancestry. The recorded current branch and commit do.
- Supervisor MUST create `ai/<task-id>` from `parent_commit` when it does not
  exist.
- If `ai/<task-id>` already exists, Supervisor MUST verify its recorded Task
  identity and ancestry before reuse. It MUST NOT silently reset or recreate it.
- The Task worktree MUST check out `ai/<task-id>`.
- `feature/<task-id>` is not part of the default automated topology.
- A future parallel Work Item design MAY create temporary child branches, but
  that is a separate capability and MUST NOT change the default topology.
- Merge conflicts MUST reuse the existing Task branch and worktree for human
  resolution. Delivery MUST NOT create a temporary conflict branch or worktree.
- Raw Git operations and `aiw wt` MUST share the same workspace registry so
  neither can create an untracked duplicate worktree.

### Parent Checkout Write Fence

Automated supervision MUST NOT directly modify Git-managed files in the parent
checkout while Work Items are advancing. The parent checkout is a protected
workspace, not an execution workspace.

Allowed writes associated with the parent checkout are limited to:

- the canonical Task state directory under `.ai/<task-id>/`, or its configured
  external `state_root` equivalent; and
- shared `.git/` metadata that Git must update for registered worktrees, task
  branch refs, and commits.

All other writes MUST be routed to the Task worktree or rejected:

```text
parent checkout
|-- .ai/<task-id>/**     Core runtime writes allowed
|-- .git/**              Git-managed metadata writes allowed
`-- tracked files        direct writes forbidden during supervision

.wt/<task-id>/
|-- production paths    Coder writes under its lease
|-- test paths          Tester writes under its lease
`-- Task artifacts      semantic ownership rules apply
```

Enforcement requirements:

- `supervise` MUST use an isolated Task worktree and MUST NOT offer a primary
  workspace execution mode.
- Every Actor process MUST use the verified Task worktree as its working
  directory.
- Every Core write path MUST be resolved and verified as a descendant of the
  canonical Task state directory before writing.
- Supervisor MUST record the parent branch, parent HEAD, and a snapshot of
  pre-existing parent-checkout changes before the first mutating Actor starts.
- Pre-existing human changes in the parent checkout MUST be preserved exactly.
- A Core or Actor write whose resolved target escapes its allowed roots is a
  confirmed `parent_write_fence_violated` condition. Supervisor MUST stop, open
  that Gate, report the attempted or changed paths, and MUST NOT attempt
  automatic rollback.
- A new parent-checkout change observed after startup is not, by itself, proof
  that Supervisor caused it. When the writer cannot be attributed to Core or an
  Actor, Supervisor MUST record `parent_checkout_drift` with the changed paths
  and continue the current Work Item.
- Human or external-process changes to the parent checkout MUST NOT terminate
  Analysis, Coder, Compiler, Tester, or Verifier work in the Task worktree.
- A parent-checkout snapshot is diagnostic and delivery Evidence; it is not a
  lock that prevents concurrent human work.
- Task branch commits may update shared Git refs, but they MUST NOT move the
  parent branch ref.
- The parent branch and its checkout may change only during an explicit local
  Delivery operation. A future `aiw-github` Plugin delivery MUST leave the
  local parent branch unchanged until a later verified synchronization.

The governing invariant is:

> Supervisor execution never directly modifies managed files in the parent
> checkout. External parent-checkout changes are recorded as drift and do not
> interrupt Work Item execution. AIW changes parent managed files only through
> an explicit Delivery operation.

Suggested workspace record:

```json
{
  "schema_version": 1,
  "task_id": "example-task",
  "repo_root": "C:/work/project",
  "parent_branch": "develop",
  "parent_commit": "0123456789abcdef",
  "task_branch": "ai/example-task",
  "worktree": "C:/work/project/.wt/example-task",
  "created_at": "2026-09-14T12:00:00+09:00"
}
```

## Task State Directory

All runtime data for a Task MUST be colocated under one canonical directory.
The default is `<repo>/.ai/<task-id>/`. `state_root` and `worktree_root` MAY be
configured, but one Task MUST NOT write authoritative state to multiple roots.

```text
.ai/<task-id>/
|-- state.json
|-- events.jsonl
|-- execution-policy.json
|-- workspace.json
|-- implementation-contract.md
|-- locks/
|   |-- supervisor.json
|   `-- write-lease.json
|-- skills/
|   |-- manifest.json
|   `-- snapshots/
|-- work-items/
|   `-- wi-0001/
|       |-- state.json
|       |-- attempts/
|       |-- handoffs/
|       |-- observable-interfaces.json
|       `-- test-cases.json
|-- actors/
|   |-- analysis/
|   |-- coder/
|   |-- compiler/
|   |-- tester/
|   `-- verifier/
|-- verification/
|-- delivery/
`-- notifications/
```

Every Agent request, live event stream, final output, stderr record, structured
handoff, and Session reference MUST be reachable from this directory. A backend
MAY store large content elsewhere only when the Task directory contains a
durable reference with integrity metadata.

The resolved absolute `state_root`, Task directory, worktree path, task branch,
and parent branch MUST be printed at startup and recorded in `workspace.json`.

### Atomicity And Leases

- State mutation MUST use a single Task writer guarded by a write lease.
- A state transition and its event append MUST be recoverable as one logical
  operation through a persisted `pending_event` or equivalent journal record.
- Temporary files MUST be created in the target directory and atomically
  replaced where the platform supports it.
- A sharing violation or access-denied rename MUST pause mutation, preserve the
  lease and pending record, and produce a diagnostic. It MUST NOT start another
  Attempt or tell the user to delete the lease file.
- Recovery MUST finish or roll back the recorded pending transition
  idempotently before Supervisor can resume.
- Stop releases only the Supervisor lease. It does not fabricate completion or
  discard the owning Work Item Attempt.

## Artifact Ownership And Projection

Authoritative runtime state lives in `.ai/<task-id>/`; it MUST NOT be split
between `tasks.md`, `task.toml`, Session files, and unrelated lock directories.

- OpenSpec proposal, design, specs, and human checklist prose remain
  human-owned requirement artifacts.
- `task.toml` is static Task definition and mapping data during supervision. It
  MUST NOT be rewritten for every runtime transition.
- `tasks.md` MUST NOT be used as a second runtime database.
- Supervisor records Work Item state in the Task directory while running.
- After a Work Item is accepted, Workflow Core MUST update its checkbox in the
  Task worktree's `tasks.md`. Coder and Tester MUST NOT mark their own Work Item
  complete.
- Checklist projection MUST use stable Work Item identifiers and a semantic
  update rather than rewriting the entire file.
- Any checklist or Task-artifact projection MUST target the Task worktree. It
  MUST NOT write the corresponding file in the parent checkout.
- Generated summaries MUST live in marked, Core-owned regions.
- Duplicate checklist identifiers, missing markers, changed source hashes, or
  concurrent human edits MUST produce a reconciliation Gate. Core MUST NOT
  overwrite human prose.
- Final reconciliation MUST verify all projected Work Item states before
  delivery. Git text-conflict resolution is not a substitute for semantic
  checklist reconciliation.

These rules replace the previous mechanism in which `tasks.md` and `task.toml`
could both be repeatedly modified during supervision and then conflict during
branch delivery.

## Skill Discovery And Injection

Actors need predictable access to required Skills even when repository-local
legacy Skill directories have restrictive permissions.

Skill discovery priority SHOULD be:

1. Repository canonical `<repo>/skills/`.
2. `<aiw-install-root>/skills/`, where `aiw-install-root` is resolved from the
   running AIW executable.
3. Configured user Skill roots.
4. A configured and verified legacy installation root, such as
   `C:/green/aiw/skills`, only as a fallback.

Supervisor MUST NOT depend on `.agents/skills/` being readable. Selected Skills
SHOULD be recorded in `skills/manifest.json` with name, source path, version or
content hash, selection reason, and Actor routes. A snapshot MAY be copied into
the Task directory when a stable, readable input is required for later resume
or audit.

Example manifest entry:

```json
{
  "name": "implement",
  "source": "C:/green/aiw/skills/implement",
  "sha256": "...",
  "actors": ["coder"],
  "reason": "production code implementation"
}
```

## Implementation Contract

Before Coder starts, Analysis MUST produce or validate
`implementation-contract.md` with status `READY`. Durable architecture decisions
remain in OpenSpec `design.md`; this contract is the operational handoff.

```md
# Implementation Contract

Status: READY | NEEDS_INPUT | COMPLETE

## Functional Surface
- Requirement and acceptance-scenario identifiers.
- Observable behavior.

## Public Interfaces
- Commands, services, controllers, exported utilities, APIs, or events.
- Inputs, outputs, errors, compatibility, and ordering constraints.

## Test Seams
- Public seams available to Tester.
- Fixture, fake, and isolation conventions.
- Internal details that must not be tested directly.

## Constraints
- Allowed production paths.
- Dependency, compatibility, migration, and security restrictions.

## Open Questions
%% NEEDS_INPUT: ...
```

Missing critical behavior, public interfaces, acceptance scenarios, or test
seams MUST set `NEEDS_INPUT` and open a Gate. Coder may report a deviation but
MUST NOT silently redefine this contract.

## Task-Level Workflow

```text
supervise start
  -> capture parent branch and commit
  -> create or reuse ai/<task-id> and worktree
  -> resolve state root and Skill manifest
  -> synchronize requirement Work Items once
  -> for each Work Item
       -> Analysis
       -> Coder
       -> Compiler planning and compile execution
       -> repair on compile failure
       -> observable-interface inventory
       -> Tester authors or updates unit tests
       -> record test cases
  -> Tester executes the collected final unit-test plan
  -> repair and retest on failure
  -> optional Verifier project suite, report-only
  -> final artifact reconciliation
  -> commit
  -> delivery policy
  -> notification
  -> conditional cleanup and Task completion
```

Suggested Work Item states:

```text
READY
  -> ANALYZING
  -> ANALYZED
  -> CODING
  -> COMPILING
  -> INTERFACES_RECORDED
  -> TESTS_AUTHORED
  -> READY_FOR_FINAL_TEST
  -> ACCEPTED
```

Any stage MAY enter `BLOCKED`, `NEEDS_INPUT`, or `FAILED_RETRYABLE`. Only the
coordinator advances stages. Actor text alone cannot change state.

## Actor Responsibilities

### Analysis Actor

Inputs:

- Task requirement sources and selected Work Item.
- Relevant OpenSpec proposal, design, specifications, and decisions.
- Existing implementation contract and nearby code.
- Applicable Skill routes.

Outputs:

- Ready or blocked implementation contract.
- Acceptance-scenario mapping.
- Recommended implementation and test seams.
- Risks and explicit `%% NEEDS_INPUT` items.

Restrictions:

- Read-only.
- Does not edit code, invoke compilation, run tests, or mutate lifecycle state.

### Coder Actor

Inputs:

- Ready implementation contract.
- Analysis handoff.
- Current Work Item and prior failure Evidence when repairing.
- Coder Skill routes.

Outputs:

- Production-code changes.
- Structured implementation report listing changed files, implemented
  behavior, actual interfaces, deviations, risks, and recommended compile scope.

Restrictions:

- Only Coder may edit approved production paths.
- Coder does not run tests, project verification, packaging, or release builds.
- Coder does not choose an arbitrary executable command. It proposes scope;
  deterministic runners execute commands selected under policy.

### Compiler Actor

Compiler is a planning and interpretation role. Command execution belongs to
the VerificationCoordinator.

Inputs:

- Implementation report and changed production files.
- Repository language and module layout.
- Execution policy.

Outputs:

- Selected compile strategy and rationale.
- Expected scope and artifact behavior.
- Interpretation of compiler diagnostics.
- Compile Evidence reference.

Compile command selection order:

1. An approved repository-root or `scripts/compile*` script appropriate for the
   operating system.
2. The narrowest language-level compiler operation for the affected module.

Rules:

- A `build*` script is not treated as a compile script.
- The compile stage MUST NOT execute tests, formatters, linters, packaging,
  deployment, or a final distributable build.
- For Java, valid strategies may include narrow `mvn compile`, Gradle compile
  tasks, or `javac`, provided tests and packaging are not invoked.
- For Go, `go build` MAY be used only as compile validation with output directed
  to a temporary location and removed afterward; no distributable artifact is
  retained. `go vet` is not compilation and belongs to optional verification.
- Compiler failure returns diagnostics to Coder and consumes the current Work
  Item Attempt cycle.
- Compiler success is required before Tester receives the Work Item.

### Observable-Interface Inventory

After compilation succeeds, Supervisor MUST persist the actual observable
interfaces that Tester may exercise. Typical Java candidates include public
Service methods, Controller behavior, and public utility functions. Other
languages SHOULD use exported functions, commands, handlers, public types, and
stable error behavior appropriate to that language.

Example:

```json
{
  "schema_version": 1,
  "work_item_id": "wi-0001",
  "interfaces": [
    {
      "kind": "service_method",
      "symbol": "OrderService.createOrder",
      "source": "src/main/java/example/OrderService.java",
      "inputs": ["CreateOrderRequest"],
      "outputs": ["Order"],
      "errors": ["DuplicateOrderException"],
      "scenarios": ["AC-1", "AC-2"],
      "test_seam": "OrderRepository fake"
    }
  ]
}
```

### Tester Actor: Authoring Stage

Inputs:

- Requirement and implementation contracts.
- Analysis and Coder handoffs.
- Observable-interface inventory.
- Approved test paths and testing Skills.

Outputs:

- New or updated unit-test source files.
- Structured test-case records mapped to Work Item and acceptance scenarios.
- Proposed final test command scope.

Restrictions:

- Tester may edit only approved test paths.
- Tester MUST NOT edit production code.
- Tester authors tests for the current Work Item but does not automatically run
  them during this stage.
- Tester MUST NOT invent an unrestricted shell command.
- Integration and system tests are excluded from the Work Item unit-test plan.

Example test-case record:

```json
{
  "id": "TC-wi-0001-001",
  "work_item_id": "wi-0001",
  "scenario_ids": ["AC-1"],
  "interface": "OrderService.createOrder",
  "test_file": "src/test/java/example/OrderServiceTest.java",
  "test_name": "createsOrderForValidRequest",
  "kind": "unit",
  "status": "authored"
}
```

### Actor Write-Scope Enforcement

Prompts alone are not sufficient enforcement. Supervisor MUST combine
platform-supported prevention with a portable, mandatory changed-path check.

- Analysis, Compiler, and Verifier are read-only.
- Coder may write only production paths declared by the Implementation
  Contract.
- Tester may write only declared test paths.
- Before each mutating Actor starts, Supervisor MUST snapshot tracked,
  untracked, deleted, and renamed paths in the Task worktree.
- After the Actor returns, Supervisor MUST compare the complete changed-path
  set with that Actor's allowed path patterns.
- Where the Agent backend supports filesystem sandboxing, Supervisor SHOULD
  restrict writable roots before execution. The post-execution path check is
  still mandatory on every platform.
- When project layout does not allow Analysis to determine safe path patterns,
  it MUST open a Gate rather than grant repository-wide write access.
- A confirmed violation MUST persist Evidence, open an
  `actor_write_scope_violated` Gate, preserve the worktree, and stop stage
  advancement. Supervisor MUST NOT automatically roll back the files.

Example contract fragment:

```yaml
write_scopes:
  coder:
    - internal/**
    - cmd/**
    - go.mod
  tester:
    - internal/**/*_test.go
    - tests/**
  analysis: []
  compiler: []
  verifier: []
```

### Tester Actor: Final Execution Stage

After all Work Items reach `READY_FOR_FINAL_TEST`, Tester selects the narrowest
command or bounded command set covering the recorded unit-test cases. Invocation
of `supervise start` under a policy that enables final unit tests is the
authorization for this Task-scoped execution; broader scope still requires a
separate policy or Gate resolution.

Requirements:

- Record the exact command, working directory, environment restrictions,
  duration, exit code, and output Evidence.
- Map each failure to its owning Work Item and interface when possible.
- Return mapped production failures to Coder.
- Recompile repaired production code before rerunning affected tests.
- Rerun the smallest affected test scope first.
- Do not silently widen to integration, system, repository-wide, or
  network-dependent tests.

Final unit tests MUST use a structured, immutable `TestExecutionPlan`. Tester
describes test intent; it MUST NOT supply an arbitrary shell command. A trusted
runner Adapter validates the plan and materializes an argv array without a
shell.

```json
{
  "schema_version": 1,
  "profile": "unit",
  "checks": [
    {
      "id": "unit-order-service",
      "runner": "go-test",
      "working_directory": ".",
      "targets": ["./internal/order"],
      "selectors": ["TestCreateOrder", "TestDuplicateOrder"],
      "timeout_seconds": 120
    }
  ],
  "network": {
    "mode": "external-denied",
    "loopback": "allowed"
  }
}
```

Plan requirements:

- `working_directory` MUST resolve inside the Task worktree.
- `runner` MUST be registered and allowed by policy.
- Targets and selectors MUST map to recorded Task unit-test cases.
- Environment variables MUST use an allowlist.
- The validated plan digest MUST be persisted before execution. Any plan
  change requires validation and a new digest.
- The exact materialized argv, directory, timeout, environment policy, output,
  duration, and exit code MUST be recorded as Evidence.

The default network policy denies external network, LAN access, and DNS while
allowing loopback and local IPC for in-process unit-test fixtures. An execution
backend MUST report whether it can enforce that policy. Container networking,
Linux network namespaces, or an Agent sandbox MAY provide enforcement.

If enforcement is unavailable, Supervisor MUST fail closed with a
`network_isolation_unavailable` Gate. Only a human may grant a recorded,
one-time `best-effort` waiver. Environment capability failure and waiver do not
consume a Work Item Attempt.

### Verifier Actor

Verifier is optional and initially MAY be implemented as a scaffold.

- Runs only after collected unit tests pass.
- May select a project-level test suite or additional checks such as `go vet`
  when enabled by policy.
- Records command and result Evidence.
- Reports failures, including obsolete tests and external-service dependencies.
- Does not return the Task to Coder automatically and does not block delivery
  unless a future policy explicitly promotes a Verifier result to a Gate.

## Attempts And Retry Policy

An Attempt represents one bounded Work Item implementation-and-repair cycle,
not each individual Actor process. Analysis clarification, Coder work, compile,
test authoring, and failure feedback remain linked to the same Attempt until it
passes or a new repair cycle begins.

- Default maximum Attempts per Work Item: `3`.
- Configurable range: `1` through `5`.
- A repository or Task MAY choose another value in that range.
- Plugin notification retry, state-file recovery, and delivery retry do not
  consume Work Item Attempts.
- A duplicate prepared Agent request for the same Work Item and Attempt MUST be
  resumed or finished; Supervisor MUST NOT create a second Attempt.
- Exhaustion opens a `retry_exhausted` Gate and pauses Supervisor.
- Human-directed force close is separate from retry exhaustion.

## Execution Policy

The new workflow does not preserve the previous blanket prohibition on
automated commit, merge, cleanup, or Task-scoped tests. Authorization is
captured in a persisted Task execution policy.

Example:

```json
{
  "schema_version": 1,
  "compile": "allowed",
  "final_unit_tests": "allowed",
  "project_suite": "report_only",
  "delivery_mode": "local-merge",
  "auto_commit": true,
  "auto_merge": true,
  "push": false,
  "create_pull_request": false,
  "cleanup_after_merge": true,
  "network": "denied",
  "max_work_item_attempts": 3
}
```

The effective policy MUST be printed and persisted before the first mutating
Actor starts. A policy change during execution MUST be recorded as an event and
MUST NOT retroactively rewrite Evidence.

### Configuration And Authorization Precedence

Ordinary configuration uses this precedence, from lowest to highest:

```text
built-in defaults
  < user configuration
  < repository configuration
  < Task request configuration
  < supervise command-line overrides
```

Ordinary values include model selection, Plugin name, timeouts, Attempt limits,
and state/worktree roots. Values remain subject to hard validation such as the
`1` through `5` Attempt limit.

Capabilities with security or external side effects MUST use
`forbidden`, `requires-confirmation`, or `allowed` semantics instead of ordinary
last-writer-wins precedence. This includes tests, network, commit, parent-branch
mutation, push, pull-request publication, cleanup, and force-close delivery.

- A repository `forbidden` value cannot be loosened by Task configuration,
  environment variables, or command-line arguments.
- Task policy may grant a capability only within repository policy.
- An explicit command-line choice may satisfy a `requires-confirmation` state
  when it identifies the human authorization and is persisted.
- Environment variables MUST NOT widen capabilities. They may provide runtime
  paths, backend capability declarations, credential references, or stricter
  restrictions.
- Secret values MUST NOT be persisted. State records only a reference such as
  `env:GITHUB_TOKEN`.

Supervisor resolves and displays the effective policy once before the first
mutating Actor, then stores its immutable snapshot and digest. Resume uses that
snapshot rather than silently re-resolving changed configuration or environment
variables. A later policy change requires an explicit operation and a
`policy.updated` event containing old and new digests, actor, and reason.

## Acceptance

A Work Item is accepted only when:

1. Its implementation contract is ready.
2. Coder produced valid structured output.
3. Compile Evidence passed.
4. Observable interfaces were recorded.
5. Required unit tests were authored and mapped to acceptance scenarios.
6. Final Tester Evidence passed for its mapped tests.
7. No unresolved Work Item Gate remains.

A Task is ready for delivery only when all Work Items are accepted, required
final unit tests pass, final artifact reconciliation succeeds, and no blocking
Task Gate remains. Optional Verifier failures are reported separately and MUST
NOT be presented as unit-test success or failure.

## Git Commit And Delivery

After acceptance, DeliveryCoordinator MUST:

1. Inspect the Task diff and reject unrelated or unresolved-conflict files.
2. Reconcile managed requirement projections.
3. Commit Task worktree changes to `ai/<task-id>`.
4. Record commit IDs and tree identity.
5. Execute the initial `local-merge` delivery.

### `local-merge`

- Re-read the recorded `parent_branch` and its current HEAD immediately before
  delivery. `parent_commit` remains the immutable Task development baseline;
  the branch's current HEAD is the delivery target.
- If the parent checkout changed branch, is dirty, or otherwise no longer meets
  safe merge preconditions, open a `parent_delivery_precondition_changed` Gate
  instead of terminating or overwriting the external work.
- Probe whether `ai/<task-id>` can merge into the latest valid HEAD of the
  recorded `parent_branch`.
- This is the only initial delivery mode allowed to update the local parent
  branch or its checkout.
- On a clean result, merge into the local parent branch.
- The initial merge direction is Task branch into `parent_branch`. Merging the
  parent into the Task worktree is a conflict-recovery step, not a prerequisite
  for every delivery.
- Verify that the Task commit is an ancestor of the resulting parent commit.
- Persist delivery Evidence.
- Clean the worktree and task branch only after verified merge.
- Notify the human that the local parent branch is ready to review and push.

### Future `aiw-github` Plugin Delivery

Pull-request delivery is outside the initial Core implementation. A future
delivery policy MAY select the existing Python `aiw-github` Plugin instead of
`local-merge`; the two paths are mutually exclusive for one delivery attempt.

When selected, `aiw-github` owns all GitHub publication behavior required for
the pull request:

- pushing `ai/<task-id>` to the configured GitHub remote;
- creating or reusing the pull request;
- querying pull-request and merge status;
- resolving GitHub credentials through its own configuration or environment;
- returning a structured, idempotent result to Core.

Core owns authorization, the persisted delivery handoff, request/result
Evidence, and cleanup eligibility. Core MUST NOT read or persist credential
values, implement GitHub-specific commands, independently push the branch, or
perform `local-merge` after Plugin delivery has started. The Plugin MUST NOT
force push. Local resources remain until the Plugin reports a verified merge.

### Merge Conflicts

Target behavior below differs from current local delivery: the implementation
map above describes the existing-worktree candidate and manual Skill handoff.

Conflict policy: preserve the conflict state and notify a human.

If a merge probe or merge reports conflicts:

1. Stop automatic delivery and preserve the recorded Task branch and worktree.
   Do not create a temporary branch or worktree, invoke a conflict-resolution
   Agent, or automatically select either side of a conflict.
2. If this delivery started a merge in the parent checkout, abort only that
   merge to restore the parent checkout. If restoration fails, stop immediately
   and report the actual remaining state; do not reset, clean, or continue.
3. In the existing Task worktree, merge the recorded `parent_branch` into the
   Task branch with `--no-commit --no-ff`. Preserve its unresolved index and
   working files for human resolution. Do not restart or abort a merge already
   awaiting human resolution. If the conflict cannot be reproduced, report that
   result and wait for review instead of claiming it was resolved.
4. Persist failure evidence and notify the human through NotificationDispatcher.
   Include the Task ID, both branch names and commit IDs, the actual workspace
   path, conflict files, merge direction, parent-restoration result, and recovery
   commands. In the Task worktree, current/ours is the Task branch and
   incoming/theirs is the recorded parent branch. Report other failures as their
   actual error, not as an assumed merge conflict.
5. Keep delivery pending and supervision paused. Do not commit a resolution,
   retry delivery, or clean Task resources automatically while human action is
   outstanding. Notification retries do not retry Git operations.

Human recovery:

1. Open the reported Task worktree and inspect `git status` and each conflict.
2. Resolve the files while preserving the required behavior of both branches;
   validate the combined result under the project's execution policy.
3. Stage the resolved paths (including intended deletions), confirm
   `git diff --name-only --diff-filter=U` is empty, and commit the pending merge.
4. Return to the primary workspace, review the result, and explicitly run
   `aiw workflow local-merge <task-id> "Complete Task <task-id>"`.
5. Delivery rechecks the latest recorded parent branch and its clean-workspace
   preconditions. Only a verified successful merge permits Task resource cleanup.

Humans may request Agent assistance separately. Using
`resolving-merge-conflicts` is optional and is not an automatic delivery stage
or evidence that the combined behavior is correct.

## Notification Through Existing AIW Plugins

AIW already discovers and executes Python `aiw-plugins`. Notification MUST use
that mechanism rather than introduce a second Plugin framework.

The stable Plugin command interface SHOULD be:

```text
aiw notify emit --event-file <absolute-path>
```

Core dispatches it through the existing Plugin discovery and execution modules,
equivalent to discovering `aiw-notify.py`. AIW distributions SHOULD ship a
default Console implementation. A configured replacement Plugin may route to
Slack, GitHub, email, or another system without changing Supervisor Core.

Core MUST persist before dispatch:

```text
Supervisor -> notification event file -> Python Plugin -> delivery receipt
```

Suggested environment:

```text
AIW_TASK_ID
AIW_TASK_DIR
AIW_REPO_ROOT
AIW_EVENT_ID
AIW_EVENT_PATH
```

Example notification event:

```json
{
  "schema_version": 1,
  "event_id": "evt-20260914-001",
  "task_id": "example-task",
  "type": "local_merge_completed",
  "severity": "info",
  "created_at": "2026-09-14T12:00:00+09:00",
  "message": "Task branch was merged into the local parent branch.",
  "context": {
    "source_branch": "ai/example-task",
    "target_branch": "develop",
    "merge_commit": "def456"
  },
  "actions": [
    {
      "type": "command",
      "label": "Review parent branch",
      "value": "git status"
    }
  ]
}
```

Notification delivery states:

```text
PENDING -> DISPATCHING -> DELIVERED
                      `-> FAILED_RETRYABLE
```

- Notification retry MUST NOT consume a Work Item Attempt.
- Ordinary notification failure MUST NOT rerun Coder, Compiler, or Tester.
- A human-action notification failure leaves the Task in its correct
  `AWAITING_HUMAN` state and adds a visible `notification_failed` condition.
- Existing Plugin metadata is informative unless Workflow Core enforces it.
  Supervisor policy, not Plugin self-declaration, decides whether network,
  push, pull-request creation, or another external side effect is allowed.

## Stop, Recovery, And Force Close

### Stop

Stop prevents further scheduling and releases the Supervisor lease when safe.
It preserves the current Work Item, Attempt, task branch, worktree, state, and
prepared Agent request.

### Recovery

Recovery MUST inspect and reconcile:

- pending state events;
- Supervisor and write leases;
- prepared or running Actor requests;
- Session final output;
- partially recorded Evidence;
- Plugin delivery receipts;
- partial Git delivery state.

Recovery is idempotent. It resumes or finalizes the owning Attempt and MUST NOT
create another Attempt for the same prepared work.

### Force Close

A human MAY explicitly force close a Task that cannot complete normally. Force
close MUST NOT fabricate compile, test, verification, or merge success.

It MUST:

1. Stop new Actor scheduling.
2. Record the force-close actor, reason, unresolved Gates, skipped stages, and
   last valid Evidence.
3. Inspect and preserve already completed changes.
4. Apply an explicitly selected delivery action, such as commit current work,
   local merge, preserve the Task branch, or discard. No delivery action may be
   inferred. Future GitHub publication remains an `aiw-github` Plugin action.
5. Clean the worktree and branch only when the selected delivery action makes
   cleanup safe and ancestry or discard Evidence is recorded.
6. Close the Task with a distinct `FORCE_CLOSED` outcome, not `PASSED`.
7. Persist and dispatch a final notification.

## Suggested Task State

```json
{
  "schema_version": 1,
  "task_id": "example-task",
  "phase": "IMPLEMENTING",
  "outcome": null,
  "current_work_item": "wi-0002",
  "current_stage": "COMPILING",
  "supervisor_lease": "sup-...",
  "write_lease": {
    "owner_type": "attempt",
    "owner_id": "attempt-..."
  },
  "pending_event": null,
  "delivery_mode": "local-merge",
  "notification_condition": null,
  "updated_at": "2026-09-14T12:00:00+09:00"
}
```

Suggested Task phases:

```text
DRAFT
READY
IMPLEMENTING
FINAL_TESTING
VERIFYING
DELIVERING
AWAITING_HUMAN
COMPLETED
FORCE_CLOSED
```

## Implementation Slices

Implementation SHOULD proceed in small vertical slices:

1. Canonical `.ai/<task-id>/` store, migration/read compatibility, atomic
   events, and recovery.
2. WorkspaceCoordinator with `ai/<task-id>` and recorded parent ancestry.
3. SkillRegistry and persisted Actor routing manifest.
4. Implementation Contract validation and Analysis/Coder handoff.
5. Compiler Actor, deterministic compile runner, and repair loop.
6. Observable-interface inventory and Tester authoring handoff.
7. Collected final unit-test execution, failure mapping, and acceptance Gate.
8. Verifier scaffold and report-only Evidence.
9. Artifact ownership/projection changes for `tasks.md` and `task.toml`.
10. Commit and verified `local-merge` delivery.
11. Existing Python Plugin-based notification and retry.
12. Conflict-resolution candidate workspace.
13. Stop, recover, and explicit force-close delivery choices.
14. Future `aiw-github` Plugin handoff for push and pull-request delivery;
    outside the initial Core implementation.

## Confirmed Decisions

- Actor writes use platform sandboxing when available plus mandatory Git
  changed-path validation. Violations preserve evidence and open a Gate without
  automatic rollback.
- Final unit tests use a structured `TestExecutionPlan`. External network is
  denied, loopback is allowed, unavailable enforcement fails closed, and only a
  human may grant a recorded one-time waiver.
- Workflow Core updates the Task worktree's `tasks.md` after each accepted Work
  Item. Supervisor never directly writes the parent checkout's `tasks.md`;
  normal Git Delivery may merge the committed update into the parent branch.
- Ordinary configuration uses defaults, user, repository, Task, then CLI
  precedence. Capability authorization uses explicit states; environment
  variables cannot widen permissions, and effective policy is snapshotted.
- The initial Core implementation provides only verified `local-merge` delivery.
  Future push, pull-request creation, and status queries belong to the existing
  Python `aiw-github` Plugin, including the push required to publish the Task
  branch. Core never persists provider credential values.
