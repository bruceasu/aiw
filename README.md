# aiw

AIW is a workflow-first CLI for organizing work, preserving task state, and exposing reusable capabilities through a small core, Skills, AI support, and plugins.

## Feature Overview

* Initialize the OpenSpec-backed directory structure and default instruction files
* Automatically create or append `.wt/` entries to `.gitignore`
* Generate or merge AI prompt files from `docs/agent-templates/`
* Create, view, and update tasks
* Capture, approve, and promote durable Requirement records before Task creation
* Create dedicated Git worktrees for tasks
* Output task-specific context prompts
* Create and maintain long-lived specification documents
* Archive completed tasks

## Directory Structure

After running `aiw init`, the following structure will be created (if missing):

```text
repo/
|-- openspec/
|   |-- changes/
|   |-- specs/
|   `-- archive/
|-- .wt/
|-- AGENTS.md
`-- .github/
    `-- copilot-instructions.md
```

Notes:

* `AGENTS.md` and `.github/copilot-instructions.md` are created only if they do not already exist.
* Active Task metadata is discovered from `openspec/changes/*` task folders.
  `openspec/changes/archive/` is reserved for archived changes.
  Metadata discovery prefers `task.toml` and falls back to legacy `tasks.toml`.

## Build and Installation

```bash
go build -o aiw .
```

Windows executable:

```powershell
.\aiw.exe init
```

## Commands

```text
aiw --help
aiw help [command|topic]

aiw init [--no-setup] [--prompts] [--merge] [--force] [--template <name>]
aiw new <task-id> [--allow-unrelated-dirty] [--backend auto|openspec|native]
aiw list
aiw show <task-id>
aiw status <task-id> <status>
aiw done <task-id>
aiw archive <task-id> [--push] [--cleanup-wt] [--delete-branch] [--finalize] [--force]

aiw requirement new <requirement-id> [title]
aiw requirement chat [requirement-id] [--provider NAME] [--model MODEL]
aiw requirement show <requirement-id>
aiw requirement capture <requirement-id> <artifact> --file <path>
aiw requirement approve <requirement-id> <APPROVED|DEFERRED|REJECTED> --by <actor> --reason <reason>
aiw requirement promote <requirement-id> --task <task-id>

aiw wt add <task-id> [base-branch]
aiw wt rm <task-id> [--delete-branch] [--force]
aiw wt commit <task-id> "message"
aiw wt pull <task-id> [--resolve=agent]
aiw wt status <task-id>
aiw wt list [--porcelain]
aiw wt prune [--dry-run]
aiw wt lock <task-id> [reason]
aiw wt unlock <task-id>
aiw wt repair
aiw wt ignore

aiw context <task-id>
aiw decision <task-id>
aiw spec <spec-id>
aiw turn <task-id> [--handoff PATH] [--provider NAME] [--model MODEL] [--takeover] [--yes]
aiw chat <task-id> [--handoff PATH] [--provider NAME] [--model MODEL] [--takeover] [--yes]
aiw workflow <operation> <task-id>
aiw workspace <operation> <task-id>
aiw task workflow <plan|sync|advance|run|supervise|repair|attempt|evidence|gate|complete|diagnose|recover> <task-id>
aiw task workflow run <task-id> [--execute] [--primary] [--provider NAME] [--model MODEL]
aiw task workflow supervise <task-id> <start|status|stop> [--provider NAME] [--model MODEL]
aiw completion <powershell|bash|zsh|fish>
aiw session <status|list|get|finish|archive|delete|memory|handoff>

aiw ask [--chat|--resume] [--system-prompt TEXT] [--system-prompt-file FILE] [--allow-path PATH] "PROMPT"

aiw prompts list
aiw prompts [template] [--merge] [--force]

aiw tcc [args...]       # TCC wrapper with automatic include/lib defaults
aiw git <subcommand>    # run: aiw git help
```

Every built-in command accepts `-h`/`--help` where detailed command help is
available. Use `aiw help <command>` for the same overview from the top-level
help router. Unknown commands are resolved as plugins and use the same
`aiw <plugin> --help` convention.

See [AIW Ask](docs/usage/aiw-ask.md) for safe usage guidance, chat controls,
private session storage, and provider limitations.

## Common Workflows

### Start a normal Task

Use this flow when the work can be done in the current workspace:

```powershell
aiw init
aiw new payment-retry
aiw show payment-retry
aiw context payment-retry
aiw status payment-retry IN_PROGRESS
```

Use `aiw decision payment-retry` when the Task needs a design decision, and
`aiw spec <spec-id>` when you need to create or update a long-lived
specification. `aiw done payment-retry` records Task completion; it does not
commit, push, merge, or publish code.

### Work in an isolated worktree

Use a worktree when the Task should not modify the parent workspace, or when
multiple Tasks or agents need to work in parallel:

```powershell
aiw wt add payment-retry office
cd .wt\payment-retry

# edit files in this directory
git status
git add internal\retry.go
git commit -m "implement retry logic"
```

The equivalent AIW command stages all changes in that Task worktree:

```powershell
aiw wt commit payment-retry "implement retry logic"
```

`aiw wt` resolves the worktree from Task metadata, so you do not need to pass
`-C` when using `aiw wt commit`, `aiw wt status`, or `aiw wt pull`. Use `git -C`
only when running Git directly:

```powershell
git -C .wt\payment-retry status
git -C .wt\payment-retry add internal\retry.go
```

### Check before merging

Run status before delivery to see the worktree branch, changed files, latest
commits, parent workspace state, ahead/behind counts, and merge preview:

```powershell
aiw wt status payment-retry
```

If the worktree is clean and the merge preview is ready, merge the Task branch
back into its parent branch:

```powershell
aiw wt pull payment-retry
```

If that merge conflicts, AIW preserves Git's unresolved merge by default. An
operator may explicitly request a bounded, agent-assisted proposal for eligible
text files:

```powershell
aiw wt pull payment-retry --resolve=agent
```

The proposal is for review only. It never changes protected lifecycle or
metadata files, binary files, lockfiles, dependency lockfiles, or configured
sensitive paths; those conflicts require manual resolution. Reviewing or
accepting a proposal can stage only the approved eligible files. It never
creates a commit or completes the merge, so inspect the staged result and use
the normal Git merge flow deliberately.

After `pull`, review the result and archive explicitly when the Task is ready:

```powershell
aiw status payment-retry DONE
aiw archive payment-retry --cleanup-wt --delete-branch
```

Do not use `--delete-branch` until the branch has been merged or is otherwise
safe to remove. Use `aiw wt rm <task-id> --force` only when deliberately
discarding an isolated worktree.

### Run one bounded workflow step

Use the managed workflow when implementation should be prepared, evidenced,
and executed one bounded step at a time:

```powershell
aiw workflow plan payment-retry
aiw workflow advance payment-retry
aiw workflow run payment-retry
```

The first `run` is a preview. Execute one authorized step with:

```powershell
aiw workflow run payment-retry --execute
```

AIW uses an isolated worktree by default for an executing step. Use
`--primary` only when you explicitly intend to run in the parent workspace:

```powershell
aiw workflow run payment-retry --execute --primary
```

Inspect a blocked or interrupted workflow with:

```powershell
aiw workflow diagnose payment-retry
aiw workflow recover payment-retry
aiw workflow repair payment-retry
```

### Preserve a requirement before creating a Task

Use Requirement Management when the request still needs clarification,
business approval, metric definition, or engineering option review:

```powershell
aiw requirement chat daily-withdrawal-report
aiw requirement show daily-withdrawal-report
aiw requirement approve daily-withdrawal-report APPROVED --by alice --reason "scope approved"
aiw requirement promote daily-withdrawal-report --task daily-withdrawal-report
```

Promotion requires an explicit approval and creates or reuses the linked Task.
It does not mean that the implementation is complete or approved for release.
See [Requirement Management](docs/usage/aiw-requirement.md) for artifact
capture, recovery, and revision rules.

## Shell Completion

The completion generator supports PowerShell, Bash, Zsh, and Fish. It completes
top-level commands, common subcommands, and task IDs discovered from the
current directory's `openspec/changes` folder.

`aiw setup-project` installs persistent completion for the detected shell by
default. When the marked completion block is already present, it makes no
change. It cannot modify the parent shell process, so it prints the command to
copy and run when you want completion to take effect immediately. Set
`AIW_SHELL` to `powershell`, `bash`, `zsh`, or `fish` when automatic shell
detection is unsuitable.

### PowerShell

Enable it for the current session:

```powershell
aiw completion powershell | Out-String | Invoke-Expression
```

To enable it for future sessions, add the same command to `$PROFILE`. If
`aiw.exe` is not on `PATH`, use its full path, for example:

```powershell
.\aiw.exe completion powershell | Out-String | Invoke-Expression
```

### Bash

Current session:

```bash
source <(aiw completion bash)
```

Persistent setup, add the same line to `~/.bashrc`.

### Zsh

Current session:

```zsh
eval "$(aiw completion zsh)"
```

Persistent setup, add the same line to `~/.zshrc`.

### Fish

Current session:

```fish
aiw completion fish | source
```

For persistent setup, install the generated script into Fish's completion
directory:

```fish
aiw completion fish > ~/.config/fish/completions/aiw.fish
```

Run completion commands from the repository root, or from another directory
that contains the relevant `openspec/changes` folder, so task ID completion can
discover the available tasks.

## Workflow backend selection

Task workflow commands use `auto` by default. When a verified OpenSpec CLI is
available, `new` and `archive` delegate to it; otherwise AIW uses its native
implementation. `decision` and `spec` currently have no direct OpenSpec CLI
mapping and use native fallback in `auto` mode.

Use `--backend native` to force the built-in behavior, or
`--backend openspec` to require OpenSpec and fail if delegation is unavailable.
Configure a specific executable with `AIW_OPENSPEC_BIN`.

## Requirement Management

Requirement Management preserves the human requirement discussion before it
becomes an engineering Task. Start with `aiw requirement chat [requirement-id]`:
it creates or resumes a durable Flow Session, selects the smallest useful
discussion phase, and asks for confirmation before each durable action.
Records are stored under `requirements/<requirement-id>/`; promotion requires
an explicit `APPROVED` decision and creates or reuses one AIW Task with
`artifacts/requirement-handoff.md`.

Supported captured artifacts are `problem-brief`, `business-case`,
`metric-brief`, `engineering-options`, and `requirement-plan`. Captured content
is copied from an explicit source file and recorded with a digest, so later
untracked edits are not silently handed to engineering.

See the [Requirement Management user guide](docs/usage/aiw-requirement.md) for
the lifecycle, commands, examples, recovery behavior, and Skill integration.

### Use the Requirement Management Skill

Use `$requirement-management` when you want an AI conversation to lead the
Requirement workflow. It starts or resumes `aiw requirement chat`, selects the
needed finance discussion or deep discovery, and prepares durable actions for
explicit confirmation. Users do not need to remember artifact types, paths, or
CLI parameters.

Install the canonical repository Skill into the current project's managed Skill
location when needed:

```text
aiw skills install requirement-management
```

Examples:

```text
$requirement-management I need a daily withdrawal report.
aiw requirement chat daily-withdrawal-report
```

The conversation shows a confirmation checkpoint before it creates, captures,
approves, defers, rejects, or promotes. Confirm a displayed action with
`confirm` or `确认`; the lower-level `aiw requirement ...`
commands remain available for scripts and automation.

## Sequential agent handoff

`aiw turn <task-id>` consumes a handoff and starts one bounded Agent turn.
If the Task already exists, its Session, branch, and worktree are reused. If it
does not exist, AIW creates them from a handoff:

```text
handoff -> Task -> Session -> branch/worktree -> fresh Thread
```

Usage:

```text
aiw turn <task-id> [--handoff PATH] [--provider NAME] [--model MODEL] [--takeover] [--yes]
```

An existing Task normally needs metadata like:

```toml
session = "TASK-123"
worktree = ".wt/TASK-123"
```

The command acquires a per-task lease, resolves handoff sources in the order
`--handoff`, Task artifact, and Session artifact, and starts a fresh Codex
Thread. New Tasks copy the handoff into their own `artifacts/handoff.md`.
Lineage is recorded in `openspec/changes/<task-id>/agent-lineage.json`.
Running Sessions are refused unless `--takeover` is explicit. Invalid Task IDs
require confirmation with `--yes` and an interactive `y` response.
Use Workflow diagnostics to inspect the recorded Attempt and Session
transition. `aiw chat <task-id>` starts an interactive Codex or Copilot CLI in
the Task worktree and returns when that CLI exits.

This workflow is sequential. Use `aiw wt` to give parallel agents separate
worktrees and sessions. `aiw session` is the AIW Core Session inspection and
handoff surface;
`aiw cxs` remains focused on native Codex session navigation.
`task.toml` is canonical; `tasks.toml` is a legacy fallback.

## Automatic development workflow

AIW separates Task orchestration from AI Session execution:

* `aiw task workflow` owns the managed Task plan, Work Items, Attempts, Gates,
  Evidence, write leases, recovery, and derived progress.
* `aiw turn` hands one selected Work Item to a fresh or resumed Session.
* `aiw session` manages persisted AIW Sessions, their memory, handoffs, and backend resume IDs.
* OpenSpec owns proposal, design, capability specs, and the human-authored
  checklist in `tasks.md`.

The normal flow is:

```text
Requirement -> approved Task -> OpenSpec artifacts -> Work Items
  -> prepared Attempt -> one bounded agent turn -> Evidence/Gate
  -> next Work Item or human decision
```

### End-to-end example

Start with a Task whose `tasks.md` contains numbered checklist items. For a
requirement-led change, use Requirement Management first:

```text
aiw requirement chat daily-withdrawal-report
aiw requirement approve daily-withdrawal-report APPROVED --by alice --reason "scope approved"
aiw requirement promote daily-withdrawal-report --task daily-withdrawal-report
```

Prepare and inspect the managed execution plan:

```text
aiw task workflow plan daily-withdrawal-report
aiw show daily-withdrawal-report
aiw task workflow advance daily-withdrawal-report
```

`advance` selects one dependency-satisfied Work Item and prepares one
Attempt-bound agent request. It does not start a model. Preview the same
bounded step with:

```text
aiw task workflow run daily-withdrawal-report
```

Execute exactly one managed agent turn only when that is authorized. Automated
execution creates or reuses an isolated `.wt/<task-id>` worktree by default:

```text
aiw task workflow run daily-withdrawal-report --execute
```

To deliberately execute in the verified primary workspace, use the explicit
opt-out:

```text
aiw task workflow run daily-withdrawal-report --execute --primary
```

`--primary` is rejected without `--execute`; preview commands never create a
worktree.

The Runner creates a minimal Task-local `artifacts/handoff.md` when needed and
hands the selected Work Item to `aiw turn`. Existing handoff content
is preserved. A second write-capable Attempt for the same workspace is refused
while its lease is active.

### Supervised bounded execution

For a long-running local loop, start supervision explicitly. Supervisor
execution uses the same isolated-worktree default:

```text
aiw task workflow supervise daily-withdrawal-report start
aiw task workflow supervise daily-withdrawal-report status
aiw task workflow supervise daily-withdrawal-report stop
```

The Supervisor re-evaluates the Task only after durable changes and invokes one
bounded `run --execute` step at a time. It pauses on a Gate, authorization
requirement, failed or incomplete Session result, active lease, repair item, or
terminal no-work result. It does not run as a background scheduler unless the
operator starts it explicitly.

### Evidence, completion, and recovery

Agent output is evidence for review; it is not an automatic authorization to
declare the work complete. Use the managed workflow commands to record the
result and satisfy explicit preconditions:

```text
aiw task workflow evidence <task-id> <evidence-id> <work-item-id> <kind> <state> [reference]
aiw task workflow gate <task-id> <gate-id> <resolved|waived>
aiw task workflow complete <task-id> <work-item-id>
```

When a transition or projection is interrupted, inspect and repair the durable
record before continuing:

```text
aiw task workflow diagnose <task-id>
aiw task workflow recover <task-id>
aiw task workflow repair <task-id>
```

These commands do not replay an external agent turn. A pending Gate or missing
Evidence remains a blocker until the required human decision or authorized
validation is recorded.

### Retry limits and terminal cancellation

Each Work Item has an automatic Attempt limit of three by default. The operator
may set a limit from one through five, but only for that Work Item:

```powershell
aiw workflow retry-policy daily-withdrawal-report wi-0001 3
```

Failed or no-progress turns count toward that limit. At exhaustion, AIW blocks
the Work Item, clears its prepared request, releases its lease, and retains the
last output reference for diagnosis. It does not silently retry or mark the
checklist item complete. Reopen an exhausted Work Item only with an explicit
reason; reopening resets that item's count and does not affect other Work
Items:

```powershell
aiw workflow reopen daily-withdrawal-report wi-0001 "validation authorization granted"
```

When a managed Task must stop without fabricating completion, use a reasoned
force-close with an explicit delivery outcome:

```powershell
aiw workflow force-close daily-withdrawal-report discarded "superseded by a replacement task"
```

Force-close cancels managed Attempts, releases execution ownership, and records
`CANCELLED` with either `merged` or `discarded` delivery. It never marks
incomplete Work Items or pending validation as passed. `merged` is allowed only
after clean preflight confirms committed Task-branch content and the recorded
parent branch; a failed preflight or merge leaves the Task, branch, worktree,
conflict state, and recovery evidence intact. Cleanup, branch deletion, and
archive are separate, explicit operations that are permitted only after terminal
delivery is durable.

### Automation boundaries

Automatic development is deliberately bounded. AIW does not automatically:

* approve or promote a Requirement;
* resolve a Gate or grant validation authorization;
* run tests, builds, migrations, or broad verification without authorization;
* commit, push, merge, delete branches, or publish external changes;
* create a background scheduler.

Use `aiw task workflow run` to preview the next action and
`aiw task workflow run --execute` or `supervise ... start` only with the
appropriate authorization. Git delivery remains separate from Task completion;
`aiw done` and `aiw archive` do not mean that code has been pushed or merged.

### Focused-test pilot

The task-bound `focused-test` path is a pilot for this AIW repository only. It
is disabled by default for every Task. It runs one selected check only after a
human has explicitly authorized the current normalized Verification Plan
digest for the `focused-test` profile. The authorization records the approver
and approval time; changing the Plan makes that approval stale.

For a participating Task, keep the human-authored Plan at
`openspec/changes/<task-id>/artifacts/verification-plan.json`. A test agent
may select only a Plan `check_id`, with rationale and supplied evidence
references. It cannot provide a command, argv, directory, environment, or
network override. The controlled runner reads that stored selection and runs
the Plan entry in the owning Attempt worktree:

```text
aiw task workflow focused-test <task-id> <attempt-id>
```

The command does not grant authorization. Invalid Plan or selection data, and
an Attempt worktree that cannot contain the declared directory, stop before
process creation. Missing or stale approval, and an unenforceable
`network: deny` boundary, also stop before process creation and leave an
actionable Gate. A completed invocation stores bounded output and a result
artifact, then Workflow Core records the related command Evidence.

This pilot does not permit network-enabled tests, Git delivery, generic shell
or custom-script execution, automatic repair or retry loops, permission
escalation, or automatic Gate resolution.

For Session inspection, handoff, or native-resume-ID lookup, use `aiw session`. For inspecting or resuming native Codex
sessions, use `aiw cxs`. See the command-specific `--help` output before using
an unfamiliar operation.

# Plugin System

`aiw` supports extending subcommands through external executable plugins.

## Managed Skills

Use `aiw skills` to list and safely install canonical Portable Skills or
path-based Skill bundles into the current project's `.agents/skills`
directory:

```text
aiw skills list
aiw skills install tdd --dry-run
aiw skills install ./bundle.zip
aiw skills install tdd
aiw skills install --all
aiw skills discover [--json] [--scope user]
aiw skills adopt [--json] [--scope user]
aiw skills sync <skill> [--json] [--scope user]
```

The installer protects unmanaged same-name directories and records verified
AIW-managed copies in `.agents/skills/.aiw-skills.json`. Path-based installs
use the same managed pipeline as canonical Skill installs. Run
`aiw skills --help` for constraints and JSON automation options.

Use `aiw skills discover` to inspect installed and unmanaged Skills,
`aiw skills adopt` to record valid existing directories as managed, and
`aiw skills sync <skill>` to republish an already managed Skill. The default
scope is the current project; use `--scope user` for the shared
`~/.agents/skills` catalog. Project and user scopes keep independent
`.aiw-skills.json` manifests. `--json` returns one machine-readable result,
and `--dry-run` previews installation without writing files.

The installer validates frontmatter and copyable filesystem entries, protects
unmanaged same-name directories, stages and verifies content, and records
ownership in `.aiw-skills.json`. Path-based installs use the same managed
pipeline as canonical Skill installs. Run `aiw skills --help` for complete
constraints and JSON automation options.

Canonical Skill packages are maintained in the repository-root `skills/`
directory. Release layouts keep `skills/` beside `program/` and `plugins/`.
`aiw-install-skill` is deprecated; use `aiw skills install` for both canonical
names and local bundle sources.

When an unknown subcommand is invoked, `aiw` searches for an executable named `aiw-<plugin-name>` and executes it.

## Plugin Search Paths

Search order:

1. `plugins/` directory next to the `aiw` executable
2. `$HOME/.config/aiw/plugins`
3. System `PATH`

## Naming Convention

Plugin filenames must follow:

```text
aiw-<plugin-name>
```

Supported extensions:

```text
.exe
.py
.sh
.bat
.cmd
.ps1
.js
.jar
(no extension)
```

If subdirectories exist under `plugins/`, `aiw` recursively searches one level deeper.

## Execution Priority

When multiple matching plugins exist:

1. `.bat` / `.cmd` / `.sh`
2. `.py`
3. Extensionless scripts (shebang)
4. Native binaries (`.exe` / ELF)

## Interpreters and Shebang

* Extensionless scripts with a `#!` shebang are executed using the specified interpreter.
* For `.js` files, `bun` is preferred when available; otherwise `node` is used.

### Python Interpreter Configuration

Python plugins use the first available interpreter in this order:

1. The absolute path in `AIW_PYTHON`
2. `[runtime].python` in the user `aiw.toml`
3. `[runtime].python` in `aiw.toml` beside the AIW executable
4. `python/python.exe` on Windows, or `python/python` on other platforms, beside
   the AIW executable
5. `python`, then `python3`, from `PATH`

The program-directory configuration provides defaults. User configuration
overrides those defaults, and `AIW_PYTHON` provides a temporary environment
override.

```toml
[runtime]
python = "C:/Python312/python.exe"
```

Configured interpreter paths must be absolute paths to existing files. An
invalid explicit path produces an error instead of silently selecting another
Python runtime. An empty value is treated as unset.

The canonical user configuration locations are:

* Windows: `%APPDATA%\aiw\aiw.toml`
* Linux and other XDG platforms:
  `$XDG_CONFIG_HOME/aiw/aiw.toml`, or `$HOME/.config/aiw/aiw.toml` when
  `XDG_CONFIG_HOME` is unset
* macOS: `$HOME/Library/Application Support/aiw/aiw.toml`

If the canonical file does not exist, AIW also checks
`$HOME/.config/aiw/aiw.toml` as a compatibility fallback. AIW reads only the
first existing user configuration file and does not merge user files. It does
not read project-root configuration for interpreter selection, and it does not
create a missing user configuration file or directory.

## Environment Variables

The following variables are injected into plugin processes:

| Variable                | Description                                |
| ----------------------- | ------------------------------------------ |
| `AIW_PLUGIN_NAME`       | Plugin name without the `aiw-` prefix      |
| `AIW_PLUGIN_PATH`       | Absolute path to the executed plugin       |
| `AIW_CMDLINE`           | Original command line after the subcommand |
| `AIW_HOME` / `AIW_ROOT` | AIW configuration or installation root     |

## Example

Place `plugins/aiw-hello.sh` in the repository's `plugins/` directory:

```bash
aiw hello arg1 arg2
```

The plugin should process arguments via standard `argv` and write output to stdout/stderr. Its exit code becomes the exit code of `aiw`.

## Security Notice

Plugins execute arbitrary external code and may pose security risks. Only install trusted plugins. Consider signature verification or allowlists in production environments.

# Command Behavior Details

## 1. `aiw init`

* Creates directories such as `openspec/` and `.wt/`
* Writes default template files only when missing
* Creates or appends `.wt/` to `.gitignore`
* Does not automatically merge templates from `docs/agent-templates/`

Options:

* `--prompts`
  Run prompt synchronization immediately after initialization.

* `--merge`
  Valid only with `--prompts`. Merge content into existing prompt files.

* `--force`
  Valid only with `--prompts`. Overwrite existing prompt files.

* `--template <name>`
  Valid only with `--prompts`. Explicitly specify the template directory (`go`, `java`, or `python`).

## 2. `aiw new <task-id>`

Creates:

```text
openspec/changes/<task-id>/
|-- task.toml
|-- tasks.md
`-- notes.md
```

Default metadata:

```toml
type = "task"
status = "TODO"
branch = "<current-branch>"
parent_branch = "<current-branch>"
worktree = "."
workspace_kind = "primary"
delivery = "unmanaged"
```

如果工作区有未提交改动，默认会拒绝创建。仅当所有脏路径都与新建
`openspec/changes/<task-id>/` 无关时，才可显式确认：

```text
aiw new <task-id> --allow-unrelated-dirty
```

目标目录存在脏改动时始终拒绝；此规则不改变已有工件编辑、worktree、归档、
提交或推送等操作的保护策略。

## 3. `aiw decision <task-id>`

Creates `design.md` for the task if it does not already exist.

## 4. `aiw spec <spec-id>`

Creates:

```text
openspec/specs/<spec-id>/
  - spec.toml
  - spec.md
```

## 5. `aiw status <task-id> <status>`

Updates:

* `task.toml` (or legacy `tasks.toml` if present)
* `status` (converted to uppercase)
* `updated`

## 6. `aiw done <task-id>`

Equivalent to:

```bash
aiw status <task-id> DONE
```

Does not archive the task automatically.

## 7. `aiw archive <task-id>`

Moves:

```text
openspec/changes/<task-id>
```

to:

```text
openspec/archive/<YYYY-MM-DD>-<task-id>
```

Options:

* `--push`
  Execute:

  ```bash
  git push -u origin feature/<task-id>
  ```

* `--cleanup-wt`
  Remove the task worktree.

* `--delete-branch`
  Delete the local feature branch.

* `--finalize`
  Deprecated. It warns and performs only the local cleanup equivalent to:

  ```text
  --cleanup-wt --delete-branch
  ```

  It never pushes implicitly; use `--push` explicitly when that is intended.

## 8. `aiw wt <subcommand>`

Worktree management commands (`aiw wt help` for details).

| Subcommand             | Description                                                                 |
| ---------------------- | --------------------------------------------------------------------------- |
| `add <task-id> [base]` | Explicitly isolate a Task on branch `feature/<task-id>` using its recorded parent branch by default |
| `commit <task-id> "message"` | Stage and commit changes in the Task worktree |
| `pull <task-id>` | Merge the Task branch into its recorded parent branch |
| `status <task-id>` | Show worktree state and merge readiness |
| `discard <task-id> --yes` | Discard a confirmed isolated experiment |
| `rm <task-id>`         | Remove a worktree                                                           |
| `list`                 | List all worktrees                                                          |
| `prune`                | Remove stale worktree metadata                                              |
| `lock`                 | Protect a worktree from accidental removal                                  |
| `unlock`               | Unlock a worktree                                                           |
| `repair`               | Repair worktree links after path relocation                                 |
| `ignore`               | Add `.wt/` to `.gitignore`                                                  |

`add` executes:

```bash
git worktree add .wt/<task-id> -b feature/<task-id> <base>
```

and updates:

```text
branch
worktree
updated
```

in `task.toml`.

## 9. `aiw context <task-id>`

Prints recommended files to review and execution constraints for the task.

## 10. `aiw prompts [template] [--merge] [--force]`

Features:

* `aiw prompts list` lists available templates under `docs/agent-templates/`
* Generates or merges repository-level AI prompt files

Auto-detected templates:

| Template | Detection                                        |
| -------- | ------------------------------------------------ |
| `go`     | `go.mod`                                         |
| `java`   | `pom.xml`, `build.gradle`, `build.gradle.kts`    |
| `python` | `pyproject.toml`, `requirements.txt`, `setup.py` |

Output files:

```text
AGENTS.md
.github/copilot-instructions.md
CODEX.md
```

Behavior:

* Default: create missing files only
* `--merge`: merge into AIW-managed sections
* `--force`: overwrite target files

Summary output reports:

```text
created
merged
wrote
skipped existing
```

## 11. `aiw wt ignore`

Creates `.gitignore` or appends:

```text
.wt/
```

If the rule already exists, no duplicate entry is added.

# Git Utilities

## `aiw git cz` (Conventional Commit Wizard)

### Default Behavior

* LLM disabled by default
* Interactive bilingual wizard for:

  * type
  * scope
  * subject
  * body
  * breaking changes
  * footer

### Long-Text Editing

For body/breaking/footer fields:

```text
/edit
Ctrl+E
```

launches an external editor.

### AI Provider Support

Enabled only with `--llm`.

Uses the globally configured AI provider through the shared provider module.

```bash
set OPENAI_API_KEY=your_api_key

# optional
set OPENAI_MODEL=gpt-4o-mini
set OPENAI_BASE_URL=https://api.openai.com/v1
```

### Configuration Priority

```text
CLI
-> Project Root Configuration
-> Program Directory Configuration
```

Supported configuration files:

```text
aiw.toml
.aiw.toml
```

### AI Provider Configuration Priority

```text
[ai] section in config
-> environment variables
-> .env in current directory
-> .env in program directory
-> defaults
```

Supported shared options include:

```toml
provider
model
base_url
api_key
codex_command
copilot_command
```

Mapping:

```text
OPENAI_MODEL
OPENAI_BASE_URL
OPENAI_API_KEY
```

Example:

```toml
[ai]
provider = "openai"
llm = false
candidates = 3
emoji = false
EDITOR = "code --wait"
model = "gpt-4o-mini"
base_url = "https://api.openai.com/v1"
api_key = ""

[[cz.types]]
value = "feat"
name = "feat:     New Feature | A new feature"

[[cz.types]]
value = "fix"
name = "fix:      Bug Fix | A bug fix"
```

The provider settings are global and are shared by `aiw ask`, managed workflow
turns, and CZ. Existing provider settings under `[cz]` remain supported as a
compatibility fallback.

### CZ language

CZ includes `en`, `zh`, and `ja` language packs. Set a global default for the
commands that support internationalization:

```toml
[i18n]
default_language = "zh"
```

`aiw cz --lang LANGUAGE` overrides that default for one invocation. A
non-built-in language such as French is configured by name; its unspecified
messages, types, and scopes fall back to English:

```toml
[i18n]
default_language = "fr"

[cz.locales.fr.messages]
type = "Choisissez le type de commit :"
subject = "Sujet :"

[[cz.locales.fr.types]]
value = "feat"
name = "feat:     Nouvelle fonctionnalité"
```

### New Features

* Supports `--retry` (`-r`) to restore the most recent commit as a draft for amendment or resubmission.
* Interactive `issue-prefix` selection now supports both predefined options and custom input.

## task.toml Format

```toml
id = "payment-retry"
type = "task"
status = "TODO"
created = "2026-05-28"
updated = "2026-05-28"
branch = "feature/payment-retry"
worktree = ".wt/payment-retry"
```

Compatibility note:

* Existing repositories that still use `tasks.toml` are supported.

## task-id / spec-id Rules

Allowed characters:

```text
a-z
A-Z
0-9
-
_
.
```

Any other character is considered invalid.

## Quick Start

```bash
# Initialize repository
aiw init
aiw init --prompts --merge
aiw init --prompts --template go

# Task workflow
aiw new payment-retry
aiw wt add payment-retry
aiw context payment-retry
aiw status payment-retry IN_PROGRESS
aiw done payment-retry
aiw archive payment-retry --cleanup-wt --delete-branch

# Worktrees
aiw wt list
aiw wt prune --dry-run
aiw wt lock payment-retry "in review"
aiw wt rm payment-retry --delete-branch
aiw wt ignore

# Git utilities
aiw git st
aiw git save "feat: add retry"
aiw git sync
aiw git update main
aiw git log
aiw git help

# TCC wrapper
aiw tcc hello.c -o hello.exe
aiw tcc dll hello.c -o hello.dll
aiw tcc x86_64 hello.c -o hello.exe
aiw tcc run hello.c

# Prompts
aiw prompts list
aiw prompts go --merge

# Codex sessions (plugin)
aiw cxs list -n 20
aiw cxs list --current-workspace --json -n 20
aiw cxs gui
aiw cxs exec "summarize current diff"
aiw cxs exec --session payment-retry "continue implementation"

# Help
aiw --help
aiw help git
aiw session --help
aiw github --help
aiw skills --help
```

For full `aiw cxs` usage, see `docs/usage/aiw-cxs.md`.

## Available Plugins

Plugins are discovered beside the `aiw` executable. The repository currently
ships these common entry points as external plugins:

| Command | Purpose | Detailed help |
| --- | --- | --- |
| `aiw wt` | Create and maintain task worktrees | `aiw wt --help` |
| `aiw git` | Git helpers and discoverable Git subcommands | `aiw git help` |
| `aiw session` | Persisted Session inspection and native resume-ID lookup | `aiw session --help` |
| `aiw cxs` | Auxiliary inspection and continuation for Codex CLI sessions | `aiw cxs --help` |
| `aiw skills` | List and install canonical Skills | `aiw skills --help` |
| `aiw exec-java` | Run a Java main class through Maven in single- or multi-module projects | `aiw exec-java --help` |
| `aiw github` | Read and publish GitHub Issues and PRs | `aiw github --help` |
| `aiw cz` | Run the Conventional Commit wizard | `aiw cz --help` |
| `aiw tcc` | Compile or run Tiny C Compiler programs | `aiw tcc --help` |

Plugin HELP is intentionally generated by each plugin so its examples stay
next to the parser. If a plugin is not installed, the top-level help reports
the missing discovery location instead of showing stale commands.





### Requirement history management

Requirement records remain independent from implementation and can be moved out of the active directory when their decision lifecycle is complete:

```text
requirements/<id>/              active
requirements/archive/<id>/      completed and archived
requirements/cancelled/<id>/    cancelled and retained
```

Use `aiw requirement archive <id> --reason <reason> [--by <actor>]` for `DECIDED`, `APPROVED`, or `PROMOTED` requirements. Options may appear in either order; when `--by` is omitted, AIW uses the current OS user. Use the same form for `aiw requirement cancel`. `aiw requirement list` shows active records; add `--all`, `--archived`, or `--cancelled` to query history. `show` continues to resolve records by ID after they move.
