#!/usr/bin/env python3
"""Manage canonical AIW Skills and managed Skill imports."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import stat
import subprocess
import sys
import tempfile
import zipfile
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, Iterable, List, Optional, Sequence, Tuple, cast


FRONTMATTER_RE = re.compile(
    r"\A---[ \t]*\r?\n(.*?)\r?\n---[ \t]*(?:\r?\n|\Z)",
    re.DOTALL,
)
NAME_RE = re.compile(r"^[a-z0-9][a-z0-9-]*$")


@dataclass(frozen=True)
class Skill:
    name: str
    description: str
    source: Path


class SkillError(ValueError):
    """Canonical Skill metadata or catalog state is invalid."""


def configure_utf8_stdio() -> None:
    for stream in (sys.stdout, sys.stderr):
        reconfigure = getattr(stream, "reconfigure", None)
        if reconfigure is not None:
            reconfigure(encoding="utf-8")


def source_root() -> Path:
    configured = os.environ.get("AIW_SKILLS_SOURCE_ROOT")
    if configured and configured.strip():
        return Path(configured).expanduser().resolve()
    return Path(__file__).resolve().parents[2] / "skills"


def find_skill_md(path: Path) -> Optional[Path]:
    skill_md = path / "SKILL.md"
    if skill_md.is_file():
        return skill_md
    skill_md_lower = path / "skill.md"
    if skill_md_lower.is_file():
        return skill_md_lower
    return None


def read_skill(skill_dir: Path) -> Skill:
    skill_md = skill_dir / "SKILL.md"
    text = skill_md.read_text(encoding="utf-8").lstrip("\ufeff")
    match = FRONTMATTER_RE.search(text)
    if not match:
        raise SkillError("Missing YAML frontmatter")

    fields = {}
    for raw_line in match.group(1).splitlines():
        if not raw_line or raw_line[0].isspace() or ":" not in raw_line:
            continue
        key, value = raw_line.split(":", 1)
        key = key.strip()
        if key in {"name", "description"}:
            fields[key] = value.strip()

    name = parse_scalar(fields.get("name"))
    if not name or not NAME_RE.fullmatch(name):
        raise SkillError("Missing or invalid frontmatter name")
    description = parse_scalar(fields.get("description"))
    if not description:
        raise SkillError("Missing or invalid frontmatter description")
    return Skill(name=name, description=description, source=skill_dir.resolve())


def parse_scalar(raw: Optional[str]) -> Optional[str]:
    if raw is None:
        return None
    value = raw.strip()
    if not value:
        return None
    if value[0] in {"'", '"'}:
        if len(value) < 2 or value[-1] != value[0]:
            return None
        delimiter = value[0]
        value = value[1:-1].strip()
        if delimiter in value:
            return None
    elif value[-1] in {"'", '"'}:
        return None
    else:
        if value[0] in "-?:,[]{}#&*!|>'\"%@`":
            return None
        if ": " in value or " #" in value:
            return None
    if not value:
        return None
    return value


def discover_skills(root: Path) -> Tuple[List[Skill], List[str]]:
    if not root.is_dir():
        raise SkillError("Canonical Skill source does not exist: {}".format(root))

    skills: List[Skill] = []
    issues: List[str] = []
    for candidate in sorted(root.iterdir(), key=lambda path: path.name.lower()):
        if candidate.is_symlink():
            issues.append("{}: Unsupported symlink".format(candidate))
            continue
        if not candidate.is_dir() or not (candidate / "SKILL.md").is_file():
            continue
        try:
            skill = read_skill(candidate)
            if skill.name != candidate.name:
                raise SkillError(
                    "Declared name `{}` does not match folder `{}`".format(
                        skill.name,
                        candidate.name,
                    )
                )
            skills.append(skill)
        except (OSError, UnicodeError, SkillError) as exc:
            issues.append("{}: {}".format(candidate, exc))
    skills.sort(key=lambda skill: skill.name)
    return skills, issues


def validate_skill_dir(skill_dir: Path) -> Skill:
    if not skill_dir.is_dir():
        raise SkillError("Not a directory: {}".format(skill_dir))
    skill = read_skill(skill_dir)
    if skill.name != skill_dir.name:
        raise SkillError(
            "Declared name `{}` does not match folder `{}`".format(
                skill.name,
                skill_dir.name,
            )
        )
    validate_copyable_tree(skill_dir)
    return skill


def validate_copyable_tree(root: Path) -> None:
    pending = [root]
    while pending:
        current = pending.pop()
        with os.scandir(str(current)) as entries:
            for entry in entries:
                path = Path(entry.path)
                if entry.is_symlink():
                    raise SkillError("Unsupported symlink in Skill: {}".format(path))
                entry_stat = entry.stat(follow_symlinks=False)
                if stat.S_ISDIR(entry_stat.st_mode):
                    pending.append(path)
                elif not stat.S_ISREG(entry_stat.st_mode):
                    raise SkillError(
                        "Unsupported filesystem entry in Skill: {}".format(path)
                    )


def is_zip(path: Path) -> bool:
    return path.is_file() and path.suffix.lower() == ".zip"


def extract_zip(zip_path: Path, work_dir: Path) -> Path:
    if not zipfile.is_zipfile(zip_path):
        raise SkillError("Invalid zip file: {}".format(zip_path))
    out_dir = work_dir / zip_path.stem
    out_dir.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(zip_path, "r") as zf:
        zf.extractall(out_dir)
    return out_dir


def iter_nested_skill_zips(path: Path) -> Iterable[Path]:
    if not path.is_dir():
        return []
    return path.rglob("skill.zip")


def direct_skill_dirs_under(path: Path) -> List[Path]:
    result: List[Path] = []
    if path.is_dir() and find_skill_md(path):
        result.append(path)
        return result
    if path.is_dir():
        for child in sorted(path.iterdir(), key=lambda entry: entry.name.lower()):
            if child.is_dir() and find_skill_md(child):
                result.append(child)
    return result


def deep_skill_dirs(path: Path) -> List[Path]:
    if path.is_dir() and find_skill_md(path):
        return [path]

    candidates: List[Path] = []
    for skill_md in path.rglob("SKILL.md"):
        candidates.append(skill_md.parent)
    for skill_md in path.rglob("skill.md"):
        candidates.append(skill_md.parent)

    unique: List[Path] = []
    seen = set()
    for candidate in sorted(candidates, key=lambda item: item.as_posix().lower()):
        resolved = candidate.resolve()
        if resolved in seen:
            continue
        seen.add(resolved)
        inside_existing = False
        for parent in unique:
            try:
                candidate.relative_to(parent)
                inside_existing = True
                break
            except ValueError:
                pass
        if not inside_existing:
            unique.append(candidate)
    return unique


def collect_skill_dirs_from_path(source: Path, work_dir: Path) -> List[Path]:
    source = source.expanduser().resolve()
    if not source.exists():
        raise SkillError("Source does not exist: {}".format(source))

    collected: List[Path] = []
    if is_zip(source):
        extracted = extract_zip(source, work_dir)
        nested_zips = list(iter_nested_skill_zips(extracted))
        for nested_zip in nested_zips:
            nested_extracted = extract_zip(nested_zip, work_dir / "nested")
            collected.extend(deep_skill_dirs(nested_extracted))
        collected.extend(deep_skill_dirs(extracted))
    elif source.is_dir():
        collected.extend(direct_skill_dirs_under(source))
        nested_zips = list(iter_nested_skill_zips(source))
        for nested_zip in nested_zips:
            nested_extracted = extract_zip(nested_zip, work_dir / "nested")
            collected.extend(deep_skill_dirs(nested_extracted))
        if not collected:
            collected.extend(deep_skill_dirs(source))
    else:
        raise SkillError("Unsupported source type: {}".format(source))

    unique: List[Path] = []
    seen = set()
    for skill_dir in collected:
        resolved = skill_dir.resolve()
        if resolved not in seen:
            seen.add(resolved)
            unique.append(skill_dir)

    if not unique:
        raise SkillError("No skill directories found in: {}".format(source))
    return unique


def directory_digest(root: Path) -> str:
    digest = hashlib.sha256()
    files = sorted(
        (path for path in root.rglob("*") if path.is_file()),
        key=lambda path: path.relative_to(root).as_posix(),
    )
    for path in files:
        relative = path.relative_to(root).as_posix().encode("utf-8")
        digest.update(len(relative).to_bytes(8, byteorder="big"))
        digest.update(relative)
        with path.open("rb") as source:
            while True:
                chunk = source.read(1024 * 1024)
                if not chunk:
                    break
                digest.update(chunk)
    return digest.hexdigest()


def source_revision(source: Path) -> Optional[str]:
    try:
        completed = subprocess.run(
            ["git", "-C", str(source), "rev-parse", "HEAD"],
            capture_output=True,
            text=True,
            encoding="utf-8",
            timeout=5,
        )
    except (OSError, subprocess.SubprocessError):
        return None
    if completed.returncode != 0:
        return None
    revision = completed.stdout.strip()
    return revision or None


def write_manifest(path: Path, manifest: object) -> None:
    descriptor, temp_name = tempfile.mkstemp(
        prefix=".aiw-manifest-",
        dir=str(path.parent),
        text=True,
    )
    temp_path = Path(temp_name)
    try:
        with os.fdopen(descriptor, "w", encoding="utf-8", newline="\n") as output:
            json.dump(
                manifest,
                output,
                ensure_ascii=False,
                indent=2,
                sort_keys=True,
            )
            output.write("\n")
            output.flush()
            os.fsync(output.fileno())
        os.replace(str(temp_path), str(path))
    finally:
        try:
            temp_path.unlink()
        except FileNotFoundError:
            pass


def load_manifest(path: Path) -> Dict[str, object]:
    if not os.path.lexists(str(path)):
        return {"schema_version": 1, "skills": {}}
    if path.is_symlink():
        raise SkillError("Managed manifest must not be a symlink: {}".format(path))
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        raise SkillError("Invalid managed manifest: {}".format(exc))
    if (
        not isinstance(data, dict)
        or data.get("schema_version") != 1
        or not isinstance(data.get("skills"), dict)
    ):
        raise SkillError("Invalid managed manifest schema: {}".format(path))
    return cast(Dict[str, object], data)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="aiw skills",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        description=(
            "Manage canonical AIW Skills.\n\n"
            "Portable Skills install to ./.agents/skills by default. "
            "This first release installs one Skill at a time."
        ),
        epilog=(
            "Quick start:\n"
            "  aiw skills list\n"
            "  aiw skills install tdd --dry-run\n"
            "  aiw skills install tdd\n\n"
            "Run `aiw skills COMMAND --help` for command details."
        ),
    )
    commands = parser.add_subparsers(dest="command", required=True)

    list_command = commands.add_parser(
        "list",
        help="List canonical Portable Skills without changing the project.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        description=(
            "List canonical Portable Skills without changing the project.\n\n"
            "Use this when you want to check which Skills AIW can install."
        ),
        epilog=(
            "Examples:\n"
            "  aiw skills list\n"
            "  aiw skills list --json"
        ),
    )
    list_command.add_argument(
        "--json",
        action="store_true",
        help="Write one machine-readable JSON result.",
    )

    install = commands.add_parser(
        "install",
        help="Safely install one or more Skills into ./.agents/skills.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        description=(
            "Safely install canonical Skills or path-based Skill bundles.\n\n"
            "Use this when a project needs one AIW-maintained Skill or a local "
            "source bundle. The default target is ./.agents/skills. A same-name "
            "unmanaged directory is protected and will not be replaced."
        ),
        epilog=(
            "Examples:\n"
            "  aiw skills install tdd --dry-run\n"
            "  aiw skills install ./bundle.zip\n"
            "  aiw skills install ./skills\n"
            "  aiw skills install tdd --json"
        ),
    )
    install.add_argument("source", help="Canonical Skill name or local source path.")
    install.add_argument(
        "--dry-run",
        action="store_true",
        help="Show the source and target without changing the project.",
    )
    install.add_argument(
        "--json",
        action="store_true",
        help="Write one machine-readable JSON result.",
    )
    return parser


def print_json(payload: object) -> None:
    print(json.dumps(payload, ensure_ascii=False, sort_keys=True))


def install_skills_from_sources(
    skill_dirs: List[Path],
    *,
    dry_run: bool,
    json_output: bool,
    source_label: str,
) -> int:
    destination_root = Path.cwd() / ".agents" / "skills"
    manifest_path = destination_root / ".aiw-skills.json"
    manifest = load_manifest(manifest_path)
    managed_skills = cast(Dict[str, object], manifest["skills"])

    skill_entries = [validate_skill_dir(skill_dir) for skill_dir in skill_dirs]
    destinations = [destination_root / skill.name for skill in skill_entries]

    if dry_run:
        if json_output:
            print_json(
                {
                    "action": "install",
                    "destination": str(destinations[0]) if len(destinations) == 1 else str(destination_root),
                    "name": skill_entries[0].name if len(skill_entries) == 1 else source_label,
                    "ok": True,
                    "source": source_label,
                    "status": "would_install",
                }
            )
        else:
            if len(skill_entries) == 1:
                print(
                    "Would install {} from {} to {}".format(
                        skill_entries[0].name,
                        source_label,
                        destinations[0],
                    )
                )
            else:
                print(
                    "Would install {} from {} to {}".format(
                        ", ".join(skill.name for skill in skill_entries),
                        source_label,
                        destination_root,
                    )
                )
        return 0

    installed: List[str] = []
    already_installed: List[str] = []
    last_digest: Optional[str] = None

    for skill, destination in zip(skill_entries, destinations):
        if os.path.lexists(str(destination)):
            managed = managed_skills.get(skill.name)
            if not isinstance(managed, dict):
                raise SkillError(
                    "Target is unmanaged and will not be replaced: {}".format(
                        destination
                    )
                )
            validate_copyable_tree(destination)
            source_digest = directory_digest(skill.source)
            installed_digest = directory_digest(destination)
            if (
                managed.get("mode") == "copy"
                and managed.get("sha256") == source_digest == installed_digest
            ):
                already_installed.append(skill.name)
                last_digest = source_digest
                continue
            raise SkillError(
                "Installed Skill differs from its managed record: {}".format(
                    destination
                )
            )

        if skill.name in managed_skills:
            raise SkillError(
                "Managed manifest entry has no installed Skill: {}".format(
                    skill.name
                )
            )

    destination_root.mkdir(parents=True, exist_ok=True)

    with tempfile.TemporaryDirectory(prefix=".aiw-install-") as tmp:
        tmp_root = Path(tmp)
        for skill, destination in zip(skill_entries, destinations):
            if skill.name in already_installed and destination.exists():
                continue
            source_digest_before = directory_digest(skill.source)
            stage_root = Path(tempfile.mkdtemp(prefix=".aiw-stage-", dir=str(destination_root)))
            staged_skill = stage_root / skill.name
            try:
                shutil.copytree(str(skill.source), str(staged_skill), symlinks=True)
                validate_copyable_tree(staged_skill)
                staged_digest = directory_digest(staged_skill)
                source_digest_after = directory_digest(skill.source)
                if not (
                    source_digest_before == staged_digest == source_digest_after
                ):
                    raise SkillError("Canonical Skill changed during installation.")
                if os.path.lexists(str(destination)):
                    raise SkillError("Target already exists: {}".format(destination))
                os.replace(str(staged_skill), str(destination))
                last_digest = staged_digest
            finally:
                shutil.rmtree(str(stage_root), ignore_errors=True)

            managed_skills[skill.name] = {
                "mode": "copy",
                "sha256": last_digest,
                "source_identity": str(skill.source),
                "source_revision": source_revision(skill.source),
            }
            installed.append(skill.name)

    if installed:
        try:
            write_manifest(manifest_path, manifest)
        except (OSError, TypeError, ValueError) as exc:
            for skill_name in installed:
                shutil.rmtree(str(destination_root / skill_name), ignore_errors=True)
            raise SkillError("Unable to write managed manifest: {}".format(exc))

    if json_output:
        if installed:
            payload = {
                "action": "install",
                "destination": str(destinations[0]) if len(installed) == 1 else str(destination_root),
                "name": installed[0] if len(installed) == 1 else source_label,
                "ok": True,
                "sha256": last_digest,
                "status": "installed",
            }
        else:
            payload = {
                "action": "install",
                "destination": str(destinations[0]),
                "name": already_installed[0],
                "ok": True,
                "sha256": last_digest,
                "status": "already_installed",
            }
        print_json(payload)
    else:
        if installed:
            if len(installed) == 1:
                print(
                    "Installed {} to {} (sha256: {})".format(
                        installed[0],
                        destinations[0],
                        last_digest,
                    )
                )
            else:
                print(
                    "Installed {} skills to {}".format(
                        len(installed),
                        destination_root,
                    )
                )
        else:
            print(
                "Already installed {} at {}".format(
                    already_installed[0],
                    destinations[0],
                )
            )
    return 0

def command_install(source_text: str, *, dry_run: bool, json_output: bool) -> int:
    source = Path(source_text)
    if source.exists():
        with tempfile.TemporaryDirectory(prefix=".aiw-install-") as tmp:
            skill_dirs = collect_skill_dirs_from_path(source, Path(tmp))
            return install_skills_from_sources(
                skill_dirs,
                dry_run=dry_run,
                json_output=json_output,
                source_label=str(source.resolve()),
            )

    if not NAME_RE.fullmatch(source_text):
        raise SkillError("Invalid canonical Skill name: {}".format(source_text))
    root = source_root()
    candidate = root / source_text
    if candidate.is_symlink():
        raise SkillError("Unsupported symlink in Skill: {}".format(candidate))
    if not candidate.is_dir() or not (candidate / "SKILL.md").is_file():
        raise SkillError("Canonical Skill not found: {}".format(source_text))

    return install_skills_from_sources(
        [candidate],
        dry_run=dry_run,
        json_output=json_output,
        source_label=str(candidate),
    )


def command_list(*, json_output: bool) -> int:
    skills, issues = discover_skills(source_root())
    if json_output:
        print_json(
            {
                "action": "list",
                "issues": issues,
                "ok": True,
                "skills": [
                    {
                        "description": skill.description,
                        "name": skill.name,
                        "source": str(skill.source),
                    }
                    for skill in skills
                ],
            }
        )
        return 0
    print("Canonical Skills:")
    for skill in skills:
        print("  {} - {}".format(skill.name, skill.description))
    for issue in issues:
        print("Warning: {}".format(issue), file=sys.stderr)
    return 0


def main(argv: Sequence[str]) -> int:
    args = build_parser().parse_args(argv)
    try:
        if args.command == "list":
            return command_list(json_output=args.json)
        if args.command == "install":
            return command_install(
                args.source,
                dry_run=args.dry_run,
                json_output=args.json,
            )
        raise SkillError("Unknown command: {}".format(args.command))
    except (OSError, UnicodeError, SkillError) as exc:
        if getattr(args, "json", False):
            print_json(
                {
                    "action": args.command,
                    "error": str(exc),
                    "ok": False,
                }
            )
        else:
            print("error: {}".format(exc), file=sys.stderr)
        return 1


if __name__ == "__main__":
    configure_utf8_stdio()
    raise SystemExit(main(sys.argv[1:]))

