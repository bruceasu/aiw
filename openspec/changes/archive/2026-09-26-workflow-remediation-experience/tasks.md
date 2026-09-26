## 1. Remediation Core contract

- [x] 1.1 Add schema 9-compatible Remediation problem, diagnosis, action, option, and response types with strict validation and stable problem identity.
- [x] 1.2 Persist immutable Remediation reports and human response templates under the Task reports directory; keep original FailureReport evidence intact and make writes idempotent.

## 2. Analysis and safe actions

- [x] 2.1 Add the read-only `RemediationAnalyzer` seam and an execution adapter that returns structured, evidence-bound AI analysis without Store write access.
- [x] 2.2 Add Core validation and execution for the finite safe-action catalog, including existing Session repair and projection repair; reject unknown state and unsafe actions.
- [x] 2.3 Add bounded Supervisor remediation rounds and idempotent verification after each action, without changing schema 10 activation or existing retry/Gate rules.

## 3. Human response workflow

- [x] 3.1 Add `continue` and `resume` CLI commands that read, validate, and consume the current response file through one shared path.
- [x] 3.2 Render actionable failure output and response templates with choices, scope, risks, resource impact, authorization, rollback, and exact next commands.
- [x] 3.3 Ensure restart, duplicate response, stale response, and missing response cases remain safe and do not create a second Attempt.

## 4. Documentation and verification

- [x] 4.1 Add focused tests for report identity, strict response validation, idempotent actions, unknown execution state, and continue/resume behavior.
- [x] 4.2 Update this change's Verification/TODO notes and document that schema 10 production activation evidence remains outside this change.
- [x] 4.3 Compile the changed Go packages with the repository compile script; do not run tests without explicit authorization.

## Verification

- Static review MUST confirm that AI analysis has no Store write seam, human responses are digest-bound, unknown execution states do not auto-dispatch, and schema 9 remains the active default.
- Compile-only evidence: `GOMAXPROCS=2 go build ./internal/workflow/...` completed without compiler output. The broader `go build ./cmd/...` reached linker execution but the Windows environment ran out of memory; the repository helper `scripts/compile.py` also targets a missing root `main.go` and failed during temporary-cache cleanup.
- Focused test attempt: `GOMAXPROCS=2 GOCACHE=.ai/test-cache/go-remediation go test ./internal/workflow ./internal/workflow/cli` was authorized and reached test-package compilation, but the repository's pre-existing `internal/workflow/verification_plan_test.go` is missing `reflect` imports and `internal/workflow/cli/workflow_test_adapter_test.go` redeclares production functions; no new remediation test failure was reached.
- Production/real-model activation evidence is not implied by implementation completion.

%% Real Worker/Host, real model capacity, cost metering, E01-E04 joint validation, and schema 10 activation remain unresolved by design and belong to a separate change.
