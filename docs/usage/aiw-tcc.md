# aiw tcc

Short: Tiny C Compiler wrapper with run and cross-architecture support.

Description:
Wrapper for Tiny C Compiler (TCC). Supports fast compilation, direct execution (-run), shared libraries, and optional x86_64 build mode. Designed for minimal and fast C workflows.

Usage:
aiw tcc [mode] [args...]

Arguments:
- run — Compile and execute immediately (-run).
- dll — Build shared library (-shared).
- x86_64 — Use 64-bit compiler variant.
- amd64 — Alias for x86_64 mode.
- x64 — Alias for x86_64 mode.

Examples:
- aiw tcc hello.c -run
- aiw tcc dll plugin.c -o plugin.dll
- aiw tcc x86_64 hello.c -o app.exe
- aiw tcc hello.c -o app.exe -luser32

For full help run: generate_new_plugin_docs.py -h