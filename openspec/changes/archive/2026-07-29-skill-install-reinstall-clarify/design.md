# Design: Clarify Skill ownership and publish semantics

## Ownership model

The managed manifest remains the authoritative record for installed Skills.
The manifest distinguishes between:

- unmanaged destinations that were not yet recorded;
- managed destinations that can be republished from canonical source;
- adopted destinations that were recorded after the fact.

The manifest content should continue to include a stable content digest for each
managed Skill so the installer can tell whether a republished destination is the
same content or a new version.

## Command responsibilities

### `discover`

`discover` is read-only. It inspects installed Skills and reports whether each
one is managed or unmanaged. It does not change the filesystem.

### `adopt`

`adopt` records an existing valid installed Skill as managed without changing
its content. It is intended for existing directories that the user trusts.

### `install`

`install` publishes canonical source content into the project target. It must
still refuse to overwrite a same-name unmanaged destination by default.

When the target is already managed, `install` may replace that managed
destination with the canonical source content and update the manifest digest.
That behavior makes `install` usable for reinstall and upgrade flows.

### `sync`

`sync` is the explicit republish command for managed destinations. It exists to
make the upgrade intent obvious and to provide a stable operation for future
automation. `sync` should fail when the same-name destination is unmanaged.

## Manifest updates

Managed entries should remain versioned JSON records with:

- installation mode;
- content digest;
- source identity;
- source revision when available.

The change should preserve the manifest schema, unless the new command surface
requires a new field to distinguish adopted records from canonical installs.
If a new field is needed, it should be minimal and backward compatible.

## Risks

- A managed directory may have local edits that the user wants to preserve.
  `install` and `sync` need to make overwrite intent clear in their help text.
- The repository currently contains adopted manifests in the workspace, so the
  implementation must avoid breaking existing managed entries.
- Discovery logic must avoid treating non-Skill directories as installed Skills.
