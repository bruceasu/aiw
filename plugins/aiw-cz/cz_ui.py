"""Interactive GUI/TUI commit wizard with graceful tkinter fallback."""

from __future__ import annotations

import os
import subprocess
import sys
import tempfile
from pathlib import Path

from cz_config import Choice, Config
from cz_core import Draft


def gui_available() -> bool:
    if not sys.stdin.isatty() or not sys.stdout.isatty():
        return False
    if os.name != "nt" and not os.environ.get("DISPLAY") and not os.environ.get("WAYLAND_DISPLAY"):
        return False
    try:
        import tkinter  # noqa: F401
        return True
    except (ImportError, OSError):
        return False


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


def gui(config: Config, existing: Draft | None = None) -> Draft:
    import tkinter as tk
    from tkinter import messagebox

    result: list[Draft] = []
    root = tk.Tk()
    root.title("aiw cz")
    fields: dict[str, tk.Entry] = {}
    labels = (("scope", "Scope"), ("subject", "Subject"), ("body", "Body"), ("breaking", "Breaking"), ("footer", "Footer"))
    tk.Label(root, text="Type").grid(row=0, column=0, sticky="w")
    type_var = tk.StringVar(value=existing.type if existing else config.types[0].value)
    tk.OptionMenu(root, type_var, *(item.value for item in config.types)).grid(row=0, column=1, sticky="ew")
    for row, (key, label) in enumerate(labels, 1):
        tk.Label(root, text=label).grid(row=row, column=0, sticky="w")
        entry = tk.Entry(root, width=72)
        entry.insert(0, getattr(existing, key, "") if existing else "")
        entry.grid(row=row, column=1, sticky="ew")
        fields[key] = entry

    def accept() -> None:
        subject = fields["subject"].get().strip()
        if not subject or len(subject) > config.max_subject_length:
            messagebox.showerror("aiw cz", f"Subject must be 1-{config.max_subject_length} characters")
            return
        result.append(Draft(type_var.get(), *(fields[key].get().strip() for key, _ in labels)))
        root.destroy()

    tk.Button(root, text="Commit", command=accept).grid(row=len(labels) + 1, column=1, sticky="e")
    root.columnconfigure(1, weight=1)
    root.mainloop()
    if not result:
        raise KeyboardInterrupt
    return result[0]


def wizard(config: Config, existing: Draft | None = None) -> Draft:
    if gui_available():
        try:
            import tkinter
        except (ImportError, OSError):
            return tui(config, existing)
        try:
            return gui(config, existing)
        except (OSError, tkinter.TclError):
            return tui(config, existing)
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


def review_and_commit(root: Path, draft: Draft) -> None:
    print("\n--- Commit message preview ---\n" + _message(draft) + "------------------------------")
    answer = input("Commit this message? [Y/n]: ").strip().lower()
    if answer not in ("", "y", "yes"):
        return
    branch = subprocess.run(
        ["git", "branch", "--show-current"], cwd=root, capture_output=True,
        text=True, encoding="utf-8", errors="replace", check=True,
    ).stdout.strip().lower()
    protected = {"main", "master", "develop"}
    if branch in protected:
        print(f"aiw-cz: local commit allowed on protected branch '{branch}'; push is not performed")
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
