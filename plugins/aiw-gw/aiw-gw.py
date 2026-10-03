#!/usr/bin/env python3
"""Launch the precompiled Agent Gateway beside this plugin entry."""

import os
import subprocess
import sys
from pathlib import Path


def main() -> int:
    executable = Path(__file__).resolve().parent / (
        "agent-gateway.exe" if os.name == "nt" else "agent-gateway"
    )
    if not executable.is_file():
        print(
            f"gw: executable not found: {executable}. "
            "Compile program/agent-gateway and copy the binary to this directory.",
            file=sys.stderr,
        )
        return 1
    arguments = sys.argv[1:]
    options = arguments[:arguments.index("--")] if "--" in arguments else arguments
    has_config = any(
        argument in ("--config", "-config")
        or argument.startswith(("--config=", "-config="))
        for argument in options
    )
    if not has_config:
        position = 1 if arguments and arguments[0] in ("start", "stop") else 0
        arguments[position:position] = ["--config", str(executable.parent / "gateway.json")]
    command = [str(executable), *arguments]
    try:
        if os.name == "nt":
            return subprocess.call(command)
        os.execv(str(executable), command)
    except OSError as error:
        print(f"gw: cannot start {executable}: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
