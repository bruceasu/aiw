## Context

aiw-flow already persists instructions, memory, prompts, outputs, events, and artifacts for each Codex session. It also resumes a Codex thread in the declared workspace. The new capabilities should compose these existing primitives instead of introducing a second conversation loop or a direct OpenAI SDK dependency.

The repository supports Python 3.9+, Windows, Linux, and macOS. It forbids `shell=True`, requires atomic state writes and locks for session mutations, and prefers the standard library.

## Goals / Non-Goals

**Goals:**

- Start a guided requirement interview as a normal aiw-flow session and Codex thread.
- Collect a small, deterministic, redacted workspace summary before the first Grill turn.
- Generate a reproducible Markdown handoff from stored session facts.
- Keep every generated artifact owned by its session.
- Preserve all existing CLI and status behavior.

**Non-Goals:**

- Add the OpenAI Python SDK or call Chat Completions directly.
- Reconstruct an unrestricted transcript from external chat products.
- Scan arbitrary files or upload the full workspace.
- Automatically decide that requirements are complete without user confirmation.
- Replace `memory.md` as the canonical curated session summary.

## Decisions

### Reuse the existing exec backend and thread lifecycle

`grill` will create a normal session with built-in Grill instructions, save a workspace context artifact, and execute the first turn through the existing exec backend. Later answers use the existing `continue` command with phase `grill`.

This keeps model configuration, sandboxing, event capture, timeout handling, and thread persistence in one implementation. A separate OpenAI SDK loop was rejected because it would duplicate configuration and add a dependency.

### Use prompt rules, not punctuation counting, for one-question behavior

The built-in Grill instructions will require at most one decision question per response, a recommended answer with rationale, and a final specification after explicit user confirmation. The CLI will not count question marks because punctuation is language-dependent and does not reliably represent semantic questions.

### Collect only bounded, allow-listed context

The context collector will walk directory names to a bounded depth and read snippets only from an allow-list of project metadata files. It will skip hidden, dependency, cache, and VCS directories. It will cap entry count, per-file bytes, and total bytes.

Potential credential assignments in collected text will be deterministically replaced before the content is saved or sent to Codex. Collection failures will be recorded in the context artifact rather than silently ignored.

### Keep handoff deterministic

`handoff create` will render a fixed Markdown structure from `status.json`, `memory.md`, saved artifact paths, and a bounded excerpt of the latest final output. It will not invoke a model.

The result will be written atomically to `artifacts/handoff.md` under the session lock. `handoff show` will display that exact artifact and fail clearly if it has not been created.

### Add small focused modules

- `workspace_context.py` owns bounded collection and redaction.
- `grill.py` owns built-in instructions and the initial requirement prompt.
- `handoff_manager.py` owns deterministic rendering.
- `SessionStore` exposes locked artifact text read/write helpers.
- `cli.py` only parses arguments and coordinates these components.

### Treat `program/aiw-flow` as source and mirror the plugin package

Implementation and tests will be performed in `program/aiw-flow`. After verification, changed runtime files will be copied to `plugins/aiw-flow`, followed by a runtime source parity check. The shorter plugin README will retain its entry-point-specific structure and document the new commands separately.

## Risks / Trade-offs

- [Prompt instructions cannot absolutely enforce one semantic question] → Make the contract explicit, keep each answer as a separate resumed turn, and test the built-in prompt.
- [Redaction patterns cannot identify every secret] → Read only allow-listed project metadata, impose small byte limits, and document that the artifact is a summary rather than a security boundary.
- [Deterministic handoff is less fluent than an AI summary] → Prefer factual reproducibility; users can continue the session to request a narrative summary when needed.
- [Two source trees can drift] → Add a verification test or parity check for changed files and document the canonical source.
- [Concurrent handoff/context creation could race] → Route artifact writes through the existing per-session lock and atomic writer.

## Migration Plan

1. Add modules and tests without changing the status schema.
2. Add new CLI commands while preserving existing parser arguments.
3. Update the development source and mirror it into the plugin package.
4. Roll back by removing the new commands and modules; existing sessions remain valid because only optional artifacts and normal session files are added.

## Open Questions

%% A future change may add automatic memory extraction after each turn. It is intentionally excluded because it would change the source-of-truth semantics of `memory.md`.

%% A future change may add a read-only ephemeral AI handoff renderer as an explicit opt-in. The first version remains deterministic.
