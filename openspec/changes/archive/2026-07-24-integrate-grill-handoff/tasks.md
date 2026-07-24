## 1. Workspace Context

- [x] 1.1 Implement bounded cross-platform workspace tree and metadata collection
- [x] 1.2 Implement deterministic credential redaction and context collector tests

## 2. Session Artifacts

- [x] 2.1 Add locked atomic artifact text read/write helpers to SessionStore
- [x] 2.2 Implement deterministic handoff rendering with bounded output references

## 3. CLI Workflows

- [x] 3.1 Add built-in Grill instructions, initial prompt composition, and `grill` command
- [x] 3.2 Add `handoff create` and `handoff show` commands
- [x] 3.3 Add CLI tests for successful and failing Grill and handoff paths

## 4. Distribution and Documentation

- [x] 4.1 Document the Grill, context, and handoff workflows
- [x] 4.2 Mirror runtime changes into the packaged `plugins/aiw-flow` source

## 5. Verification

- [x] 5.1 Run Python compile checks and focused unit tests
- [x] 5.2 Run the complete aiw-flow test suite and CLI smoke tests
- [x] 5.3 Verify development and packaged runtime source parity

## TODO

- [x] Record any implementation follow-ups as `%%` notes without expanding this change

%% Follow-up: automatic Memory extraction and optional AI-rendered handoff remain separate future changes, as recorded in design.md.

%% Environment: pytest is not installed in the available Python 3.9 runtime. No dependency was added; the complete unittest-compatible suite was run with standard-library discovery.

## Verification

- [x] Record the exact commands and observed results before completion

- `python -m py_compile src/codex_flow/cli.py src/codex_flow/grill.py src/codex_flow/handoff_manager.py src/codex_flow/workspace_context.py src/codex_flow/session_store.py` — passed.
- `$env:PYTHONPATH='src'; python -m unittest discover -s tests -p "test_workspace_context.py" -v` — 4 tests passed.
- `$env:PYTHONPATH='src'; python -m unittest discover -s tests -p "test_session_store.py" -v` — 5 tests passed.
- `$env:PYTHONPATH='src'; python -m unittest discover -s tests -p "test_handoff_manager.py" -v` — 2 tests passed.
- `$env:PYTHONPATH='src'; python -m unittest discover -s tests -p "test_grill.py" -v` — 3 tests passed.
- `$env:PYTHONPATH='src'; python -m unittest discover -s tests -p "test_cli.py" -v` — 6 tests passed.
- `python -m compileall -q src tests` — passed.
- `$env:PYTHONPATH='src'; python -m unittest discover -s tests -v` — 27 tests passed.
- `$env:PYTHONPATH='src'; python -m codex_flow --help` — passed and listed `grill` and `handoff`.
- `$env:PYTHONPATH='src'; python -m codex_flow grill --help` — passed.
- `$env:PYTHONPATH='src'; python -m codex_flow handoff --help` — passed.
- `python plugins\aiw-flow\aiw-flow.py --help`, `grill --help`, and `handoff --help` — passed against the packaged plugin entry point.
- `git diff --check` — passed; Git reported only line-ending conversion warnings.
- Development/package SHA-256 parity check for all top-level runtime Python files — `PARITY_OK`.
- `openspec.cmd validate integrate-grill-handoff --type change --strict --no-interactive` — passed.
