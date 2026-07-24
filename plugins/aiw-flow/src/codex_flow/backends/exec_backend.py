from __future__ import annotations

import asyncio
from pathlib import Path
from typing import List, Optional

from codex_flow.backends.base import ParsedExecOutput, parse_exec_jsonl
from codex_flow.models import AppConfig, TurnRequest, TurnResult, utc_now
from codex_flow.process_utils import build_codex_exec_command, run_command, terminate_process
from codex_flow.file_utils import atomic_write_text


class ExecBackendError(RuntimeError):
    """Exec backend failed."""


class ExecCodexBackend:
    def __init__(self, config: AppConfig):
        self.config = config
        self._started = False

    async def start(self) -> None:
        if self._started:
            return
        try:
            run_command(["cmd", "/c", "codex", "exec", "--help"])
        except Exception as exc:
            raise ExecBackendError("Unable to access codex exec: {}".format(exc))
        self._started = True

    async def close(self) -> None:
        self._started = False

    async def run_turn(self, request: TurnRequest) -> TurnResult:
        if not self._started:
            await self.start()
        output_dir = request.output_dir or request.workspace
        output_dir.mkdir(parents=True, exist_ok=True)
        events_file = output_dir / "{:04d}-events.jsonl".format(request.turn_number)
        stderr_file = output_dir / "{:04d}-stderr.log".format(request.turn_number)
        output_file = output_dir / "{:04d}-final.txt".format(request.turn_number)
        command = build_codex_exec_command(
            thread_id=request.thread_id,
            config=self.config,
            output_file=output_file,
            json_output=True,
            ephemeral=request.ephemeral,
        )
        started_at = utc_now()
        proc = await asyncio.create_subprocess_exec(
            *command,
            cwd=str(request.workspace),
            stdin=asyncio.subprocess.PIPE,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout_lines: List[str] = []
        stderr_chunks: List[str] = []

        async def read_stdout() -> None:
            assert proc.stdout is not None
            while True:
                line = await proc.stdout.readline()
                if not line:
                    break
                stdout_lines.append(line.decode("utf-8", errors="replace"))

        async def read_stderr() -> None:
            assert proc.stderr is not None
            while True:
                chunk = await proc.stderr.read(4096)
                if not chunk:
                    break
                stderr_chunks.append(chunk.decode("utf-8", errors="replace"))

        prompt_bytes = request.prompt.encode("utf-8")
        assert proc.stdin is not None
        proc.stdin.write(prompt_bytes)
        await proc.stdin.drain()
        proc.stdin.close()

        readers = [asyncio.create_task(read_stdout()), asyncio.create_task(read_stderr())]
        interrupted = False
        try:
            if request.timeout_seconds:
                await asyncio.wait_for(proc.wait(), timeout=request.timeout_seconds)
            else:
                await proc.wait()
        except asyncio.TimeoutError:
            interrupted = True
            await terminate_process(proc)
        finally:
            await asyncio.gather(*readers)

        atomic_write_text(events_file, "".join(stdout_lines))
        atomic_write_text(stderr_file, "".join(stderr_chunks))
        parsed = parse_exec_jsonl(stdout_lines)
        final_output = self._resolve_final_output(output_file, parsed)
        if not output_file.exists():
            atomic_write_text(output_file, final_output)
        return TurnResult(
            thread_id=parsed.thread_id or request.thread_id,
            final_output=final_output,
            exit_code=proc.returncode if proc.returncode is not None else 1,
            events_file=events_file,
            output_file=output_file,
            started_at=started_at,
            completed_at=utc_now(),
            interrupted=interrupted,
            stderr_file=stderr_file,
            metadata={"command": command, "invalid_json_lines": len(parsed.invalid_lines)},
        )

    def _resolve_final_output(self, output_file: Path, parsed: ParsedExecOutput) -> str:
        if output_file.exists():
            text = output_file.read_text(encoding="utf-8").strip()
            if text:
                return text
        return parsed.final_output

