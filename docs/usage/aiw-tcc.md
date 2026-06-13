# aiw tcc

Tiny C Compiler wrapper with run and cross-architecture support.

## Usage

aiw tcc [mode] [args...]

## Description

Wrapper for Tiny C Compiler (TCC). Supports fast compilation, direct execution (-run), shared libraries, and optional x86_64 build mode. Designed for minimal and fast C workflows.

## Examples

- `aiw tcc hello.c -run`
- `aiw tcc dll plugin.c -o plugin.dll`
- `aiw tcc x86_64 hello.c -o app.exe`
- `aiw tcc hello.c -o app.exe -luser32`
