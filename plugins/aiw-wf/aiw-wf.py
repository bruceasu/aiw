#!/usr/bin/env python3
"""Run the platform-specific standalone AIW workflow program."""

from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path


def _binary_name() -> str:
    if os.name == "nt":
        return "aiw-wf.exe"
    if sys.platform.startswith("linux"):
        return "aiw-wf"
    raise RuntimeError(f"unsupported platform: {sys.platform}")


def main() -> int:
    plugin_dir = Path(__file__).resolve().parent
    override = os.environ.get("AIW_WF_BINARY")
    binary = Path(override) if override else plugin_dir / _binary_name()
    if not binary.is_file():
        print(
            f"aiw-wf binary not found: {binary}\n"
            "Run build.bat wf (Windows) or build.bat plugins to create "
            "the bundled workflow binaries.",
            file=sys.stderr,
        )
        return 1

    completed = subprocess.run([str(binary), *sys.argv[1:]])
    return completed.returncode


if __name__ == "__main__":
    raise SystemExit(main())
