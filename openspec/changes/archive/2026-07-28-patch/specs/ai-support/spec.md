# Patch Application Specification

## Purpose

Provide reliable, Git-backed patch application across Windows terminal encodings.

### Requirement: Normalize patch input

The system SHALL accept patch input from a file or standard input and SHALL normalize supported UTF-8 and UTF-16 input before invoking Git.

#### Scenario: UTF-8 patch file

- **WHEN** a user runs aiw patch check patch.diff with a UTF-8 patch
- **THEN** the system invokes Git with equivalent UTF-8 patch text and does not modify the worktree

#### Scenario: UTF-8 BOM or UTF-16 patch

- **WHEN** a user supplies a patch encoded as UTF-8 with BOM or UTF-16
- **THEN** the system removes transport-specific markers as needed and Git receives valid patch text

### Requirement: Delegate validation and application to Git

The system SHALL use git apply --check for check and preflight validation and SHALL use git apply for normal application.

#### Scenario: Apply a valid patch

- **WHEN** a user runs aiw patch apply patch.diff
- **THEN** the system validates the patch with Git and applies it only after validation succeeds

#### Scenario: Reverse a valid patch

- **WHEN** a user runs aiw patch reverse patch.diff
- **THEN** the system applies the patch using Git reverse mode

### Requirement: Protect state-changing options

The system SHALL NOT enable index modification or three-way application by default.

#### Scenario: Explicit index update

- **WHEN** a user supplies --index
- **THEN** the system passes the index option to Git

#### Scenario: Explicit three-way application

- **WHEN** a user supplies --3way
- **THEN** the system passes the three-way option to Git

### Requirement: Report failures safely

The system SHALL preserve Git failure status, show actionable diagnostics, and remove temporary normalized patch files after completion.

#### Scenario: Invalid patch

- **WHEN** Git rejects a patch
- **THEN** the system returns a non-zero status and reports Git diagnostics

#### Scenario: Missing Git

- **WHEN** Git is unavailable on PATH
- **THEN** the system reports that Git is required and returns a non-zero status
### Requirement: AI patch integration

The system SHALL expose the patch adapter as the default application path for AI-generated code patches.

#### Scenario: AI applies a generated patch

- **WHEN** an AI coding workflow produces a patch for repository changes
- **THEN** it invokes the patch adapter, which normalizes input, runs Git preflight validation, and applies the patch through Git

#### Scenario: AI receives an application failure

- **WHEN** Git rejects an AI-generated patch
- **THEN** the adapter returns a structured failure with the Git exit status and diagnostics, and the AI workflow SHALL NOT report the change as applied

#### Scenario: AI receives a successful application

- **WHEN** Git applies an AI-generated patch successfully
- **THEN** the adapter returns a structured success result with the applied operation and affected paths when available
### Requirement: Convert AI patch syntax

The system SHALL recognize the AI patch envelope and convert supported operations into a standard unified diff before invoking Git.

#### Scenario: Convert an update operation

- **WHEN** an input contains a Begin Patch envelope with an Update File operation
- **THEN** the system produces a standard unified diff for that file and passes the converted patch to Git

#### Scenario: Convert an add or delete operation

- **WHEN** an input contains an Add File or Delete File operation
- **THEN** the system produces a standard unified diff representing the file creation or deletion

#### Scenario: Convert a move operation

- **WHEN** an input contains a Move to File operation with a supported source and target
- **THEN** the system produces a standard Git rename-style patch or an equivalent delete-and-add patch

#### Scenario: Conversion failure

- **WHEN** the AI patch syntax is malformed or contains an unsupported operation
- **THEN** the system returns a conversion error naming the operation and path, and SHALL NOT invoke Git apply