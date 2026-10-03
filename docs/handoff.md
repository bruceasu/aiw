# Handoff: Codex-based Shared AI Gateway

## 1. Objective

Build a small AI gateway that allows other people or applications to use AI capabilities backed by Codex, without requiring them to have their own OpenAI account and without exposing the host machine, local files, credentials, or development environment.

The gateway is intended to expose only AI capability.

It is **not** intended to expose:

- the host filesystem
- local projects
- SSH credentials
- AWS credentials
- Git credentials
- shell access to the host
- arbitrary command execution on the host
- access to existing Codex workspaces

The most important security principle is:

> Users must never be able to read or modify files from the host machine. Codex must run inside an isolated and preferably ephemeral execution environment that contains only data explicitly supplied for that request.

# 2. Technology decision

Use:

```
Gateway backend: Go
AI engine:       Codex CLI
Production isolation: Docker or equivalent isolated runner
Local development: process runner in an empty temporary directory
Frontend/UI: optional later, likely React/TypeScript
```

Do not use TypeScript/Node.js for the gateway unless requirements later become heavily UI-oriented.

Go is preferred because the gateway mainly handles:

- HTTP
- authentication
- streaming
- subprocess execution
- timeouts
- cancellation
- concurrency
- temporary directories
- container lifecycle
- cleanup
- rate limiting

The deployment target should remain very small.

Avoid unnecessary infrastructure such as:

- Kubernetes
- Redis initially
- message queues initially
- AWS API Gateway
- Cognito
- Lambda
- DynamoDB

Start with a standalone Go binary.

# 3. High-level architecture

Target architecture:

```
                User / Application
                        |
                  HTTPS + API key
                        |
                        v
               +------------------+
               |   AI Gateway     |
               |      Go          |
               +--------+---------+
                        |
                Authentication
                Rate limits
                Quotas
                Request validation
                        |
                        v
               +------------------+
               | Runner Manager   |
               +--------+---------+
                        |
                isolated execution
                        |
              +---------+----------+
              |                    |
        Dev mode              Production
              |                    |
       temp directory       Docker container
              |                    |
              +---------+----------+
                        |
                      Codex
                        |
                    OpenAI
```

The host filesystem must not be visible to the Codex runtime.

# 4. Security model

The security boundary must be structural, not prompt-based.

Do not rely on instructions such as:

```
"Do not read files outside the working directory."
```

Instead, ensure Codex physically cannot access those files.

For production:

```
Host
|
+-- normal local files
+-- project directories
+-- ~/.ssh
+-- ~/.aws
+-- Git credentials
+-- other secrets
|
+-- isolated Codex container
      |
      +-- /workspace
            temporary only
```

Never mount:

```
/home
/root
~/.ssh
~/.aws
local project directories
Docker socket
host filesystem
```

into the Codex container.

Particularly avoid:

```
-v /:/host
-v ~/projects:/projects
-v ~/.ssh:/root/.ssh
-v /var/run/docker.sock:/var/run/docker.sock
```

The gateway must not expose parameters that allow the user to disable these restrictions.

# 5. Separate security profiles

There are two conceptually different Codex use cases.

## AI Dev Manager Codex Worker

This worker may access an explicitly authorized project workspace.

Example:

```
AI Dev Manager
      |
      v
authorized project workspace
      |
      v
Codex worker
```

## Shared AI Gateway

This gateway must not access the host's project workspaces.

Example:

```
Shared user
     |
     v
AI Gateway
     |
     v
ephemeral empty sandbox
     |
     v
Codex
```

These should remain separate security profiles even if they reuse some implementation code.

# 6. Public API philosophy

Do not expose Codex CLI semantics directly.

Avoid exposing things such as:

```
codex exec
approval_policy
sandbox_mode
working directory
CLI arguments
shell command flags
internal Codex configuration
```

The external API should be stable and simple.

The user should think they are talking to an AI service, not controlling a Codex process.

# 7. Minimal V1 API

Start with only:

```
POST /v1/responses
GET  /v1/models
GET  /v1/usage
```

Potential later additions:

```
POST   /v1/sessions
POST   /v1/sessions/{id}/responses
DELETE /v1/sessions/{id}
```

Do not initially expose:

```
/files
/shell
/workspaces
/git
/commands
/system
```

# 8. Primary endpoint

Example:

```
POST /v1/responses
Authorization: Bearer gw_xxxxxxxxx
Content-Type: application/json
```

Request:

```
{
  "model": "coding",
  "input": "Write a Java function that parses this CSV."
}
```

Response:

```
{
  "id": "resp_01K...",
  "status": "completed",
  "output_text": "..."
}
```

The API should intentionally resemble a normal AI API.

# 9. Streaming

Support streaming using Server-Sent Events.

Example:

```
POST /v1/responses
Accept: text/event-stream
```

Possible events:

```
event: response.started
data: {"id":"resp_123"}

event: output.delta
data: {"text":"The "}

event: output.delta
data: {"text":"solution is..."}

event: response.completed
data: {"id":"resp_123"}
```

Prefer SSE over WebSocket initially.

Reasons:

- easier implementation
- easier reverse proxy support
- enough for one-way AI output
- simple client support
- works with curl

# 10. API authentication

The gateway uses its own credentials.

Example:

```
gw_alice_xxxxx
gw_bob_xxxxx
gw_internal_service_xxxxx
```

These are not OpenAI credentials.

Architecture:

```
Alice
  |
  | gw_alice_xxx
  v
Gateway
  |
  | upstream credential
  v
Codex / OpenAI
```

Each user/service should get a separate key.

Do not create one global gateway key shared by everyone.

Store only a hash/HMAC of the key.

Recommended DB representation:

```
key_id
principal_id
key_hash
enabled
created_at
last_used_at
```

# 11. Principal model

Use a generic principal abstraction.

Example:

```
principal
-----------------------------
alice              HUMAN
bob                HUMAN
ci-system          SERVICE
internal-agent     AGENT
```

This supports future usage accounting and permissions.

Suggested type:

```
HUMAN
SERVICE
AGENT
```

# 12. Rate limits and quotas

At minimum support:

```
requests per minute
concurrent requests
daily request limit
optional daily token/cost limit
```

Example:

```
users:
  - name: alice
    rpm: 10
    max_concurrent: 2
    daily_requests: 100

  - name: bob
    rpm: 5
    max_concurrent: 1
    daily_requests: 50
```

Do not introduce Redis initially.

Use in-memory rate limiting first.

If the gateway later runs as multiple replicas, replace it with a shared limiter.

# 13. Runner abstraction

Do not couple the HTTP API directly to Codex CLI.

Define an internal abstraction.

Example:

```
type Runner interface {
    Run(ctx context.Context, req Request) (<-chan Event, error)
}
```

Implementations:

```
type ProcessCodexRunner struct{}
type DockerCodexRunner struct{}
```

Possible future implementations:

```
type RemoteCodexRunner struct{}
type OpenAIRunner struct{}
type ClaudeRunner struct{}
type CopilotRunner struct{}
```

The HTTP API should not change when the backend changes.

# 14. Development runner

Development mode should be easy.

Configuration:

```
runner:
  mode: process
```

Each request should create a unique temporary directory.

Example:

```
/tmp/ai-gateway/
    resp_01KABC/
```

Then run Codex with that temporary directory as the working directory.

Conceptual flow:

```
HTTP request
    |
    v
create temp directory
    |
    v
launch Codex
    |
    v
stream result
    |
    v
terminate process
    |
    v
delete temp directory
```

Do not launch Codex from:

```
~/projects
D:\03_projects
repository root
user home directory
```

even during development.

# 15. Production runner

Production configuration:

```
runner:
  mode: docker
```

For each request/session:

```
create container
      |
      v
create empty workspace
      |
      v
pass only prompt/request data
      |
      v
run Codex
      |
      v
return output
      |
      v
destroy container
```

Container requirements:

```
no host filesystem mount
no SSH credentials
no AWS credentials
no Git credentials
no Docker socket
no user home mount
temporary filesystem
strict timeout
memory limit
CPU limit
process limit
```

Network access should ideally be restricted to what Codex actually requires.

# 16. Attachments

Users may eventually need to ask Codex to analyze files.

They must explicitly upload those files.

Example request:

```
{
  "model": "coding",
  "input": "Find the bug in this Java file.",
  "attachments": [
    {
      "name": "PaymentService.java",
      "content": "..."
    }
  ]
}
```

The gateway should write those files only inside the request sandbox.

Example:

```
sandbox/
   PaymentService.java
```

Then:

```
Codex analyzes uploaded file
        |
        v
response returned
        |
        v
sandbox deleted
```

The host filesystem remains inaccessible.

Future support may include:

```
ZIP
tar.gz
small source repositories
text documents
```

with upload-size restrictions.

# 17. Sessions

Sessions are optional and should not block V1.

Possible future API:

```
POST /v1/sessions
```

Response:

```
{
  "id": "ses_01K..."
}
```

Then:

```
POST /v1/sessions/ses_01K/responses
```

A session may retain:

```
conversation context
user-uploaded temporary files
Codex-generated temporary files
```

It must never retain or inherit host files.

When session expires:

```
destroy workspace
destroy container
delete temporary files
delete session state
```

# 18. Model/agent abstraction

Externally, do not expose implementation-specific Codex models unless necessary.

Prefer logical names:

```
general
coding
review
fast
deep
```

Example:

```
{
  "model": "coding",
  "input": "..."
}
```

Internal mapping:

```
models:
  general:
    backend: codex
    profile: general

  coding:
    backend: codex
    profile: coding

  review:
    backend: codex
    profile: readonly
```

This allows backend changes without changing client code.

# 19. Configuration

Suggested initial configuration:

```
server:
  listen: ":8080"

runner:
  mode: process
  max_concurrent: 2
  timeout: 300s

codex:
  executable: codex

models:
  general:
    profile: shared-ai

  coding:
    profile: shared-ai

limits:
  max_input_bytes: 1048576
  max_output_bytes: 4194304
```

Production:

```
runner:
  mode: docker
```

# 20. Suggested repository structure

Use:

```
ai-gateway/
├── cmd/
│   └── gateway/
│       └── main.go
│
├── internal/
│   ├── api/
│   │   ├── responses.go
│   │   ├── models.go
│   │   ├── usage.go
│   │   └── middleware.go
│   │
│   ├── auth/
│   │   ├── apikey.go
│   │   └── principal.go
│   │
│   ├── runner/
│   │   ├── runner.go
│   │   ├── event.go
│   │   └── request.go
│   │
│   ├── codex/
│   │   ├── process_runner.go
│   │   ├── docker_runner.go
│   │   └── parser.go
│   │
│   ├── sandbox/
│   │   ├── sandbox.go
│   │   ├── tempdir.go
│   │   └── cleanup.go
│   │
│   ├── quota/
│   │   ├── limiter.go
│   │   └── usage.go
│   │
│   ├── config/
│   │   └── config.go
│   │
│   └── service/
│       └── response_service.go
│
├── config.example.yaml
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

Keep dependencies minimal.

# 21. Request lifecycle

Recommended flow:

```
POST /v1/responses
       |
       v
validate API key
       |
       v
load principal
       |
       v
check rate limit
       |
       v
check concurrency quota
       |
       v
validate request
       |
       v
select logical model/profile
       |
       v
create isolated sandbox
       |
       v
start Codex runner
       |
       v
stream events/output
       |
       v
collect usage
       |
       v
destroy sandbox
       |
       v
record request summary
```

Cleanup must happen even after:

```
timeout
client disconnect
Codex crash
gateway cancellation
invalid output
panic
```

Use `defer` and context cancellation carefully.

# 22. Logging

Do not log full prompts/responses by default.

Recommended fields:

```
request_id
principal_id
model
start_time
duration_ms
status
input_size
output_size
runner
exit_code
error_type
```

For example:

```
{
  "request_id": "resp_01K...",
  "principal": "alice",
  "model": "coding",
  "duration_ms": 8211,
  "status": "completed",
  "runner": "docker"
}
```

Prompt/content logging should be opt-in.

# 23. Concurrency

For V1, use a semaphore instead of a queue.

Example concept:

```
var slots = make(chan struct{}, maxConcurrent)
```

Before running Codex:

```
slots <- struct{}{}
```

After completion:

```
<-slots
```

Combine this with:

```
context.WithTimeout(...)
```

Do not introduce Kafka/SQS/RabbitMQ initially.

# 24. Error model

Return stable error responses.

Example:

```
{
  "error": {
    "code": "rate_limit_exceeded",
    "message": "Too many requests."
  }
}
```

Suggested error codes:

```
invalid_api_key
forbidden
rate_limit_exceeded
quota_exceeded
invalid_request
unsupported_model
runner_unavailable
runner_timeout
runner_failed
internal_error
```

Do not leak:

```
host paths
shell commands
container IDs
environment variables
OpenAI credentials
internal stack traces
```

# 25. Cancellation

The runner interface must support context cancellation.

Cases:

```
client disconnect
timeout
gateway shutdown
explicit future cancel request
```

The gateway must terminate the Codex process/container.

Do not leave orphan Codex processes running.

# 26. Deployment

Initial development:

```
Go
Codex CLI
config.yaml
```

Run:

```
go run ./cmd/gateway
```

Production:

```
Caddy
   |
   v
Go Gateway
   |
   v
Docker Codex runners
```

Potential deployment host:

```
small Linux VM
EC2
mini-PC
dedicated Linux server
```

A dedicated machine is safer than exposing the user's personal development PC.

# 27. Reverse proxy

Use Caddy initially.

Responsibilities:

```
TLS
HTTPS
request-size limit
basic connection limits
reverse proxy
```

Architecture:

```
Internet / LAN
      |
     443
      |
    Caddy
      |
   localhost:8080
      |
  AI Gateway
```

Do not expose the Codex runner/container directly.

# 28. Non-goals for V1

Do not implement yet:

```
Web UI
organization management
OAuth
SSO
multi-region
distributed queue
Redis
Kubernetes
complex billing
GitHub integration
repository checkout
host workspace access
plugin ecosystem
tool marketplace
multi-agent orchestration
AI Dev Manager integration
```

These can be added after the secure basic gateway works.

# 29. Recommended implementation stages

## Stage 1 — Basic gateway

Implement:

```
POST /v1/responses
GET /v1/models
API key auth
ProcessCodexRunner
temporary working directory
request timeout
basic JSON response
```

Acceptance criteria:

- user can call gateway with curl
- Codex answers
- Codex starts inside an empty temporary directory
- no host project files are visible
- temp directory is deleted after request

## Stage 2 — Streaming

Add:

```
SSE
Codex output streaming
client disconnect cancellation
```

Acceptance criteria:

- curl `-N` displays incremental output
- process stops if client disconnects
- no zombie Codex process remains

## Stage 3 — Quota and concurrency

Add:

```
per-key rate limit
per-key concurrent limit
global concurrent limit
usage accounting
```

No Redis.

## Stage 4 — Docker isolation

Add:

```
DockerCodexRunner
ephemeral container
CPU/memory limits
no host mounts
automatic cleanup
```

Production should default to Docker runner.

## Stage 5 — Attachments

Add:

```
text/source file upload
size limit
sandbox-only storage
cleanup
```

Do not allow user-specified host paths.

## Stage 6 — Sessions

Only if there is clear demand.

Add ephemeral session workspace with TTL.

# 30. Important implementation rule

Never allow a request to contain something equivalent to:

```
{
  "cwd": "/some/path"
}
```

or:

```
{
  "mounts": [...]
}
```

or:

```
{
  "sandbox": "disabled"
}
```

or:

```
{
  "hostAccess": true
}
```

Such controls are server-side only.

# 31. Core design principles

Keep these principles fixed:

1. Codex is an implementation detail.
2. The public API represents AI capabilities, not host-machine capabilities.
3. Users cannot supply host filesystem paths.
4. Shared AI uses ephemeral isolated workspaces only.
5. No local credentials are inherited.
6. No host project directories are mounted.
7. Authentication is per user/service.
8. Rate limits and quotas are enforced by the gateway.
9. The HTTP API stays stable even if the backend changes.
10. Start with minimal infrastructure.

# 32. Initial target

The next implementation should focus only on this vertical slice:

```
curl
  |
  v
POST /v1/responses
  |
  v
API key authentication
  |
  v
temporary empty directory
  |
  v
Codex process
  |
  v
AI response
  |
  v
cleanup
```

Do not start with Docker, persistence, UI, sessions, or advanced quota management until this path works reliably.

The first milestone should prove:

> A remote client can use Codex as an AI service while having zero ability to inspect or modify the gateway host's local files.