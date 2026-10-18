#!/usr/bin/env python3
"""Run the bundled standalone packer without building or downloading anything."""
import os
from pathlib import Path
import subprocess
import sys


def main():
    plugin = Path(__file__).resolve().parent
    name = "aiw-packer.exe" if os.name == "nt" else "aiw-packer"
    binary = Path(os.environ["AIW_PACKER_BINARY"]) if os.environ.get("AIW_PACKER_BINARY") else plugin / name
    if not binary.is_file():
        print(f"aiw-packer binary not found: {binary}\n"
              "Prepare it explicitly with python src/programs/aiw-packer/scripts/package.py; "
              "see src/programs/aiw-packer/README.md.", file=sys.stderr)
        return 1
    try:
        return subprocess.run([str(binary.resolve()), *sys.argv[1:]], check=False).returncode
    except OSError as error:
        print(f"aiw-packer could not start: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
