## 1. AI Support Surface

- [x] 1.1 Add aiw patch check, apply, and reverse dispatch.
- [x] 1.2 Support patch files and - for standard input.
- [x] 1.3 Add explicit --encoding, --3way, and --index options.
- [x] 1.4 Route AI-generated patches through the patch adapter by default from AI support surfaces.

## 2. Encoding and Git Adapter

- [x] 2.1 Implement UTF-8, UTF-8 BOM, and UTF-16 normalization.
- [x] 2.2 Convert Begin Patch, Update File, Add File, Delete File, and Move to File into standard unified diff.
- [x] 2.3 Reject malformed or unsupported AI patch syntax before invoking Git.
- [x] 2.4 Invoke Git with an ephemeral normalized patch file.
- [x] 2.5 Clean up temporary files on success and failure.
- [x] 2.6 Preserve Git exit status and diagnostics.
