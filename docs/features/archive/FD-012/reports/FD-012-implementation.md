# FD-012 Implementation Report

## Result

- Moved `prompts/` to `.agents/prompts/`.
- Moved `docs/agents/` to `.agents/agents/`.
- Moved `docs/agent-templates/` to `agent-templates/`.
- Updated both monorepo prompts, CLI template lookup, README, Skills, plugin
  setup text, the Skills release fixture, and Windows install packaging.
- Replaced the missing `cp-mirror.bat` references in both install branches with
  built-in recursive `xcopy /E /I /Y` operations for `agent-templates`
  and `.agents/prompts`; replaced the remaining install-time `docs/usage`
  helper call as well.
- Kept the template package's nested prompts library as the source distributed
  by `aiw init`; it continues to write to `.agents/prompts/` in the target
  project.

## Evidence

- Static search found no current references to the old source paths outside
  this FD and archived history.
- Reviewed the scoped diffs and confirmed all three old directories are absent
  while their new `.agents/` locations exist.
- Static review confirmed install-time copies use quoted source and target
  paths and no longer depend on the missing `cp-mirror.bat`. The Windows
  installer was not executed, so runtime copy behavior is unverified.
- Ran `python scripts/compile.py` once; no compiler diagnostics were returned.
- Tests, final-artifact builds, and runtime checks were not run.

## Review status

Reviewer r1 requested changes for the missing install-copy helper. The finding
is addressed; independent review is requested for this updated implementation.
