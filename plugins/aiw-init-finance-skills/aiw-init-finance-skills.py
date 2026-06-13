#!/usr/bin/env python3
"""
aiw-init-finance-skills.py

Initialize Financial Admin & Analytics Codex Skills for the current project.

Expected directory layout:

aiw-init-finance-skills/
├── aiw-init-finance-skills.py
├── install_codex_skills.py
└── assets/
    └── dist.zip

Default behavior:
- Install bundled finance skills from assets/dist.zip
- Target current working directory:
  ./.codex/skills

Usage:
    python aiw-init-finance-skills.py

Dry run:
    python aiw-init-finance-skills.py --dry-run

Overwrite existing skills:
    python aiw-init-finance-skills.py --force

Backup existing skills before replacing:
    python aiw-init-finance-skills.py --backup

Custom project root:
    python aiw-init-finance-skills.py --project-root /path/to/repo

Custom installer:
    python aiw-init-finance-skills.py --installer /path/to/install_codex_skills.py

Custom dist zip:
    python aiw-init-finance-skills.py --dist /path/to/dist.zip
"""

from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Install Financial Admin & Analytics Codex Skills into the current project."
    )

    parser.add_argument(
        "--project-root",
        default=".",
        help="Project root where .codex/skills will be created. Default: current directory.",
    )

    parser.add_argument(
        "--installer",
        default=None,
        help="Path to install_codex_skills.py. Default: same directory as this script.",
    )

    parser.add_argument(
        "--dist",
        default=None,
        help="Path to bundled finance skills zip. Default: assets/dist.zip beside this script.",
    )

    parser.add_argument(
        "--force",
        action="store_true",
        help="Overwrite existing installed skills.",
    )

    parser.add_argument(
        "--backup",
        action="store_true",
        help="Backup existing installed skills before replacing them.",
    )

    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show what would be installed without changing files.",
    )

    parser.add_argument(
        "--python",
        default=sys.executable,
        help="Python executable used to run install_codex_skills.py. Default: current Python.",
    )

    return parser.parse_args()


def fail(message: str, code: int = 1) -> None:
    print(f"error: {message}", file=sys.stderr)
    raise SystemExit(code)


def main() -> int:
    args = parse_args()

    if args.force and args.backup:
        fail("Use only one of --force or --backup.")

    script_dir = Path(__file__).resolve().parent

    installer = (
        Path(args.installer).expanduser().resolve()
        if args.installer
        else "aiw"
    )

    dist_zip = (
        Path(args.dist).expanduser().resolve()
        if args.dist
        else script_dir / "assets" / "dist.zip"
    )

    project_root = Path(args.project_root).expanduser().resolve()
    target_dir = project_root / ".codex" / "skills"

    #if not installer.is_file():
    #    fail(f"installer not found: {installer}")

    if not dist_zip.is_file():
        fail(f"dist zip not found: {dist_zip}")

    if not project_root.exists():
        fail(f"project root does not exist: {project_root}")

    if not project_root.is_dir():
        fail(f"project root is not a directory: {project_root}")
    if installer.startswith("aiw"):
        command = [
        f"{installer}",
        "install-skill", 
        f"{dist_zip}",
        "--dest",
        f"{target_dir}",
    ]
    else:
        command = [
            args.python,
            f"{installer}",
            f"{dist_zip}", 
            "--dest",
            f"{target_dir}",
        ]

    if args.force:
        command.append("--force")

    if args.backup:
        command.append("--backup")

    if args.dry_run:
        command.append("--dry-run")

    print("Installing Financial Admin & Analytics Skills")
    print()
    print(f"Project root : {project_root}")
    print(f"Target dir   : {target_dir}")
    print(f"Installer    : {installer}")
    print(f"Dist zip     : {dist_zip}")
    print()
    print("Command:")
    print(" ".join(command))
    print()

    result = subprocess.run(command, cwd=str(project_root))

    if result.returncode != 0:
        fail(f"installation failed with exit code {result.returncode}", result.returncode)

    print()
    print("Done.")
    print()
    print("Installed into:")
    print(f"  {target_dir}")
    print()
    print("Try:")
    print('  codex exec "Use metrics-review to review specs/metrics.md"')
    print('  codex exec "Use release-review to review this release"')
    print('  codex exec "Use autoplan-finance to generate PLAN.md"')

    return 0


if __name__ == "__main__":
    raise SystemExit(main())