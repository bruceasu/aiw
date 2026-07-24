from __future__ import annotations

import asyncio
import os
import subprocess
from pathlib import Path
from typing import Dict, Iterable, List, Optional

from codex_flow.models import AppConfig


class CommandError(RuntimeError):
    """External command failed."""


def codex_cli_prefix() -> List[str]:
    if os.name == "nt":
        return ["cmd", "/c", "codex"]
    return ["codex"]


def build_codex_exec_command(
    *,
    thread_id: Optional[str],
    config: AppConfig,
    output_file: Path,
    json_output: bool = True,
    ephemeral: bool = False,
) -> List[str]:
    command = codex_cli_prefix() + ["exec"]
    if thread_id:
        command.extend(["resume", thread_id, "-"])
    else:
        command.append("-")
    if json_output:
        command.append("--json")
    command.extend(["-o", str(output_file)])
    if config.model:
        command.extend(["--model", config.model])
    if config.profile:
        command.extend(["--profile", config.profile])
    if config.sandbox:
        command.extend(["--sandbox", config.sandbox])
    if ephemeral:
        command.append("--ephemeral")
    command.extend(config.additional_codex_args)
    return command


def run_command(args: List[str], cwd: Optional[Path] = None) -> str:
    try:
        completed = subprocess.run(
            args,
            cwd=str(cwd) if cwd else None,
            check=True,
            capture_output=True,
            text=True,
            encoding="utf-8",
        )
    except subprocess.CalledProcessError as exc:
        raise CommandError(exc.stderr.strip() or exc.stdout.strip() or str(exc))
    return completed.stdout


async def terminate_process(proc: asyncio.subprocess.Process, grace_period: float = 2.0) -> None:
    if proc.returncode is not None:
        return
    proc.terminate()
    try:
        await asyncio.wait_for(proc.wait(), timeout=grace_period)
        return
    except asyncio.TimeoutError:
        proc.kill()
        await proc.wait()
