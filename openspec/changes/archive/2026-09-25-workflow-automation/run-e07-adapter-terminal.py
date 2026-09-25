"""Run the four offline adapter tests once and preserve their exact inputs."""

import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
from datetime import datetime, timezone


def main():
    base = Path(__file__).resolve().parent
    root = base.parents[2]
    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S%fZ")
    output = base / "verification-results" / ("e07-adapter-terminal-" + stamp)
    output.mkdir(parents=True, exist_ok=False)
    digest = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()

    def save(name, value):
        (output / name).write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    plan = base / "e07-adapter-verification-plan.md"
    paths = [root / "plugins" / name for name in ("test_notify_managed.py", "aiw-notify.py", "send_teams_msg.py")]
    paths += [Path(__file__).resolve(), plan]
    inputs = {path.relative_to(root).as_posix(): digest(path) for path in paths}
    save("inputs.json", inputs)
    (output / "approved-plan.md").write_bytes(plan.read_bytes())
    command = [sys.executable, "-I", "-B", str(root / "plugins/test_notify_managed.py"), "-v"]
    run = {"command": command, "cwd": str(root), "toolchain": sys.version,
           "started_at": datetime.now(timezone.utc).isoformat(), "timeout_seconds": 60,
           "input_manifest_sha256": digest(output / "inputs.json"),
           "plan_sha256": digest(output / "approved-plan.md"), "plan_reference": "../../e07-adapter-verification-plan.md",
           "authorization_context": "Explicit operator invocation or approval of these four offline adapter tests; no product grant is implied."}
    save("run.json", run)
    print("Evidence:", output, flush=True)
    try:
        with (output / "stdout.txt").open("wb") as stdout, (output / "stderr.txt").open("wb") as stderr:
            result = subprocess.run(command, cwd=root, env=os.environ.copy(), stdout=stdout, stderr=stderr, timeout=60)
        run["exit_code"] = result.returncode
    except subprocess.TimeoutExpired:
        run["exit_code"], run["failure"] = 124, "test process exceeded 60 seconds"
    except OSError:
        run["exit_code"], run["failure"] = 127, "test process could not start"
    finally:
        run["ended_at"] = datetime.now(timezone.utc).isoformat()
        run["changed_inputs"] = [name for name, old in inputs.items() if not (root / name).is_file() or digest(root / name) != old]
        for name in ("stdout.txt", "stderr.txt"):
            if (output / name).exists():
                run[name + "_sha256"] = digest(output / name)
        save("run.json", run)
    print("Exit code:", run["exit_code"], flush=True)
    return run["exit_code"]


if __name__ == "__main__":
    sys.exit(main())
