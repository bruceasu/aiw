from __future__ import annotations

import json
from pathlib import Path
from typing import Dict

from codex_flow.process_utils import CommandError, run_command
from codex_flow.file_utils import atomic_write_text


class ArtifactManager:
    def create(self, workspace: Path, artifacts_dir: Path) -> Dict[str, str]:
        artifacts_dir.mkdir(parents=True, exist_ok=True)
        summary = {"workspace": str(workspace)}
        try:
            git_status = run_command(["git", "status", "--short"], cwd=workspace)
            git_diff = run_command(["git", "diff"], cwd=workspace)
            git_diff_binary = run_command(["git", "diff", "--binary"], cwd=workspace)
            commit = run_command(["git", "rev-parse", "HEAD"], cwd=workspace).strip()
            atomic_write_text(artifacts_dir / "git-status.txt", git_status)
            atomic_write_text(artifacts_dir / "git-diff.txt", git_diff)
            atomic_write_text(artifacts_dir / "final.patch", git_diff_binary)
            summary.update(
                {
                    "git_status_file": "git-status.txt",
                    "git_diff_file": "git-diff.txt",
                    "patch_file": "final.patch",
                    "head_commit": commit,
                }
            )
        except CommandError as exc:
            summary["error"] = str(exc)
        atomic_write_text(artifacts_dir / "summary.json", json.dumps(summary, indent=2, ensure_ascii=False) + "\n")
        return summary

