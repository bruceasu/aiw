"""CLI provider discovery and invocation for Codex and Copilot."""

from __future__ import annotations

import json
import os
import shutil
import subprocess
from dataclasses import dataclass
from pathlib import Path
import shlex


@dataclass
class CLIInfo:
    provider: str
    command: list[str]
    version: str
    help_text: str


class ProviderUnavailable(RuntimeError):
    pass


def _command(provider: str) -> list[str] | None:
    env_name = f"CZ_{provider.upper()}_COMMAND"
    configured = os.environ.get(env_name, "").strip()
    if configured:
        return shlex.split(configured, posix=os.name != "nt")
    executable = shutil.which(provider)
    return [executable] if executable else None


def detect_cli(provider: str, timeout: float = 5.0) -> CLIInfo:
    command = _command(provider)
    if not command:
        raise ProviderUnavailable(f"{provider} CLI not found")
    try:
        version = subprocess.run(
            [*command, "--version"], capture_output=True, text=True,
            encoding="utf-8", errors="replace", timeout=timeout, check=False,
        )
        help_result = subprocess.run(
            [*command, "--help"], capture_output=True, text=True,
            encoding="utf-8", errors="replace", timeout=timeout, check=False,
        )
        help_text = (help_result.stdout or help_result.stderr)
        if provider == "codex":
            exec_help = subprocess.run(
                [*command, "exec", "--help"], capture_output=True, text=True,
                encoding="utf-8", errors="replace", timeout=timeout, check=False,
            )
            help_text += "\n" + (exec_help.stdout or exec_help.stderr)
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise ProviderUnavailable(f"{provider} capability detection failed: {exc}") from exc
    if version.returncode != 0 and help_result.returncode != 0:
        raise ProviderUnavailable(f"{provider} does not provide usable version/help output")
    return CLIInfo(
        provider=provider,
        command=command,
        version=(version.stdout or version.stderr).strip(),
        help_text=help_text,
    )


def _invocation(info: CLIInfo, prompt: str) -> list[str]:
    configured = os.environ.get(f"CZ_{info.provider.upper()}_ARGS", "").strip()
    if configured:
        args = shlex.split(configured, posix=os.name != "nt")
    elif info.provider == "codex":
        if "--json" not in info.help_text:
            raise ProviderUnavailable("codex CLI has no detected --json mode")
        args = ["exec", "--json", prompt]
    else:
        if "--output-format" not in info.help_text:
            raise ProviderUnavailable("copilot CLI has no detected JSON output mode")
        args = ["-p", prompt, "--output-format", "json"]
    return [*info.command, *args]


def run_cli(provider: str, prompt: str, timeout: float = 60.0) -> tuple[str, CLIInfo]:
    info = detect_cli(provider)
    before = _git_status()
    env = os.environ.copy()
    env["AIW_CZ_READ_ONLY"] = "1"
    try:
        result = subprocess.run(
            _invocation(info, prompt), capture_output=True, text=True,
            encoding="utf-8", errors="replace", timeout=timeout,
            cwd=Path.cwd(), env=env, check=False,
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise ProviderUnavailable(f"{provider} invocation failed: {exc}") from exc
    if result.returncode != 0:
        detail = (result.stderr or result.stdout).strip()[-500:]
        raise ProviderUnavailable(f"{provider} exited {result.returncode}: {detail}")
    after = _git_status()
    if after != before:
        raise ProviderUnavailable(f"{provider} changed the working tree; refusing non-read-only result")
    return extract_json(result.stdout), info


def _git_status() -> str:
    try:
        result = subprocess.run(
            ["git", "status", "--porcelain", "--untracked-files=all"],
            cwd=Path.cwd(), capture_output=True, text=True, encoding="utf-8",
            errors="replace", timeout=5, check=False,
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise ProviderUnavailable(f"cannot establish read-only git guard: {exc}") from exc
    if result.returncode != 0:
        raise ProviderUnavailable("cannot establish read-only git guard")
    return result.stdout


def extract_json(output: str) -> str:
    decoder = json.JSONDecoder()
    candidates: list[str] = []
    for index, char in enumerate(output):
        if char != "{":
            continue
        try:
            value, end = decoder.raw_decode(output[index:])
        except json.JSONDecodeError:
            continue
        if isinstance(value, dict) and "candidates" in value:
            candidates.append(output[index:index + end])
    if not candidates:
        raise ProviderUnavailable("CLI output contains no candidate JSON")
    return candidates[-1]
