"""Run selected Go tests with an isolated, worktree-local temporary cache."""

from __future__ import annotations

import os
from pathlib import Path
import shutil
import subprocess
import sys


def main() -> int:
    if len(sys.argv) < 2:
        print("usage: python scripts/test.py <go-test-arguments>", file=sys.stderr)
        return 2

    go = shutil.which("go")
    if go is None:
        print("test failed: Go executable was not found on PATH", file=sys.stderr)
        return 127

    root = Path(__file__).resolve().parent.parent
    module_root = root / "src"
    test_root = root / ".ai" / "tmp" / "go-tests"
    go_cache = test_root / "cache"
    temporary = test_root / "tmp"
    go_cache.mkdir(parents=True, exist_ok=True)
    temporary.mkdir(parents=True, exist_ok=True)

    environment = os.environ.copy()
    environment["GOCACHE"] = str(go_cache)
    environment["GOTMPDIR"] = str(temporary)
    environment["TMP"] = str(temporary)
    environment["TEMP"] = str(temporary)
    environment["TMPDIR"] = str(temporary)
    environment["GOPROXY"] = "off"
    environment["GOSUMDB"] = "off"
    return subprocess.run([go, "test", *sys.argv[1:]], cwd=module_root, env=environment).returncode


if __name__ == "__main__":
    raise SystemExit(main())
