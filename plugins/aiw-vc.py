#!/usr/bin/env python3
"""
aiw-vc plugin: lightweight wrapper for Microsoft cl.exe (MSVC).
Auto-detects cl.exe on PATH or via VC_ROOT/VC_HOME/VSINSTALLDIR.
Supports modes: dll, release, debug.
Optionally compresses output with UPX.
"""
import os
import shutil
import sys
import subprocess
from pathlib import Path

DEFAULT_ROOT = r"D:\green\VCompiler"
USE_UPX = True  # set False to disable automatic UPX
META = {
    'name': 'aiw vc',
    'short': 'Microsoft Visual C++ (cl.exe) wrapper with build modes.',
    'long': (
        'Lightweight wrapper for Microsoft Visual C++ compiler (cl.exe). '
        'Automatically detects Visual Studio environment, configures include and lib paths, '
        'and supports DLL, release, and debug build modes. Optional UPX compression available.'
    ),
    'usage': 'aiw vc [mode] [args...]',
    'args': [
        {'flag': 'dll', 'description': 'Build dynamic library (/LD).'},
        {'flag': 'release', 'description': 'Optimized build (/O2, /W3).'},
        {'flag': 'debug', 'description': 'Debug build (/Zi, /Od, /W3).'},
    ],
    'examples': [
        'aiw vc hello.c /Fehello.exe',
        'aiw vc dll plugin.c /Feplugin.dll',
        'aiw vc release app.c /Feapp.exe',
        'aiw vc debug app.c /Feapp_dbg.exe'
    ],
}

def detect_root():
    for key in ("VC_HOME", "VC_ROOT", "VSINSTALLDIR"):
        v = os.environ.get(key, "").strip()
        if v:
            return v
    p = shutil.which("cl.exe")
    if p:
        return str(Path(p).parent)
    return DEFAULT_ROOT

def is_help_arg(arg: str) -> bool:
    return arg.lower() in ("help", "-h", "--help")

def mode_for(args):
    if not args:
        return "", []
    a = args[0].lower()
    if a in ("dll", "release", "debug"):
        return a, args[1:]
    return "", args

def include_dir(root: str):
    p = Path(root) / "Include"
    return str(p) if p.exists() else None

def lib_dir(root: str):
    p = Path(root) / "Lib"
    return str(p) if p.exists() else None

def executable(root: str):
    p = shutil.which("cl.exe")
    if p:
        return p
    default_exe = Path(root) / "bin" / "cl.exe"
    if default_exe.exists():
        return str(default_exe)
    return "cl.exe"  # fallback

def args_for(mode, args, root):
    result = []
    out_file = None

    # Detect output file in arguments (simple -Fe or /Fe)
    for i, a in enumerate(args):
        if a.startswith("/Fe") or a.startswith("-Fe"):
            out_file = a[3:] if len(a) > 3 else args[i + 1] if i + 1 < len(args) else None

    if mode == "dll":
        result.append("/LD")
    elif mode == "release":
        result.extend(["/O2", "/W3"])
    elif mode == "debug":
        result.extend(["/Zi", "/Od", "/W3"])

    inc = include_dir(root)
    if inc:
        result.append("/I" + inc)
    ld = lib_dir(root)
    if ld:
        result.extend(["/link", "/LIBPATH:" + ld])

    result.extend(args)
    return result, out_file

def print_help(root):
    inc = include_dir(root)
    ld = lib_dir(root)
    comp = Path(executable(root)).name
    print("Usage:")
    print("  aiw vc [mode] [args...]")
    print("")
    print("Modes:")
    print("  vc dll [args...]       Shortcut for /LD (build DLL)")
    print("  vc release [args...]   Optimize for speed (/O2 /W3)")
    print("  vc debug [args...]     Debug mode (/Zi /Od /W3)")
    print("")
    print("Auto paths:")
    print(f"  root: {root}")
    print(f"  compiler: {comp}")
    print(f"  include: {inc if inc else '<missing>'}")
    print(f"  lib: {ld if ld else '<missing>'}")
    print("")
    print("Examples:")
    print("  aiw vc dll foo.c /Fefoo.dll")
    print("  aiw vc release foo.c /Fefoo.exe")
    print("  aiw vc debug foo.c /Fefoo_dbg.exe")

def run(name, argv):
    try:
        p = subprocess.Popen([name, *argv])
        p.communicate()
        return p.returncode
    except FileNotFoundError:
        print(f"executable not found: {name}", file=sys.stderr)
        return 2

def upx_compress(file: str):
    if not file or not USE_UPX:
        return
    upx_exe = shutil.which("upx")
    if not upx_exe:
        print("UPX not found, skipping compression")
        return
    cmd = [upx_exe, "--best", "--lzma", "--strip-relocs=0", file]
    subprocess.run(cmd)

def main():
    args = sys.argv[1:]
    if not args or is_help_arg(args[0]):
        print_help(detect_root())
        return 0

    root = detect_root()
    mode, rest = mode_for(args)
    exe = executable(root)
    cmd_args, out_file = args_for(mode, rest, root)

    rc = run(exe, cmd_args)
    if rc == 0 and out_file:
        upx_compress(out_file)

    return rc if rc is not None else 0

if __name__ == "__main__":
    sys.exit(main())