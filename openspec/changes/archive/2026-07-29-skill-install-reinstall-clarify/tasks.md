## 1. Specification Update

- [x] 1.1 Update the skill installation spec to distinguish `install`,
      `adopt`, `discover`, and `sync`.
- [x] 1.2 Define managed reinstall and upgrade behavior for existing managed
      destinations.
- [x] 1.3 Keep unmanaged destination protection explicit and unchanged.
- [x] 1.4 Clarify the manifest as the authoritative record for installed Skill
      ownership.
- [x] 1.5 Note any open edge cases with `%%` markers instead of guessing.

## 2. Command Surface

- [x] 2.1 Add or clarify a discovery command for installed Skills.
- [x] 2.2 Add or clarify an adoption command for existing installed Skills.
- [x] 2.3 Add a sync command for managed reinstall/upgrade flows.
- [x] 2.4 Update install semantics so managed destinations can be republished.
- [x] 2.5 Update help text and README examples to explain the new ownership
      boundary.

## 3. Verification

- [x] 3.1 Add focused CLI coverage for managed reinstall.
- [x] 3.2 Add focused CLI coverage for unmanaged refusal.
- [x] 3.3 Add focused CLI coverage for adoption.
- [x] 3.4 Add focused CLI coverage for sync.
- [x] 3.5 Update command help and README references to the new semantics.
