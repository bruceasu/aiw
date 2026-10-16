# GitHub Issue projection

## Purpose

Publish explicitly selected local work as a remote summary while preserving
local ownership, human content and recoverable publication identity.

## Requirements

### Requirement: Local Issue and FD sources without a required change

Publication MUST support a resolved local Issue/REQ or FD without an OpenSpec
change. A change MAY supply optional specification evidence or remain an
explicit legacy source. Local Issue/FD records MUST retain scope/progress and
lifecycle authority; publication MUST NOT approve, merge, archive or close them.
Local completion MUST NOT automatically publish or close a remote Issue.

#### Scenario: Linked FDs share an Issue target

- WHEN an Issue has multiple linked FDs and selected progress is published
- THEN the Issue-owned mapping MUST be reused by default
- AND each projected source MUST use its own managed block
- AND another FD's block and human text MUST remain untouched

#### Scenario: Explicit independent FD publication

- WHEN a standalone FD is published or the user explicitly requests separate tracking
- THEN an FD-owned independent mapping MAY identify a separate remote target
- AND that policy MUST be recorded rather than silently inferred from conflicting mappings

### Requirement: Durable mapping ownership and legacy compatibility

New mappings MUST live under docs/external/github/issue or fd by resolved
canonical ID, independently of source archival paths. Mappings MUST identify
the owner and validated repository/Issue number/URL. A linked FD's projected
source MAY differ from the Issue mapping owner. Unexplained target conflicts,
invalid mappings or unavailable mapped Issues MUST block publication rather
than creating replacements. Valid legacy version-1/OpenSpec mappings and
helper interfaces MUST remain usable; migrations MUST be explicit and retain
prior evidence. Source marker migration MUST NOT silently adopt unrelated text.

#### Scenario: FD archive preserves publication identity

- WHEN an FD moves to its local archive
- THEN its external mapping identity MUST remain valid at the stable mapping path
- AND archival MUST NOT create a new remote Issue or close the existing one

### Requirement: Isolated and bounded managed content

New source updates MUST require a unique complete non-nested source marker
pair, preserve bytes outside that block, and reject malformed/duplicate/reversed
markers or mismatched generated blocks. The completed body MUST remain bounded
to 64 KiB without truncating human content. Existing human titles MUST remain
unless an explicit update includes them. The legacy two-argument helper MAY
retain its historical whitespace behavior.

#### Scenario: Update one FD block

- WHEN the remote body contains an Issue block, two FD blocks and human notes
- THEN updating one FD MUST preserve all content outside its exact block
- AND partial or duplicate target markers MUST fail without a replacement write

### Requirement: Explicit publication with recoverable unknown outcomes

Remote writes MUST use actual user authority and applicable network rules.
Before create/update, the publisher MUST exclusively and atomically persist a
pending operation receipt containing target repository, owner/projected sources,
operation identity/time/body hash, and an Issue number for updates. Pending
receipts MUST block new writes. Success MUST record the returned target before
committing the validated mapping, then clear only the completed operation's
receipt. Unknown remote outcomes or local commit failures MUST retain evidence
and require reconciliation before another remote write. An absent mapping MUST
NOT imply that creation never occurred. Body re-reading MUST minimize concurrent
edit loss; full-body writes MUST NOT be claimed to provide CAS protection.

#### Scenario: Create succeeds but mapping save fails

- WHEN the response identifies a new Issue but the local mapping cannot be saved
- THEN the response and pending receipt MUST be retained for recovery
- AND recovery MUST complete the local mapping for the verified Issue without creating another
- AND publication MUST be reported incomplete until the local commit is finished
