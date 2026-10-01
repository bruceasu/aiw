# FD-012 Review r1

## Result

`changes-requested` for source event `FD-012-000004-implementation-ready`.

- Reviewer session reference: `fd012-reviewer-20261002-f63b90c7`.
- Reviewed diff base: current uncommitted workspace versus `HEAD`, limited to FD-012 paths and the implementation report. The FD and report are untracked; no implementation commit was recorded.

## Finding

1. **Windows install packaging depends on a missing helper.** `build.bat:77` and `build.bat:78` call `cp-mirror.bat` to copy `agent-templates` and `.agents/prompts`; the Linux branch repeats this at lines 91–92. No `cp-mirror.bat` exists in the repository (`Test-Path cp-mirror.bat` returned false, and the scoped filename search found none). The required Windows installation packaging behavior is therefore not established and the install command is expected to stop at the first copy call unless that external helper is an explicit prerequisite. Confirm a checked-in helper or replace these calls with an available copy operation before passing verification.

## Scope and evidence

- Confirmed old source paths `prompts/`, `docs/agents/`, and `docs/agent-templates/` are absent; corresponding `.agents/` trees exist. The new trees contain 60 files total, matching the 60 tracked source-tree files at `HEAD`.
- Static search over relevant source, docs, Skills, plugins, and templates found no remaining references to the old source paths. Search encountered an inaccessible generated `scripts/aiw-fd-smoke-*` directory, so that generated directory was not included in the search evidence.
- Inspected `internal/commands/initcmd/command.go`, `prompts.go`, `cmd/aiw/main.go`, `build.bat`, moved template references, root/template guidance, and the Worker report. Template lookup resolves beside the executable under `agent-templates`; prompt synchronization targets `.agents/prompts`.
- Excluded the unrelated Windows Skill-publishing retry changes in `plugins/aiw-skills/aiw-skills.py` and its README from FD-012 scope; they do not implement the path migration described by this FD.
- Commands run included `python plugins/aiw-fd.py show FD-012`, `python plugins/aiw-fd.py claim FD-012 FD-012-000004-implementation-ready --session fd012-reviewer-20261002-f63b90c7`, targeted `git status`/`git diff`, `rg` path searches, `Get-ChildItem` counts, and `Test-Path` checks. One attempt to read `cp-mirror.bat` confirmed the path was absent.

## Skipped checks and residual risk

- Did not run tests, runtime checks, or final builds. The Worker reports `python scripts/compile.py` completed without diagnostics; the Reviewer did not rerun it.
- The implementation report claims a successful stale-path search, while the Reviewer search could not traverse one generated smoke-test directory. No source acceptance path was found stale in the searched files, but the generated directory is unverified.
