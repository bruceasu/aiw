## 1. Session Metadata and Workspace Scope

- [x] 1.1 Extend session metadata and tolerant JSONL inspection with optional
  normalized `original_cwd` extraction.
- [x] 1.2 Evolve metadata caching so legacy records without `original_cwd` are
  rescanned while alias indexes and session files remain unchanged.
- [x] 1.3 Add reusable current-workspace filtering with explicit
  all-workspaces mode and unknown-workspace handling.

## 2. Reusable Session Operations

- [x] 2.1 Separate bounded user/assistant preview generation from CLI printing
  so CLI and GUI callers receive the same readable content.
- [x] 2.2 Add reusable alias create, atomic rename, and remove operations with
  existing validation and explicit conflict protection.
- [x] 2.3 Add isolated interactive `codex resume <session-id>` command
  construction and validate the installed CLI syntax before enabling launch.
- [x] 2.4 Add terminal and original-directory precondition handling that
  returns a copyable recovery command instead of launching an unsafe process.

## 3. Desktop GUI

- [x] 3.1 Add the `aiw cxs gui` parser entry point and lazy-load Tkinter with an
  actionable unavailable-toolkit error.
- [x] 3.2 Build the workspace-scoped session table, selection state, refresh,
  and explicit all-workspaces toggle.
- [x] 3.3 Add bounded conversation preview and alias create, rename, and remove
  dialogs.
- [x] 3.4 Add interactive resume handoff that closes the GUI and starts Codex in
  the selected session's original directory.

## 4. Resume Skill

- [x] 4.1 Create the repository-managed `resume-ext` skill with concise trigger
  metadata and generated `agents/openai.yaml`.
- [x] 4.2 Define the numbered workspace-session selection workflow using
  machine-readable `aiw cxs` output and Easy English prompts.
- [x] 4.3 Ensure the skill emits a copyable command when native thread handoff
  is unavailable and never starts nested Codex.

## 5. Documentation and Verification

- [x] 5.1 Update `aiw cxs` help and usage documentation for workspace scope,
  GUI behavior, interactive resume, fallback behavior, and skill invocation.
- [x] 5.2 Inspect the final diff, trace cache/index compatibility and GUI/CLI
  call paths, and record unresolved JSONL or CLI-version risks with `%%` notes.
- [x] 5.3 Record any focused runtime command separately for user authorization;
  do not run tests, builds, formatters, linters, or GUI launch by default.

## Verification

- `python -m unittest plugins.test_aiw_cxs`: failed first with seven missing
  behavior errors, passed 7 tests after implementation, then passed 10 tests
  after review fixes.
- `python -m py_compile plugins/aiw-cxs.py plugins/test_aiw_cxs.py`: passed.
- `python -m unittest discover`: completed but discovered 0 tests at the
  repository root.
- `python -m unittest discover -s plugins -p 'test*.py'`: passed 7 discovered
  plugin tests, then passed 10 after review fixes.
- `quick_validate.py skills/resume-ext`: passed.
- `python plugins/aiw-cxs.py gui --help`: passed without opening the GUI.
- `codex resume --help`: confirmed support for `codex resume [SESSION_ID]`.
- Standards review reported two hard findings and two judgement calls; Spec
  review reported four findings. The hard/spec behavior findings were fixed.
  The large single-file GUI controller remains documented as a `%%` risk.
- GUI launch and interactive resume were not executed because they would open
  an interactive desktop/terminal process.
