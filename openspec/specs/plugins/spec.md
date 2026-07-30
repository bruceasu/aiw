# plugins Specification

## Purpose
Define shared runtime compatibility, discovery, help, and plugin-specific behavior for AIW plugin commands.
## Requirements
### Requirement: Python 3.9 subcommand discovery
The `aiw-git` dispatcher SHALL load and discover its bundled Python subcommands under Python 3.9 without failing because of unsupported dataclass options.

#### Scenario: Discover the guide subcommand
- **WHEN** the dispatcher scans the bundled `plugins/aiw-git` directory under Python 3.9
- **THEN** discovery completes and includes the `guide` subcommand

### Requirement: Guide model behavior remains compatible
The guide subcommand MUST preserve the fields and default values of its search match and generated-answer models after the compatibility change.

#### Scenario: Use default model values
- **WHEN** guide code creates its internal models without overriding optional fields
- **THEN** a search match defaults `extracted` to false and a generated answer defaults `save` to true

### Requirement: Explicit Python environment override
AIW SHALL use the absolute executable path in `AIW_PYTHON` for Python plugins when the environment variable contains a non-empty value.

#### Scenario: Environment overrides file configuration
- **WHEN** `AIW_PYTHON` names a valid Python executable and both user and program configuration specify different Python executables
- **THEN** AIW executes the Python plugin with the executable named by `AIW_PYTHON`

#### Scenario: Empty environment value is unset
- **WHEN** `AIW_PYTHON` is absent, empty, or contains only whitespace
- **THEN** AIW continues to file-based interpreter configuration

### Requirement: Layered AIW runtime configuration
AIW SHALL read `[runtime].python` from the user `aiw.toml` before using the program-directory default, while logically treating the user value as an override of the program value.

#### Scenario: User configuration overrides program defaults
- **WHEN** the user configuration and the `aiw.toml` beside the AIW executable specify different valid Python executable paths
- **THEN** AIW executes the Python plugin with the user-configured executable

#### Scenario: Program configuration supplies a default
- **WHEN** no environment or user value is configured and the `aiw.toml` beside the AIW executable specifies a valid Python executable
- **THEN** AIW executes the Python plugin with the program-configured executable

#### Scenario: Empty configured value is unset
- **WHEN** a discovered `[runtime].python` value is empty or whitespace-only
- **THEN** AIW continues to the next interpreter source

### Requirement: Platform user configuration discovery
AIW SHALL discover at most one user configuration file using the platform configuration location and the existing home-directory compatibility location.

#### Scenario: Windows canonical user configuration
- **WHEN** AIW runs on Windows and `%APPDATA%\aiw\aiw.toml` exists
- **THEN** AIW uses it as the user configuration file

#### Scenario: XDG user configuration
- **WHEN** AIW runs on an XDG platform, `XDG_CONFIG_HOME` is set, and `$XDG_CONFIG_HOME/aiw/aiw.toml` exists
- **THEN** AIW uses it as the user configuration file

#### Scenario: Home compatibility fallback
- **WHEN** the canonical user configuration does not exist and `$HOME/.config/aiw/aiw.toml` exists at a different path
- **THEN** AIW uses the home compatibility file

#### Scenario: Canonical file prevents compatibility merge
- **WHEN** both the canonical and home compatibility user configuration files exist
- **THEN** AIW uses only the canonical file and does not merge the compatibility file

### Requirement: Configuration reads have no write side effects
AIW MUST NOT create a user configuration directory or file while resolving a Python interpreter.

#### Scenario: User configuration is absent
- **WHEN** no user configuration directory or file exists and AIW resolves a Python interpreter
- **THEN** AIW leaves the user filesystem unchanged

### Requirement: Explicit interpreter validation
AIW MUST require every non-empty explicitly configured Python interpreter to be an absolute path to an existing non-directory file.

#### Scenario: Configured interpreter is valid
- **WHEN** the selected explicit value is an absolute path to an existing file
- **THEN** AIW uses that file as the Python interpreter

#### Scenario: Configured interpreter is relative
- **WHEN** the selected explicit value is a relative path
- **THEN** AIW stops before starting the plugin and reports which configuration source contains the invalid value

#### Scenario: Configured interpreter is missing
- **WHEN** the selected explicit value is an absolute path that does not identify an existing non-directory file
- **THEN** AIW stops before starting the plugin and reports which configuration source contains the invalid value

### Requirement: Backward-compatible interpreter fallback
AIW SHALL preserve bundled and system interpreter discovery when no explicit Python interpreter is configured.

#### Scenario: Bundled Python is available
- **WHEN** no explicit or bundled interpreter is available and a supported Python executable exists in the `python` directory beside the AIW executable
- **THEN** AIW uses the bundled executable

#### Scenario: System Python is used
- **WHEN** no explicit or bundled interpreter is available and `python` is available through `PATH`
- **THEN** AIW uses the resolved `python` executable

#### Scenario: Python3 is the final command candidate
- **WHEN** no explicit or bundled interpreter is available, `python` is absent from `PATH`, and `python3` is available
- **THEN** AIW uses the resolved `python3` executable

#### Scenario: No interpreter is available
- **WHEN** no explicit, bundled, or system Python interpreter is available
- **THEN** AIW returns an interpreter-not-found error without starting the plugin

### Requirement: Task-oriented plugin HELP
Every AIW plugin SHALL explain its purpose, recommend common usage scenarios, summarize every top-level command, provide quick-start examples, and direct users to command-specific HELP.

#### Scenario: First-time user requests plugin HELP
- **WHEN** a user runs an AIW plugin with `--help`
- **THEN** the output explains when to use the plugin and shows at least one representative workflow example

#### Scenario: User scans available commands
- **WHEN** a plugin's top-level HELP is displayed
- **THEN** every supported top-level command has a plain-language summary

### Requirement: Actionable command HELP
Every plugin command SHALL state its purpose, describe all positional and optional arguments, explain relevant usage constraints, recommend when the command is useful, and provide at least one valid example.

#### Scenario: User requests command HELP
- **WHEN** a user runs an AIW plugin command with `--help`
- **THEN** the output explains every argument, important constraints, a recommended usage scenario, and a complete command example

#### Scenario: Command has a risky option
- **WHEN** a command HELP page documents an option that can replace, delete, or otherwise invalidate existing state
- **THEN** the option description explains the risk before showing an example

### Requirement: Discoverable nested HELP
Every plugin command group SHALL summarize its child actions, and every child action SHALL describe its arguments, recommended usage scenarios, constraints, and examples.

#### Scenario: User explores a command group
- **WHEN** a user requests HELP for an AIW plugin command group
- **THEN** the output summarizes every child action and explains how to request detailed HELP

#### Scenario: User explores one nested action
- **WHEN** a user requests HELP for an AIW plugin nested action
- **THEN** the output describes every argument and shows a valid example and a recommended usage scenario

### Requirement: HELP terminology and layout
Plugin HELP SHALL use Easy English, consistent domain terminology, meaningful metavariables, and preserved multiline formatting for workflows and examples.

#### Scenario: HELP renders examples
- **WHEN** a plugin's argument parser renders any command HELP
- **THEN** multiline examples remain readable and use the actual command and option names

### Requirement: Parser compatibility
HELP improvements MUST NOT change command names, option names, required arguments, defaults, parsed destination names, or dispatch behavior.

#### Scenario: Existing command invocation
- **WHEN** an existing valid plugin argument list is parsed
- **THEN** it produces the same command and option values as before the HELP change

#### Scenario: Existing invalid invocation
- **WHEN** a required argument is omitted
- **THEN** the plugin's parser rejects the invocation as before

### Requirement: GitHub Issue body file input
The `aiw-github` Issue creation and update commands SHALL accept a Markdown body from a filesystem path or standard input without requiring the complete body as a command-line argument.

#### Scenario: Read body from a file
- **WHEN** a caller supplies a readable body file to an Issue creation or update command
- **THEN** the plugin sends the file contents as the GitHub Issue body

#### Scenario: Read body from standard input
- **WHEN** a caller selects standard input as the body source
- **THEN** the plugin reads the complete standard input stream and sends it as the GitHub Issue body

#### Scenario: Body source is unreadable
- **WHEN** the selected body file cannot be read
- **THEN** the plugin exits with a clear error before sending a GitHub request

### Requirement: Update an existing GitHub Issue
The `aiw-github` plugin SHALL provide an Issue update command that can replace the title or body of an existing Issue without creating a new Issue.

#### Scenario: Update a projected Issue
- **WHEN** a caller supplies a valid repository, Issue number, and updated title or body
- **THEN** the plugin patches that Issue and returns the updated Issue

#### Scenario: No update field is supplied
- **WHEN** the update command receives neither a title nor a body source
- **THEN** the plugin rejects the command without sending a GitHub request

### Requirement: Emit stable machine-readable Issue identity
In JSON mode, GitHub Issue creation, retrieval, update, and close commands SHALL emit machine-readable output containing at least the repository, Issue number, Issue URL, and Issue state.

#### Scenario: Capture a created Issue mapping
- **WHEN** an OpenSpec publication workflow creates an Issue with JSON output
- **THEN** it can read the repository, Issue number, URL, and state without parsing human-oriented terminal rendering

### Requirement: Document the actual aiw-github command surface
The `aiw-github` README and command HELP SHALL use the command names and option placement accepted by the parser.

#### Scenario: Follow an Issue listing example
- **WHEN** a user copies an Issue listing command from the README
- **THEN** the parser accepts the command without requiring an undocumented alias

#### Scenario: Use JSON output from documentation
- **WHEN** a user follows a documented JSON example
- **THEN** the example places the JSON option where the parser accepts it and produces machine-readable output

### Requirement: Preserve existing GitHub authentication and repository resolution
The new publication primitives MUST preserve `GITHUB_TOKEN` authentication, explicit `owner/repo` arguments, and current-repository discovery from the Git `origin` remote.

#### Scenario: Publish from the target repository
- **WHEN** a caller omits the repository inside a Git repository with a supported `origin` URL
- **THEN** the plugin resolves the repository as before

#### Scenario: GitHub token is absent
- **WHEN** a caller invokes a GitHub API command without `GITHUB_TOKEN`
- **THEN** the plugin exits before sending a GitHub request and reports the missing credential

### Requirement: File commit history
The system SHALL provide a read-only file history view that follows renames by default and SHALL support default, concise, patch, statistics, graph, and full-evolution output modes.

#### Scenario: Follow a renamed file
- **WHEN** a user requests history for a file that was renamed
- **THEN** the view follows the rename chain by default and shows the relevant commits

#### Scenario: Request concise output
- **WHEN** a user selects the concise output mode
- **THEN** the view produces a compact commit list with hashes, dates, and summaries

### Requirement: Confirm native Git fallback

The `aiw-git` dispatcher SHALL require explicit user confirmation before
delegating a subcommand that is not defined by `aiw-git` to native Git.

#### Scenario: Known local command takes precedence

- **WHEN** the requested subcommand is defined by `aiw-git`
- **THEN** the dispatcher uses the local command implementation and does not
  offer native fallback

#### Scenario: Unknown command asks before delegation

- **WHEN** the requested subcommand is not defined by `aiw-git` and an
  interactive terminal is available
- **THEN** the dispatcher displays the exact candidate `git <subcommand> ...`
  invocation and asks for confirmation before starting Git

#### Scenario: Fallback is refused by default

- **WHEN** the user enters an empty answer, EOF, or any answer other than `y`
  or `yes`
- **THEN** the dispatcher does not start Git and returns a non-zero refusal
  result

#### Scenario: Non-interactive fallback is refused

- **WHEN** the requested subcommand is unknown and no interactive terminal is
  available
- **THEN** the dispatcher reports that confirmation is required, does not
  block for input, and does not start Git

#### Scenario: Git arguments do not approve fallback

- **WHEN** an unknown command's arguments contain `--force`, `--yes`, or another
  Git option
- **THEN** those arguments are forwarded only after separate explicit fallback
  confirmation

#### Scenario: Approved fallback preserves native execution

- **WHEN** the user explicitly approves fallback
- **THEN** the dispatcher invokes `git` with the original arguments as distinct
  argv values, preserves native output, and returns Git's exit status

#### Scenario: Native Git is unavailable

- **WHEN** fallback is requested but no native Git executable is available
- **THEN** the dispatcher reports the missing executable and does not prompt or
  start a subprocess

#### Scenario: Unknown help also requires confirmation

- **WHEN** the user requests help for a subcommand that is not defined by
  `aiw-git`
- **THEN** the dispatcher treats `git help <subcommand>` as a native fallback
  candidate and applies the same confirmation policy

