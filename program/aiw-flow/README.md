# aiw-flow

`aiw-flow` is a Python CLI for running AI coding tasks. It manages Codex sessions, prompts, memory, threads, and execution results. It does not manage task branches, Git worktrees, or repository lifecycle.

Git tasks and worktrees should be managed by `aiw` / `aiw-wt`. `aiw-flow` only accepts an existing `--workspace` as the Codex working directory.

## Installation and Prerequisites

```bash
python -m pip install -e .
```

You need Python 3.9+ and a working `codex exec` CLI. When running through the aiw plugin, the commands below may be invoked as `aiw flow` instead of `aiw-flow`.

## State Directory

The default state directory is `.ai/` under the current working directory. You can also set a custom root with the global `--root` option:

```text
.ai/
??? sessions/<session-id>/
?   ??? status.json
?   ??? instructions.md
?   ??? memory.md
?   ??? events.jsonl
?   ??? prompts/
?   ??? outputs/
?   ??? artifacts/
??? locks/
??? logs/
??? archive/
```

```bash
aiw-flow --root D:/aiw-state new   --id BUG-1001-login   --title "Fix login timeout"   --workspace D:/repos/web   --instructions examples/coding-agent-instructions.md
```

`--root` must appear before the subcommand. The `.ai/` directory may also be used by other aiw features, so the team should standardize the state layout or give `aiw-flow` its own dedicated `--root`.

## Session Lifecycle

```text
new -> run -> continue (repeat) -> finish -> archive
  ?-> loop (interactive, repeatable)
grill -> loop (optional)
                                      ?-> delete
```

- `new`: register a task and save persistent instructions.
- `grill`: create a requirement-clarification session, collect a limited workspace summary, and start the first interview turn immediately.
- `run`: execute the first Codex prompt and record the thread ID.
- `continue`: reuse the existing thread for the next stage.
- `finish`: mark the session complete and optionally create execution artifacts.
- `archive`: archive a completed session.
- `delete`: delete only the session state saved by `aiw-flow`.

## Creating a Task: `new`

```text
aiw-flow new --id ID --title TITLE --workspace PATH --instructions FILE
  [--ephemeral] [--loop] [--phase PHASE] [--timeout SECONDS]
```

| Parameter | Required | Description |
| --- | --- | --- |
| `--id SESSION_ID` | Yes | Unique session ID, for example `GWJ-1234-order-slip`. Used only to identify the task and its state directory. |
| `--title TITLE` | No | Task title. Defaults to the session ID. |
| `--workspace PATH` | No | Existing AI execution directory. Defaults to the current directory. `aiw-flow` does not create, delete, repair, or switch worktrees. |
| `--instructions FILE` | No | UTF-8 persistent execution rules file. Defaults to `AGENTS.md`, `instructions.md`, or `README.md` in the workspace or current directory. |
| `--ephemeral` | No | Mark the task as temporary. |
| `--loop` | No | Enter the interactive loop immediately after creating the session. The first input will create and bind the Codex thread. |
| `--phase PHASE` | No | Phase used by `--loop`. Defaults to `interactive`. |
| `--timeout SECONDS` | No | Timeout for each Codex turn in the loop. |

Example: prepare the worktree first, then hand it to `aiw-flow`:

```bash
aiw wt add FEAT-204-export main

aiw-flow new   --id FEAT-204-export   --workspace .wt/FEAT-204-export
```

If you are not using a worktree, you can point `--workspace` directly at an existing repository:

```bash
aiw-flow new   --id BUG-1001-login   --workspace D:/repos/web
```

Real-world example: if I want to build a generic OAuth2.0 authentication and authorization service, I can organize the task like this:

```bash
aiw wt add AUTH-200-oauth2 main

aiw-flow new --id AUTH-200-oauth2 --workspace .wt/AUTH-200-oauth2

aiw-flow run AUTH-200-oauth2 --prompt "Design a reusable OAuth2.0 authentication and authorization service. The service should support client registration, authorization code flow, token issuance and refresh, token revocation, scope-based access control, and audit logging. Identify the minimal module boundaries, data model, and API endpoints before implementation."
```

If you want to clarify the scope before implementation, you can first use `grill` and then move into `run`:

```bash
aiw-flow grill --id AUTH-200-oauth2 --workspace .wt/AUTH-200-oauth2 --requirement "Build a generic OAuth2.0 authentication and authorization service for multiple client applications."
```

## Requirement Clarification: `grill`

```text
aiw-flow grill --id ID --title TITLE --workspace PATH
  (--requirement TEXT | --requirement-file FILE)
  [--timeout SECONDS] [--ephemeral]
```

`grill` uses built-in Easy English interview rules to create a normal `aiw-flow` session and immediately start the first Codex turn. The rules ask Codex to:

- inspect the workspace first and avoid asking about facts that local files already confirm,
- ask at most one user decision question per turn,
- provide a recommended answer and a reason for each question,
- emit `SUCCESS: Ready to execute.` and the final spec only when the user explicitly ends the grill,
- stay in clarification mode and not implement code.

```bash
aiw-flow grill   --id FEAT-204-export   --workspace .wt/FEAT-204-export   --requirement "Add an export workflow for operations users."   --loop
```

When `--loop` is present, the first question will wait for a reply immediately after it is completed. Without `--loop`, the command remains single-shot, and you can continue the next question with `continue`:

```bash
aiw-flow continue FEAT-204-export   --phase grill   --prompt "CSV is sufficient for the first release."
```

The first run creates `artifacts/workspace-context.md`. That summary:

- reads only explicitly allowed project metadata files, such as `README.md`, `AGENTS.md`, `go.mod`, and `pyproject.toml`,
- skips hidden directories, version control directories, dependency directories, caches, and build directories,
- limits directory depth, entry count, single-file byte count, and total byte count,
- replaces common passwords, tokens, secrets, and API key assignments before saving and sending to Codex,
- does not read `.env`, private keys, or arbitrary business files.

These limits reduce accidental exposure, but they are not a full secret scanner. Do not store real credentials in metadata files that are allowed to be read.

## First Turn: `run`

```text
aiw-flow run SESSION_ID --phase PHASE [--prompt TEXT] [--prompt-file FILE] [--timeout SECONDS] [--force-new-thread]
```

| Parameter | Required | Description |
| --- | --- | --- |
| `SESSION_ID` | Yes | Session ID created by `new`. |
| `--phase PHASE` | No | Stage name, such as `analyze`, `implement`, or `fix-tests`. Defaults to the current session phase, or `analyze` if none exists. |
| `--prompt TEXT` | No | Provide the prompt directly. |
| `--prompt-file FILE` | No | Read the prompt from a UTF-8 file. |
| `--timeout SECONDS` | No | Timeout for this Codex turn. |
| `--force-new-thread` | No | Ignore any existing thread ID and recreate the context. Use only when the original thread is unavailable. |

The prompt must come from at least one of `--prompt`, `--prompt-file`, or stdin. Multiple sources are concatenated in command-line, file, stdin order:

```bash
aiw-flow run BUG-1001-login   --phase analyze   --prompt "Find the root cause and propose a minimal fix."

aiw-flow run BUG-1001-login   --phase analyze   --prompt-file examples/analyze.md

Get-Content .	ask.md | aiw-flow run BUG-1001-login --phase analyze
```

## Continuing a Task: `continue`

```text
aiw-flow continue SESSION_ID --phase PHASE [--prompt TEXT] [--prompt-file FILE] [--timeout SECONDS]
```

`continue` requires a session that already has a thread ID and does not support selecting a different thread. A staged progression looks like this:

```bash
aiw-flow run GWJ-1234-order-slip   --phase analyze   --prompt-file examples/analyze.md

aiw-flow continue GWJ-1234-order-slip   --phase implement   --prompt-file examples/implement.md

aiw-flow continue GWJ-1234-order-slip   --phase fix-tests   --prompt-file examples/fix-tests.md
```

Each turn sent to Codex is composed from four parts: persistent instructions, session memory, the phase name, and the current prompt. Every prompt is saved under `prompts/`, the final output is saved under `outputs/`, and events are saved under `events.jsonl`.

## Interactive Session: `loop`

Loop is an optional interactive shell on top of single-shot commands:

```text
aiw-flow loop SESSION_ID [--phase PHASE] [--timeout SECONDS]
```

You can enter it from three places:

```bash
# 1. Create a normal session and immediately interact; the first input executes the first turn
aiw-flow new   --id FEAT-300-refactor   --title "Interactive refactor"   --workspace .wt/FEAT-300-refactor   --instructions examples/coding-agent-instructions.md   --loop   --phase analyze

# 2. Create a grill session and keep interacting after the first question
aiw-flow grill   --id FEAT-301-export   --title "Clarify export"   --workspace .wt/FEAT-301-export   --requirement "Add export support."   --loop

# 3. Resume an existing session
aiw-flow loop FEAT-301-export --phase grill
```

If `loop` does not specify `--phase`, it uses the session's current phase. If the session does not have a current phase, it uses `interactive`.

```text
Interactive loop for FEAT-301-export (phase: grill). Type /help for commands.
You> CSV is sufficient for the first release.
...
You> /done
SUCCESS: Ready to execute.
...
```

Loop supports the following local controls and Skill commands. Except for `/skill` and `/done`, local commands do not execute a Codex turn:

| Command | Behavior |
| --- | --- |
| `/help` | Show loop help. |
| `/status` | Show session status. |
| `/memory` | Show session memory. |
| `/handoff` | Generate `artifacts/handoff.md`. |
| `/fork` | Generate a handoff, use it as the business context for a new thread, run one new thread, then exit the loop. |
| `/skills` | List discoverable Codex Skills for the project and user scopes without executing a turn. |
| `/skill NAME MESSAGE` | Use Codex's native `$NAME` syntax to call a discovered skill and execute one normal turn. |
| `/done` | Only available in the `grill` phase; sends `Grill Done`, shows the final response, and exits. |
| `/exit` | Exit immediately without sending a new turn. |
| `//text` | Send a normal message that starts with `/`, for example `//review` sends `/review`. |

Skill discovery does not need extra configuration. Candidate directories are:

- `.agents/skills` at each level from the session workspace up to the Git repository root,
- `.codex/skills` at the Git repository root; non-Git workspaces use the workspace itself,
- `~/.agents/skills` under the user home directory,
- `skills` under the active Codex home, which defaults to `~/.codex/skills`.

Every candidate Skill must be a direct child directory containing `SKILL.md`, and its frontmatter must provide valid `name` and `description` fields. `/skills` shows scope, source path, invalid candidate warnings, and duplicate-name markers. When the same skill name appears in multiple locations, `/skill` reports all conflicting paths and refuses to guess priority.

```text
You> /skills
Project Skills:
  metrics-review - Review financial metric definitions.
    D:epos\demo\.agents\skills\metrics-review

You> /skill metrics-review Review the revenue metrics
```

You can also enter a Codex-native invocation directly, for example `$metrics-review Review the revenue metrics`; `aiw-flow` treats it as a normal message. `/skill` does not copy, install, or permanently activate a skill; the full `SKILL.md` and linked resources are still loaded on demand by Codex.

Empty input is ignored. EOF, `Ctrl+C` while waiting for input, and `/exit` all exit normally without changing session state. `running`, `completed`, `archived`, and `deleted` sessions cannot enter the loop.

Loop keeps one `aiw-flow` process alive, but each normal input still goes through the existing execution path and launches a single `codex exec`. Thread, prompt, output, event, timeout, and error behavior therefore match `run` and `continue`. The current version uses single-line input; for long prompts, prefer `--prompt-file` or stdin on the single-shot commands.

## Inspecting Task State

### `status`

```text
aiw-flow status SESSION_ID [--json]
```

Show status, thread ID, current phase, recent exit code, last output, and error information. `--json` is suitable for scripts or CI.

```bash
aiw-flow status GWJ-1234-order-slip
aiw-flow status GWJ-1234-order-slip --json
```

### `list`

```text
aiw-flow list [--state STATE]
```

List AI sessions, optionally filtered by state:

```bash
aiw-flow list
aiw-flow list --state active
```

### `inspect`

```text
aiw-flow inspect SESSION_ID
```

Print the full state, recent events, and a Memory summary. This is useful when investigating a failed turn or an unbound thread.

## Memory Management

Memory records only AI task context; it does not manage the repository or branches.

```text
aiw-flow memory show SESSION_ID
aiw-flow memory append SESSION_ID --text TEXT
aiw-flow memory replace SESSION_ID --file FILE
```

```bash
aiw-flow memory append BUG-1001-login   --text "Confirmed: timeout occurs only when the refresh token is expired."

aiw-flow memory show BUG-1001-login
aiw-flow memory replace BUG-1001-login --file notes/confirmed-findings.md
```

## Session Handoff: `handoff`

```text
aiw-flow handoff create SESSION_ID [--focus TEXT]
aiw-flow handoff show SESSION_ID
```

`handoff create` does not call a model. It produces a deterministic `artifacts/handoff.md` from the session state, memory, recent output, and existing artifact paths.

```bash
aiw-flow handoff create FEAT-204-export   --focus "Continue from validation and resolve the encoding decision."

aiw-flow handoff show FEAT-204-export
```

The handoff document includes Goal, Current State, Confirmed Findings, Decisions, Modified Files, Validation State, Open Issues, Recommended Next Action, Suggested Skills, and Artifact References. Only a limited excerpt of the latest output is saved; the full content is still referenced through the `outputs/` path.

Handoff writes use session locks and atomic replacement, so they do not depend on a system temp directory or a `next-agent` shell function, and they do not accidentally read another workspace's "latest file".

## Completion and Archiving

```text
aiw-flow finish SESSION_ID [--create-patch]
aiw-flow archive SESSION_ID
aiw-flow delete SESSION_ID --yes
```

`finish` marks the task complete. `--create-patch` only generates review artifacts from the current workspace; it does not create or clean worktrees. Git branch and worktree cleanup is handled by `aiw-wt`:

```bash
aiw-flow finish FEAT-204-export --create-patch
aiw archive FEAT-204-export --cleanup-wt --delete-branch
```

`delete` removes only `aiw-flow` session state; it does not delete the workspace, branch, or worktree.

## Configuration File

The global configuration file is located at:

- Windows: `%APPDATA%iw-flow\config.toml`
- Linux/macOS: `$XDG_CONFIG_HOME/aiw-flow/config.toml`, or `~/.config/aiw-flow/config.toml` when unset.

```toml
model = "gpt-5-codex"
profile = "default"
sandbox = "workspace-write"
codex_home = "D:/codex-home"
additional_codex_args = ["--color", "never"]
```

## Security Constraints

- `aiw-flow` does not run `commit`, `push`, `git reset --hard`, or `git clean -fd`.
- It does not use `shell=True`; Codex command arguments are passed as arrays.
- Do not write API keys, passwords, or other secrets into instructions, prompts, memory, or event logs.
- Parallel AI tasks should use `aiw-wt` to create independent workspaces and then create one session per workspace.

## Testing

```bash
python -m pytest
```

The tests do not require a real Codex installation. The backend uses a fake backend or a mock process.
