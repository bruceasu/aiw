#!/usr/bin/env python3
"""
aiw-tcc plugin: lightweight wrapper for Tiny C Compiler.
Supports modes: dll/run/x86_64/release/upx/clean
Auto-detects TCC root and include/lib paths.
"""
import os, sys, shutil, subprocess, time
from pathlib import Path

DEFAULT_ROOT = r"c:\green\tcc"
MODES = {"dll","run","x86_64","amd64","x64","release","upx","clean","quiet"}
META = {
    'name': 'aiw tcc',
    'short': 'Tiny C Compiler wrapper with run and cross-architecture support.',
    'long': (
        'Wrapper for Tiny C Compiler (TCC). '
        'Supports fast compilation, direct execution (-run), shared libraries, '
        'and optional x86_64 build mode. Designed for minimal and fast C workflows.'
    ),
    'usage': 'aiw tcc [mode] [args...]',
    'args': [
        {'flag': 'run', 'description': 'Compile and execute immediately (-run).'},
        {'flag': 'dll', 'description': 'Build shared library (-shared).'},
        {'flag': 'x86_64', 'description': 'Use 64-bit compiler variant.'},
        {'flag': 'amd64', 'description': 'Alias for x86_64 mode.'},
        {'flag': 'x64', 'description': 'Alias for x86_64 mode.'},
    ],
    'examples': [
        'aiw tcc hello.c -run',
        'aiw tcc dll plugin.c -o plugin.dll',
        'aiw tcc x86_64 hello.c -o app.exe',
        'aiw tcc hello.c -o app.exe -luser32'
    ],
}
# ------------------- Helpers -------------------
def detect_root():
    for key in ("TCC_HOME","TCC_ROOT","TCC_DIR"):
        v = os.environ.get(key,"").strip()
        if v: return v
    for name in ("tcc.exe","tcc"):
        p = shutil.which(name)
        if p: return str(Path(p).parent)
    return DEFAULT_ROOT

def is_help_arg(arg): return arg.lower() in ("help","-h","--help")

def mode_for(args):
    if not args: return "",[]
    a=args[0].lower()
    if a in MODES: return a,args[1:]
    return "",args

def is64bit(mode): return mode in ("x86_64","amd64","x64")

def include_dir(root):
    p = Path(root)/"include"
    return str(p) if p.exists() else None

def lib_dir(root):
    p = Path(root)/"lib"
    return str(p) if p.exists() else None

def executable(root,prefer64):
    candidates = [Path(root)/"tcc.exe", Path(root)/"x86_64-win32-tcc.exe"]
    if prefer64: candidates=list(reversed(candidates))
    for c in candidates:
        if c.exists(): return str(c)
    for name in ("tcc.exe","tcc"):
        p=shutil.which(name)
        if p: return p
    return str(Path(root)/"tcc.exe")

def args_for(mode,args,root):
    result=[]
    if mode=="dll": result.append("-shared")
    if mode=="run": result.append("-run")
    result.extend(args)
    inc=include_dir(root)
    if inc: result.append("-I"+inc)
    ld=lib_dir(root)
    if ld: result.append("-L"+ld)
    return result

def show_size(path):
    p = Path(path)
    if p.exists(): print(f"size: {p.stat().st_size/1024:.1f} KB")

def run(name,argv):
    try:
        p=subprocess.Popen([name,*argv])
        p.communicate()
        return p.returncode
    except FileNotFoundError:
        print(f"executable not found: {name}",file=sys.stderr)
        return 2

def print_help(root):
    inc=include_dir(root)
    ld=lib_dir(root)
    comp=Path(executable(root,False)).name
    print("Usage: aiw tcc [mode] [args...]")
    print("Modes: dll/run/x86_64/release/upx/clean/quiet")
    print(f"Root: {root}, Compiler: {comp}")
    print(f"Include: {inc if inc else '<missing>'}, Lib: {ld if ld else '<missing>'}")
    print("Examples: aiw tcc hello.c -o hello.exe")
    print("          aiw tcc dll hello.c -o hello.dll")
    print("          aiw tcc x86_64 hello.c -o hello.exe")
    print("          aiw tcc run hello.c")
    print("          aiw tcc release hello.c")

def clean(src):
    stem = Path(src).stem
    for ext in (".o",".exe",".dll",".map"):
        f = Path(stem+ext)
        if f.exists(): f.unlink()

# ------------------- Main -------------------
def main():
    args=sys.argv[1:]
    root=detect_root()

    if not args or is_help_arg(args[0]):
        print_help(root)
        return 0

    mode,rest = mode_for(args)

    # release preset
    if mode=="release":
        mode="run"
        rest.insert(0,"-run")  # optional
        upx=True
    else: upx=False

    exe=executable(root,is64bit(mode))

    # build timer
    t0=time.time()
    rc=run(exe,args_for(mode,rest,root))
    dt=time.time()-t0

    # UPX compress
    if upx and Path(rest[-1]).suffix.lower()==".exe":
        exe_path=rest[-1]
        upx_path=shutil.which("upx")
        if upx_path:
            subprocess.run([upx_path,"--best","--lzma","--strip-relocs=0",exe_path])
            print("UPX applied")

    # show size
    if rest and Path(rest[-1]).suffix.lower()==".exe":
        show_size(rest[-1])

    if "quiet" not in mode:
        print(f"time: {dt:.2f}s")

    # clean mode
    if mode=="clean" and rest:
        clean(rest[0])

    return rc if rc is not None else 0

if __name__=="__main__":
    sys.exit(main())