# Capability: Reviewed Skill Contract

## Requirements

### Requirement: Predictable invocation
Each reviewed Skill MUST define trigger conditions, required inputs, outputs, and completion criteria.

### Requirement: Explicit routing
Related Skills MUST describe their handoff and routing relationship; consultation, specification, ticketing, implementation, testing, review, and publishing MUST remain distinguishable.

### Requirement: Lifecycle ownership
A Skill MUST NOT create or mutate AIW Tasks, OpenSpec changes, worktrees, commits, or external projections unless its contract explicitly authorizes that action.

### Requirement: Uncertainty and evidence
Missing or unverified information MUST be represented with `%%` notes or an explicit blocking/incomplete state; a Skill MUST NOT claim skipped work or unavailable sibling work as completed.

### Requirement: Verification scope
Runtime validation MUST remain opt-in unless explicitly requested by the user or required by the task type; static evidence and commands actually run MUST be reportable.

### Requirement: Scope preservation
Remediation MUST be limited to findings in `docs/skills-review.md` and MUST preserve backward-compatible behavior unless a change is explicitly specified.
