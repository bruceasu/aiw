## Why

aiw-flow can run and resume Codex sessions, but it does not provide a guided requirement interview or a portable session handoff. Adding these capabilities makes early requirement discovery and cross-session continuation reliable without introducing another model SDK or storing state outside the session.

## What Changes

- Add a built-in Grill workflow that starts a session with a requirement, inspects the declared workspace, and asks at most one decision question per turn.
- Add deterministic workspace context collection with strict file, byte, and sensitivity limits.
- Add `handoff create` and `handoff show` commands that build and display a session-owned Markdown handoff artifact.
- Store context and handoff artifacts inside the existing session directory using atomic writes and session locks.
- Keep all existing commands and status schema compatible.
- Update the packaged plugin copy, documentation, and tests.

## Capabilities

### New Capabilities

- `grill-workflow`: Guided, resumable, one-question-at-a-time requirement discovery using the existing Codex exec backend.
- `session-handoff`: Deterministic creation and display of a session handoff document from stored session facts.
- `workspace-context`: Cross-platform, bounded, sensitive-file-aware collection of workspace metadata.

### Modified Capabilities

None.

## Impact

- Affects `program/aiw-flow` CLI, prompt composition, workspace helpers, session artifacts, tests, and README.
- Mirrors the completed source changes into `plugins/aiw-flow`.
- Adds no third-party dependency and makes no direct OpenAI API call.
- Adds new CLI commands and options without changing existing command behavior.
