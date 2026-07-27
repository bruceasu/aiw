## 1. Shared OpenSpec Work Management

- [x] 1.1 Add shared work-management guidance that defines OpenSpec artifact
  roles, deterministic active-change resolution, AIW workspace validation, and
  external trackers as explicit projections.
- [x] 1.2 Replace the repository's Local Markdown issue-tracker guidance with
  concise OpenSpec-canonical guidance under `docs/agents/`, without deleting
  existing `.scratch` user content.
- [x] 1.3 Update the engineering-Skill setup flow to select OpenSpec
  automatically in an AIW/OpenSpec repository and stop asking users to choose a
  local, GitHub, or GitLab tracker.

## 2. Planning and Routing Skills

- [x] 2.1 Update `to-spec` to create or update proposal, design, capability
  specs, and tasks according to the shared OpenSpec artifact contract.
- [x] 2.2 Update `to-tickets` to write ordinary tracer bullets as numbered
  `tasks.md` checklist items and propose a separate change only for independently
  managed work.
- [x] 2.3 Update `ask-matt` and other affected routing text to describe
  OpenSpec-canonical local work without embedding `.scratch` or issue-tracker
  backend choices.
- [x] 2.4 Add prompt-level examples covering an existing change, a new change,
  an ambiguous change context, and a slice that requires an independent
  worktree.

## 3. Execution, Review, and Handoff Skills

- [x] 3.1 Update `implement` to read the complete resolved OpenSpec context,
  verify AIW branch/worktree metadata, implement one selected task item, and
  update TODO, verification, notes, and lifecycle state.
- [x] 3.2 Update `code-review` to resolve originating requirements from
  OpenSpec artifacts before falling back to external Issue or PR references.
- [x] 3.3 Update `handoff` to reference OpenSpec artifacts and prefer AIW
  Session artifacts when available, retaining OS temporary storage only as a
  non-AIW fallback.
- [x] 3.4 Verify affected Skills stop before mutation when work context or
  workspace resolution is ambiguous.

## 4. aiw-github Issue Transport

- [x] 4.1 Add focused parser and transport-boundary tests for body-file input,
  standard-input bodies, Issue updates, missing update fields, JSON identity
  output, missing credentials, and repository discovery.
- [x] 4.2 Add body-file and standard-input support to GitHub Issue creation and
  update commands without adding or upgrading dependencies.
- [x] 4.3 Add an Issue update command that patches title or body and rejects an
  empty update before making a request.
- [x] 4.4 Normalize JSON Issue results to include repository, Issue number, URL,
  and state while preserving human-friendly output.
- [x] 4.5 Align `aiw-github` README and HELP examples with the parser's actual
  command names, global option placement, authentication source, and supported
  configuration.

## 5. Explicit OpenSpec-to-GitHub Publication

- [x] 5.1 Add a `publish-github-issue` Skill that resolves one OpenSpec change,
  renders a bounded managed projection, and invokes `aiw-github` only after an
  explicit publication request.
- [x] 5.2 Define and validate the versioned
  `external/github.json` mapping record without storing unknown fields in
  `task.toml`.
- [x] 5.3 Implement first-publication creation and repeat-publication update
  behavior, including validation that prevents silent duplicate creation when a
  stored mapping is stale or inaccessible.
- [x] 5.4 Preserve human-authored GitHub Issue body content outside the managed
  OpenSpec markers during repeat publication.
- [x] 5.5 Add publication tests using a fake GitHub transport for initial
  creation, repeat update, managed-marker replacement, preserved human content,
  and invalid mappings.

## 6. Compatibility and Verification

- [x] 6.1 Confirm no affected engineering Skill creates new canonical
  specifications or tickets under `.scratch`, while documenting that existing
  `.scratch` content is left untouched.
- [x] 6.2 Run the repository-standard Skill validation or focused prompt
  fixtures and inspect all affected Skill definitions for consistent
  work-management terminology.
- [x] 6.3 Run focused `aiw-github` tests and HELP/parser smoke tests for every
  changed Issue command.
- [x] 6.4 Run applicable Go and Python formatting, focused tests, full
  repository tests, vet/static checks, and build verification, recording any
  environment limitations.
- [x] 6.5 Run strict OpenSpec validation, inspect the scoped diff, and update
  this checklist with completed TODOs, verification evidence, and remaining
  `%%` risks or questions.

## Notes

%% Do not add or upgrade dependencies without separate approval.
%% GitLab publication, bidirectional synchronization, and AIW task dependency
metadata are outside this change.
%% Existing `.scratch` files are user data and must not be deleted or rewritten
automatically.
%% Proposal verification: all four spec-driven artifacts are complete;
`openspec.cmd validate align-skills-with-openspec --type change --strict
--no-interactive` and the scoped Git whitespace check passed.
%% Implementation verification: Skill CLI tests (25, 3 skipped), aiw-github
tests (6), projection tests (3), go test ./..., go vet ./..., and
go build -buildvcs=false ./... passed. Plain go build was blocked only by Git
VCS stamping under the sandbox's repository ownership rules.
%% Repository layout note: this checkout stores bundled Skills under `skills/`
and has no pre-existing `docs/agents/` directory; implementation used those
actual paths without moving or deleting content.
