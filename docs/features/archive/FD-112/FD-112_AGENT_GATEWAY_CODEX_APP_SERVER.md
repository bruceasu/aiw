# FD-112: Remove the Agent Gateway Codex App Server backend

**Status:** Complete
**Revision:** 18
**Priority:** Medium
**Review policy:** Independent
**Evidence policy:** Markdown

## Problem

Agent Gateway currently implements the Codex App Server, the AIW-managed Codex login backend, and an OpenAI-compatible proxy. App Server adds a separate JSON-RPC subprocess pool, sandbox/workspace management, and MCP state checks. Given the user's decision to use ChatGPT Codex login for the Codex integration, this execution path adds complexity without enough value.

## Solution

Remove the Gateway's Codex App Server startup and execution backend. Keep `openai_codex` and `openai_proxy`. Startup configuration must explicitly select a supported `backend_mode`. An omitted value or the legacy value `codex` must fail with a clear configuration error that instructs the operator to set `backend_mode` to `openai_codex`; the Gateway must not silently switch execution backends. The stop command may continue recognizing legacy configuration to shut down an already-running old Gateway gracefully.

## Scope

- Remove App Server process pool, JSON-RPC, execution runner, platform process launch code, and their startup, routing, and shutdown dependencies.
- Remove `workspace_dir`, `codex_path`, and `codex_script` from new Gateway startup configuration. Retain only the minimal parsing needed to stop an old process.
- Preserve AIW-managed credentials and existing Responses/Chat Completions, limits, and persistence behavior in `openai_codex`; preserve `openai_proxy`.
- Update Gateway examples, README, and the stable Agent Gateway specification.
- Do not remove AIW Codex login, `src/pkg/openai-codex`, other Codex CLI callers, or `openai_proxy`.

## Work items

- [x] 1.1 Confirm removal scope and migration behavior: startup must explicitly select `openai_codex` or `openai_proxy`; omitted or `codex` must instruct migration to `openai_codex`.
- [x] 1.2 Remove App Server runtime code and backend dependencies from config, Gateway routing, startup, and shutdown; preserve old-process stop compatibility.
- [x] 1.3 Update Gateway docs, example configs, and stable spec to describe supported backends and the migration error.
- [x] 1.4 Statically inspect affected call paths, run the allowed Go compile-only check, and record review and skipped checks.

## Acceptance

- Starting with `backend_mode` omitted or set to `codex` fails clearly and tells the operator to set `backend_mode` to `openai_codex`; no App Server is launched.
- New configs accept only `openai_codex` and `openai_proxy`; their existing authentication, limits, persistence, proxy, and shutdown behavior remains intact.
- `stop` can use a legacy config to contact and gracefully stop an already-running old Gateway. This compatibility does not allow a new process to start App Server.
- Gateway source, examples, and stable spec no longer declare or depend on the App Server execution backend, its sandbox, or workspace configuration.

## Verification

- Plan: statically trace config parsing, start/stop, Responses/Chat Completions execution paths, proxy mode, and stable spec; then run the compile-only script under `src/programs/agent-gateway`.
- Review policy is Independent because this removes a public backend configuration and changes old-config startup behavior across Gateway modules and operator migration.
- Per Go subtree rules, do not create or run Go tests. Do not run builds, network calls, installs, or runtime acceptance.
- `python scripts/compile.py` from `src/programs/agent-gateway` initially found a missing shared `RunResult` type after deleting `app_runner.go` and a missing `filepath` import. Moved `RunResult` to `result.go`, restored the import, and reran the same compile-only command successfully for Windows and Linux; the script writes no distributable artifact.
- `git diff --check` passed. Static reference inspection confirms the App Server runtime and start path are removed; old `codex` and workspace/executable fields remain only in explicit migration errors and the stop parser for already-running older processes.
- Reviewer event `FD-112-000006-changes-requested` found two documentation gaps: request persistence was stated without a backend qualifier, and the active development/shared listener rules had been removed with the App Server lifecycle text. Updated the README to scope persistence to `openai_codex` and restored the listener/peer rules as a separate stable requirement. Worker session: `fd112-worker-20261108-docfix`.
- The review repairs changed documentation and stable requirements only; implementation code is unchanged from the successful compile-only revision.
- Reviewer event `FD-112-000009-changes-requested` found that the shutdown specification still required process-tree cleanup. Reworded it to preserve active-request cancellation, HTTP shutdown, proxy connection closure, content cleanup completion, and state-lock release without App Server process-tree semantics. Worker session: `fd112-worker-20261108-shutdownspec`.
- Reviewer reports: `reviews/FD-112-review-r1.md` requested the README persistence and deployment-listener corrections; `reviews/FD-112-review-r2.md` requested the remaining shutdown-spec correction; `reviews/FD-112-review-r3.md` passed after those corrections. Final revision review `reviews/FD-112-review-r4.md` found no findings and passed verification.
- After recording the required review links, PM reset the event-derived `Complete` status to `In Progress` to permit a fresh current-revision review. CLI audit: `.ai/fd/FD-112/operations/6302c9cd70dc40939b16588cc0034087.json`. This status operation did not change implementation or waive independent review.
- No tests, runtime validation, dependency installation, or network calls were run.

## TODO

- No implementation or verification work remains. PM may archive after the final `verification-passed` event is recorded.

## Sources

- `openspec/specs/agent-gateway/spec.md`
- `src/programs/agent-gateway/config.go`
- `src/programs/agent-gateway/server.go`
- `src/programs/agent-gateway/main.go`
- `src/programs/agent-gateway/shutdown.go`

**Completed:** 2026-11-08
