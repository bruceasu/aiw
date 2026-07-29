# capabilities Specification

## Purpose
Define how AIW discovers, lists, installs, and invokes reusable Skills as capability assets within the workflow-first model.

## Requirements
### Requirement: Discover Skills from standard locations
The system SHALL discover Skill directories from project and user `.agents/skills` locations and from compatible project and user `.codex/skills` locations without requiring configurable search paths.

#### Scenario: Discover project Skills
- **WHEN** a Session workspace or its repository ancestry contains a valid Skill under `.agents/skills`
- **THEN** the system includes that Skill in the available project Skills

#### Scenario: Discover compatible project Skills
- **WHEN** the Session project root contains a valid Skill under `.codex/skills`
- **THEN** the system includes that Skill in the available project Skills

#### Scenario: Discover user Skills
- **WHEN** the user home contains a valid Skill under `.agents/skills` or the effective Codex home contains a valid Skill under `skills`
- **THEN** the system includes that Skill in the available user Skills

#### Scenario: Resolve a non-Git workspace
- **WHEN** the Session workspace is not inside a Git repository
- **THEN** the system treats the workspace as the project root when resolving project Skill locations

### Requirement: List Skills without executing a turn
The interactive loop SHALL process `/skills` locally and display discovered Skills with their scope and source without calling Codex.

#### Scenario: List valid Skills
- **WHEN** the user enters `/skills` and valid project or user Skills are discoverable
- **THEN** the system displays their names, descriptions, scopes, and source paths without incrementing the Session turn number

#### Scenario: List an empty catalog
- **WHEN** the user enters `/skills` and no valid Skills are discoverable
- **THEN** the system reports that no Skills were found and waits for another input without calling Codex

#### Scenario: Report malformed candidates
- **WHEN** the user enters `/skills` and a candidate has missing or invalid required frontmatter
- **THEN** the system reports a warning containing the candidate path while continuing to display other valid Skills

### Requirement: Invoke a discovered Skill
The interactive loop SHALL accept `/skill <name> <message>` and execute a valid request through the normal turn path using Codex's native `$<name>` invocation syntax.

#### Scenario: Invoke a project Skill
- **WHEN** the user enters `/skill metrics-review Review the revenue metrics` and exactly one discovered Skill is named `metrics-review`
- **THEN** the system executes `$metrics-review Review the revenue metrics` as one normal Session turn

#### Scenario: Preserve normal turn behavior
- **WHEN** a valid `/skill` request executes
- **THEN** the system persists its Prompt, Output, Event, status, timeout result, and Thread changes exactly as an ordinary loop message

#### Scenario: Preserve direct invocation
- **WHEN** the user enters an ordinary message beginning with `$metrics-review`
- **THEN** the system sends that message unchanged through the normal turn path

### Requirement: Reject invalid Skill commands safely
The interactive loop SHALL reject malformed, missing, or ambiguous Skill invocations without calling Codex.

#### Scenario: Missing Skill message
- **WHEN** the user enters `/skill metrics-review` without a task message
- **THEN** the system displays the required syntax and waits for another input without incrementing the turn number

#### Scenario: Unknown Skill name
- **WHEN** the user invokes a name that is not present in the current discovery result
- **THEN** the system reports that the Skill was not found and waits for another input without calling Codex

#### Scenario: Duplicate Skill name
- **WHEN** more than one discovered Skill declares the requested name
- **THEN** the system reports every conflicting source path and refuses to execute the turn

#### Scenario: Skill changes before invocation
- **WHEN** the discovery contents change after an earlier `/skills` command and before `/skill` is entered
- **THEN** the system uses a fresh discovery result to validate the invocation

### Requirement: Preserve existing loop command behavior
The new Skill commands MUST coexist with all existing loop messages, escapes, and local commands without changing their behavior.

#### Scenario: Escape a Skill-like slash message
- **WHEN** the user enters `//skill metrics-review Review this text`
- **THEN** the system sends `/skill metrics-review Review this text` as an ordinary message instead of processing a local Skill command

#### Scenario: Use an existing command
- **WHEN** the user enters an existing local command such as `/status`
- **THEN** the system processes that command with its existing behavior

### Requirement: Install canonical Skills
The system SHALL list valid canonical portable Skills and install them into the current project's `.agents/skills` directory when requested.

#### Scenario: Install a valid Skill
- **WHEN** the user installs a valid canonical Skill into a project where its destination does not exist
- **THEN** the complete Skill directory is copied to the default project target and becomes discoverable there

#### Scenario: Reject an unknown Skill
- **WHEN** the user names a Skill that is not in the valid canonical catalog
- **THEN** the command fails without creating the destination root or manifest

### Requirement: Validate before writing
The system MUST validate required Skill metadata and all copyable source entries before it changes the destination.

#### Scenario: Reject invalid metadata
- **WHEN** the selected canonical candidate has a missing or invalid `name` or `description`
- **THEN** the command fails without writing project state

#### Scenario: Reject unsupported filesystem entries
- **WHEN** the selected Skill contains a symlink or non-regular entry
- **THEN** the command fails before publishing an installed Skill

### Requirement: Preserve work management behavior
The system SHALL treat skill work management as a local AIW capability and optional external Issue projection behavior.

#### Scenario: Local work management
- **WHEN** a Skill manages its local work without external projection
- **THEN** the system keeps the skill work in the AIW-controlled local workflow model

#### Scenario: External Issue projection
- **WHEN** a Skill is configured to project work to an external tracker
- **THEN** the system preserves local workflow state while allowing the external projection behavior

<!-- archived spec: skill-work-management -->

## ADDED Requirements

### Requirement: Use OpenSpec as canonical local work state
Bundled engineering Skills running in an AIW/OpenSpec repository SHALL treat
OpenSpec changes, stable specs, and AIW task metadata as the canonical local
work state.

#### Scenario: Create local specification work
- **WHEN** a specification Skill creates local work in an AIW/OpenSpec
  repository
- **THEN** it creates or updates artifacts under
  `openspec/changes/<change-id>/` and does not create a parallel `.scratch`
  specification

#### Scenario: Preserve stable requirements
- **WHEN** a Skill identifies a stable requirement that outlives the current
  change
- **THEN** it records the requirement in the appropriate OpenSpec capability
  specification rather than treating an Issue body as normative

### Requirement: Configure one default local work manager
The engineering Skill setup SHALL select OpenSpec as the local work manager in
an AIW/OpenSpec repository without asking the user to choose among Local
Markdown, GitHub, or GitLab trackers.

#### Scenario: Set up an OpenSpec repository
- **WHEN** setup detects a valid AIW/OpenSpec repository
- **THEN** it writes OpenSpec-canonical work-management guidance and describes
  GitHub and GitLab as optional explicit projections

### Requirement: Resolve work context before mutation
A Skill that changes work artifacts or implementation state MUST resolve one
active OpenSpec change and its applicable workspace before writing.

#### Scenario: User names a change
- **WHEN** the user explicitly supplies an existing change identifier
- **THEN** the Skill uses that change as its work context

#### Scenario: Session already established a change
- **WHEN** the user does not name a change and the active session has exactly
  one established change
- **THEN** the Skill uses the session change as its work context

#### Scenario: Worktree identifies a task
- **WHEN** no change is established by the request or session and the current
  worktree or branch maps uniquely to an AIW task
- **THEN** the Skill uses the mapped task and its OpenSpec artifacts

#### Scenario: Work context is ambiguous
- **WHEN** no change can be resolved uniquely
- **THEN** the Skill stops before writing and requests a change identifier

### Requirement: Respect AIW execution workspace
An implementation Skill SHALL verify the branch and worktree declared for the
resolved AIW task before modifying implementation files.

#### Scenario: Execute in the declared worktree
- **WHEN** the current workspace and branch match the resolved task metadata
- **THEN** the Skill may implement the selected task item and update its
  OpenSpec progress

#### Scenario: Workspace does not match task metadata
- **WHEN** the current workspace or branch conflicts with the resolved task
  metadata
- **THEN** the Skill stops before modifying implementation files and reports
  the expected task workspace

### Requirement: Map Skill outputs to defined OpenSpec artifacts
Engineering Skills SHALL store motivation and scope in `proposal.md`, technical
decisions in `design.md`, normative behavior in capability specs,
implementation progress and verification in `tasks.md`, and temporary findings
in `notes.md` or `%%` notes.

#### Scenario: Convert a conversation to a specification
- **WHEN** a specification Skill synthesizes an approved conversation
- **THEN** it updates the applicable proposal, design, capability specs, and
  tasks according to their artifact roles without duplicating normative
  requirements in a local Issue file

#### Scenario: Finish implementation work
- **WHEN** an implementation Skill completes a selected work item
- **THEN** it updates the corresponding TODO, verification evidence, remaining
  `%%` risks or questions, and applicable AIW task status

### Requirement: Keep ordinary slices inside one change
A ticketing Skill SHALL represent work that shares one goal, branch, lifecycle,
and delivery boundary as numbered checklist items in the change `tasks.md`.

#### Scenario: Split one change into tracer bullets
- **WHEN** approved tracer-bullet slices can be implemented within the same AIW
  task and worktree
- **THEN** the Skill writes them as ordered, independently verifiable checklist
  items in `tasks.md`

#### Scenario: Slice needs an independent lifecycle
- **WHEN** a slice requires an independent worktree, status, archive lifecycle,
  or delivery boundary
- **THEN** the Skill proposes a separate OpenSpec change instead of representing
  it as a local `.scratch` ticket

### Requirement: Publish external Issues only on explicit request
Engineering Skills MUST NOT publish or synchronize GitHub or GitLab Issues
unless the user explicitly requests external publication.

#### Scenario: Perform normal local planning
- **WHEN** a user creates a specification or task breakdown without requesting
  external publication
- **THEN** the Skill changes only local OpenSpec artifacts

#### Scenario: User requests GitHub publication
- **WHEN** the user explicitly asks to publish a resolved OpenSpec change to
  GitHub Issues
- **THEN** the GitHub publishing workflow renders a bounded projection and
  invokes the configured GitHub transport

### Requirement: Keep OpenSpec authoritative after publication
An external Issue projection SHALL identify its source OpenSpec change, while
local OpenSpec artifacts remain authoritative for requirements, task progress,
and AIW status.

#### Scenario: Remote Issue changes independently
- **WHEN** a GitHub Issue is edited, labeled, commented on, or closed outside
  AIW
- **THEN** the local OpenSpec change is not silently modified

### Requirement: Publish GitHub Issues idempotently
The GitHub publishing workflow SHALL persist the target repository, Issue
number, URL, and publication metadata under the source OpenSpec change and use
that mapping for later publications.

#### Scenario: Publish a change for the first time
- **WHEN** the source change has no GitHub mapping
- **THEN** the workflow creates one Issue and records its mapping under
  `external/github.json`

#### Scenario: Republish a mapped change
- **WHEN** the source change has a valid GitHub mapping
- **THEN** the workflow updates the mapped Issue instead of creating another
  Issue

#### Scenario: Mapped Issue cannot be validated
- **WHEN** the stored GitHub mapping cannot be read or the target Issue cannot
  be validated
- **THEN** the workflow reports the failure and does not silently create a
  replacement Issue

### Requirement: Preserve human content outside managed markers
When updating a projected GitHub Issue, the publishing workflow SHALL replace
only its managed OpenSpec projection block and preserve Issue body content
outside that block.

#### Scenario: Republish after a human adds notes
- **WHEN** a mapped Issue contains human-authored content outside the managed
  markers
- **THEN** the workflow updates the generated projection and leaves the
  human-authored content unchanged

<!-- archived spec: skill-installation -->

## ADDED Requirements

### Requirement: List canonical Portable Skills
The system SHALL list valid canonical Portable Skills through the `aiw skills`
CLI without modifying the project.

#### Scenario: List valid Skills
- **WHEN** the canonical source contains Skills with valid single-line `name`
  and `description` frontmatter
- **THEN** the command displays each Skill name and description in stable name
  order

#### Scenario: Report an invalid candidate
- **WHEN** a canonical candidate is missing valid required frontmatter
- **THEN** the command reports the candidate as invalid without listing it as
  installable

### Requirement: Resolve the packaged canonical Skill root
The system SHALL use the `skills` directory beside `program` and `plugins` as
the default canonical Skill source in repository and release layouts.

#### Scenario: Run from a packaged plugin layout
- **WHEN** `aiw-skills.py` runs from
  `<install-root>/plugins/aiw-skills/aiw-skills.py` without a source override
- **THEN** the command reads canonical Skills from `<install-root>/skills`

### Requirement: Install one Skill to the default project target
The system SHALL copy one named canonical Portable Skill into the current
project's `.agents/skills` directory when no scope or target is specified.

#### Scenario: Install a valid Skill
- **WHEN** the user installs a valid canonical Skill into a project where its
  destination does not exist
- **THEN** the complete Skill directory is copied to the default project target
  and becomes discoverable there

#### Scenario: Reject an unknown Skill
- **WHEN** the user names a Skill that is not in the valid canonical catalog
- **THEN** the command fails without creating the destination root or manifest

#### Scenario: Reinstall a managed Skill
- **WHEN** the destination already exists and has a matching managed manifest
  entry
- **THEN** the command republishes the selected canonical source to the managed
  destination and refreshes the manifest entry

### Requirement: Validate before writing
The system MUST validate required Skill metadata and all copyable source
entries before it changes the destination.

#### Scenario: Reject invalid metadata
- **WHEN** the selected canonical candidate has a missing or invalid `name` or
  `description`
- **THEN** the command fails without writing project state

#### Scenario: Reject unsupported filesystem entries
- **WHEN** the selected Skill contains a symlink or non-regular entry
- **THEN** the command fails before publishing an installed Skill

### Requirement: Preview installation without side effects
The system SHALL provide a dry-run that reports the resolved source,
destination, and intended action without changing the filesystem.

#### Scenario: Dry-run a new installation
- **WHEN** the user installs a valid Skill with `--dry-run`
- **THEN** the command reports that the Skill would be installed and creates no
  destination directory, staging directory, or manifest

### Requirement: Stage and verify copied content
The system MUST stage a new installation on the destination filesystem and
verify its deterministic SHA-256 digest before publishing it.

#### Scenario: Publish verified staged content
- **WHEN** the source and staged content have the same deterministic digest
- **THEN** the system atomically renames the staged directory to the final
  destination

#### Scenario: Source changes during installation
- **WHEN** the source and staged content digests differ
- **THEN** the system removes staging, reports failure, and does not publish the
  destination

### Requirement: Record managed ownership
The system SHALL atomically maintain a versioned JSON manifest that records
each installed Skill's source identity, source revision when available,
installation mode, and content digest.

#### Scenario: Record a new installation
- **WHEN** a Skill is successfully published
- **THEN** the managed manifest contains a copy-mode entry whose digest matches
  the installed content

#### Scenario: Manifest update fails
- **WHEN** the manifest cannot be committed after a new Skill is published
- **THEN** the system removes only that newly published Skill and reports
  failure

### Requirement: Protect existing destinations
The system MUST NOT overwrite or delete a same-name destination that lacks a
matching managed manifest entry.

#### Scenario: Unmanaged destination exists
- **WHEN** a same-name destination exists without matching AIW ownership
- **THEN** the command fails and preserves the destination unchanged

#### Scenario: Identical managed installation exists
- **WHEN** the destination, source, and managed manifest all have the same
  digest
- **THEN** the command succeeds as an idempotent no-op without rewriting files

### Requirement: Adopt existing installed Skills
The system SHALL record a valid existing Skill directory under
`.agents/skills` as AIW-managed without changing the directory content.

#### Scenario: Adopt a valid installed Skill
- **WHEN** the user adopts an installed Skill directory that contains valid
  Skill metadata and copyable filesystem entries
- **THEN** the system writes a managed manifest entry for that Skill and leaves
  the directory content unchanged

### Requirement: Discover installed Skills
The system SHALL provide a discovery command that reports installed Skills and
their managed status without modifying the filesystem.

#### Scenario: Report installed status
- **WHEN** the user runs the discovery command
- **THEN** the system reports each installed Skill as managed or unmanaged

### Requirement: Synchronize a managed Skill
The system SHALL provide a sync behavior that republishes the canonical source
for a managed Skill into its destination and updates the managed digest.

#### Scenario: Sync a managed Skill after source changes
- **WHEN** the canonical source for a managed Skill has changed since the last
  installation
- **THEN** the sync operation replaces the managed destination with the new
  source content and records the new digest

#### Scenario: Reject sync for unmanaged destinations
- **WHEN** a same-name destination exists without a matching managed manifest
  entry
- **THEN** the sync operation fails without modifying the destination

### Requirement: Provide script-stable command results
The system SHALL provide human-readable results by default, one structured JSON
result when requested, and stable success or failure exit behavior.

#### Scenario: Request JSON success output
- **WHEN** a list, dry-run, new install, or idempotent install succeeds with
  `--json`
- **THEN** stdout contains one valid JSON object describing the action and
  stderr is empty

#### Scenario: Request JSON operational error output
- **WHEN** an install operation fails with `--json`
- **THEN** stdout contains one valid JSON error object and the process returns
  a nonzero operational exit code
