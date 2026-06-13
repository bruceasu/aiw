# aiw bcc

Borland BCC32 compiler wrapper with tiny and GUI modes.

## Usage

aiw bcc [mode] [args...]

## Description

Lightweight wrapper for Borland C++ Builder (bcc32). Supports DLL builds, GUI applications, and tiny optimization mode for minimal executable size. Optional UPX compression supported.

## Examples

- `aiw bcc hello.cpp`
- `aiw bcc tiny hello.cpp`
- `aiw bcc gui tiny app.cpp`
- `aiw bcc dll plugin.cpp`
- `aiw bcc tiny upx app.cpp`
