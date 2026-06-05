#!/usr/bin/env python3
"""
aiw-tcc plugin: implements the tcc wrapper as a plugin.
This mirrors the behavior of the previous Go implementation: auto-detect TCC root,
add -I/-L from include/lib directories, and support modes: dll, run, x86_64/amd64/x64.
"""
import os
import shutil
import sys
import subprocess
from pathlib import Path

DEFAULT_ROOT = r"c:\green\tcc"


def detect_root():
    for key in ("TCC_HOME", "TCC_ROOT", "TCC_DIR"):
        v = os.environ.get(key, "").strip()
        if v:
            return v
    for name in ("tcc.exe", "tcc"):
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
    if a in ("dll", "run", "x86_64", "amd64", "x64"):
        return a, args[1:]
    return "", args


def is64bit(mode: str) -> bool:
    return mode in ("x86_64", "amd64", "x64")


def include_dir(root: str):
    p = Path(root) / "include"
    return str(p) if p.exists() else None


def lib_dir(root: str):
    p = Path(root) / "lib"
    return str(p) if p.exists() else None


def executable(root: str, prefer64: bool):
    candidates = [Path(root) / "tcc.exe", Path(root) / "x86_64-win32-tcc.exe"]
    if prefer64:
        candidates = list(reversed(candidates))
    for c in candidates:
        if c.exists():
            return str(c)
    for name in ("tcc.exe", "tcc"):
        p = shutil.which(name)
        if p:
            return p
    return str(Path(root) / "tcc.exe")


def args_for(mode, args, root):
    result = []
    if mode == "dll":
        result.append("-shared")
    if mode == "run":
        result.append("-run")
    result.extend(args)
    inc = include_dir(root)
    if inc:
        result.append("-I" + inc)
    ld = lib_dir(root)
    if ld:
        result.append("-L" + ld)
    return result


def print_help(root):
    inc = include_dir(root)
    ld = lib_dir(root)
    comp = Path(executable(root, False)).name
    print("Usage:")
    print("  aiw tcc [args...]")
    print("")
    print("Modes:")
    print("  tcc dll [args...]         Shortcut for -shared")
    print("  tcc run [args...]         Shortcut for -run")
    print("  tcc x86_64 [args...]      Use x86_64-win32-tcc.exe")
    print("  default compiler: tcc.exe (32-bit)")
    print("")
    print("Auto paths:")
    print(f"  root: {root}")
    print(f"  compiler: {comp}")
    print(f"  include: {inc if inc else '<missing>'}")
    print(f"  lib: {ld if ld else '<missing>'}")
    print("")
    print("Examples:")
    print("  aiw tcc hello.c -o hello.exe")
    print("  aiw tcc dll hello.c -o hello.dll")
    print("  aiw tcc x86_64 hello.c -o hello.exe")
    print("  aiw tcc run hello.c")
    print("  aiw tcc hello.c -o app.exe -luser32")


def run(name, argv):
    try:
        p = subprocess.Popen([name, *argv])
        p.communicate()
        return p.returncode
    except FileNotFoundError:
        print(f"executable not found: {name}", file=sys.stderr)
        return 2


def main():
    # aiw will pass args after the subcommand
    args = sys.argv[1:]
    if not args or is_help_arg(args[0]):
        print_help(detect_root())
        return 0
    root = detect_root()
    mode, rest = mode_for(args)
    exe = executable(root, is64bit(mode))
    rc = run(exe, args_for(mode, rest, root))
    return rc if rc is not None else 0


if __name__ == "__main__":
    sys.exit(main())
