#!/usr/bin/env python3
"""
Install Codex-style Skills from a folder or zip archive.

Supported inputs:
1. A single skill folder:
   metrics-review/
   ├── SKILL.md
   └── references/

2. A single skill.zip:
   skill.zip
   └── metrics-review/
       ├── SKILL.md
       └── references/

3. A folder containing multiple skills:
   skills/
   ├── metrics-review/
   │   └── SKILL.md
   └── release-review/
       └── SKILL.md

4. A bundle containing dist/*/skill.zip:
   dist/
   ├── metrics-review/skill.zip
   └── release-review/skill.zip

Default target:
- project scope: ./.codex/skills

Optional target:
- user scope: ~/.codex/skills
- custom path: --dest /path/to/skills
"""

from __future__ import annotations

import argparse
import datetime as dt
import os
import re
import shutil
import sys
import tempfile
import zipfile
from pathlib import Path
from typing import Iterable, List, Optional, Tuple


FRONTMATTER_RE = re.compile(r"^---\s*\n(.*?)\n---\s*", re.DOTALL)
NAME_RE = re.compile(r"^name:\s*([a-z0-9][a-z0-9-]*)\s*$", re.MULTILINE)


class InstallError(Exception):
    pass


def now_stamp() -> str:
    return dt.datetime.now().strftime("%Y%m%d-%H%M%S")


def is_zip(path: Path) -> bool:
    return path.is_file() and path.suffix.lower() == ".zip"


def find_skill_md(path: Path) -> Optional[Path]:
    skill_md = path / "SKILL.md"
    if skill_md.is_file():
        return skill_md

    skill_md_lower = path / "skill.md"
    if skill_md_lower.is_file():
        return skill_md_lower

    return None


def read_skill_name(skill_dir: Path) -> str:
    skill_md = find_skill_md(skill_dir)
    if not skill_md:
        raise InstallError(f"Missing SKILL.md in {skill_dir}")

    text = skill_md.read_text(encoding="utf-8")
    match = FRONTMATTER_RE.search(text)
    if not match:
        raise InstallError(f"Missing YAML frontmatter in {skill_md}")

    frontmatter = match.group(1)
    name_match = NAME_RE.search(frontmatter)
    if not name_match:
        raise InstallError(f"Missing or invalid frontmatter name in {skill_md}")

    return name_match.group(1).strip()


def validate_skill_dir(skill_dir: Path) -> str:
    if not skill_dir.is_dir():
        raise InstallError(f"Not a directory: {skill_dir}")

    name = read_skill_name(skill_dir)

    # Basic name/path sanity check. We allow folder name mismatch but warn later.
    if not re.match(r"^[a-z0-9][a-z0-9-]*$", name):
        raise InstallError(f"Invalid skill name '{name}' in {skill_dir}")

    return name


def safe_copytree(src: Path, dest: Path, force: bool, backup: bool, dry_run: bool) -> None:
    if dest.exists():
        if force:
            if dry_run:
                print(f"[dry-run] remove existing: {dest}")
            else:
                shutil.rmtree(dest)
        elif backup:
            backup_path = dest.with_name(f"{dest.name}.backup-{now_stamp()}")
            if dry_run:
                print(f"[dry-run] backup existing: {dest} -> {backup_path}")
            else:
                shutil.move(str(dest), str(backup_path))
        else:
            raise InstallError(
                f"Target already exists: {dest}\n"
                f"Use --force to overwrite or --backup to move existing copy aside."
            )

    if dry_run:
        print(f"[dry-run] copy: {src} -> {dest}")
    else:
        shutil.copytree(src, dest)


def extract_zip(zip_path: Path, work_dir: Path) -> Path:
    if not zipfile.is_zipfile(zip_path):
        raise InstallError(f"Invalid zip file: {zip_path}")

    out_dir = work_dir / zip_path.stem
    out_dir.mkdir(parents=True, exist_ok=True)

    with zipfile.ZipFile(zip_path, "r") as zf:
        zf.extractall(out_dir)

    return out_dir


def iter_nested_skill_zips(path: Path) -> Iterable[Path]:
    """
    Find nested skill.zip files, especially dist/<skill>/skill.zip.
    """
    if not path.is_dir():
        return []

    return path.rglob("skill.zip")


def direct_skill_dirs_under(path: Path) -> List[Path]:
    """
    Return immediate child directories that are skills, or the path itself if it is a skill.
    """
    result: List[Path] = []

    if path.is_dir() and find_skill_md(path):
        result.append(path)
        return result

    if path.is_dir():
        for child in sorted(path.iterdir()):
            if child.is_dir() and find_skill_md(child):
                result.append(child)

    return result


def deep_skill_dirs(path: Path) -> List[Path]:
    """
    Find skill directories by scanning for SKILL.md.
    Avoid returning nested children inside an already detected skill.
    """
    candidates: List[Path] = []

    if path.is_dir() and find_skill_md(path):
        return [path]

    for skill_md in path.rglob("SKILL.md"):
        candidates.append(skill_md.parent)

    # Also support lowercase skill.md, though SKILL.md is preferred.
    for skill_md in path.rglob("skill.md"):
        candidates.append(skill_md.parent)

    unique: List[Path] = []
    seen = set()
    for c in sorted(candidates):
        resolved = c.resolve()
        if resolved not in seen:
            seen.add(resolved)
            unique.append(c)

    # Remove candidates that are inside another skill dir.
    filtered: List[Path] = []
    for c in unique:
        inside_existing = False
        for parent in filtered:
            try:
                c.relative_to(parent)
                inside_existing = True
                break
            except ValueError:
                pass
        if not inside_existing:
            filtered.append(c)

    return filtered


def collect_skills_from_path(source: Path, work_dir: Path) -> List[Path]:
    """
    Collect installable skill directories from a file or folder.
    """
    source = source.expanduser().resolve()

    if not source.exists():
        raise InstallError(f"Source does not exist: {source}")

    collected: List[Path] = []

    if is_zip(source):
        extracted = extract_zip(source, work_dir)

        # First, support bundles that contain nested dist/*/skill.zip.
        nested_zips = list(iter_nested_skill_zips(extracted))
        if nested_zips:
            for nested_zip in nested_zips:
                nested_extracted = extract_zip(nested_zip, work_dir / "nested")
                collected.extend(deep_skill_dirs(nested_extracted))

        # Then, also support normal zip containing one or more skill directories.
        collected.extend(deep_skill_dirs(extracted))

    elif source.is_dir():
        # Support a single skill dir or a directory containing multiple skill dirs.
        collected.extend(direct_skill_dirs_under(source))

        # Support bundles containing dist/*/skill.zip.
        nested_zips = list(iter_nested_skill_zips(source))
        for nested_zip in nested_zips:
            nested_extracted = extract_zip(nested_zip, work_dir / "nested")
            collected.extend(deep_skill_dirs(nested_extracted))

        # If nothing found directly, do a deeper scan.
        if not collected:
            collected.extend(deep_skill_dirs(source))

    else:
        raise InstallError(f"Unsupported source type: {source}")

    # Deduplicate by resolved directory.
    unique: List[Path] = []
    seen = set()
    for skill_dir in collected:
        resolved = skill_dir.resolve()
        if resolved not in seen:
            seen.add(resolved)
            unique.append(skill_dir)

    if not unique:
        raise InstallError(f"No skill directories found in: {source}")

    return unique


def resolve_target_dir(scope: str, dest: Optional[str]) -> Path:
    if dest:
        return Path(dest).expanduser().resolve()

    if scope == "project":
        return (Path.cwd() / ".codex" / "skills").resolve()

    if scope == "user":
        return (Path.home() / ".codex" / "skills").resolve()

    raise InstallError(f"Unknown scope: {scope}")


def install_skills(
    sources: List[Path],
    target_dir: Path,
    force: bool,
    backup: bool,
    dry_run: bool,
) -> None:
    if dry_run:
        print(f"[dry-run] target directory: {target_dir}")
    else:
        target_dir.mkdir(parents=True, exist_ok=True)

    installed: List[Tuple[str, Path]] = []

    with tempfile.TemporaryDirectory(prefix="codex-skill-install-") as tmp:
        work_dir = Path(tmp)

        for source in sources:
            skill_dirs = collect_skills_from_path(source, work_dir)

            for skill_dir in skill_dirs:
                name = validate_skill_dir(skill_dir)
                dest = target_dir / name

                folder_name = skill_dir.name
                if folder_name != name:
                    print(
                        f"[warn] folder name '{folder_name}' differs from skill name '{name}'. "
                        f"Installing as '{name}'."
                    )

                safe_copytree(skill_dir, dest, force=force, backup=backup, dry_run=dry_run)
                installed.append((name, dest))

    print()
    if dry_run:
        print("Dry run complete. No files were changed.")
    else:
        print("Install complete.")

    print()
    print("Installed skills:")
    for name, dest in installed:
        print(f"- {name}: {dest}")

    print()
    print("Next check:")
    print(f"- Confirm your Codex environment is configured to read: {target_dir}")
    print("- Then try a prompt such as:")
    print('  codex exec "Use metrics-review to review specs/metrics.md"')


def parse_args(argv: List[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Install Codex-style skills from folders or zip files."
    )

    parser.add_argument(
        "sources",
        nargs="+",
        help="Skill folder, skill.zip, skills directory, or bundle zip.",
    )

    parser.add_argument(
        "--scope",
        choices=["project", "user"],
        default="project",
        help="Install target scope. project => ./.codex/skills, user => ~/.codex/skills. Default: project.",
    )

    parser.add_argument(
        "--dest",
        help="Custom destination directory. Overrides --scope.",
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

    args = parser.parse_args(argv)

    if args.force and args.backup:
        parser.error("Use only one of --force or --backup.")

    return args


def main(argv: List[str]) -> int:
    args = parse_args(argv)

    try:
        sources = [Path(s) for s in args.sources]
        target_dir = resolve_target_dir(args.scope, args.dest)

        install_skills(
            sources=sources,
            target_dir=target_dir,
            force=args.force,
            backup=args.backup,
            dry_run=args.dry_run,
        )

        return 0

    except InstallError as e:
        print(f"error: {e}", file=sys.stderr)
        return 1

    except KeyboardInterrupt:
        print("cancelled", file=sys.stderr)
        return 130


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))