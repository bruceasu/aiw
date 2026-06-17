# AIW CLI Refactoring and `cz` Rebuild Implementation Plan

## Objective

Refactor the current CLI into the structure:

```text
main -> internal/commands/* -> internal/*
```

and rebuild the top-level `aiw cz` capability within the new architecture.

The implementation is divided into two phases:

1. Complete command structure migration.
2. Gradually rebuild a TUI + AI experience inside `internal/commands/cz` that approaches the reference project's UX.

---

## Confirmed Constraints

* `cz` resides in `internal/commands/cz`
* `git`, `wt`, and `tcc` reside in:

  * `internal/commands/git`
  * `internal/commands/wt`
  * `internal/commands/tcc`
* OpenSpec-like commands reside in `internal/commands/task`
* Shared functionality resides under `internal/*`
* `aiw git cz` will not be retained
* `cz` only supports this project's own configuration format
* TUI is a key selling point and should closely resemble the current interaction model of the reference project
* Dependency changes are considered high-risk; modifications to `go.mod` or `go.sum` require explicit approval before implementation

---

## Current Progress

The following structural migration has already been completed:

* `main.go` has been delegated to the new `internal/commands/*` hierarchy
* `internal/commands/task` now handles task-like top-level commands
* `internal/commands/wt` now handles worktree commands
* `internal/commands/tcc` now handles tcc commands
* `internal/commands/git` now handles git shortcut commands
* `internal/commands/cz` has completed the initial linear-version migration
* Initial shared utilities have been extracted into:

  * `internal/envx`
  * `internal/gitx`
  * `internal/taskx`
  * `internal/fsx`

The current focus is now limited to:

1. Further splitting the internal structure of `internal/commands/cz`
2. Introducing a real searchable TUI
3. Completing advanced `cz` functionality and documentation

---

## Target Directory Structure

```text
aiw/
├─ main.go
├─ internal/commands/
│  ├─ cz/
│  ├─ git/
│  ├─ wt/
│  ├─ tcc/
│  └─ task/
└─ internal/
   ├─ envx/
   ├─ fsx/
   ├─ gitx/
   ├─ taskx/
   ├─ textx/
   └─ cmdx/
```

---

# Remaining Tasks

## Task 9: `internal/commands/cz` TUI Phase 1

### Objective

Introduce the first batch of genuine interactive TUI capabilities without breaking the existing linear `cz` implementation.

### Approval Gate

This task will modify:

* `go.mod`
* potentially `go.sum`

Therefore it is considered a high-risk change and must receive user approval before implementation.

### Recommended Dependency

Preferred:

* `bubbletea`

Alternative (not recommended):

* `tview`
* Direct implementation on top of `tcell`

Reasons for choosing `bubbletea`:

* Better suited for custom searchable workflows
* Easier to emulate the `cz-git` style UX
* Lower risk than building directly on top of a low-level event system

---

### Phase 1 Scope

The first phase is intentionally small and delivers only a minimal end-to-end workflow.

1. Keep the existing `lineUI`
2. Add a new `teaUI`
3. Implement only `SearchList`
4. Use it only for the `type` selection step
5. Continue automatically falling back to `lineUI` in non-interactive terminals

---

### Planned File Changes

* Modify: `go.mod`
* Create or Modify: `internal/commands/cz/tui.go`
* Modify: `internal/commands/cz/command.go`
* Modify: `internal/commands/cz/flow.go`
* Modify: `internal/commands/cz/command_test.go`
* Create: `internal/commands/cz/tui_test.go`

---

### Implementation Steps

#### Step 1: Define a Unified UI Abstraction

Inside `internal/commands/cz`:

* Create a common UI abstraction
* Allow both `lineUI` and future `teaUI` to share the same interface
* Initially cover only `SearchList`

---

#### Step 2: Introduce TUI Dependencies

* Update `go.mod`
* Add only the minimal dependencies required to implement `SearchList`

---

#### Step 3: Implement `SearchList`

Requirements:

* Input-based filtering
* Up/down navigation
* Enter-to-confirm
* Empty-result feedback
* Default-item highlighting

---

#### Step 4: Switch Type Selection to `SearchList`

Only replace:

```text
type selection
```

All remaining steps continue to use the current linear workflow.

---

#### Step 5: Add Tests

Cover:

* Filtering logic
* Workflow behavior
* Fallback behavior

---

#### Step 6: Full Validation

```bash
go test ./internal/commands/cz
go test ./...
go build ./...
go vet ./...
```

---

### Definition of Done

Task 9 Phase 1 is complete only when all of the following are true:

1. `aiw cz` uses a searchable list for `type` selection in interactive terminals
2. Non-interactive terminals continue to work
3. Existing `internal/commands/cz` tests do not regress
4. Repository-wide build, test, and vet all pass

---

## Task 10: `internal/commands/cz` TUI Phase 2

### Objective

Evolve the TUI from "usable" to "visibly close to the reference project's experience."

---

### Scope

* `SearchCheckbox`
* Custom scope / predefined scope selection
* AI candidate selection
* Subject length feedback
* Preview / edit / cancel workflow

---

### Planned File Changes

* Modify: `internal/commands/cz/tui.go`
* Modify: `internal/commands/cz/flow.go`
* Modify: `internal/commands/cz/session.go`
* Modify: `internal/commands/cz/message.go`
* Create or Modify: `internal/commands/cz/flow_test.go`
* Create or Modify: `internal/commands/cz/tui_test.go`

---

### Implementation Steps

#### Step 1

Implement `SearchCheckbox`.

#### Step 2

Switch `scope` selection to searchable single-select or multi-select.

#### Step 3

Implement AI candidate list selection.

#### Step 4

Add subject length feedback during input.

#### Step 5

Implement preview / edit / cancel interaction flow.

#### Step 6

Add workflow and edge-case tests.

#### Step 7

Run full validation.

---

## Task 11: Advanced `cz` Features

### Objective

Complete advanced behaviors and configuration capabilities.

---

### Scope

* Issue-prefix workflow
* Breaking-change mode
* Retry support
* More complete `aiw.toml` configuration
* Help text and README updates

---

### Planned File Changes

* Modify: `internal/commands/cz/config.go`
* Modify: `internal/commands/cz/flow.go`
* Modify: `internal/commands/cz/help.go`
* Modify: `README.md`
* Modify: `docs/design.md`

---

### Implementation Steps

#### Step 1

Implement:

* Issue prefix support
* Breaking-change mode

#### Step 2

Implement retry support.

#### Step 3

Extend `aiw.toml`.

#### Step 4

Update help documentation and README.

#### Step 5

Run full validation.

---

# Validation Matrix

## Repository-Level Validation

```bash
go test ./...
go build ./...
go vet ./...
```

Checklist:

* [ ] `go test ./...`
* [ ] `go build ./...`
* [ ] `go vet ./...`

---

## `cz`-Specific Validation

* [ ] Clear message when no staged changes exist
* [ ] Searchable `type` selection works
* [ ] Searchable `scope` selection works
* [ ] Multi-select checkbox works
* [ ] AI candidate selection works
* [ ] Subject length feedback works
* [ ] External editor integration still works
* [ ] Preview supports commit, edit, and cancel
* [ ] Automatic fallback in non-interactive terminals works

---

# Risk Control

* Structural migration is largely complete; avoid incidental changes to unrelated modules
* Replace TUI functionality incrementally:

  * Start with `SearchList`
  * Expand gradually to the full workflow
* Keep `lineUI` until `teaUI` is fully mature
* Do not modify `go.mod` before approval is granted
* Windows terminal compatibility must be included in validation

---

# OpenSpec Notes

The current repository does not contain usable content under:

```text
openspec/changes/<task>/
openspec/specs/
```

for further alignment.

If strict OpenSpec-lite traceability is desired later, it is recommended to create a dedicated task directory for the `cz` rebuild and move the TODO items and verification checklist into that structure.
