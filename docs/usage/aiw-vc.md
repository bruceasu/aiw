# aiw vc

Microsoft Visual C++ (cl.exe) wrapper with build modes.

## Usage

aiw vc [mode] [args...]

## Description

Lightweight wrapper for Microsoft Visual C++ compiler (cl.exe). Automatically detects Visual Studio environment, configures include and lib paths, and supports DLL, release, and debug build modes. Optional UPX compression available.

## Examples

- `aiw vc hello.c /Fehello.exe`
- `aiw vc dll plugin.c /Feplugin.dll`
- `aiw vc release app.c /Feapp.exe`
- `aiw vc debug app.c /Feapp_dbg.exe`
