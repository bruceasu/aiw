"""Run the approved E06 generation tests once and preserve local evidence; never retry."""

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
    now = lambda: datetime.now(timezone.utc).isoformat()
    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S%fZ")
    output = base / "verification-results" / ("e06-generation-terminal-" + stamp)
    output.mkdir(parents=True, exist_ok=False)

    def digest(path):
        return hashlib.sha256(path.read_bytes()).hexdigest()

    def save(name, value):
        (output / name).write_text(
            json.dumps(value, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
        )

    command = [
        "go", "test", "./internal/workflow", "-run",
        "^(TestKnowledgeExtractionRequiresExplicitNoNewEvidence|TestKnowledgePartialDraftPreservesCoverageAndHistory|TestKnowledgeGeneratedVersionsPreservePriorReview)$",
        "-count=1", "-timeout=60s", "-vet=off", "-json",
    ]
    environment = os.environ.copy()
    overrides = {
        "GOPROXY": "off", "GOSUMDB": "off", "GOTOOLCHAIN": "local",
        "GOCACHE": str(root / ".ai" / "compile-cache" / "go"),
    }
    environment.update(overrides)
    paths = set(root.glob("internal/**/*.go")) | set(root.glob("cmd/**/*.go"))
    paths |= set(root.glob("*.go")) | set(base.rglob("*.md"))
    paths |= {root / "go.mod", root / "go.sum", Path(__file__).resolve()}
    inputs = {
        path.relative_to(root).as_posix(): digest(path)
        for path in sorted(paths)
        if path.is_file() and "verification-results" not in path.parts
    }
    save("inputs.json", inputs)
    save("invocation.json", {
        "scope": "One original three-test E06 generation run from the operator terminal",
        "authorization_context": "Execution requires explicit operator approval for this bounded E06 generation group; the runner does not verify approval and implies no product grant",
        "product_runner_grant": False,
    })
    run = {
        "command": command, "cwd": str(root), "started_at": now(),
        "environment_overrides": overrides,
        "temporary_environment": {key: environment.get(key) for key in ("TMP", "TEMP", "TMPDIR")},
        "input_manifest_sha256": digest(output / "inputs.json"),
        "plan_reference": "../../e06-generation-verification-plan.md",
    }
    (output / "approved-plan.md").write_bytes((base / "e06-generation-verification-plan.md").read_bytes())
    run["plan_sha256"] = digest(output / "approved-plan.md")
    save("run.json", run)
    print("Evidence:", output, flush=True)
    try:
        version = subprocess.run(
            ["go", "version"], cwd=root, env=environment, capture_output=True, text=True
        )
        run["toolchain"] = version.stdout.strip()
        run["toolchain_error"] = version.stderr.strip()
        with (output / "stdout.jsonl").open("wb") as stdout, (output / "stderr.txt").open("wb") as stderr:
            result = subprocess.run(command, cwd=root, env=environment, stdout=stdout, stderr=stderr)
        run["exit_code"] = result.returncode
    except OSError as error:
        run["launch_error"] = str(error)
        run["exit_code"] = 127
    finally:
        run["ended_at"] = now()
        run["changed_inputs"] = [
            name for name, old in inputs.items()
            if not (root / name).is_file() or digest(root / name) != old
        ]
        for name in ("stdout.jsonl", "stderr.txt"):
            path = output / name
            if path.exists():
                run[name + "_sha256"] = digest(path)
        save("run.json", run)
    print("Exit code:", run["exit_code"], flush=True)
    return run["exit_code"]


if __name__ == "__main__":
    sys.exit(main())
