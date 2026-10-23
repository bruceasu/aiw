# Issue Handoff: Python function calls through agent-gateway

Status: Captured as `ISSUE-00012`; awaiting Issue approval. Capture does not authorize implementation.

## User, problem, and goal

The user runs an OpenAI-compatible service at `http://127.0.0.1:43127/v1` and wants Python function callers to use it for convenient testing. The service is the AIW `agent-gateway`, whose backend is Codex. A plain Responses API request succeeds, but the Agents SDK function-tool case receives `BadRequestError`. The user also confirmed that the deployed `process-python` worker is expected to target this gateway.

Goal: allow the current Python OpenAI Agents SDK and the AIW `process-python` runtime, when configured to use the Codex-backed gateway, to complete non-streaming, sequential function-call round trips while preserving schema validation and the gateway's safety boundaries.

## Confirmed facts and evidence

- Using Python 3.12, `openai-agents` 0.24.0 and `openai` 3.36.1, the demo's plain Responses request to the configured gateway returned `AGENTS_SDK_OK`.
- The demo's tool-call request using `@function_tool` failed with `BadRequestError`. The test harness intentionally reports only the exception type; the HTTP status/body and gateway rejection code were not captured.
- The installed Agents SDK generates this schema for the demo's `add_numbers(a: int, b: int)` function:
  `{"additionalProperties":false,"properties":{"a":{"title":"A","type":"integer"},"b":{"title":"B","type":"integer"}},"required":["a","b"],"title":"add_numbers_args","type":"object"}`
- The user confirmed that the deployed `process-python` runtime is expected to target this gateway, so its sequential-call request setting is in scope.
- In the Codex backend, `src/programs/agent-gateway/protocol.go` accepts Responses `tools`, `tool_choice`, and `parallel_tool_calls`; it rejects unsupported strict schemas and `parallel_tool_calls=true`.
- `src/programs/agent-gateway/schema.go` validates a bounded JSON Schema subset and rejects unrecognized keywords. `title` is not in the accepted subset, so the SDK-generated strict schema is expected to be rejected. This is the leading diagnosis, not a confirmed server error code.
- `src/programs/agent-gateway/app_runner.go` maps Codex structured output to a Responses `function_call`; `protocol.go` also parses subsequent function-call and function-call-output input items. The README documents non-streaming sequential function calls, with the caller responsible for executing the function and submitting its result.
- The separate Python worker runtime in `workers/process-python/openai_api_runtime.py` explicitly sets `parallel_tool_calls=True`, which conflicts with the gateway's current sequential-only behavior when that runtime targets this gateway.
- The deployed `process-python` runtime pins `openai-agents` 0.23.1 and `openai` 3.26.0; the demo reproduction's reported versions are 0.24.0 and 3.36.1.

Sources:

- `src/programs/agent-gateway/protocol.go` (`decodeRequest`, `responseObject`)
- `src/programs/agent-gateway/schema.go` (`validateFunctionSchema`, `parseSchema`)
- `src/programs/agent-gateway/app_runner.go` (`appOutputSchema`, `mapToolOutput`)
- `src/programs/agent-gateway/README.md` (documented Responses/tool-call limits)
- `../aiw-de/workers/demo-python-with-openai-agents/test_tool_call.py` (repro case)
- `../aiw-de/workers/process-python/openai_api_runtime.py` (parallel-tool setting)

Source snapshot reviewed: gateway sources at AIW revision `4ddd9931935084f69d561b889c6207dff2624322`; Python worker sources at `D:\03_projects\AI-tools\aiw-de` revision `31abb1b8b560eab5cc35e1c0675ec114461b7946`. The referenced demo file `workers/demo-python-with-openai-agents/test_tool_call.py` is present locally but is not tracked by Git; `workers/process-python/openai_api_runtime.py` is tracked and unchanged between the previously reviewed and current revisions.

## Scope

- Make strict function schemas emitted by the current Agents SDK compatible with the gateway's bounded schema validator. Ignore or normalize harmless schema annotations such as `title`; keep validating parameter types, required fields, and values.
- Align the `process-python` runtime with sequential-only gateway tool calls when configured to use this gateway. Do not request parallel tool calls on this route; preserve behavior for other supported backends.
- Preserve the existing Responses function-call contract: gateway returns a function-call item, the SDK executes the local function, then submits the call and output for a final model response.
- Add or update focused compatibility coverage for the SDK-generated schema and one complete tool-call round trip.

## Non-goals and constraints

- Do not make Codex execute caller-provided Python functions. The Agents SDK remains responsible for executing its local function tools.
- Do not add parallel or streaming tool-call execution in this Issue. Keep the gateway's current sequential/non-streaming boundary explicit.
- Do not change parallel-call behavior for non-gateway backends as part of this Issue.
- Do not accept arbitrary JSON Schema keywords or remove strict argument validation. Support only SDK metadata that does not change parameter semantics, unless the Planner identifies a separate compatibility requirement.
- Do not change the gateway's authentication, limits, persistence, or safety policy as part of this compatibility fix.

## Acceptance examples

- Given the current Agents SDK-generated strict schema for `add_numbers(a: int, b: int)`, the gateway accepts the tool definition without weakening validation of the integer arguments.
- Given an Agents SDK Agent required to call `add_numbers(19, 23)`, the gateway returns a valid Responses `function_call`; the SDK executes the local tool exactly once, submits its output, and receives `TOOL_CALL_OK:42`.
- The focused demo `test_tool_call.py` reports `status=ok` against the Codex-backed gateway.
- When `process-python` is configured to target this gateway, its request does not require parallel tool calls and can use the documented sequential function-call flow.
- Plain text Responses requests continue to succeed.

## Assumptions and open gates

- Assumption: the tested listener on `127.0.0.1:43127` was this repository's Codex-backed `agent-gateway`, rather than `openai_proxy` mode or another compatible service.
- `%% NEEDS_INPUT: Capture the gateway's structured 400 error code for the failing tool request before final engineering design, to confirm whether strict-schema rejection is the runtime cause or whether another request option (including parallel calls) is rejected first. The current source evidence supports the schema diagnosis, but the observed response body was not captured.`

## Size and split assessment

Keep this as one medium-sized integration Issue: the confirmed caller set includes both the Agents SDK demo and the deployed `process-python` runtime, and both must follow the same sequential function-call contract when using this gateway. The Planner can create separate Work Items for the gateway and Python runtime while keeping one acceptance outcome: supported Python callers can complete function calls through `agent-gateway` without weakening its validation or safety boundaries.

## Recommended next action

Review and resolve or explicitly retain the remaining Gate, then decide whether to approve `ISSUE-00012`. After approval, promote it to an FD for cross-repository engineering design. Do not treat capture as approval or implementation authorization.
