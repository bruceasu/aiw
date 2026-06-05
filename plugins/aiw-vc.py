#!/usr/bin/env python3
"""
aiw-vc plugin: lightweight wrapper for Microsoft cl.exe (MSVC).
Auto-detects `cl` on PATH or via VC_ROOT/VC_HOME and adds `/LD` for `dll` mode.
"""
import os
import shutil
import sys
import subprocess
from pathlib import Path

DEFAULT_ROOT = r"C:\Program Files (x86)\Microsoft Visual Studio"


def detect_root():
    for key in ("VC_HOME", "VC_ROOT", "VSINSTALLDIR"):
        v = os.environ.get(key, "").strip()
        if v:
            return v
    for name in ("cl.exe", "cl"):
        p = shutil.which(name)
        if p:
            return str(Path(p).parent)
    return DEFAULT_ROOT


def is_help_arg(arg: str) -> bool:
    return arg.lower() in ("help", "-h", "--help")


def mode_for(args):
    if not args:
        return "", []
    a = args[0].lower()
    if a in ("dll",):
        return a, args[1:]
    return "", args


def include_dir(root: str):
    p = Path(root) / "Include"
    return str(p) if p.exists() else None


def lib_dir(root: str):
    p = Path(root) / "Lib"
    return str(p) if p.exists() else None


def executable(root: str):
    for name in ("cl.exe", "cl"):
        p = shutil.which(name)
        if p:
            return p
    return str(Path(root) / "cl.exe")


def args_for(mode, args, root):
    result = []
    # cl uses /LD to produce DLL
    if mode == "dll":
        result.append("/LD")
    result.extend(args)
    inc = include_dir(root)
    if inc:
        # cl expects multiple /I flags; here we append one if found
        result.append("/I" + inc)
    ld = lib_dir(root)
    if ld:
        # linker lib path can be passed via /link /LIBPATH:...
        result.extend(["/link", "/LIBPATH:" + ld])
    return result


def print_help(root):
    inc = include_dir(root)
    ld = lib_dir(root)
    comp = Path(executable(root)).name
    print("Usage:")
    print("  aiw vc [args...]")
    print("")
    print("Modes:")
    print("  vc dll [args...]         Shortcut for /LD (build DLL)")
    print("")
    print("Auto paths:")
    print(f"  root: {root}")
    print(f"  compiler: {comp}")
    print(f"  include: {inc if inc else '<missing>'}")
    print(f"  lib: {ld if ld else '<missing>'}")


def run(name, argv):
    try:
        p = subprocess.Popen([name, *argv])
        p.communicate()
        return p.returncode
    except FileNotFoundError:
        print(f"executable not found: {name}", file=sys.stderr)
        return 2


def main():
    args = sys.argv[1:]
    if not args or is_help_arg(args[0]):
        print_help(detect_root())
        return 0
    root = detect_root()
    mode, rest = mode_for(args)
    exe = executable(root)
    rc = run(exe, args_for(mode, rest, root))
    return rc if rc is not None else 0


if __name__ == "__main__":
    sys.exit(main())
