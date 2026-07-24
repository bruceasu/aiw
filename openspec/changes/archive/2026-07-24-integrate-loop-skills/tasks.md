## 1. Skill Discovery

- [x] 1.1 Add a read-only Skill discovery module for repository-ancestry `.agents/skills`, project `.codex/skills`, user `.agents/skills`, and effective Codex-home `skills`
- [x] 1.2 Parse and validate required `SKILL.md` frontmatter without adding a dependency
- [x] 1.3 Detect malformed candidates, symlinked Skill directories, and duplicate names with source-aware diagnostics
- [x] 1.4 Add focused discovery tests for Git and non-Git workspaces, all supported scopes, malformed metadata, symlinks, and collisions

## 2. Interactive Commands

- [x] 2.1 Extend the loop input model to parse `/skills` and `/skill <name> <message>` while preserving `//text` escaping and existing commands
- [x] 2.2 Implement local `/skills` output grouped by scope with warnings and no Codex turn
- [x] 2.3 Re-discover and validate `/skill` requests immediately before transforming them to native `$<name> <message>` prompts
- [x] 2.4 Route valid Skill prompts through the existing `_execute_turn` path and reject missing, unknown, malformed, or ambiguous requests locally
- [x] 2.5 Add parser and CLI tests covering listing, invocation, normal persistence, refreshed discovery, validation failures, direct `$name` messages, and regression behavior

## 3. Documentation and Packaging

- [x] 3.1 Document supported `.agents/skills` and `.codex/skills` scopes, `/skills`, `/skill`, direct `$name` invocation, collisions, and limitations
- [x] 3.2 Update Loop help text and examples without changing existing one-shot command documentation
- [x] 3.3 Mirror changed runtime files and documentation from `program/aiw-flow` to `plugins/aiw-flow`

## 4. Verification

- [x] 4.1 Run formatting or compile checks and focused Skill discovery and Loop command tests
- [x] 4.2 Run the complete aiw-flow regression suite and packaged CLI smoke tests
- [x] 4.3 Verify development/package runtime parity and run strict OpenSpec validation
- [x] 4.4 Record exact commands and observed results in the Verification section before completing the change

## TODO

- [x] Record implementation discoveries, unresolved risks, and follow-up scope as `%%` notes instead of guessing

%% No dependency change is expected. Pause for approval if implementation reveals that a YAML parser dependency is necessary.

%% Preserve the existing completed interactive-loop behavior; do not fold unrelated Loop enhancements into this change.

%% Implementation: frontmatter discovery accepts unquoted, single-quoted, and double-quoted single-line `name` and `description`; YAML block descriptions are reported as malformed.

%% Environment: `scripts/verify.sh` is unavailable, so repository-standard Python compile, unittest, CLI smoke, parity, Git whitespace, and OpenSpec validation commands were used.

## Verification

- [x] Record exact commands and observed results before completion

- `python -m compileall -q src tests` — passed.
- Focused unittest discovery for `test_skill_discovery.py`, `test_interactive_loop.py`, and `test_loop_cli.py` — 6, 5, and 13 tests passed respectively.
- `$env:PYTHONPATH='src'; python -m unittest discover -s tests -v` — 51 tests passed.
- Packaged plugin `--help` and `loop --help` smoke commands — passed.
- SHA-256 parity check across all `program/aiw-flow/src/codex_flow/**/*.py` files and packaged mirrors — `PARITY_OK`.
- `cmd /c openspec validate integrate-loop-skills --type change --strict --no-interactive` — passed.
- `git diff --check` — passed with only expected Windows line-ending conversion warnings.
