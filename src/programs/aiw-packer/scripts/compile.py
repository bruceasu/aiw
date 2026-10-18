"""Compile only, offline; keep no executable or Python bytecode."""
import os
from pathlib import Path
import subprocess

module = Path(__file__).resolve().parents[1]
repo = module.parents[2]
for source in (repo / "src/plugins/aiw-packer").glob("*.py"):
    compile(source.read_text(encoding="utf-8"), str(source), "exec")
for source in (module / "scripts").glob("*.py"):
    compile(source.read_text(encoding="utf-8"), str(source), "exec")
environment = os.environ.copy()
environment.update(GOPROXY="off", GOSUMDB="off", GOTOOLCHAIN="local", CGO_ENABLED="0",
                   GOFLAGS="", GOWORK="off", GOCACHE=str(repo / ".ai/compile-cache/packer"))
raise SystemExit(subprocess.run(["go", "build", "-o", os.devnull, "."],
                               cwd=module, env=environment, check=False).returncode)
