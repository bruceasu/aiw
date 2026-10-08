"""Run FD-035 black-box cases and measure branches in child CLI processes."""

from __future__ import annotations

import os
from pathlib import Path
import subprocess
import sys
import tempfile


def main() -> int:
    package_dir = os.environ.get("FD035_COVERAGE_PACKAGE_DIR", "")
    env = os.environ.copy()
    if package_dir:
        sys.path.insert(0, package_dir)
        env["PYTHONPATH"] = os.pathsep.join(
            item for item in (package_dir, env.get("PYTHONPATH", "")) if item
        )
    try:
        import coverage
    except ImportError:
        print("coverage.py is unavailable; no tests or coverage command was run.", file=sys.stderr)
        return 2

    root = Path(__file__).resolve().parents[1]
    with tempfile.TemporaryDirectory(prefix="fd035-coverage-") as temporary:
        temp = Path(temporary)
        bootstrap = temp / "bootstrap"
        data_dir = temp / "data"
        bootstrap.mkdir()
        data_dir.mkdir()
        config = temp / ".coveragerc"
        data_file = data_dir / ".coverage"
        config.write_text(
            "[run]\n"
            "branch = True\n"
            "parallel = True\n"
            "relative_files = True\n"
            "source = plugins\n"
            f"data_file = {data_file}\n",
            encoding="utf-8",
        )
        (bootstrap / "sitecustomize.py").write_text(
            "import coverage\ncoverage.process_startup()\n",
            encoding="utf-8",
        )
        env.update(
            {
                "FD035_COVERAGE_CONFIG": str(config),
                "FD035_COVERAGE_BOOTSTRAP": str(bootstrap),
                "FD035_COVERAGE_FILE": str(data_file),
            }
        )
        result = subprocess.run(
            [sys.executable, "-B", "-m", "unittest", "tests.test_fd032_risk_decision_blackbox", "-v"],
            cwd=root,
            env=env,
            timeout=180,
            check=False,
        )
        if result.returncode:
            return result.returncode

        measured = coverage.Coverage(config_file=str(config))
        measured.load()
        measured.combine(data_paths=[str(data_dir)], keep=True)
        measured.save()
        files = sorted(
            path for path in measured.get_data().measured_files()
            if Path(path).name == "aiw-fd.py" and "plugins" in Path(path).parts
        )
        if not files:
            print("No CLI source file was measured by child-process coverage.", file=sys.stderr)
            return 2
        for path in files:
            branch_stats = measured.branch_stats(path)
            total = sum(total for total, _ in branch_stats.values())
            taken = sum(taken for _, taken in branch_stats.values())
            percentage = (taken * 100 / total) if total else 100.0
            print(
                f"Branch coverage: {path}: {taken}/{total} branches "
                f"({percentage:.1f}%)."
            )
        return 0


if __name__ == "__main__":
    raise SystemExit(main())
