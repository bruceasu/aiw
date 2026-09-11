#!/usr/bin/env python3
"""Configure the AIW project documents for aiw init."""

from __future__ import annotations

import argparse
import os
import sys
from pathlib import Path


COMPLETION_BEGIN = "# >>> aiw completion >>>"
COMPLETION_END = "# <<< aiw completion <<<"


AGENT_SKILLS_BLOCK = """## Agent skills

- Read `docs/agents/work-management.md` for lifecycle ownership.
- Read `docs/agents/domain.md` for project-domain conventions.
"""

WORK_MANAGEMENT_FALLBACK = """# Work Management

AIW owns Task lifecycle, branch, worktree, Session, and handoff state.
OpenSpec owns proposal, design, capability specifications, and `tasks.md`.
External Issues are optional projections used only on explicit request.
"""

DOMAIN_CONTENT = """# Domain

Record project-specific domain terminology and conventions here.
"""


def is_interactive() -> bool:
    return sys.stdin.isatty() and sys.stdout.isatty()


def select_action(path: Path) -> str:
    while True:
        answer = input(f"{path}: [m]erge, [r]eplace, or [s]kip? ").strip().lower()
        if answer in {"m", "merge"}:
            return "merge"
        if answer in {"r", "replace"}:
            return "replace"
        if answer in {"s", "skip"}:
            return "skip"
        print("Choose merge, replace, or skip.", file=sys.stderr)


def write_target(path: Path, content: str, *, base_agents_created: bool) -> None:
    if not path.exists():
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8", newline="\n")
        print(f"created {path}")
        return
    if base_agents_created and path.name == "AGENTS.md":
        action = "merge"
    elif not is_interactive():
        print(f"skipped existing {path} (non-interactive)")
        return
    else:
        action = select_action(path)
    if action == "skip":
        print(f"skipped existing {path}")
        return
    if action == "replace":
        path.write_text(content, encoding="utf-8", newline="\n")
        print(f"replaced {path}")
        return
    existing = path.read_text(encoding="utf-8")
    if content.strip() in existing:
        print(f"unchanged {path}")
        return
    path.write_text(existing.rstrip() + "\n\n" + content, encoding="utf-8", newline="\n")
    print(f"merged {path}")


def source_work_management() -> str:
    source = Path(__file__).resolve().parents[1] / "skills" / "work-management.md"
    if source.is_file():
        return source.read_text(encoding="utf-8")
    return WORK_MANAGEMENT_FALLBACK


def detect_shell() -> str | None:
    explicit = os.environ.get("AIW_SHELL", "").strip().lower()
    if explicit in {"powershell", "bash", "zsh", "fish"}:
        return explicit
    shell = Path(os.environ.get("SHELL", "")).name.lower()
    if shell in {"bash", "zsh", "fish"}:
        return shell
    if os.name == "nt" and os.environ.get("PSModulePath"):
        return "powershell"
    return None


def completion_target(shell: str) -> Path:
    home = Path.home()
    if shell == "powershell":
        modules = os.environ.get("PSModulePath", "").lower()
        directory = "WindowsPowerShell" if "windowspowershell" in modules else "PowerShell"
        return home / "Documents" / directory / "Microsoft.PowerShell_profile.ps1"
    if shell == "bash":
        return home / ".bashrc"
    if shell == "zsh":
        return home / ".zshrc"
    if shell == "fish":
        return home / ".config" / "fish" / "conf.d" / "aiw-completion.fish"
    raise ValueError(f"unsupported shell: {shell}")


def completion_command(shell: str) -> str:
    commands = {
        "powershell": "aiw completion powershell | Out-String | Invoke-Expression",
        "bash": "source <(aiw completion bash)",
        "zsh": 'eval "$(aiw completion zsh)"',
        "fish": "aiw completion fish | source",
    }
    return commands[shell]


def install_completion(shell: str) -> tuple[Path, bool]:
    target = completion_target(shell)
    existing = target.read_text(encoding="utf-8") if target.exists() else ""
    has_begin = COMPLETION_BEGIN in existing
    has_end = COMPLETION_END in existing
    if has_begin and has_end:
        return target, False
    if has_begin or has_end:
        raise RuntimeError(f"incomplete AIW completion block in {target}; repair it before rerunning setup-project")
    target.parent.mkdir(parents=True, exist_ok=True)
    separator = "" if not existing or existing.endswith("\n") else "\n"
    block = f"{COMPLETION_BEGIN}\n{completion_command(shell)}\n{COMPLETION_END}\n"
    target.write_text(existing + separator + block, encoding="utf-8", newline="\n")
    return target, True


def configure_completion() -> None:
    shell = detect_shell()
    if shell is None:
        print("shell completion skipped: unable to identify the current shell; set AIW_SHELL to powershell, bash, zsh, or fish.")
        return
    try:
        target, changed = install_completion(shell)
    except (OSError, RuntimeError) as err:
        print(f"shell completion not installed: {err}", file=sys.stderr)
        return
    if changed:
        print(f"installed {shell} completion in {target}")
    else:
        print(f"{shell} completion already configured in {target}")
    print(f"enable completion in the current shell: {completion_command(shell)}")


def main() -> int:
    parser = argparse.ArgumentParser(description="Configure AIW engineering-skill documents.")
    parser.add_argument("--base-agents-created", action="store_true")
    args = parser.parse_args()
    root = Path.cwd()
    write_target(root / "AGENTS.md", AGENT_SKILLS_BLOCK, base_agents_created=args.base_agents_created)
    write_target(root / "docs" / "agents" / "work-management.md", source_work_management(), base_agents_created=False)
    write_target(root / "docs" / "agents" / "domain.md", DOMAIN_CONTENT, base_agents_created=False)
    configure_completion()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
