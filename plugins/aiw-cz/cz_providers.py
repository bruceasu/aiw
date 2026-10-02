"""CLI provider discovery and invocation for Codex and Copilot."""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import tempfile
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
    if provider == "copilot":
        return None
    executable = shutil.which(provider)
    return [executable] if executable else None


def detect_cli(provider: str, timeout: float = 5.0) -> CLIInfo:
    command = _command(provider)
    if not command:
        raise ProviderUnavailable(
            "copilot CLI not configured" if provider == "copilot" else f"{provider} CLI not found"
        )
    probe_env = os.environ.copy()
    probe_env["CI"] = "1"
    probe_env["GIT_TERMINAL_PROMPT"] = "0"
    try:
        version = subprocess.run(
            [*command, "--version"], capture_output=True, text=True, stdin=subprocess.DEVNULL,
            env=probe_env,
            encoding="utf-8", errors="replace", timeout=timeout, check=False,
        )
        help_result = subprocess.run(
            [*command, "--help"], capture_output=True, text=True, stdin=subprocess.DEVNULL,
            env=probe_env,
            encoding="utf-8", errors="replace", timeout=timeout, check=False,
        )
        help_text = (help_result.stdout or help_result.stderr)
        if provider == "codex":
            exec_help = subprocess.run(
                [*command, "exec", "--help"], capture_output=True, text=True, stdin=subprocess.DEVNULL,
                env=probe_env,
                encoding="utf-8", errors="replace", timeout=timeout, check=False,
            )
            if exec_help.returncode != 0:
                raise ProviderUnavailable("codex exec is unavailable")
            help_text = exec_help.stdout or exec_help.stderr
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise ProviderUnavailable(f"{provider} capability detection failed: {exc}") from exc
    if version.returncode != 0 or not (version.stdout or version.stderr).strip():
        raise ProviderUnavailable(f"{provider} does not provide usable version output")
    if help_result.returncode != 0 or not help_text.strip():
        raise ProviderUnavailable(f"{provider} does not provide usable help output")
    return CLIInfo(
        provider=provider,
        command=command,
        version=(version.stdout or version.stderr).strip(),
        help_text=help_text,
    )


def _invocation(info: CLIInfo, prompt: str, model: str = "", last_message: str = "") -> list[str]:
    if os.environ.get(f"CZ_{info.provider.upper()}_ARGS", "").strip():
        raise ProviderUnavailable(f"CZ_{info.provider.upper()}_ARGS cannot override the read-only CLI contract")
    if info.provider == "codex":
        if "--json" not in info.help_text or "--sandbox" not in info.help_text:
            raise ProviderUnavailable("codex CLI has no detected read-only JSON mode")
        args = ["exec", "--sandbox", "read-only", "--json"]
        if last_message and "--output-last-message" in info.help_text:
            args.extend(("--output-last-message", last_message))
        args.append("-")
    else:
        if "--output-format" not in info.help_text or "--prompt" not in info.help_text:
            raise ProviderUnavailable("copilot CLI has no detected non-interactive text mode")
        args = ["--output-format", "text", "--prompt", prompt]
    if model and "--model" not in args:
        args = ["--model", model, *args] if info.provider == "copilot" else [args[0], "--model", model, *args[1:]]
    return [*info.command, *args]


def run_cli(provider: str, prompt: str, model: str = "", root: Path | None = None, timeout: float = 60.0) -> tuple[str, CLIInfo]:
    info = detect_cli(provider)
    workspace = root or Path.cwd()
    before = _git_status(workspace)
    env = os.environ.copy()
    env["AIW_CZ_READ_ONLY"] = "1"
    env["CI"] = "1"
    env["GIT_TERMINAL_PROMPT"] = "0"
    last_message = ""
    if provider == "codex":
        with tempfile.NamedTemporaryFile(prefix="aiw-cz-message-", suffix=".txt", delete=False) as file:
            last_message = file.name
    try:
        try:
            result = subprocess.run(
                _invocation(info, prompt, model, last_message), capture_output=True, text=True,
                input=prompt if provider == "codex" else None,
                stdin=subprocess.DEVNULL if provider != "codex" else None,
                encoding="utf-8", errors="replace", timeout=timeout,
                cwd=workspace, env=env, check=False,
            )
        except (OSError, subprocess.TimeoutExpired) as exc:
            raise ProviderUnavailable(f"{provider} invocation failed: {exc}") from exc
        final_text = ""
        if last_message:
            try:
                final_text = Path(last_message).read_text(encoding="utf-8").strip()
            except OSError:
                pass
    finally:
        if last_message:
            Path(last_message).unlink(missing_ok=True)
    if result.returncode != 0:
        if provider == "copilot":
            raise ProviderUnavailable(f"copilot exited {result.returncode}")
        detail = (result.stderr or result.stdout).strip()[-500:]
        raise ProviderUnavailable(f"{provider} exited {result.returncode}: {detail}")
    after = _git_status(workspace)
    if after != before:
        raise ProviderUnavailable(f"{provider} changed the working tree; refusing non-read-only result")
    if provider == "codex":
        final_text = final_text or _codex_final_output(result.stdout)
    else:
        final_text = result.stdout
    return extract_json(final_text), info


def _git_status(root: Path) -> str:
    try:
        result = subprocess.run(
            ["git", "status", "--porcelain", "--untracked-files=all"],
            cwd=root, capture_output=True, text=True, encoding="utf-8",
            errors="replace", timeout=5, check=False,
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise ProviderUnavailable(f"cannot establish read-only git guard: {exc}") from exc
    if result.returncode != 0:
        raise ProviderUnavailable("cannot establish read-only git guard")
    return result.stdout


def _codex_final_output(events: str) -> str:
    output = ""
    for line in events.splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if not isinstance(event, dict) or event.get("type") != "item.completed":
            continue
        item = event.get("item")
        if not isinstance(item, dict) or item.get("type") != "agent_message":
            continue
        if isinstance(item.get("text"), str) and item["text"].strip():
            output = item["text"]
            continue
        content = item.get("content")
        if isinstance(content, str):
            output = content
        elif isinstance(content, list):
            text = "\n".join(
                part["text"] for part in content
                if isinstance(part, dict) and part.get("type") in ("text", "output_text")
                and isinstance(part.get("text"), str)
            )
            if text.strip():
                output = text
    return output


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
