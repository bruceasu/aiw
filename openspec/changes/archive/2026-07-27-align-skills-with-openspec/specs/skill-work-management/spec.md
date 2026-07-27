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
