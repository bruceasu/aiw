"""Compile AIW without retaining a binary, using a worktree-local Go cache."""

from __future__ import annotations

import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile


def worktree_go_cache(root: Path) -> Path:
    """Return a stable cache in this worktree unless explicitly overridden."""
    override = os.environ.get("AIW_GOCACHE")
    if override:
        return Path(override).expanduser()

    return root / ".ai" / "compile-cache" / "go"


def main() -> int:
    go = shutil.which("go")
    if go is None:
        print("compile failed: Go executable was not found on PATH", file=sys.stderr)
        return 127

    root = Path(__file__).resolve().parent.parent
    output_name = "aiw.exe" if os.name == "nt" else "aiw"
    go_cache = worktree_go_cache(root)
    go_cache.mkdir(parents=True, exist_ok=True)
    temporary_root = root / ".ai" / "compile-cache" / "tmp"
    temporary_root.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="aiw-compile-", dir=temporary_root) as temporary:
        output = Path(temporary) / output_name
        environment = os.environ.copy()
        environment["GOCACHE"] = str(go_cache)
        completed = subprocess.run(
            [go, "build", "-o", str(output), "main.go"],
            cwd=root,
            env=environment,
        )
    return completed.returncode


if __name__ == "__main__":
    raise SystemExit(main())
