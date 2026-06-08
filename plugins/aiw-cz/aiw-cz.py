#!/usr/bin/env python3
import os
import platform
import stat
import subprocess
import sys
from pathlib import Path

META = {
    'name': 'aiw cz',
    'short': 'Commitizen plugin for aiw.',
    'long': (
        'Commitizen plugin for aiw. '
        'Provides a standardized way to write commit messages.'
    ),
    'usage': 'aiw cz [args...]',
    'args': [
        {'flag': '--help', 'description': 'Show this help message and exit.'},
        {'flag': '--version', 'description': 'Show version information and exit.'},
    ],
    'examples': [
        'aiw cz',
        'aiw cz --help',
        'aiw cz --version'
    ],
}

def target_binary(base: Path) -> Path:
    system = platform.system().lower()
    if system.startswith("win"):
        return base / "cz.exe"
    return base / "cz"


def ensure_executable(path: Path) -> None:
    if os.name == "nt":
        return
    mode = path.stat().st_mode
    if mode & stat.S_IXUSR:
        return
    path.chmod(mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)


def main() -> int:
    base = Path(__file__).resolve().parent
    target = target_binary(base)
    if not target.exists():
        print(f"missing cz binary: {target}", file=sys.stderr)
        return 2

    ensure_executable(target)
    proc = subprocess.run([str(target), *sys.argv[1:]])
    return proc.returncode


if __name__ == "__main__":
    sys.exit(main())
