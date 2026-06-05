#!/usr/bin/env python3
"""
aiw-gcc plugin: lightweight wrapper around gcc/g++ to add auto include/lib paths.
Supports mode `dll` -> `-shared` and otherwise forwards args to gcc.
"""
import os
import shutil
import sys
import subprocess
from pathlib import Path

DEFAULT_ROOT_UNIX = "/usr"
DEFAULT_ROOT_WIN = r"C:\MinGW"


def detect_root():
    for key in ("GCC_HOME", "GCC_ROOT"):
        v = os.environ.get(key, "").strip()
        if v:
            return v
    for name in ("gcc.exe", "gcc"):
        p = shutil.which(name)
        if p:
            return str(Path(p).parent.parent)
    return DEFAULT_ROOT_WIN if os.name == "nt" else DEFAULT_ROOT_UNIX


def is_help_arg(arg: str) -> bool:
    return arg.lower() in ("help", "-h", "--help")


def mode_for(args):
    if not args:
        return "", []
    a = args[0].lower()
    if a in ("dll", "run"):
        return a, args[1:]
    return "", args


def include_dir(root: str):
    p = Path(root) / "include"
    if p.exists():
        return str(p)
    return None


def lib_dir(root: str):
    p = Path(root) / "lib"
    if p.exists():
        return str(p)
    # try lib64
    p2 = Path(root) / "lib64"
    if p2.exists():
        return str(p2)
    return None


def executable(root: str):
    for name in ("gcc", "gcc.exe"):
        p = shutil.which(name)
        if p:
            return p
    g = Path(root) / "bin" / ("gcc.exe" if os.name == "nt" else "gcc")
    return str(g)


def args_for(mode, args, root):
    result = []
    if mode == "dll":
        result.append("-shared")
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
    comp = Path(executable(root)).name
    print("Usage:")
    print("  aiw gcc [args...]")
    print("")
    print("Modes:")
    print("  gcc dll [args...]         Shortcut for -shared")
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
