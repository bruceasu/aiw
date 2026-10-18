# FD-097: Load installed dotenv defaults before command dispatch

**Status:** Pending Verification
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem and authorized intent

User requests installation-directory .env loading because plugins and built-in
features consume environment variables. aiw main currently dispatches without
loading dotenv; plugins only inherit process variables. The Go LLM resolver reads
dotenv separately for its own configuration, not the child process environment.

## Decision and compatibility

At the start of aiw main, before help/version, built-ins or plugin discovery,
load only filepath.Join(filepath.Dir(os.Executable()), .env). Do not use cwd,
AIW_ROOT overrides, parent-directory searches or profile files for startup loading.
A missing file is harmless; other read/parse/export failures return a contextual
stderr error and exit 1 before dispatch. Never log values or file contents.

Reuse the existing Go env.Loader syntax and expansion. Add private defaults-only
mode: existing process variables, including empty values, are retained during
parse/expansion and export. File-defined variables can reference previously parsed
file defaults or parent environment. Respect Windows case-insensitive keys for
new defaults and expansion, while preserving existing Loader behavior by default.
Handle a first-line UTF-8 BOM and ignore empty pseudo-keys when seeding from the
Windows process environment; do not alter those actual inherited process entries.
Replace expansion panics with returned errors so startup fails clearly on cycles.
No command substitution, shell invocation or new dependency.

Plugins launched through aiw inherit effective defaults through existing ExecPlugin.
Engine-owned invocation metadata still overrides its reserved variables. Directly
launching a Python/plugin executable bypasses aiw startup and does not acquire this
new automatic loading. build.py keeps its checkout-local loader unchanged.

Installed defaults now belong to effective process environment. Existing LLM
explicit config precedence stays unchanged; nonempty installed defaults therefore
precede that resolver's cwd/AIW_ROOT/executable dotenv fallback values, just like
other effective environment variables. Document this compatibility effect rather
than claiming the old project-dotenv priority is unchanged. Empty inherited values
remain empty; consumers' existing empty-value fallback rules are not rewritten.

Continue in user-selected primary workspace. Preserve the existing unrelated dirty
src/plugins/aiw-git/git-guide.py and never stage it. No branch/worktree/receipt.
Independent bounded design feedback checks initialization timing, precedence,
expansion errors, Windows casing, plugin inheritance and compatibility before code.
No linked Issue or OpenSpec change.

## Work Items

- [x] 1.1 Add executable-directory defaults loading and safe error propagation.
  Size small, difficulty medium, no dependencies. Completion: defaults-only parsing,
  inherited-empty/case/expansion behavior, absent file and returned cycle errors traced.
- [x] 1.2 Load defaults before all aiw command paths and document inheritance.
  Size small, difficulty low, depends on 1.1. Completion: main pre-dispatch hook,
  shared plugin inheritance, direct-plugin boundary and priority docs/spec aligned.
- [x] 1.3 Prepare offline Go compile-only and static evidence.
  Size small, difficulty low, depends on 1.2. Completion: scripts/compile.py to NUL
  and one static diff check. Independent review remains a completion gate.

## Acceptance

- aiw built-ins and child plugins can read installation .env defaults from os.Getenv.
- Existing parent variables, including empty values and Windows alternate casing,
  are not overwritten; expansion sees those inherited values.
- Only the executable-directory .env is loaded by this startup hook.
- Missing .env does not stop commands; actual load failures do not expose values.
- Plugin invocation metadata, cwd/args/stdio and direct-plugin behavior remain defined.
- No installed binary, package ZIP or unrelated user edit is changed by this work.

## TODO

- Independent design ready; implementation/docs and offline Go compile-only complete.
- Independent static review required before completion; unrelated git-guide edit preserved.

## Verification

Static review plus one offline root scripts/compile.py to NUL. No Go tests,
formatters, runtime commands, final builds, installation or network.
Preserve unrelated user changes and report installed launcher still needs updating.
Offline scripts/compile.py returned exit 0 for both entrypoints to NUL. One static
diff/read batch inspected default parsing/export, main timing, docs and new helper.
No runtime dotenv or plugin invocation executed. See implementation report.

## Sources

- src/cmd/aiw/main.go
- src/internal/util/env/loader.go and src/internal/ai/llm.go
- src/internal/plugin/{environment,exec}.go
- openspec/specs/cli-and-plugins/spec.md; docs/usage/cli-reference.md
