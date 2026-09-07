# Tasks

## Plan and contracts

- [x] Create the migration plan and define Session/Core/backend ownership.
- [x] Define the Go Session state schema and valid lifecycle transitions.
- [x] Define Codex, Copilot, and OpenAI backend request/result contracts.

## Go Session Core

- [x] Add canonical `.ai/sessions` storage, atomic status writes, and per-Session locks.
- [x] Add prompt snapshots, memory updates, event records, outputs, and artifacts.
- [x] Add finish, archive, delete, and deterministic handoff operations.

## AI backends

- [x] Implement Codex CLI backend with thread extraction and raw event capture.
- [x] Implement Copilot CLI backend with session resume support.
- [x] Implement OpenAI API backend without persisting credentials.
- [x] Add backend selection and configuration through AIW flow options/environment.

## CLI and integration

- [x] Add `aiw flow` command routing and lifecycle subcommands.
- [x] Rewire `task agent next` to call the Go Session Core.
- [x] Keep `cxs` as native Codex session navigation and remove flow overlap.
- [x] Remove the Python `aiw-flow` implementation and stale plugin packaging.
- [x] Update help, README, and AI-support specification references.

## Verification

- [x] Static review confirms no production Go code invokes `aiw-flow`.
- [x] Static review confirms secrets are not written to Session status or logs.
- [x] Run one focused Go static validation command.
- [x] Record remaining provider-format risks in `plan.md`.
