## 1. Loop Input Model

- [x] 1.1 Implement deterministic interactive input and slash-command parsing
- [x] 1.2 Add parser tests for messages, empty input, known commands, and unknown commands

## 2. Loop Coordination

- [x] 2.1 Add `loop SESSION_ID` argument parsing and phase resolution
- [x] 2.2 Implement sequential turn execution and clean idle exit behavior
- [x] 2.3 Implement local help, status, memory, handoff, done, and exit commands
- [x] 2.4 Reject unavailable Session states before reading input

## 3. Creation Integration

- [x] 3.1 Add `new --loop [--phase PHASE]` without changing one-shot creation
- [x] 3.2 Add `grill --loop` without changing one-shot Grill behavior

## 4. Tests and Documentation

- [x] 4.1 Add CLI tests for new, Grill, resume, first-turn, command, exit, and failure paths
- [x] 4.2 Document all Loop entry paths and local commands
- [x] 4.3 Mirror runtime and documentation changes into the packaged plugin

## 5. Verification

- [x] 5.1 Run compile checks and focused Loop tests
- [x] 5.2 Run the complete aiw-flow regression suite and CLI smoke tests
- [x] 5.3 Verify runtime source parity and strict OpenSpec validation

## TODO

- [x] Record follow-ups as `%%` notes without expanding this change

%% Follow-up: add multiline input or external-editor integration after observing real Loop usage.

%% Follow-up: consider a persistent MCP or daemon backend only if subprocess startup is measured as a bottleneck.

%% Environment: pytest is not installed in the available Python 3.9 runtime. No dependency was added; all tests use standard-library unittest discovery.

## Verification

- [x] Record exact commands and observed results before completion

- `$env:PYTHONPATH='src'; python -m unittest discover -s tests -p "test_interactive_loop.py" -v` — 4 parser tests passed.
- `$env:PYTHONPATH='src'; python -m unittest discover -s tests -p "test_loop_cli.py" -v` — 9 Loop CLI tests passed.
- `$env:PYTHONPATH='src'; python -m unittest discover -s tests -p "test_cli.py" -v` — 6 existing CLI tests passed.
- `python -m py_compile src\codex_flow\cli.py src\codex_flow\interactive_loop.py` — passed.
- `python -m compileall -q src tests` — passed.
- `$env:PYTHONPATH='src'; python -m unittest discover -s tests -v` — 40 tests passed.
- `$env:PYTHONPATH='src'; python -m codex_flow loop --help` — passed.
- `$env:PYTHONPATH='src'; python -m codex_flow new --help` — passed and listed `--loop`, `--phase`, and `--timeout`.
- `$env:PYTHONPATH='src'; python -m codex_flow grill --help` — passed and listed `--loop`.
- Packaged plugin `loop --help`, `new --help`, and `grill --help` — passed.
- Development/package SHA-256 parity check for all top-level runtime Python files — `PARITY_OK`.
- `git diff --check` — passed; Git reported only line-ending conversion warnings.
- `openspec.cmd validate add-interactive-session-loop --type change --strict --no-interactive` — passed.
- `openspec.cmd validate --specs --strict --no-interactive` — 3 main specs passed.
