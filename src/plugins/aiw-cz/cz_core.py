"""Git context, prompt construction, and candidate validation for aiw-cz."""

from __future__ import annotations

import json
import re
import subprocess
from dataclasses import dataclass
from pathlib import Path

from cz_config import Config


@dataclass
class Draft:
    type: str
    scope: str = ""
    subject: str = ""
    body: str = ""
    breaking: str = ""
    footer: str = ""


def _git(root: Path, *args: str) -> str:
    result = subprocess.run(
        ["git", *args], cwd=root, capture_output=True, text=True,
        encoding="utf-8", errors="replace", check=True,
    )
    return result.stdout.strip()


def staged_context(root: Path) -> tuple[str, str]:
    files = _git(root, "diff", "--cached", "--name-only")
    diff = _git(root, "diff", "--cached", "--")
    try:
        history = _git(root, "log", "--oneline", "-n", "5")
    except subprocess.CalledProcessError:
        history = ""
    return f"{files}\n{diff}\n{history}", (files, diff, history)


def has_staged_changes(root: Path) -> bool:
    return bool(_git(root, "diff", "--cached", "--name-only"))


def previous_draft(root: Path) -> Draft:
    text = _git(root, "log", "-1", "--pretty=%B")
    first, *rest = text.splitlines()
    match = re.match(r"^([a-z][a-z0-9-]*)(?:\(([^)]+)\))?!?:\s*(.+)$", first)
    if not match:
        return Draft("chore", subject=first, body="\n".join(rest).strip())
    return Draft(match.group(1), match.group(2) or "", match.group(3), "\n".join(rest).strip())


def build_prompt(config: Config, root: Path) -> tuple[str, str]:
    context, parts = staged_context(root)
    files, _, history = parts
    diff_preview = context.split("\n", 1)[1][:1000]
    prompt = (
        f"Generate {config.candidates} Conventional Commit candidates. Return only JSON: "
        '{"candidates":[{"type":"feat","scope":"","subject":"","body":"",'
        '"breaking":"","footer":""}]}. All six fields are strings. '
        f"Allowed types: {', '.join(item.value for item in config.types)}. "
        f"Keep subjects concise and imperative, max {config.max_subject_length} characters. "
        "Use the language of changed code and comments. Do not invent issue references. "
        "Do not execute tools or edit files.\n\n"
        f"Changed files:\n{files}\nRecent commits:\n{history}\n"
        f"Staged diff:\n{diff_preview}"
    )
    return prompt, context


def parse_candidates(raw: str, config: Config, context: str) -> list[Draft]:
    match = re.search(r"\{[\s\S]*\}", raw)
    if not match:
        raise ValueError("LLM response does not contain a JSON object")
    try:
        parsed = json.loads(match.group(0))
    except json.JSONDecodeError as exc:
        raise ValueError(f"invalid candidate JSON: {exc.msg}") from exc
    rows = parsed.get("candidates") if isinstance(parsed, dict) else None
    if not isinstance(rows, list):
        raise ValueError("candidate JSON must contain a candidates array")
    allowed = {item.value for item in config.types}
    issue_refs = set(re.findall(r"#\d+", context))
    result: list[Draft] = []
    for item in rows:
        if not isinstance(item, dict):
            continue
        values = {key: item.get(key, "") for key in ("type", "scope", "subject", "body", "breaking", "footer")}
        if not all(isinstance(value, str) for value in values.values()):
            continue
        if values["type"].strip() not in allowed or not values["subject"].strip():
            continue
        footer_refs = set(re.findall(r"#\d+", values["footer"]))
        if not footer_refs.issubset(issue_refs):
            continue
        subject = re.sub(r"[\r\n]+", " ", values["subject"].replace("\x00", "").strip())
        if len(subject) > config.max_subject_length:
            continue
        result.append(Draft(
            type=values["type"].strip(),
            scope=re.sub(r"[\r\n()]", "", values["scope"].replace("\x00", "").strip()),
            subject=subject,
            body=values["body"].replace("\x00", "").strip(),
            breaking=values["breaking"].replace("\x00", "").strip(),
            footer=values["footer"].replace("\x00", "").strip(),
        ))
    if not result:
        raise ValueError("no valid commit candidates")
    return result[: config.candidates]
