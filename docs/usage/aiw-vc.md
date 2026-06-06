# aiw vc

Short: Microsoft Visual C++ (cl.exe) wrapper with build modes.

Description:
Lightweight wrapper for Microsoft Visual C++ compiler (cl.exe). Automatically detects Visual Studio environment, configures include and lib paths, and supports DLL, release, and debug build modes. Optional UPX compression available.

Usage:
aiw vc [mode] [args...]

Arguments:
- dll — Build dynamic library (/LD).
- release — Optimized build (/O2, /W3).
- debug — Debug build (/Zi, /Od, /W3).

Examples:
- aiw vc hello.c /Fehello.exe
- aiw vc dll plugin.c /Feplugin.dll
- aiw vc release app.c /Feapp.exe
- aiw vc debug app.c /Feapp_dbg.exe

For full help run: generate_new_plugin_docs.py -h