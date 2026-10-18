# File conversion capability

## Requirements

### Requirement: Unified plugin conversion

The file-convert plugin MUST expose dependency-free Usage and Examples through
both manifest help and -h/--help. It MUST accept input, output (including the
legacy --ouput alias), explicit target format and structured JSON results.
It MUST declare supported format pairs and the backend for each pair.

### Requirement: Safe file and stream output

Omitted or empty output MUST append the target extension to the complete input
name. Existing outputs and input files MUST be preserved. Binary output MUST
reject stdout before backend execution, including JSON mode. Text stdout in
JSON mode MUST appear only in result-data; file results MUST provide metadata
without binary contents. Conversion failure MUST NOT publish partial output.

### Requirement: Local resource and rendering boundaries

Conversion MUST NOT implicitly upload documents, download dependencies or fetch
remote document resources. Local assets MUST remain inside the allowed input
resource directory after link resolution. PDF MUST await Mermaid, fonts and
images and return failure for missing resources, backend failure or timeout.
Browser printing MUST use installed supported browsers without requiring
Playwright, Office, LibreOffice or LaTeX.

### Requirement: Explicit dependency installation

Missing dependencies MUST produce actionable official sources and installation
guidance, including machine-readable errors. Optional dependencies MUST NOT
block unrelated conversions. Automatic portable installation MUST require
explicit approval of a concrete version/platform/source/checksum/destination
plan. Refusal or absent approval MUST NOT download anything. Installation MUST
verify release checksums, reject unsafe archives and preserve existing tools.
Failed installation MUST remain a failure with recovery information.

## Scenarios

- DOCX with no output becomes report.docx.md; an existing target is refused.
- PDF output with -o - --json fails before conversion or installation.
- Markdown HTML uses bundled Mermaid/CSS; remote image references fail visibly.
- Missing Pandoc does not prevent MarkItDown document extraction.
- An installation plan is previewed offline; only its matching approval digest
  allows a download into a new destination.

Implementation: FD-078; supported pairs and limits are in docs/usage/aiw-file-convert.md.
