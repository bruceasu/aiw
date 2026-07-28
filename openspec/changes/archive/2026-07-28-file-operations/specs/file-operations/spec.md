# File Operations Specification

### Requirement: Detect supported text encodings

The system SHALL detect UTF-8, UTF-16, GB18030, and Windows-31J and SHALL expose encoding and confidence metadata.

#### Scenario: Deterministic encoding

- **WHEN** a file has a supported BOM or valid UTF-8 content
- **THEN** the system reports the matching encoding with high confidence

#### Scenario: Ambiguous legacy encoding

- **WHEN** bytes can be decoded as more than one legacy encoding
- **THEN** the system reports ambiguity and requires an explicit encoding for writes

### Requirement: Preserve file format

The system SHALL preserve an existing file's BOM and newline style by default when writing text.

#### Scenario: Preserve an existing file

- **WHEN** an AI writes a detected text file without overriding format options
- **THEN** the system writes using the existing encoding, BOM, and newline style

### Requirement: Atomic text writes

The system SHALL write through a temporary file and atomically replace the destination only after encoding succeeds.

#### Scenario: Encoding failure

- **WHEN** content cannot be represented in the selected encoding
- **THEN** the original file remains unchanged and the command returns a non-zero status

### Requirement: Skill integration

AI Skills SHALL use aiw file read/info/write for text file content access and SHALL use aiw patch for generated code patches.

#### Scenario: Skill reads or modifies text

- **WHEN** a Skill needs project file content
- **THEN** it uses the shared file tools unless the tool is unavailable or the file is binary