#!/usr/bin/env python3
"""
aiw-bcc

Tiny Borland C++ wrapper.

Features:
- auto-detect bcc32
- tiny mode
- gui mode
- dll mode
- compile+run
- tiny linker optimization
- automatic output names
"""

import os
import shutil
import subprocess
import sys
from pathlib import Path

META = {
    'name': 'aiw bcc',
    'short': 'Borland BCC32 compiler wrapper with tiny and GUI modes.',
    'long': (
        'Lightweight wrapper for Borland C++ Builder (bcc32). '
        'Supports DLL builds, GUI applications, and tiny optimization mode '
        'for minimal executable size. Optional UPX compression supported.'
    ),
    'usage': 'aiw bcc [mode] [args...]',
    'args': [
        {'flag': 'tiny', 'description': 'Enable size optimization (-O2, RTTI off, exception off).'},
        {'flag': 'gui', 'description': 'Build Windows GUI application (no console window).'},
        {'flag': 'dll', 'description': 'Build dynamic library (-WD).'},
        {'flag': 'run', 'description': 'Compile and run the program immediately.'},
        {'flag': 'upx', 'description': 'Compress output binary using UPX.'},
    ],
    'examples': [
        'aiw bcc hello.cpp',
        'aiw bcc tiny hello.cpp',
        'aiw bcc gui tiny app.cpp',
        'aiw bcc dll plugin.cpp',
        'aiw bcc tiny upx app.cpp'
    ],
}

DEFAULT_ROOT = r"C:\green\Borland\BCC55"

HELP = {"help", "-h", "--help", "/?"}

MODES = {
    "dll",
    "tiny",
    "gui",
    "run",
    "dry",
    "quiet",
    "upx",
    "release",
}

def upx_executable():
    return (
        shutil.which("upx.exe")
        or shutil.which("upx")
    )

def compress_upx(exe_name):
    upx = upx_executable()

    if not upx:
        print(
            "warning: upx not found",
            file=sys.stderr,
        )
        return

    cmd = [
        upx,
        "--best",
        "--lzma",
        "--strip-relocs=0",
        exe_name,
    ]

    subprocess.call(cmd)

def detect_root():
    env = os.environ.get("BCC_ROOT", "").strip()

    if env:
        return env

    found = shutil.which("bcc32.exe")

    if found:
        return str(Path(found).parent.parent)

    return DEFAULT_ROOT


def executable(root):
    p = Path(root) / "bin" / "bcc32.exe"

    if p.exists():
        return str(p)

    return shutil.which("bcc32.exe") or str(p)


def ilink(root):
    p = Path(root) / "bin" / "ilink32.exe"

    if p.exists():
        return str(p)

    return shutil.which("ilink32.exe") or str(p)


def find_dir(root, names):
    root = Path(root)

    for name in names:
        p = root / name

        if p.exists():
            return str(p)

    return None


def include_dir(root):
    return find_dir(root, ("Include", "include"))


def lib_dir(root):
    return find_dir(root, ("Lib", "lib"))


def prepend_path(path):
    cur = os.environ.get("PATH", "")

    parts = cur.split(os.pathsep)

    if path not in parts:
        os.environ["PATH"] = path + os.pathsep + cur


def parse(argv):
    modes = set()
    args = []

    for a in argv:
        low = a.lower()

        if low in MODES:
            modes.add(low)
        else:
            args.append(a)

    if "release" in modes:
        modes.update({
            "tiny",
            "gui",
            "upx",
            "quiet",
        })

    return modes, args


def source_file(args):
    for a in args:
        if a.lower().endswith((".c", ".cpp", ".cc", ".cxx")):
            return Path(a)

    return None


def output_name(src, modes):
    if not src:
        return "a.exe"

    if "dll" in modes:
        return src.stem + ".dll"

    return src.stem + ".exe"




def compile_args(modes, args, root):
    out = []

    if "dll" in modes:
        out.append("-WD")

    if "gui" in modes:
        out.append("-tW")
    
    if "console" in modes:
        out.append("-tWC")

    if "tiny" in modes:
        out.extend([
            "-O2",
            "-RT-",
            "-E-",
            "-6",
        ])

    inc = include_dir(root)

    if inc:
        out.append("-I" + inc)

    lib = lib_dir(root)

    if lib:
        out.append("-L" + lib)

    out.extend(args)

    return out


def tiny_link(exe_name, root):
    linker = ilink(root)

    stem = Path(exe_name).stem

    obj = stem + ".obj"

    cmd = [
        linker,
        obj + "," + exe_name,
        ",,",
        "c0x32",
        ",,",
        "-x",
        "-t",
        "-6",
    ]

    subprocess.call(cmd)


def cleanup(src):
    stem = src.stem

    for ext in (
        ".obj",
        ".tds",
        ".map",
        ".map",
        ".ilc",
        ".ild",
        ".ilf",
    ):
        p = Path(stem + ext)

        if p.exists():
            try:
                p.unlink()
            except:
                pass


def print_help(root):
    print("Usage:")
    print("  aiw bcc [modes] file.cpp")
    print("")
    print("Modes:")
    print("  tiny    optimize for size")
    print("  dll     build DLL")
    print("  gui     Windows GUI app")
    print("  run     compile and run")
    print("  dry     print commands only")
    print("  quiet   suppress wrapper messages")
    print("  upx     compress output with UPX (if available)")
    print("")
    print("Examples:")
    print("  aiw bcc hello.cpp")
    print("  aiw bcc tiny hello.cpp")
    print("  aiw bcc gui tiny app.cpp")
    print("  aiw bcc run hello.cpp")
    print("")
    print("Detected:")
    print(f"  root: {root}")
    print(f"  bcc : {executable(root)}")


def main():
    argv = sys.argv[1:]

    root = detect_root()

    if not argv or argv[0].lower() in HELP:
        print_help(root)
        return 0

    prepend_path(str(Path(root) / "bin"))

    modes, args = parse(argv)

    src = source_file(args)

    if not src:
        print("source file not found", file=sys.stderr)
        return 2

    exe_name = output_name(src, modes)

    cmd = [
        executable(root),
        "-e" + exe_name,
        *compile_args(modes, args, root),
    ]

    if "dry" in modes:
        print(" ".join(cmd))
        return 0

    if "quiet" not in modes:
        print("build:", exe_name)

    rc = subprocess.call(cmd)

    if rc != 0:
        return rc

    if "tiny" in modes:
        tiny_link(exe_name, root)

    if "upx" in modes:
        compress_upx(exe_name)


    if "run" in modes:
        subprocess.call([exe_name])

    cleanup(src)

    return 0


if __name__ == "__main__":
    sys.exit(main())

