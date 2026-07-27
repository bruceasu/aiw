## ADDED Requirements

### Requirement: GitHub Issue body file input
The `aiw-github` Issue creation and update commands SHALL accept a Markdown body
from a filesystem path or standard input without requiring the complete body as
a command-line argument.

#### Scenario: Read body from a file
- **WHEN** a caller supplies a readable body file to an Issue creation or update
  command
- **THEN** the plugin sends the file contents as the GitHub Issue body

#### Scenario: Read body from standard input
- **WHEN** a caller selects standard input as the body source
- **THEN** the plugin reads the complete standard input stream and sends it as
  the GitHub Issue body

#### Scenario: Body source is unreadable
- **WHEN** the selected body file cannot be read
- **THEN** the plugin exits with a clear error before sending a GitHub request

### Requirement: Update an existing GitHub Issue
The `aiw-github` plugin SHALL provide an Issue update command that can replace
the title or body of an existing Issue without creating a new Issue.

#### Scenario: Update a projected Issue
- **WHEN** a caller supplies a valid repository, Issue number, and updated title
  or body
- **THEN** the plugin patches that Issue and returns the updated Issue

#### Scenario: No update field is supplied
- **WHEN** the update command receives neither a title nor a body source
- **THEN** the plugin rejects the command without sending a GitHub request

### Requirement: Emit stable machine-readable Issue identity
In JSON mode, GitHub Issue creation, retrieval, update, and close commands SHALL
emit machine-readable output containing at least the repository, Issue number,
Issue URL, and Issue state.

#### Scenario: Capture a created Issue mapping
- **WHEN** an OpenSpec publication workflow creates an Issue with JSON output
- **THEN** it can read the repository, Issue number, URL, and state without
  parsing human-oriented terminal rendering

### Requirement: Document the actual aiw-github command surface
The `aiw-github` README and command HELP SHALL use the command names and option
placement accepted by the parser.

#### Scenario: Follow an Issue listing example
- **WHEN** a user copies an Issue listing command from the README
- **THEN** the parser accepts the command without requiring an undocumented
  alias

#### Scenario: Use JSON output from documentation
- **WHEN** a user follows a documented JSON example
- **THEN** the example places the JSON option where the parser accepts it and
  produces machine-readable output

### Requirement: Preserve existing GitHub authentication and repository resolution
The new publication primitives MUST preserve `GITHUB_TOKEN` authentication,
explicit `owner/repo` arguments, and current-repository discovery from the Git
`origin` remote.

#### Scenario: Publish from the target repository
- **WHEN** a caller omits the repository inside a Git repository with a
  supported `origin` URL
- **THEN** the plugin resolves the repository as before

#### Scenario: GitHub token is absent
- **WHEN** a caller invokes a GitHub API command without `GITHUB_TOKEN`
- **THEN** the plugin exits before sending a request and reports the missing
  credential
