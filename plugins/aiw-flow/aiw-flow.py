#!/usr/bin/env python3
"""aiw plugin entry point for the aiw-flow Codex workflow manager."""

from __future__ import annotations

import sys
from pathlib import Path


def main() -> int:
    plugin_dir = Path(__file__).resolve().parent
    source_dir = plugin_dir / "src"
    if not source_dir.is_dir():
        print(f"missing aiw-flow source directory: {source_dir}", file=sys.stderr)
        return 2

    sys.path.insert(0, str(source_dir))
    from codex_flow.cli import main as flow_main

    return flow_main(sys.argv[1:])


if __name__ == "__main__":
    raise SystemExit(main())
