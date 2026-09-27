"""Compile AIW without retaining a binary, using a worktree-local Go cache."""

from __future__ import annotations

import os
from pathlib import Path
import shutil
import subprocess
import sys


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
    go_cache = worktree_go_cache(root)
    go_cache.mkdir(parents=True, exist_ok=True)
    # On Windows, NUL makes the linker perform the full compile without
    # creating an executable that the managed environment may refuse to write.
    # On other platforms, /dev/null has the same no-artifact behavior.
    output = os.devnull
    environment = os.environ.copy()
    environment["GOCACHE"] = str(go_cache)
    targets = ["./cmd/aiw", "./cmd/aiw-wf", "./cmd/aiw-req", "./cmd/aiw-cz"]
    for target in targets:
        completed = subprocess.run(
            [go, "build", "-o", output, target],
            cwd=root,
            env=environment,
        )
        if completed.returncode != 0:
            return completed.returncode
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
