"""Compile converter and plugin syntax offline; retain no executable or bytecode."""
import os
from pathlib import Path
import subprocess

module = Path(__file__).resolve().parents[1]
root = module.parents[2]
for source in (root / "src/plugins/aiw-file-convert").glob("*.py"):
    compile(source.read_text(encoding="utf-8"), str(source), "exec")
environment = os.environ.copy()
environment.update(GOPROXY="off", GOSUMDB="off", GOTOOLCHAIN="local", CGO_ENABLED="0",
                   GOCACHE=str(root / ".ai/compile-cache/md-converter"))
raise SystemExit(subprocess.run(["go", "build", "-o", os.devnull, "."], cwd=module, env=environment).returncode)
