# FD-022: Agent Gateway OpenAI Agents SDK compatibility

**Status:** Deferred
**Revision:** 7
**Priority:** Medium  
**Test policy:** External  
**Evidence policy:** Dual

## Problem

`program/agent-gateway` implements a strict OpenAI Responses API subset for
single text responses. It currently rejects non-empty `tools`, accepts only
user/assistant text input, and does not support tool-call output items. This
prevents applications using the OpenAI Agents SDK from completing function
tool turns through the Gateway.

## Options and decision

- Add a separate Agent SDK-specific HTTP API: this would duplicate the SDK's
  existing client-side orchestration and invent a service endpoint.
- Expand the existing OpenAI Responses API endpoint: this keeps the SDK's
  normal `baseURL` configuration and preserves Gateway authentication, model
  authorization, quotas, and request observability.
- Implement a broad OpenAI API clone: this exceeds the Gateway's bounded
  purpose and would introduce unsupported capability and maintenance promises.

Decision: target the JavaScript/TypeScript `@openai/agents` v0.18.0 default
HTTP Responses path, starting from the characterized non-streamed text agent
with ordinary function tools. The SDK's `OpenAIResponsesModel` calls `responses.create`; its request
builder sends `input`, `instructions`, `include`, `tools`, `stream`, and optional
`tool_choice`/`parallel_tool_calls`. Its converter sends previous
`function_call` items and later `function_call_output` items in `input`.
Extend the existing Responses endpoint for this bounded flow. Do not
add a separate `/v1/agents` endpoint. Keep function execution in the calling
application; the Gateway transports model function-call requests and the
result messages required for subsequent Responses calls.

User scope decision (2026-10-04): initial support also includes streaming
Responses and structured output. Retain v0.18.0 and ordinary function tools.
The earlier non-streamed function-tool characterization covers only that
path; streaming and structured output require their own SDK contract evidence
and implementation. Handoffs remain outside this FD.

Baseline finding (2026-10-04): the repository has `openai` as a client
dependency, but no installed `@openai/agents` package. The original Gateway
Runner disabled Codex's own tools and consumed only completed `agent_message`
text items. This increment keeps Codex's tools disabled and uses its
`exec --output-schema` option plus validation to map a model decision to a
single `function_call` with stable IDs. Its actual behavior for this Gateway
flow remains unverified. The local Gateway bearer key is separate
from an OpenAI API key; the latter is not needed to inspect or compile Gateway
code, but the former is still required when calling the local HTTP service.

The intended caller-side executor is `aiw-agent`, as clarified by the user.
Its current `program/aiw-agent/src/providers.ts` implementation sends text
inputs and extracts text output; it has no function-call execution loop yet.
This FD remains scoped to Gateway. End-to-end tool execution also requires
a separate caller-side change before it can be claimed as working.

User clarification (2026-10-04): `agent-gateway` is an OpenAI API simulator;
each advertised compatible API behavior should follow OpenAI API semantics.
For this FD, that principle applies to the selected Responses function-tool
flow. The FD does not advertise endpoints or capabilities outside that flow.

## Solution

Capture the SDK's actual HTTP methods, paths, request fields, streaming
events, structured-output request and response shapes, and optional tracing
behavior for the selected flows. Use that evidence to define the bounded
contract before changing the Go protocol decoder or Runner. Preserve strict
unknown-field rejection, principal
model access, quotas, execution accounting, audit redaction, and standard
Responses error envelopes.

The SDK default HTTP transport requires `POST /v1/responses` for this flow;
it does not require a Gateway tool-execution endpoint. Set
`OPENAI_AGENTS_DISABLE_TRACING=1` (or `tracingDisabled: true`) in the caller,
because the SDK otherwise exports traces in server runtimes. Initial scope
includes streaming and structured output, and excludes handoffs, hosted tools,
sessions, and `previous_response_id`. The caller must replay its input items
for each tool turn. `aiw-agent` tool execution is a separate caller-side change.

%% NEEDS_INPUT: Characterize v0.18.0 streaming and structured-output wire
contracts before choosing event sequences, accepted format options, and
whether their combinations with function tools are supported. Reject any
unselected combination before starting the Runner.

%% NEEDS_INPUT: Runtime evidence is still required to establish that the
Codex CLI `--output-schema` path reliably emits the selected function call
and that the v0.18.0 SDK completes a tool turn against this implementation.
%% NEEDS_INPUT: Resolve `strict:true` function-tool argument behavior before
claiming API-compatible tool results. The current Runner checks only that the
arguments are a JSON object; it does not validate them against the tool's
parameters schema.

## Scope

The planned implementation is limited to the OpenAI Responses compatibility
surface in `program/agent-gateway`, its stable `agent-proxy` spec, and Gateway
usage documentation. Authentication, principal identity, model authorization,
quota accounting, audit schemas, storage lifecycle, and the separate AIW Agent
service API remain unchanged. Do not promise general OpenAI API compatibility.

## Work items

- [x] 1.1 Characterize the selected Agents SDK version and record exact request
  and response evidence for the agreed agent/tool flow. Acceptance: required
  routes, fields, stream mode, tracing behavior, and unsupported features are
  listed without inference.
- [ ] 1.2 Define the minimal Responses function-call contract and verify Runner
  feasibility. Acceptance: model function-call items, tool outputs, follow-up
  input, and state/usage accounting have explicit behavior and limits.
- [ ] 1.3 Implement the agreed non-streamed function-tool subset. Acceptance:
  supported SDK function-tool flow completes; unsupported fields still fail
  before Runner; auth, model allowlists, quotas, and redacted observability
  remain effective.
- [x] 1.4 Update the stable spec and README with supported SDK version, setup,
  tracing limits, and the previously selected non-streamed function-tool
  compatibility boundaries.
- [ ] 1.5 Record focused compile and externally authorized runtime evidence.
- [ ] 1.6 Characterize v0.18.0 streaming and structured-output contracts.
  Acceptance: exact SDK requests, supported combinations, SSE events, terminal
  states, and structured-output validation behavior are recorded from primary
  source evidence. Size: small; difficulty: medium; depends on 1.1.
- [ ] 1.7 Implement selected streaming behavior, including function-call events
  if SDK characterization requires them. Acceptance: event order, IDs,
  cancellation/failure terminals, and accounting match the selected contract.
  Size: medium; difficulty: medium; depends on 1.2 and 1.6.
- [ ] 1.8 Implement selected structured-output behavior. Acceptance: accepted
  format options and validation/failure behavior match the selected contract;
  unsupported combinations fail before Runner. Size: medium; difficulty:
  medium; depends on 1.2 and 1.6.
- [ ] 1.9 Update the stable spec and README for the new streaming and structured
  output scope. Acceptance: supported combinations and limitations match the
  implementation. Size: small; difficulty: low; depends on 1.7 and 1.8.

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- A pinned OpenAI Agents SDK agent can call the Gateway using its documented
  OpenAI-compatible Responses configuration for the selected function-tool,
  streaming, and structured-output flows.
- Function-call items, function-call outputs, and follow-up Responses requests
  round-trip using standard API shapes; function execution remains in the
  client application.
- Streaming emits events and terminal states following the characterized
  Responses contract for the selected flow.
- Structured output accepts only characterized format options and validates
  the completed result according to the selected contract.
- Unsupported routes and fields fail explicitly without starting the Runner.
- Logs and persistent audit data do not include prompts, tool arguments, tool
  results, API keys, or response text.
- No standalone Agent SDK server endpoint or broad OpenAI API compatibility is
  claimed.

## Verification

- Static evidence: `program/agent-gateway/protocol.go` rejects non-empty tools
  and permits only user/assistant text content; `openspec/specs/agent-proxy/spec.md`
  explicitly excludes tools, sessions, and full OpenAI API compatibility.
- Scope revision: the user selected streaming and structured output in addition
  to v0.18.0 function tools. Current code and stable spec still reject these
  combinations; this decision is a plan change, not implementation evidence.
- Static feasibility evidence: `program/agent-gateway/runner.go` passes
  `features.shell_tool=false`, `features.apps=false`, and
  `features.multi_agent=false` to Codex; it records only `agent_message` text
  from completed items. The local `codex exec --help` lists `--output-schema`
  for final response shape; no tool-call behavior was run or inferred from
  that help text. `program/aiw-agent/package.json` depends on `openai` rather
  than an Agents SDK, and no local `@openai/agents` installation was found.
  `program/aiw-agent/src/providers.ts` sends text and extracts text output;
  it does not execute function calls.
- No Agent SDK package, source, or behavior was installed or executed for this
  planning FD.
- Official v0.18.0 source: `packages/agents-openai/src/openaiResponsesModel.ts`
  (`_buildResponsesCreateRequest` and `_fetchResponse`) and
  `openaiResponsesConverter.ts` (`getInputItems`, function-call conversion).
  The package manifest pins `@openai/agents` to 0.18.0. This is source
  characterization, not a runtime SDK result.
- Runtime SDK and backend behavior: not run.
- `python scripts/compile.py` in `program/agent-gateway` exited 0 on the
  current source and retained no distributable artifact.
- Static code review traced `decodeRequest` → `Gateway.responses` → `execute`
  → `responseObject`, including rejection before quota acquisition and the
  existing audit/error paths. `git diff --check` reported no whitespace error.

## TODO

- Planner handoff (2026-10-04): preserve completed items 1.1 and 1.4; resolve
  the pinned SDK streaming/structured-output contract and `strict:true` tool
  semantics, then decide which remaining items are ready for Worker. Existing
  Gateway function-tool code is partial implementation, not accepted evidence.
- Confirm the structured Runner decision, JSON arguments, and failure behavior
  with runtime evidence. Function execution belongs to the caller.
- Characterize streaming events and structured-output wire shapes in the pinned
  SDK; define supported combinations before implementation.
- Define and enforce the advertised `strict:true` tool-argument guarantee, or
  explicitly reject that mode before starting the Runner.
- Inspect an installed v0.18.0 Agents SDK against the local Gateway with a
  focused offline test once runtime authorization exists. In particular,
  confirm Codex `--output-schema` output and the SDK's replayed input shape.
- Keep implementation and verification Work Items open until their acceptance
  evidence exists.

## Sources

- User request to generate an FD for Gateway interfaces needed by OpenAI Agents
  SDK.
- `program/agent-gateway/protocol.go`
- `program/agent-gateway/server.go`
- `program/agent-gateway/README.md`
- `openspec/specs/agent-proxy/spec.md`
- `https://raw.githubusercontent.com/openai/openai-agents-js/v0.18.0/packages/agents/package.json`
- `https://raw.githubusercontent.com/openai/openai-agents-js/v0.18.0/packages/agents-openai/src/openaiResponsesModel.ts`
- `https://raw.githubusercontent.com/openai/openai-agents-js/v0.18.0/packages/agents-openai/src/openaiResponsesConverter.ts`
- `https://openai.github.io/openai-agents-js/guides/tracing/`

**Closed:** 2026-10-04
**Disposition reason:** 用户决定放弃；尚有 SDK 契约、strict:true 语义及运行证据未解决。
