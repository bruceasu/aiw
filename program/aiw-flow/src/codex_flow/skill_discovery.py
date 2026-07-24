from __future__ import annotations

import re
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, Iterable, List, Optional, Tuple


_FRONTMATTER_RE = re.compile(
    r"\A---[ \t]*\r?\n(.*?)\r?\n---[ \t]*(?:\r?\n|\Z)",
    re.DOTALL,
)
_NAME_RE = re.compile(r"^[a-z0-9][a-z0-9-]*$")


@dataclass(frozen=True)
class SkillInfo:
    name: str
    description: str
    scope: str
    source: Path


@dataclass(frozen=True)
class SkillIssue:
    source: Path
    message: str


@dataclass(frozen=True)
class SkillDiscovery:
    skills: Tuple[SkillInfo, ...]
    issues: Tuple[SkillIssue, ...]

    def by_name(self) -> Dict[str, Tuple[SkillInfo, ...]]:
        grouped: Dict[str, List[SkillInfo]] = {}
        for skill in self.skills:
            grouped.setdefault(skill.name, []).append(skill)
        return {name: tuple(items) for name, items in grouped.items()}


@dataclass(frozen=True)
class _SkillRoot:
    path: Path
    scope: str


class SkillMetadataError(ValueError):
    """A SKILL.md file does not contain supported metadata."""


def discover_skills(
    workspace: Path,
    *,
    codex_home: Optional[Path] = None,
    user_home: Optional[Path] = None,
) -> SkillDiscovery:
    resolved_workspace = workspace.resolve()
    if not resolved_workspace.is_dir():
        raise ValueError("Workspace does not exist: {}".format(resolved_workspace))

    skills: List[SkillInfo] = []
    issues: List[SkillIssue] = []
    for root in _skill_roots(
        resolved_workspace,
        codex_home=codex_home,
        user_home=user_home,
    ):
        if not root.path.is_dir():
            continue
        try:
            candidates = sorted(root.path.iterdir(), key=lambda path: path.name.lower())
        except OSError as exc:
            issues.append(
                SkillIssue(
                    source=root.path,
                    message="Unable to read Skill directory: {}".format(type(exc).__name__),
                )
            )
            continue

        for candidate in candidates:
            try:
                if not candidate.is_dir():
                    continue
                skill_md = candidate / "SKILL.md"
                if not skill_md.is_file():
                    continue
                name, description = read_skill_metadata(skill_md)
                skills.append(
                    SkillInfo(
                        name=name,
                        description=description,
                        scope=root.scope,
                        source=candidate.resolve(),
                    )
                )
            except (OSError, UnicodeError, SkillMetadataError) as exc:
                issues.append(SkillIssue(source=candidate, message=str(exc)))

    return SkillDiscovery(skills=tuple(skills), issues=tuple(issues))


def read_skill_metadata(skill_md: Path) -> Tuple[str, str]:
    text = skill_md.read_text(encoding="utf-8").lstrip("\ufeff")
    match = _FRONTMATTER_RE.search(text)
    if not match:
        raise SkillMetadataError("Missing YAML frontmatter in {}".format(skill_md))

    fields = _parse_frontmatter_fields(match.group(1))
    name = _parse_scalar(fields.get("name"))
    if not name or not _NAME_RE.fullmatch(name):
        raise SkillMetadataError(
            "Missing or invalid frontmatter name in {}".format(skill_md)
        )
    description = _parse_scalar(fields.get("description"))
    if not description:
        raise SkillMetadataError(
            "Missing or invalid frontmatter description in {}".format(skill_md)
        )
    return name, description


def _parse_frontmatter_fields(frontmatter: str) -> Dict[str, str]:
    fields: Dict[str, str] = {}
    for raw_line in frontmatter.splitlines():
        if not raw_line or raw_line[0].isspace() or ":" not in raw_line:
            continue
        key, value = raw_line.split(":", 1)
        normalized_key = key.strip()
        if normalized_key in {"name", "description"}:
            fields[normalized_key] = value.strip()
    return fields


def _parse_scalar(raw: Optional[str]) -> Optional[str]:
    if raw is None:
        return None
    value = raw.strip()
    if len(value) >= 2 and value[0] == value[-1] and value[0] in {"'", '"'}:
        value = value[1:-1].strip()
    if not value or value in {"|", ">"}:
        return None
    return value


def _skill_roots(
    workspace: Path,
    *,
    codex_home: Optional[Path],
    user_home: Optional[Path],
) -> Iterable[_SkillRoot]:
    project_root = _find_project_root(workspace)
    current = workspace
    while True:
        yield _SkillRoot(current / ".agents" / "skills", "project")
        if current == project_root:
            break
        current = current.parent

    yield _SkillRoot(project_root / ".codex" / "skills", "project")

    effective_user_home = (user_home or Path.home()).expanduser().resolve()
    yield _SkillRoot(effective_user_home / ".agents" / "skills", "user")
    effective_codex_home = (
        codex_home.expanduser().resolve()
        if codex_home is not None
        else effective_user_home / ".codex"
    )
    yield _SkillRoot(effective_codex_home / "skills", "user")


def _find_project_root(workspace: Path) -> Path:
    current = workspace
    while True:
        if (current / ".git").exists():
            return current
        if current.parent == current:
            return workspace
        current = current.parent
