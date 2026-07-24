## ADDED Requirements

### Requirement: Task-oriented top-level HELP
The system SHALL explain what aiw-flow does, show the common Session workflow, summarize every top-level command, provide quick-start examples, and direct users to command-specific HELP.

#### Scenario: First-time user requests HELP
- **WHEN** a user runs `aiw-flow --help`
- **THEN** the output explains the Session workflow and shows at least one creation example and one interactive example

#### Scenario: User scans available commands
- **WHEN** top-level HELP is displayed
- **THEN** every supported top-level command has a plain-language summary

### Requirement: Actionable command HELP
Every top-level command SHALL provide its purpose, descriptions for all positional and optional arguments, relevant usage constraints, and at least one valid example.

#### Scenario: User requests creation HELP
- **WHEN** a user runs `aiw-flow new --help`
- **THEN** the output explains each required input, optional loop behavior, and shows a complete creation command

#### Scenario: User requests execution HELP
- **WHEN** a user runs `aiw-flow run --help`
- **THEN** the output explains phase, prompt sources, timeout, thread replacement risk, and valid examples

### Requirement: Discoverable nested HELP
The `memory`, `handoff`, and `daemon` command groups SHALL summarize their child actions, and every child action SHALL provide argument descriptions and examples.

#### Scenario: User explores Memory commands
- **WHEN** a user runs `aiw-flow memory --help`
- **THEN** the output explains `show`, `append`, and `replace` and how to request their detailed HELP

#### Scenario: User explores one nested action
- **WHEN** a user runs `aiw-flow handoff create --help`
- **THEN** the output explains the Session ID and focus arguments and shows a valid example

### Requirement: HELP terminology and layout
The system SHALL use Easy English, consistent Session terminology, meaningful metavariables, and preserved multiline formatting for workflows and examples.

#### Scenario: HELP renders examples
- **WHEN** argparse renders any command HELP
- **THEN** multiline examples remain readable and use the actual command and option names

### Requirement: Parser compatibility
HELP improvements MUST NOT change command names, option names, required arguments, defaults, parsed destination names, or dispatch behavior.

#### Scenario: Existing command invocation
- **WHEN** an existing valid aiw-flow argument list is parsed
- **THEN** it produces the same command and option values as before the HELP change

#### Scenario: Existing invalid invocation
- **WHEN** a required argument is omitted
- **THEN** argparse rejects the invocation as before
