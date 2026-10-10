"""Terminal commit wizard and local commit flow."""

from __future__ import annotations

import os
import subprocess
import tempfile
from pathlib import Path

from cz_config import Config
from cz_core import Draft


def _ask(prompt: str, initial: str = "") -> str:
    suffix = f" [{initial}]" if initial else ""
    value = input(f"{prompt}{suffix}: ").strip()
    return value or initial


def tui(config: Config, existing: Draft | None = None) -> Draft:
    print(config.messages.get("type", "Select commit type"))
    for index, item in enumerate(config.types, 1):
        print(f"  {index}) {item.value} {item.name}")
    selected = _ask("Type", existing.type if existing else "1")
    type_value = next((item.value for index, item in enumerate(config.types, 1) if str(index) == selected), selected)
    scope = _ask(config.messages.get("scope", "Scope (optional)"), existing.scope if existing else "")
    subject = _ask(config.messages.get("subject", "Subject"), existing.subject if existing else "")
    while not subject or len(subject) > config.max_subject_length:
        subject = _ask(f"Subject (max {config.max_subject_length})", subject)
    body = _ask(config.messages.get("body", "Body (optional)"), existing.body if existing else "")
    breaking = _ask(config.messages.get("breaking", "Breaking changes (optional)"), existing.breaking if existing else "")
    footer = _ask(config.messages.get("footer", "Related issue (optional)"), existing.footer if existing else "")
    return Draft(type_value, scope, subject.replace("\n", " "), body, breaking, footer)


def wizard(config: Config, existing: Draft | None = None) -> Draft:
    return tui(config, existing)


def choose_candidate(config: Config, drafts: list[Draft]) -> Draft:
    if len(drafts) == 1:
        return drafts[0]
    print(config.messages.get("candidate", "Select AI candidate"))
    for index, draft in enumerate(drafts, 1):
        print(f"  {index}) {draft.type}{f'({draft.scope})' if draft.scope else ''}: {draft.subject}")
    while True:
        answer = input("Candidate number: ").strip()
        if answer.isdigit() and 1 <= int(answer) <= len(drafts):
            return drafts[int(answer) - 1]


def _message(draft: Draft) -> str:
    header = f"{draft.type}{f'({draft.scope})' if draft.scope else ''}: {draft.subject}"
    blocks = [header]
    if draft.body:
        blocks.append(draft.body)
    if draft.breaking:
        blocks.append(f"BREAKING CHANGE: {draft.breaking}")
    if draft.footer:
        blocks.append(draft.footer)
    return "\n\n".join(blocks) + "\n"


def review_and_commit(root: Path, config: Config, draft: Draft) -> None:
    while True:
        print("\n--- Commit message preview ---\n" + _message(draft) + "------------------------------")
        answer = input(config.messages.get("confirm_commit", "Commit this message") + " [Y/n/e(edit)]: ").strip().lower()
        if answer in ("e", "edit"):
            draft = wizard(config, draft)
            continue
        if answer in ("n", "no"):
            return
        if answer in ("", "y", "yes"):
            break
        print("Enter y, n, or e.")
    with tempfile.NamedTemporaryFile("w", encoding="utf-8", suffix=".txt", delete=False) as handle:
        handle.write(_message(draft))
        message_path = handle.name
    try:
        subprocess.run(["git", "commit", "-F", message_path], cwd=root, check=True)
    finally:
        try:
            os.unlink(message_path)
        except OSError:
            pass
