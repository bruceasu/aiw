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
* Run bounded foreground supervision with frozen model and compile plans
* Inspect durable execution, recovery, and auxiliary knowledge state

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
* Canonical Task metadata and runtime state live under `.ai/tasks/<task-id>/`.
  OpenSpec proposal, design, specs, and checklists live under
  `openspec/changes/<task-id>/`. The compatible `tasks.toml` filename remains readable.

## Build and Installation

```bash
go build -o aiw ./cmd/aiw
```

Windows executable:

```powershell
.\aiw.exe init
```

The workflow CLI is distributed as the `aiw-wf` plugin. Build both Windows
and Linux workflow binaries into `plugins/aiw-wf` with:

```bat
build.bat wf
```

`build.bat plugins` and `build.bat all` also build the workflow binaries before
copying the plugin tree to the install directory. The command is:

```text
aiw wf <operation> ...
```

Requirement Management is also distributed as the `aiw-req` plugin.
Build its Windows and Linux binaries into `plugins/aiw-req` with:

```bat
build.bat req
```

The Conventional Commit wizard is distributed as the `aiw-cz` plugin. Build
its Windows and Linux binaries into `plugins/aiw-cz` with:

```bat
build.bat cz
```

## Commands

```text
aiw --help
aiw help [command|topic]

aiw init [--no-setup] [--prompts] [--merge] [--force] [--template <name>]
aiw new <task-id> [--allow-unrelated-dirty] [--backend auto|openspec|native]
aiw list [--all]
aiw show <task-id>
aiw status <task-id> <status>
aiw done <task-id>
aiw archive <task-id> [--push] [--cleanup-wt] [--delete-branch] [--force]

aiw req new <slug> [title]
aiw req new --id <requirement-id> [title]
aiw req chat [requirement-id] [--provider NAME] [--model MODEL]
aiw req list [--all|--archived|--cancelled]
aiw req show <requirement-id>
aiw req capture <requirement-id> <artifact> --file <path>
aiw req approve <requirement-id> <APPROVED|DEFERRED|REJECTED> --by <actor> --reason <reason>
aiw req promote <requirement-id> --task <task-id>
aiw req archive <requirement-id> --reason <reason> [--by <actor>]
aiw req cancel <requirement-id> --reason <reason> [--by <actor>]

aiw wt add <task-id> [base-branch]
aiw wt rm <task-id> [--delete-branch] [--force]
aiw wt commit <task-id> "message"
aiw wt pull <task-id> [--conflict-handoff]
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
aiw wf <operation> <task-id> [arguments/options]
aiw wf repair-metadata [task-id] [--dry-run]
aiw workspace <operation> <task-id>
aiw wf run <task-id> [--execute] [--primary] [--provider NAME] [--model MODEL]
aiw wf supervise <task-id> <start|status|stop> [--provider NAME] [--model MODEL]
aiw completion <powershell|bash|zsh|fish>
aiw session status <task-id>
aiw session list
aiw session get <task-id>
aiw session finish <task-id>
aiw session archive <task-id>
aiw session delete <task-id> --yes
aiw session memory <append|show> <task-id> [TEXT]
aiw session handoff [show] <task-id> [focus]

aiw skills <list|discover|adopt|install|sync> [options]

aiw ask [--chat|--resume] [--system-prompt TEXT] [--system-prompt-file FILE] [--allow-path PATH] "PROMPT"

aiw prompts list
aiw prompts [template] [--merge] [--force]

aiw tcc [args...]       # TCC wrapper with automatic include/lib defaults
aiw git <subcommand>    # run: aiw git help
aiw cxs <subcommand>   # inspect and resume Codex CLI sessions
aiw github <subcommand> # GitHub Issues and PRs
aiw cz <subcommand>    # Conventional Commit wizard
aiw exec-java [args...] # run a Java main class through Maven
```

Use `aiw --help` or `aiw help <command>` for command discovery. Plugins provide
their own detailed help through `aiw <plugin> --help`.

See [AIW Ask](docs/usage/aiw-ask.md) for safe usage guidance, chat controls,
private session storage, and provider limitations.

## Common Workflows

### List Tasks

Run `aiw list` to show active Tasks, sorted by Task ID, with three fields:
Task ID, Workflow status, and the actual OpenSpec Change path. Use
`aiw list --all` (also `aiw task list --all`) to include archived Tasks and
insert an `ACTIVE`/`ARCHIVED` field before the path. Archiving does not replace
the Workflow status: an archived Task can still show `DONE` or `CANCELLED`.

Columns grow to fit complete IDs and statuses, with at least two spaces
between fields. Interactive terminals show `TASK / STATUS / PATH` headers;
`--all` adds `ARCHIVE`. Empty results print nothing. Pipes and redirected files
receive aligned data rows without headers or ANSI escapes. Supported terminals
color statuses: green for done, gray for draft/cancelled and the archive marker,
cyan for ready/running, yellow for waiting, and red for errors/blocked states.
Unknown states retain their text and default color. Nonempty `NO_COLOR`,
`TERM=dumb`, or unconfirmed ANSI capability disables color.

Discovery combines Change directories with `.ai/tasks/<task-id>/`, legacy
`.ai/<task-id>/`, and the supported archive roots. It does not recurse
into sessions, evidence, or arbitrary runtime directories. `task.toml` takes
precedence over the compatible `tasks.toml` filename. Unique historical Tasks
whose Change was archived but whose runtime directory stayed active are hidden
by default; `--all` shows their real archived Change path without moving them.
An explicit `aiw archive <task-id>` can finish pairing their stored records.
Conflicting identities or multiple archive matches are reported rather than
silently selecting a record.

If a Task has runtime data but no Change record, its path field shows
`规格已删除`. Missing individual Markdown files do not mark the Change deleted.
For uniquely identified Tasks in the selected list scope, AIW/Workflow Core
recreates only missing runtime files and reports this minimal reconstruction
on stderr. Existing valid data is preserved; lost execution history, workspace
bindings, and completion results are not guessed. Archived records are repaired
in their archive location, and the default list does not repair hidden archives.
Repeated listing leaves complete records unchanged.

Corrupt records, read failures, identity conflicts, and failed reconstruction
produce diagnostics and a nonzero exit code while other valid Tasks continue
to appear. A Task whose runtime summary cannot be read shows `RUNTIME_ERROR`.
Scripts must check the exit code before treating stdout as a complete list.
Listing does not create Attempts or leases, dispatch agents, move directories,
or recreate missing specifications or Sessions.

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
aiw wt pull payment-retry --conflict-handoff
```

This option is off by default. On eligible conflicts it writes
`proposal-request.md` and prints its path as `proposal handoff: ...`.
No Agent is started: give that file to an Agent to produce `proposal.patch`
and `proposal.md`, then use `aiw wt resolve review <task-id>` and, after review,
`aiw wt resolve apply <task-id> --confirm`. The old option is not accepted.

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
aiw wf plan payment-retry
aiw wf advance payment-retry
aiw wf run payment-retry
```

The first `run` is a preview. Execute one authorized step with:

```powershell
aiw wf run payment-retry --execute
```

AIW uses an isolated worktree by default for an executing step. Use
`--primary` only when you explicitly intend to run in the parent workspace:

```powershell
aiw wf run payment-retry --execute --primary
```

Inspect a blocked or interrupted workflow with:

```powershell
aiw wf diagnose payment-retry
aiw wf recover payment-retry
aiw wf repair payment-retry
```

### Preserve a requirement before creating a Task

Use Requirement Management when the request still needs clarification,
business approval, metric definition, or engineering option review:

```powershell
aiw req chat daily-withdrawal-report
aiw req show daily-withdrawal-report
aiw req approve daily-withdrawal-report APPROVED --by alice --reason "scope approved"
aiw req promote daily-withdrawal-report --task daily-withdrawal-report
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
becomes an engineering Task. Start with `aiw req chat [requirement-id]`:
it creates or resumes a durable Flow Session, selects the smallest useful
discussion phase, and asks for confirmation before each durable action.
Records are stored under `docs/requirements/<requirement-id>/`; promotion requires
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
Requirement workflow. It starts or resumes `aiw req chat`, selects the
generic discovery or an applicable finance method, and prepares durable actions for
explicit confirmation. Users do not need to remember artifact types, paths, or
CLI parameters.

Question quality is reviewed separately from automated contract tests. See the
[three fixed human-review cases](docs/usage/requirement-discovery-review.md).

Install the canonical repository Skill into the current project's managed Skill
location when needed:

```text
aiw skills install requirement-management
```

Examples:

```text
$requirement-management I need a daily withdrawal report.
aiw req chat daily-withdrawal-report
```

The conversation shows a confirmation checkpoint before it creates, captures,
approves, defers, rejects, or promotes. Confirm a displayed action with
`confirm` or `确认`; the lower-level `aiw req ...`
commands remain available for scripts and automation.

## Automatic development workflow

AIW separates Task orchestration from AI Session execution:

* `aiw wf` owns the managed Task plan, Work Items, Attempts, Gates,
  Evidence, write leases, recovery, and derived progress.
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
aiw req chat daily-withdrawal-report
aiw req approve daily-withdrawal-report APPROVED --by alice --reason "scope approved"
aiw req promote daily-withdrawal-report --task daily-withdrawal-report
```

Prepare and inspect the managed execution plan:

```text
aiw wf plan daily-withdrawal-report
aiw show daily-withdrawal-report
aiw wf advance daily-withdrawal-report
```

`advance` selects one dependency-satisfied Work Item and prepares one
Attempt-bound agent request. It does not start a model. Preview the same
bounded step with:

```text
aiw wf run daily-withdrawal-report
```

Execute exactly one managed agent turn only when that is authorized. Automated
execution creates or reuses an isolated `.wt/<task-id>` worktree by default:

```text
aiw wf run daily-withdrawal-report --execute
```

To deliberately execute in the verified primary workspace, use the explicit
opt-out:

```text
aiw wf run daily-withdrawal-report --execute --primary
```

The workflow facade dispatches `plan`, `sync`, `advance`, `run`,
`supervise`, `recommend-routing`, `attempt`, `evidence`, `gate`,
`skip-focused-test`, `complete`, `retry-policy`, `reopen`, `force-close`,
`focused-test`, `delivery`, `local-merge`, `delivery-failed`, `report`,
`diagnose`, `recover`, and `repair`. `repair-metadata` is a special form:
`aiw wf repair-metadata [task-id] [--dry-run]`. Run and `supervise ... start`
accept `--provider NAME`
and `--model MODEL`; `run` also accepts `--execute` and `--primary`, and
`--primary` requires `--execute`. Supervisor overrides apply only to `start`.

For example, delivery failures record their stage and detail, while the
focused-test operation takes an Attempt ID:

```text
aiw wf delivery-failed payment-retry merge "conflict in parent branch"
aiw wf focused-test payment-retry attempt-123
aiw wf repair-metadata payment-retry --dry-run
```

`--primary` is rejected without `--execute`; preview commands never create a
worktree.

The Runner creates a minimal Task-local `artifacts/handoff.md` when needed and
hands the selected Work Item to the internal Agent adapter. Existing handoff
content is preserved. A second write-capable Attempt for the same workspace is refused
while its lease is active.

### Supervised bounded execution

New Tasks currently default to **schema 9**. The commands below use its
Coder/compile/repair loop. The implemented schema 10 protocol requires a
controlled host and managed activation; `supervise start` does not enable it.

For a long-running local loop, start supervision explicitly. Supervisor
execution uses the same isolated-worktree default:

```text
aiw wf recommend-routing daily-withdrawal-report
aiw wf supervise daily-withdrawal-report start
aiw wf supervise daily-withdrawal-report status
aiw wf supervise daily-withdrawal-report stop
```

Supervisor dispatches sequential, non-interactive Agent turns. A completed
Session must return a valid structured outcome bound to the expected Attempt
and Session turn. Only `completed` proceeds to the frozen Compile Plan;
`blocked` opens a Gate, and `no-progress` follows the ordinary retry policy.
Compiler failures prepare repair turns in the same Attempt. Three consecutive
failures stop repair; a successful compile resets the separate counter.

Generate routing before starting a manually created Task. Requirement
promotion invokes `recommend-routing` automatically. A missing frozen Compile
Plan opens a Gate and requires a fresh prepared request after planning.
Do not prewarm a supervise run with `advance` or `run`: those commands can
persist a request before the supervised Compile Plan is frozen. Use `status`,
`diagnose`, and `report` for observation. `--primary` belongs to single-step
`run --execute`, not to `supervise`.

When an isolated Task completes execution and satisfies validation and Gate
checks, supervise automatically commits its changes, merges into its recorded
parent branch, verifies ancestry, and removes the merged Task worktree and
branch. It does not push or archive. Delivery failures preserve recovery
information. This automatic delivery describes the default schema 9 path.
See [Supervise](docs/supervise.md) for operation and recovery, and
[automatic coding](docs/auto-coding.md) for state ownership and execution boundaries.

Named models are defined under `[ai.profiles.<name>]` with `provider` and
`model`. Default routes are `analysis=fast`, `coder/tester=balanced`, and
`verifier=reasoning`; the current supervised dispatcher selects the `coder`
route. Missing or incomplete Profiles fall back to global `[ai]`. New requests
snapshot the resolved choice, including `start --provider/--model` overrides;
recovery and compiler repairs reuse it. The default schema 9 dispatcher does
not escalate models automatically.

Raw Session output is archived before interpretation. Requests with frozen
input references additionally require a matching implementation report,
including input identity, change references, and explicit fact sections.
An invalid report can receive one report-only supplement; a further failure
opens `report-manual-review`. A saved report alone does not accept the Work Item.

### Durable multi-stage protocol (schema 10)

The Core implements `coder -> report-validation -> compile -> tester ->
test-run -> acceptance -> accepted`. Tester uses an independent Session and
test-path write scope; the Runner consumes a frozen test manifest. Controlled
execution requires authorization, result and acceptance validators, budget
services, and platform activation evidence. Missing host isolation or assertion
review capability fails closed.

**This is not the default CLI execution loop.** The generic Runner blocks
schema 10 with a controlled-stage-adapter diagnostic and disables legacy
dispatch. There is no general CLI migration, grant, or Stop-clear command;
changing `schema_version` manually does not enable execution.

* Explicit `supervise stop` persists even without a foreground lease. Restart
  does not clear it or prove that an in-flight process exited.
* Unknown results must be reconciled through the original executor. Saved
  results can be consumed without replaying the model or command.
* Actor generation and repair budgets survive restart. Configured distinct
  model tiers support escalation; unknown historical budgets remain unknown.
* Local delivery requires a frozen, grant-bound plan and controlled host.
  Legacy `local-merge` is rejected. Cleanup needs its own grant and sealed
  source evidence; conflicts are preserved without implicit abort or reset.

Supervisor `status` also exposes protocol revision, budget-known state, Stop
reason, item phases, in-flight requests, recovery counts, and auxiliary gaps.
The ordinary retry/reopen instructions below describe schema 9, not a way to
reset these durable budgets or bypass Stop.

### Auxiliary work and knowledge

The controlled auxiliary host drains persisted Verifier, Task memory, and
project knowledge jobs within a bounded run. It preserves waiting and unknown
work for a later managed launch; it is not a polling daemon. Explicit Task
authorization, reviewed provider capability evidence, and host configuration
are required. Missing capabilities remain visible as host gaps.

```text
aiw wf auxiliary inventory
aiw wf knowledge show <task-id>
```

Explicit maintenance also includes `auxiliary policy <file>`,
`auxiliary initialize`, `auxiliary settle`, and
`knowledge review|import <task-id> <root> <file>`. These operations do not enable
schema 10 or authorize model/test execution. Inspect their help before changing
policy or publishing a knowledge review.

### Evidence, completion, and recovery

Agent output is evidence for review; it is not an automatic authorization to
declare the work complete. Use the managed workflow commands to record the
result and satisfy explicit preconditions:

```text
aiw wf evidence <task-id> <evidence-id> <work-item-id> <kind> <state> [reference]
aiw wf gate <task-id> <gate-id> <resolved|waived>
aiw wf complete <task-id> <work-item-id>
```

When a transition or projection is interrupted, inspect and repair the durable
record before continuing:

```text
aiw wf diagnose <task-id>
aiw wf report <task-id>
aiw wf recover <task-id>
aiw wf repair <task-id>
```

These commands do not replay an external agent turn. A pending Gate or missing
Evidence remains a blocker until the required human decision or authorized
validation is recorded.

If runtime state is missing, `supervise start` can initialize an inactive
projection only from valid, ID-matching Task metadata. A change directory or
migration marker cannot restore missing Attempt history. `status` does not
initialize missing state.

### Retry limits and terminal cancellation

In the default schema 9 path, each Work Item has an automatic Attempt limit of three. The operator
may set a limit from one through five, but only for that Work Item:

```powershell
aiw wf retry-policy daily-withdrawal-report wi-0001 3
```

Only structured no-progress outcomes consume this limit in supervise; blocked
outcomes do not. Git preflight failures create a deduplicated workspace-access
Gate before a new Attempt starts. Compiler failures have their own three-failure
limit and do not consume ordinary retries. At exhaustion, AIW blocks
the Work Item, clears its prepared request, releases its lease, and retains the
last output reference for diagnosis. It does not silently retry or mark the
checklist item complete. Reopen an exhausted Work Item only with an explicit
reason; reopening resets that item's count and does not affect other Work
Items. A blocked outcome must have its relevant Gates resolved before explicit
reopening; a non-exhausted ordinary retry count is preserved:

```powershell
aiw wf reopen daily-withdrawal-report wi-0001 "validation authorization granted"
```

When a managed Task must stop without fabricating completion, use a reasoned
force-close with an explicit delivery outcome:

```powershell
aiw wf force-close daily-withdrawal-report discarded "superseded by a replacement task"
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
* push, publish pull requests, or archive after supervised local delivery;
* create a background scheduler.

Use `aiw wf run` to preview the next action and
`aiw wf run --execute` or `supervise ... start` only with the
appropriate authorization. Default schema 9 supervision includes compile validation and automatic
local delivery for eligible isolated Tasks; ordinary `aiw done` does not itself
perform that delivery. Task completion and verified Git delivery remain distinct
states. Review the parent branch before pushing.

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
aiw wf focused-test <task-id> <attempt-id>
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
directory. Release layouts keep `skills/`, `cmd/`, and `plugins/` together;
reusable implementation packages live under `internal/`.
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

Plugin processes inherit the parent environment, including provider settings
such as `AIW_LLM_*`, `OPENAI_*`, and `AIW_PYTHON`. AIW also injects or
overrides the following variables:

| Variable                | Description                                |
| ----------------------- | ------------------------------------------ |
| `AIW_PLUGIN_NAME`       | Plugin name without the `aiw-` prefix      |
| `AIW_PLUGIN_PATH`       | Absolute path to the executed plugin       |
| `AIW_CMDLINE`           | Original command line after the subcommand |
| `AIW_HOME`              | Parent home directory (`HOME` or `USERPROFILE`) |
| `AIW_ROOT`              | Directory containing the root `aiw` executable |
| `AIW_WORKSPACE`         | Working directory from which the plugin was launched |

Configuration files are not copied into the plugin environment. The plugin
keeps the inherited working directory and resolves project `aiw.toml` or
`.aiw.toml` there; user and provider environment variables remain inherited.

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
|-- tasks.md
`-- notes.md

.ai/tasks/<task-id>/
`-- task.toml
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

## Built-in and Plugin Commands

Task lifecycle, Requirement Management, Workflow, Worktree, Session, prompt,
completion, and `ask` commands are exposed through their documented `aiw`
entry points. Other commands are executable plugins discovered beside the
`aiw` binary; their availability depends on the installed plugin set. Plugin
help is owned by each plugin, so use `aiw <plugin> --help` for its current
arguments and options. If a plugin is missing, `aiw` reports that it could not
discover the corresponding executable.





### Requirement history management

Requirement records remain independent from implementation and can be moved out of the active directory when their decision lifecycle is complete:

```text
docs/requirements/<id>/              active
docs/requirements/archive/<id>/      completed and archived
docs/requirements/cancelled/<id>/    cancelled and retained
```

Use `aiw req archive <id> --reason <reason> [--by <actor>]` for `DECIDED`, `APPROVED`, or `PROMOTED` requirements. Options may appear in either order; when `--by` is omitted, AIW uses the current OS user. Use the same form for `aiw req cancel`. `aiw req list` shows active records; add `--all`, `--archived`, or `--cancelled` to query history. `show` continues to resolve records by ID after they move.
