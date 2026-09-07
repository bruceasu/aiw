# Go Core Flow Migration Plan

## Goal

Replace the Python `aiw-flow` plugin with a Go-native AIW Core workflow while
preserving the useful Session, prompt, memory, handoff, event, and artifact
semantics. OpenSpec-lite remains authoritative for requirements and checklists;
AIW Core becomes authoritative for Task-bound Session lifecycle.

## Scope

- Add a Go `aiw flow` command group.
- Add a reusable Go Session Core for state, locks, transitions, prompts, memory,
  events, outputs, and handoffs.
- Add execution backends for Codex CLI, GitHub Copilot CLI, and OpenAI API.
- Rewire `task agent next` to the Go Core instead of spawning `aiw-flow`.
- Keep the existing `cxs` command focused on native Codex session navigation.
- Remove the Python `aiw-flow` implementation and its standalone packaging after
  the Go path is wired.
- Use `.ai/sessions` as the canonical AIW Session state location.

## Design Decisions

1. Session Core owns AIW Session lifecycle and persisted state; it does not own
   OpenSpec proposal/design/spec content.
2. Backend implementations are adapters behind one Go interface. Provider
   authentication and model availability remain provider-owned.
3. Codex and Copilot adapters invoke their installed CLIs. The OpenAI adapter
   uses the official `github.com/openai/openai-go` SDK and maps the Session
   thread field to the Responses API `previous_response_id`.
4. Existing JSON field names are retained where practical so old Session state
   can be inspected during migration, but the Python executable is not retained
   as a runtime compatibility layer.
5. `task agent next` uses the same Session and starts a fresh backend thread when
   requested; worktree and Task lifecycle remain owned by AIW.

## Implementation Order

1. Create this plan and OpenSpec task checklist.
2. Implement `internal/session` models, store, locks, transitions, prompt and
   artifact helpers.
3. Implement backend adapters and backend selection/configuration.
4. Implement `internal/commands/flow` CLI commands: `new`, `run`, `continue`,
   `status`, `list`, `finish`, `archive`, `delete`, and `memory`.
5. Route `aiw flow` from `main.go` and update help output.
6. Replace `task agent` external `aiw-flow` calls with the Go Core.
7. Remove the Python `aiw-flow` source/package and stale plugin entry points.
8. Update stable AI-support documentation and complete the checklist.

## Verification

- Static review of changed Go call paths and state ownership.
- One focused read-only validation command after implementation.
- Confirm no production Go code invokes the `aiw-flow` executable.
- Confirm `aiw flow --help` and `task agent` routing are internally wired.
- Confirm OpenSpec TODO and Verification checklists are updated.

## Current Status

The Go Session Core, Codex/Copilot CLI adapters, official OpenAI Go SDK
backend, `aiw flow` routing, and `task agent` integration are implemented.

The Python source is removed after this migration; `aiw flow` is the only
supported flow execution surface.

## Risks

- Codex and Copilot CLI output formats can change; parsers must tolerate unknown
  events and retain raw output.
- OpenAI API authentication and endpoint configuration must never be persisted
  in Session status or printed in errors.
- Removing Python files is a breaking installation change and requires the Go
  binary/plugin installation path to be available.
- Existing Session state may contain Python-specific fields; unknown JSON fields
  must remain readable.
