"""Compile the gateway for Windows and Linux without a distributable artifact."""

import os
from pathlib import Path
import subprocess

root = Path(__file__).resolve().parent.parent
environment = os.environ.copy()
environment.update(GOPROXY="off", GOSUMDB="off", GOTOOLCHAIN="local", CGO_ENABLED="0")
for target in ("windows", "linux"):
    environment.update(GOOS=target, GOARCH="amd64")
    result = subprocess.run(["go", "build", "-o", os.devnull, "."], cwd=root, env=environment)
    if result.returncode:
        raise SystemExit(result.returncode)
