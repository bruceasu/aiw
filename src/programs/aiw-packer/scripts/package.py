"""Explicit offline packaging, requested by the person running this script.

Unlike compile.py, this produces a distributable executable. The FD workflow
does not run it automatically. Copy the complete plugin folder when distributing.
"""
import os
from pathlib import Path
import subprocess

module = Path(__file__).resolve().parents[1]
plugin = module.parents[2] / "src/plugins/aiw-packer"
target = plugin / ("aiw-packer.exe" if os.name == "nt" else "aiw-packer")
environment = os.environ.copy()
environment.update(GOPROXY="off", GOSUMDB="off", GOTOOLCHAIN="local", CGO_ENABLED="0",
                   GOFLAGS="", GOWORK="off")
raise SystemExit(subprocess.run(["go", "build", "-trimpath", "-o", str(target), "."],
                               cwd=module, env=environment, check=False).returncode)
