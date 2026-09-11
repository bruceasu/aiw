# Tasks

## 1. Command and configuration

- [x] 1.1 Add top-level `aiw ask` registration, argument parsing, friendly
  noninteractive output, and help for `--chat`, `--resume`,
  `--system-prompt`, `--system-prompt-file`, and repeatable `--allow-path`.
- [ ] 1.2 Add private ask configuration resolution from
  `$HOME/.aiw/ask/config.toml` with documented command-line precedence.
- [ ] 1.3 Reuse `ai.RunLLMWithSystemPrompt` with the version 1.0 answer schema
  and map core configuration into its `LLMConfig` input.

## 2. Safety, response, and persistence

- [x] 2.1 Implement the versioned answer model, JSON validation, status mapping,
  and friendly renderer; malformed output must become `error` without retry.
- [ ] 2.2 Implement read-root validation and one-time Chat confirmation;
  general guidance must not collect workspace files.
- [x] 2.3 Implement private session Markdown persistence, Windows-safe file
  names, successful/failed-turn rules, and latest-session lookup.
- [ ] 2.4 Pass Codex read-only/safety flags and authorized `--add-dir` paths;
  pass Copilot allowed paths; surface Codex read-scope limitations.

## 3. Chat UX

- [ ] 3.1 Implement promptui-assisted line accumulation with `/send`, `/exit`,
  empty-line preservation, and `Ctrl+C` cancellation.
- [ ] 3.2 Implement `--resume` continuation and no-session fallback behavior.

## 4. Documentation and focused tests

- [ ] 4.1 Document command usage, answer schema, memory location, privacy
  boundary, Chat shortcuts, and provider limitations.
- [ ] 4.2 Add focused tests for schema validation, path policy, persistence,
  resume selection, and Chat command parsing.

## 5. Verification

- [ ] 5.1 Statically trace CLI registration, LLM call, path policy, renderer,
  and session writer before handoff.
- [ ] 5.2 Run one authorized focused test command after implementation.
- [x] 5.3 Normalize the implementation-contract path before Workflow Core
  projects it, and cover a relative `tasks.md` path on Windows.

## Notes

%% DEFERRED: Verify Codex version-specific read-scope behavior and document the
limitation; do not treat `--add-dir` as a strict read allowlist.

## TODO

- Complete private config precedence, provider path flags, robust resume file
  selection, and focused tests.

## Verification

- Static review completed for CLI registration, structured schema request,
  renderer, and private Markdown writer.
- Runtime tests/builds intentionally not run under the project resource budget.
- Static review completed for the implementation-contract projection path fix;
  focused regression test added but not run.

<!-- aiw:workflow-summary:start -->
## AIW Workflow Summary
- Status: `DRAFT`
- Planning: `draft`; execution: `queued`; validation: `not-required`; delivery: `unmanaged`
- Work Item mappings:
  - `1.1` -> `wi-0001` (completed)
- [x] 1.1 — Add top-level `aiw ask` registration, argument parsing, friendly
  - `1.2` -> `wi-0002` (ready)
- [ ] 1.2 — Add private ask configuration resolution from
  - `2.1` -> `wi-0003` (completed)
- [x] 2.1 — Implement the versioned answer model, JSON validation, status mapping,
  - `3.1` -> `wi-0004` (ready)
- [ ] 3.1 — Implement promptui-assisted line accumulation with `/send`, `/exit`,
  - `3.2` -> `wi-0005` (ready)
- [ ] 3.2 — Implement `--resume` continuation and no-session fallback behavior.
  - `1.3` -> `wi-0006` (ready)
- [ ] 1.3 — Reuse `ai.RunLLMWithSystemPrompt` with the version 1.0 answer schema
  - `2.2` -> `wi-0007` (ready)
- [ ] 2.2 — Implement read-root validation and one-time Chat confirmation;
  - `2.3` -> `wi-0008` (completed)
- [x] 2.3 — Implement private session Markdown persistence, Windows-safe file
  - `2.4` -> `wi-0009` (ready)
- [ ] 2.4 — Pass Codex read-only/safety flags and authorized `--add-dir` paths;
  - `4.1` -> `wi-0010` (ready)
- [ ] 4.1 — Document command usage, answer schema, memory location, privacy
  - `4.2` -> `wi-0011` (ready)
- [ ] 4.2 — Add focused tests for schema validation, path policy, persistence,
  - `5.1` -> `wi-0012` (ready)
- [ ] 5.1 — Statically trace CLI registration, LLM call, path policy, renderer,
  - `5.2` -> `wi-0013` (ready)
- [ ] 5.2 — Run one authorized focused test command after implementation.
  - `5.3` -> `wi-0014` (completed)
- [x] 5.3 — Normalize the implementation-contract path before Workflow Core
- Implementation contract: `openspec/changes/aiw-ask/implementation-contract.md`
<!-- aiw:workflow-summary:end -->
