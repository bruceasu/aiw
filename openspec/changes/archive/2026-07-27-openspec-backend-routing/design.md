## Context

AIW's native task commands are useful in minimal installations, while an
installed OpenSpec CLI is the canonical implementation for spec-driven work.
The adapter must work on Windows and Linux, avoid adding dependencies, and
preserve existing scripts when native behavior is selected.

## Goals / Non-Goals

**Goals:**

- Detect a usable OpenSpec executable without requiring it.
- Support explicit `auto`, `openspec`, and `native` backend modes.
- Delegate only supported task operations and report the selected backend.
- Keep subprocess failures actionable and avoid partial native writes.

**Non-Goals:**

- Reimplementing OpenSpec inside Go.
- Changing `aiw-flow`, Session, Loop, or Handoff behavior.
- Automatically installing OpenSpec.
- Guessing mappings for unsupported OpenSpec commands.

## Decisions

### Backend selection

Use an option parser shared by task workflow commands. `auto` first checks an
explicit `AIW_OPENSPEC_BIN`, then PATH candidates (`openspec`, `openspec.cmd`),
and verifies the executable with `--version`. `openspec` fails clearly when no
verified executable exists; `native` never probes or delegates.

### Delegation boundary

The adapter delegates only operations whose artifact mapping is known. It
passes the repository working directory through unchanged and forwards the
change/task identifier as a validated argument. Native fallback remains the
existing implementation, not a second subprocess path.

### Diagnostics and compatibility

`auto` is the default and prints a concise backend diagnostic on stderr.
When OpenSpec is unavailable it preserves the native implementation and
artifacts. Scripts that require the historical path can select `--backend
native` explicitly.

## Risks / Trade-offs

- [Risk] An executable named `openspec` may be unrelated or incompatible. →
  Mitigation: require a successful `--version` probe and honor explicit path.
- [Risk] OpenSpec artifact semantics can diverge from AIW metadata. →
  Mitigation: delegate only mapped operations and keep `native` available.
- [Risk] Windows command resolution differs from Linux. → Mitigation: use
  `exec.LookPath` and include `.cmd` on Windows.

## Migration Plan

1. Add backend detection and option parsing without changing default behavior.
2. Add explicit delegation for verified operations and diagnostics.
3. Document `auto`, `openspec`, and `native`, including script guidance.
4. Add tests with fake executables; rollback by selecting `--backend native`.

## Open Questions

- Which OpenSpec subcommands should be delegated first: proposal creation,
  apply, archive, or all three as separate mappings?
