# aiw bcc

Short: Borland BCC32 compiler wrapper with tiny and GUI modes.

Description:
Lightweight wrapper for Borland C++ Builder (bcc32). Supports DLL builds, GUI applications, and tiny optimization mode for minimal executable size. Optional UPX compression supported.

Usage:
aiw bcc [mode] [args...]

Arguments:
- tiny — Enable size optimization (-O2, RTTI off, exception off).
- gui — Build Windows GUI application (no console window).
- dll — Build dynamic library (-WD).
- run — Compile and run the program immediately.
- upx — Compress output binary using UPX.

Examples:
- aiw bcc hello.cpp
- aiw bcc tiny hello.cpp
- aiw bcc gui tiny app.cpp
- aiw bcc dll plugin.cpp
- aiw bcc tiny upx app.cpp

For full help run: generate_new_plugin_docs.py -h