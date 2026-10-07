# aiw

AIW is a workflow-first CLI for organizing work with numbered Feature Designs,
role handoffs, Skills, AI support, and plugins. Existing Task and Workflow Core
records remain available for compatibility.

## Feature Overview

* Create numbered FDs and route PM, Planner, Worker, and Reviewer handoffs with `aiw fd`
* Initialize AIW Task, FD, and optional OpenSpec directories and default instruction files
* Create and deliver dedicated FD worktrees through `aiw git wt`
* Generate or merge AI prompt files from `agent-templates/`
* Create, view, and update tasks
* Capture, approve, split, and promote durable Issues before Task creation
* Create dedicated Git worktrees for tasks
* Output task-specific context prompts
* Create and maintain long-lived specification documents
* Archive completed tasks
* Read legacy bounded foreground supervision records during migration
* Inspect Task execution and recovery state

## Directory Structure

After running `aiw init`, the following structure will be created (if missing):

```text
repo/
|-- .ai/tasks/
|-- docs/features/
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
* New work uses `docs/features/FD-XXX_SLUG.md` and `.ai/fd/<fd-id>/` role
  receipts. Legacy Task metadata and runtime state live under
  `.ai/tasks/<task-id>/`; legacy FDs at `docs/features/<task-id>.md` and older
  `tasks.md` plans remain readable. Stable specs live under `openspec/specs/`.
  An OpenSpec change is optional.

See [FD-first workflow](docs/usage/aiw-fd.md) for new development work.

## Build and Installation

```bash
go build -o aiw ./cmd/aiw
```

Windows executable:

```powershell
.\aiw.exe init
```

Requirement Management is also distributed as the `aiw-req` plugin.
Build its Windows and Linux binaries into `plugins/aiw-req` with:

```bat
build.bat req
```

The Conventional Commit wizard is a TypeScript plugin requiring Node.js 22.12.0
or newer. Build its platform-specific release directory with:

```bat
build.bat cz
```

`build.bat plugins` installs the curated CZ release without its TypeScript
source or `node_modules`. See [CZ configuration](docs/usage/cz-configuration.md).

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
aiw issue <command> ... # preferred alias for aiw req
aiw issue link-parent <child-id> <parent-id>
aiw issue children <parent-id>

aiw git wt add <fd-id>
aiw git wt status <fd-id>
aiw git wt commit <fd-id> "message"
aiw git wt local-merge <fd-id>
aiw git wt list

aiw context <task-id>
aiw spec <spec-id>
Numbered FDs use explicit lifecycle and worktree commands:

```text
aiw fd list
aiw fd show FD-001
aiw git wt add FD-001
aiw git wt status FD-001
aiw git wt commit FD-001 "message"
aiw git wt local-merge FD-001
```

Use `aiw --help` or `aiw help <command>` for command discovery. Plugins provide
their own detailed help through `aiw <plugin> --help`.

See [AIW Ask](docs/usage/aiw-ask.md) for safe usage guidance, chat controls,
private session storage, and provider limitations.

## Common Workflows

### List Tasks

Run `aiw list` to show active Tasks, sorted by Task ID, with three fields:
Task ID, Workflow status, and the available FD or OpenSpec change path. Use
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

Record Task design decisions in `docs/features/payment-retry.md`. Use
`aiw spec <spec-id>` to create a new stable spec skeleton when needed;
`/to-spec` helps write or update its rules. `aiw done payment-retry` records
Task completion; it does not commit, push, merge, or publish code.

### Work in an isolated FD worktree

Commit the ready FD plan on a clean parent. Ensure `.wt/` and `.ai/` are
ignored by Git, then create the dedicated worktree. `wt add` checks both
paths before changing Git state and explains which ignore rule is missing.

```powershell
aiw git wt add FD-027
cd .wt\FD-027
```

`aiw git wt` stores the FD ID, parent branch, feature branch, and worktree path in
`.ai/fd/<fd-id>/workspace.json`. Use the FD ID for status and commits:

```powershell
aiw git wt status FD-027
aiw git wt commit FD-027 "implement feature"
```

After review, deliver one squash commit to the recorded parent branch:

```powershell
aiw git wt local-merge FD-027
```

If delivery conflicts in the parent, `local-merge` confirms the conflict,
resets that squash, verifies that the parent is clean, and merges the parent
branch into the FD worktree. Resolve and commit conflicts there, then rerun
`local-merge` explicitly. If the reset or recovery checks fail, the command
stops and preserves the current Git state. After a successful squash,
`local-merge` verifies the source SHA, then removes the recorded worktree and
branch. It preserves `.ai/fd/<FD-ID>` receipts and stops safely if a shared
`.ai` junction cannot be verified.

### Advance a numbered FD

Use explicit FD handoffs for planning, implementation, testing, and review.
Create the isolated worktree with `aiw git wt add` after committing the ready FD
plan and cleaning the parent workspace. After review, deliver with
`aiw git wt local-merge`; it creates one parent commit without retaining individual
FD commits. Parent-side content conflicts are recovered in the FD worktree and
require an explicit retry after resolution. The workflow does not rebase.
Successful delivery automatically removes the FD worktree and local feature
branch after checking the squash source; FD handoff receipts remain under
`.ai/fd/`.

```powershell
aiw fd list
aiw fd show FD-027
aiw git wt add FD-027
aiw git wt status FD-027
aiw git wt commit FD-027 "implement FD work"
aiw git wt local-merge FD-027
```

### Preserve an Issue before creating a Task

Use Issue Management when a bug, feature, or modification still needs scope,
approval, metric definition, or engineering option review:

```powershell
aiw issue chat daily-withdrawal-report
aiw issue show daily-withdrawal-report
aiw issue approve daily-withdrawal-report APPROVED --by alice --reason "scope approved"
aiw issue promote daily-withdrawal-report --task daily-withdrawal-report
```

Promotion requires an explicit approval and creates or reuses the linked Task.
It does not mean that the implementation is complete or approved for release.
See [Issue Management](docs/usage/aiw-issue.md) for artifact
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

## Issue Management

Issue Management preserves the discussion before it becomes an engineering
Task. Start with `aiw issue chat [id]`:
it creates or resumes a durable Flow Session, selects the smallest useful
discussion phase, and asks for confirmation before each durable action.
Records are stored under `docs/requirements/<requirement-id>/`; promotion requires
an explicit `APPROVED` decision and creates or reuses one AIW Task with
`artifacts/requirement-handoff.md`.

Supported captured artifacts are `problem-brief`, `business-case`,
`metric-brief`, `engineering-options`, and `requirement-plan`. Captured content
is copied from an explicit source file and recorded with a digest, so later
untracked edits are not silently handed to engineering.

See the [Issue Management user guide](docs/usage/aiw-issue.md) for
the lifecycle, commands, examples, recovery behavior, and Skill integration.

### Use the Issue Management Skill

Use `$issue-management` when you want an AI conversation to lead the
Issue workflow. It starts or resumes `aiw issue chat`, selects the
generic discovery or an applicable finance method, and prepares durable actions for
explicit confirmation. Users do not need to remember artifact types, paths, or
CLI parameters.

Question quality is reviewed separately from automated contract tests. See the
[three fixed human-review cases](docs/usage/requirement-discovery-review.md).

Install the canonical repository Skill into the current project's managed Skill
location when needed:

```text
aiw skills install issue-management
```

Examples:

```text
$issue-management I need a daily withdrawal report.
aiw issue chat daily-withdrawal-report
```

The conversation shows a confirmation checkpoint before it creates, captures,
approves, defers, rejects, or promotes. Confirm a displayed action with
`confirm` or `确认`; the compatible lower-level `aiw req ...`
commands remain available for scripts and automation.

## FD development workflow

Numbered FDs are the workflow for new engineering work. Use `aiw fd` for FD
lifecycle and handoffs, and `aiw git wt` for isolated worktrees, focused commits,
and local delivery. The removed `aiw wf` Task workflow is not an available
command. Existing Task records remain historical data; they are not converted
or used to create new work.

After a reviewed FD change, `aiw git wt local-merge <fd-id>` squash-delivers the
feature result to its recorded parent. If that squash has content conflicts,
the command resets the parent-side attempt and merges the parent into the FD
worktree. Resolve and commit there, then rerun `local-merge` to deliver.

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
.mjs
.cjs
.ts
.mts
.cts
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
5. JavaScript (`.js`, `.mjs`, `.cjs`), then TypeScript (`.ts`, `.mts`, `.cts`)

## Interpreters and Shebang

* Extensionless scripts with a `#!` shebang are executed using the specified interpreter.
* JavaScript plugins run with the configured Node runtime.
* TypeScript plugins run with the configured Node runtime and
  `--experimental-strip-types`. They require
  Node.js 22.12.0 or newer and may use only syntax supported by Node's type
  stripping; TypeScript imports must use the actual file extension.

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

### Node Interpreter Configuration

JavaScript and TypeScript plugins select Node in this order:

1. The absolute path in `AIW_NODE`
2. `[runtime].node` in the user `aiw.toml`
3. `[runtime].node` in `aiw.toml` beside the AIW executable
4. `node/node.exe` on Windows, or `node/node` on other platforms, beside the
   AIW executable
5. `node` from `PATH`

Use an absolute path to a Node executable to pin a version:

```toml
[runtime]
node = "C:/tools/node-v22.12.0/node.exe"
```

The user configuration locations and path validation rules are the same as
those for Python above. The project-root `aiw.toml` is not read for this
setting. An empty value is treated as unset.

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
* Does not automatically merge templates from `agent-templates/`

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
docs/features/<task-id>.md

.ai/tasks/<task-id>/
`-- task.toml
```

The FD starts `BLOCKED` and needs confirmed decisions and numbered work items
before creating a numbered FD. Use `--backend openspec` only when an
OpenSpec change is wanted.

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
`.ai/tasks/<task-id>/`、`docs/features/<task-id>.md` 无关时，才可显式确认：

```text
aiw new <task-id> --allow-unrelated-dirty
```

目标目录存在脏改动时始终拒绝；此规则不改变已有工件编辑、worktree、归档、
提交或推送等操作的保护策略。

## 3. Legacy `aiw decision <task-id>`

Creates `openspec/changes/<task-id>/design.md` only for an existing legacy
change. New Task decisions belong in `docs/features/<task-id>.md`. The command
remains callable for compatibility but is omitted from the main help.

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
.ai/tasks/<task-id>/                 -> .ai/archive/<YYYY-MM-DD>-<task-id>/
docs/features/<task-id>.md          -> docs/features/archive/<YYYY-MM-DD>-<task-id>.md
openspec/changes/<task-id>/         -> openspec/changes/archive/<YYYY-MM-DD>-<task-id>/  (when linked)
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

## 8. `aiw git wt <subcommand>`

`aiw git wt` is the canonical FD worktree interface and accepts numbered FD IDs.
It reads the parent branch, feature branch, and worktree path from
`.ai/fd/<fd-id>/workspace.json`; Task IDs are rejected.

| Subcommand | Description |
| --- | --- |
| `add <fd-id>` | Create `feature/<fd-id>` at `.wt/<fd-id>` and record its coordinates |
| `status <fd-id>` | Show the parent and FD worktree status |
| `commit <fd-id> "message"` | Stage and commit changes in the FD worktree |
| `local-merge <fd-id>` | Squash-deliver one commit, recover conflicts in the FD worktree, and clean up after verified delivery |
| `list` | List registered Git worktrees |

After the squash commit and its `FD-Source` are verified against the clean FD
worktree, `local-merge` removes that worktree and its `feature/<fd-id>` branch.
It retains `.ai/fd/<fd-id>` receipts. If the worktree `.ai` is a junction to
the primary workspace, cleanup removes only the verified junction itself. An
unsafe link or partial cleanup failure stops further deletion and reports the
remaining resources.

## 9. `aiw context <task-id>`

Prints the Task metadata, Issue handoff and authored plan in the bound workspace,
then relevant change/spec files, Workflow status, and the current or next ready
Work Item. It does not change Task state.

## 10. `aiw prompts [template] [--merge] [--force]`

Features:

* `aiw prompts list` lists available templates under `agent-templates/`
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
```

launches an external editor.

### AI Provider Support

Enabled only with `--llm`.

Uses CZ's own provider configuration. Automatic order: Copilot SDK, Codex SDK,
OpenAI SDK, then the manual wizard.

```bash
set OPENAI_API_KEY=your_api_key

# Configure a model under [cz.openai] in aiw.toml.
```

### Configuration Priority

```text
CLI and CZ environment variables
-> project root aiw.toml
-> AIW_ROOT aiw.toml
-> plugin directory cz.toml
```

Supported configuration files:

```text
aiw.toml
.aiw.toml
```

### CZ Provider Configuration

```text
CLI options
-> CZ_* environment variables
-> [cz] and [cz.<provider>] in aiw.toml
-> SDK authentication defaults
```

Example:

```toml
[cz]
llm = false
candidates = 3

[cz.copilot]
model = "your-copilot-model"

[cz.codex]
model = "your-codex-model"

[cz.openai]
model = "your-openai-model"
base_url = "https://api.openai.com/v1"
```

Optional environment overrides:

```text
CZ_LLM_PROVIDER
CZ_COPILOT_MODEL
CZ_CODEX_MODEL
CZ_OPENAI_MODEL
CZ_OPENAI_API_KEY
OPENAI_API_KEY
```

See [CZ configuration](docs/usage/cz-configuration.md) for provider-specific
settings and fallback behavior.

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

# FD worktree workflow
aiw git wt add FD-027
aiw git wt status FD-027
aiw git wt commit FD-027 "implement feature"
aiw git wt local-merge FD-027
aiw git wt list

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
