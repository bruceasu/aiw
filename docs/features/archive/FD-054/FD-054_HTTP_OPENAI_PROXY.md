# FD-054: 重定位 HTTP OpenAI Proxy

**Status:** Complete
**Revision:** 4
**Priority:** Medium

## Problem

`src/programs/aiw-agent` currently accepts a custom HTTP/WebSocket request and
selects Codex CLI, Copilot CLI, or OpenAI per request. This mixes a caller-facing
HTTP adapter with unrelated backend integrations and does not match the intended
role: applications call a normal HTTP endpoint, while this program uses the
OpenAI Responses API to reach OpenAI or an OpenAI-compatible upstream.

## Decision

- Reposition the program as `http-openai-proxy`.
- Use the user-renamed source directory `src/programs/http-openai-proxy`.
  Rename package metadata, CLI surface, startup entry point, and current
  documentation; keep the old launcher as a compatibility alias.
- Keep the caller-facing JSON request/response interface. The program adapts
  that request to the OpenAI Responses API; it is not a transparent arbitrary
  HTTP forwarder.
- Use the OpenAI Node API SDK for all upstream calls. Configure one upstream per
  process with namespaced environment variables; OpenAI remains the default.
- Remove Codex CLI and Copilot CLI execution paths. A legacy `provider` field may
  be omitted or set to `openai`, but it must never choose another backend.
- Retain the existing bounded result/ACK behavior for compatibility in this
  change. HTTP request/response remains the primary interface.
- Keep the service's current listener exposure and security behavior; do not
  broaden remote access as part of this FD.

## Work Items

- [x] 1.1 Replace multi-provider dispatch with one OpenAI Responses SDK path.
  Acceptance: no request can launch Codex or Copilot; `provider` is optional and
  only `openai` is accepted when present; response mapping, timeout, usage,
  result limits, and sanitized errors remain intact.
- [x] 1.2 Namespace service and upstream configuration and rename the executable
  surface to `http-openai-proxy`. Acceptance: documented new settings work;
  legacy settings and launcher continue to work as compatibility fallbacks.
- [x] 1.3 Rewrite the program README around the new purpose and protocol split.
  Acceptance: documents the caller HTTP contract, OpenAI Responses upstream,
  compatible service limitations, configuration, and retained ACK behavior.
- [x] 1.4 Inspect changed call paths and documentation, update Verification, and
  run one compile-only check. Tests and live upstream requests are not included.
- [x] 1.5 Integrate the standalone package into `build.py`. Acceptance: an
  explicit action builds from local dependencies without downloads and installs
  a runnable package under `AIW_INSTALL_DIR/http-openai-proxy`; `bin` and `all`
  include it without overwriting neighboring Gateway configuration.

## Acceptance

- The only model backend is the OpenAI Responses protocol through the OpenAI
  Node API SDK.
- Upstream is configured once per process and may be OpenAI or a compatible
  Responses endpoint.
- No request selects or invokes Codex CLI or Copilot CLI.
- Existing request IDs, JSON validation, result limits, timeout, usage audit,
  and error sanitization remain intact.
- The new command and environment variable names are documented; old names
  remain usable during migration.

## TODO

- [x] Work Items 1.1–1.5 implemented; independent review remains pending.

## Verification

- Static review command: `git diff -- <changed source and index paths>` plus a
  targeted `rg` over the implementation and README; no remaining Codex or
  Copilot execution path was found in the service source.
- Compile-only command: from `src/programs/aiw-agent`,
  `node node_modules/typescript/bin/tsc --noEmit -p tsconfig.json` (exit 0).
- Build-script compile-only command: `python -m py_compile build.py` (exit 0).
- Static review confirmed the new action stages compiled JavaScript, source,
  launchers, manifests, README, and the already-installed local `node_modules`
  under `dist/programs/http-openai-proxy`, then copies the package to
  `AIW_INSTALL_DIR/http-openai-proxy`. No download or install command is invoked.
- `build.py http-openai-proxy`, `build.py bin`, and `build.py all` were not run;
  installation to `C:\green\aiw` has not been executed or runtime-verified.
- No runtime request was made, so live OpenAI and `agent-gateway` behavior is
  unverified.
- Tests, smoke scripts, live OpenAI/Gateway calls, and final builds are outside
  this implementation request and will not be run.
- Preserve and report any existing workspace changes; do not commit them.

## Risks and Questions

- The repository already contains user changes to OpenSpec files. They are
  outside this FD's write scope and must remain untouched.
- Existing test sources exercise retired Codex/Copilot/provider behavior. They
  are not edited or run in this change; their later maintenance needs a separate
  authorized test task.
- The old remote HTTP mode still has no built-in authentication or TLS. Its
  security policy is intentionally unchanged here.

**Completed:** 2026-10-10
**Disposition reason:** PM override requested by user: archive direct main-workspace implementation; no independent Reviewer, report, or handoff evidence.
