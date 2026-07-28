# Design: Shared AI File Operations

aiw file is a thin CLI adapter around a shared codec policy. It owns text detection, format preservation, atomic writes, and structured metadata. aiw patch reuses the same policy for patch input before delegating patch semantics to Git.

Detection order:
1. explicit --encoding
2. UTF-8/UTF-16 BOM
3. strict UTF-8
4. GB18030 or Windows-31J only when unambiguous
5. otherwise fail with an explicit encoding request

The default write mode is preserve for existing files. New files default to UTF-8 without BOM and LF. The write operation must not use shell interpolation and must replace files atomically.