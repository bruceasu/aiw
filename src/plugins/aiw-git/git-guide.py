#!/usr/bin/env python3
"""aiw git guide

Provide Git HOW-TO guidance from:

1. Built-in Python handlers.
2. Files stored under the local ``how-to`` directory.
3. Codex-generated answers.

Unknown questions are searched against the local HOW-TO knowledge base.
When no suitable local answer exists, Codex generates a new Markdown
HOW-TO and the result is saved for future use.
"""

from __future__ import annotations

import importlib.util
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import textwrap
import unicodedata
import zipfile

from collections.abc import Callable, Iterable
from dataclasses import dataclass
from pathlib import Path
from typing import Final
from xml.etree import ElementTree


# ---------------------------------------------------------------------------
# Paths and configuration
# ---------------------------------------------------------------------------

HERE: Final[Path] = Path(__file__).resolve().parent
CORE_PATH: Final[Path] = HERE / "aiw-git-core.py"
HOW_TO_DIR: Final[Path] = HERE / "how-to"

CODEX_MODEL: Final[str] = os.environ.get(
    "AIW_GIT_GUIDE_MODEL",
    "gpt-5.4-mini",
)

MAX_DIRECT_MATCHES: Final[int] = 3
MAX_CODEX_CANDIDATES: Final[int] = 12
MAX_EXCERPT_CHARS: Final[int] = 6000
MAX_FILE_SIZE: Final[int] = 10 * 1024 * 1024

TEXT_EXTENSIONS: Final[set[str]] = {
    ".txt",
    ".md",
    ".markdown",
    ".org",
    ".rst",
    ".adoc",
    ".asciidoc",
    ".textile",
    ".csv",
    ".tsv",
    ".json",
    ".jsonl",
    ".yaml",
    ".yml",
    ".xml",
    ".html",
    ".htm",
    ".ini",
    ".cfg",
    ".conf",
    ".properties",
    ".log",
    ".sh",
    ".bash",
    ".zsh",
    ".fish",
    ".ps1",
    ".bat",
    ".cmd",
    ".py",
    ".java",
    ".kt",
    ".kts",
    ".js",
    ".ts",
    ".c",
    ".h",
    ".cpp",
    ".hpp",
    ".sql",
}

DOCUMENT_EXTENSIONS: Final[set[str]] = {
    ".pdf",
    ".doc",
    ".docx",
    ".odt",
    ".rtf",
}


# ---------------------------------------------------------------------------
# Core module
# ---------------------------------------------------------------------------

def load_core():
    """Load the shared aiw Git core module."""

    if not CORE_PATH.is_file():
        raise RuntimeError(f"Core module not found: {CORE_PATH}")

    spec = importlib.util.spec_from_file_location(
        "aiw_git_core",
        CORE_PATH,
    )

    if spec is None or spec.loader is None:
        raise RuntimeError(
            f"Unable to load core module: {CORE_PATH}"
        )

    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)

    return module


try:
    core = load_core()
except Exception as exc:
    print(
        f"Error loading aiw Git core: {exc}",
        file=sys.stderr,
    )
    sys.exit(1)


# ---------------------------------------------------------------------------
# Metadata
# ---------------------------------------------------------------------------

META = {
    "name": "aiw git guide",
    "short": (
        "Search built-in and local Git HOW-TO guides, "
        "or ask Codex."
    ),
    "long": (
        "Displays built-in Git guidance and searches the local "
        "'how-to' directory. If no suitable guide exists, the "
        f"question is delegated to Codex model {CODEX_MODEL}. "
        "New Codex answers are saved as Markdown for future use."
    ),
    "usage": (
        "aiw git guide "
        "[list | files | help <topic> | <how-to-or-question>]"
    ),
    "args": [
        {
            "flag": "<how-to-or-question>",
            "description": (
                "A built-in HOW-TO topic, a local guide search, "
                "or a free-form Git question."
            ),
        }
    ],
    "examples": [
        "aiw git guide",
        "aiw git guide list",
        "aiw git guide files",
        "aiw git guide rollback",
        "aiw git guide commit-recovery",
        "aiw git guide help split",
        'aiw git guide "How do I squash the last three commits?"',
        'aiw git guide "恢复误删的远程分支"',
    ],
}


# ---------------------------------------------------------------------------
# Models
# ---------------------------------------------------------------------------

@dataclass
class SearchMatch:
    path: Path
    score: int
    filename_score: int
    content_score: int
    excerpt: str
    extracted: bool = False


@dataclass
class CodexAnswer:
    title: str
    slug: str
    content: str
    save: bool = True


# ---------------------------------------------------------------------------
# Built-in guides
# ---------------------------------------------------------------------------

def commit_recovery() -> int:
    core.run_cmd(
        ["git", "rev-list", "--all", "--max-count=30"]
    )

    print()
    core.run_cmd(["git", "reflog"])

    print(
        """
How to recover after an accidental hard reset
---------------------------------------------

1. Find the SHA of the commit in the reflog above.

2. Create a recovery branch first:

     git switch -c recover-branch <SHA>

3. After verifying it, restore the original branch if needed:

     git switch <original-branch>
     git reset --hard <SHA>

Important:

- Prefer creating a recovery branch before using reset.
- Reflog is local to the repository.
- Do not run aggressive garbage collection until recovery is complete.
"""
    )

    return 0


def rollback() -> int:
    print(
        """
How to roll back or undo a recent Git operation
-----------------------------------------------

Undo the last commit and keep changes staged:

     git reset --soft HEAD~1

Undo the last commit and keep changes unstaged:

     git reset --mixed HEAD~1

Undo the last commit and discard changes:

     git reset --hard HEAD~1

Undo a pushed commit safely:

     git revert <SHA>

Abort an operation:

     git rebase --abort
     git merge --abort
     git cherry-pick --abort
     git revert --abort

General rule:

- Use reset for local unpublished history.
- Use revert for shared or pushed history.
- Create a backup branch before destructive operations.
"""
    )

    return 0


def split() -> int:
    print(
        """
How to split commits that landed on the wrong branch
=====================================================

1. Inspect commits:

     git log --oneline --graph --decorate -20

2. Preserve the current state:

     git branch backup-before-split

3. Reset the original branch:

     git reset --hard <last-good-sha>

4. Create target branches and cherry-pick commits:

     git switch -c feature-A <last-good-sha>
     git cherry-pick <sha-for-A>

     git switch -c feature-B <last-good-sha>
     git cherry-pick <sha-for-B>

5. Verify:

     git log --oneline --graph --decorate --all
     git status
"""
    )

    return 0


HOW_TO_TOPICS: Final[
    dict[str, tuple[str, Callable[[], int]]]
] = {
    "commit-recovery": (
        "Recover commits after reset, rebase, branch deletion, "
        "or a lost HEAD.",
        commit_recovery,
    ),
    "rollback": (
        "Undo commits or abort reset, merge, rebase, "
        "cherry-pick, and revert operations.",
        rollback,
    ),
    "split": (
        "Move commits that landed on the wrong branch "
        "into separate branches.",
        split,
    ),
}


TOPIC_ALIASES: Final[dict[str, str]] = {
    "recover": "commit-recovery",
    "recovery": "commit-recovery",
    "commit-recover": "commit-recovery",
    "commit-recovery": "commit-recovery",
    "undo": "rollback",
    "roll-back": "rollback",
    "split-commits": "split",
    "split-commit": "split",
}


# ---------------------------------------------------------------------------
# General utility functions
# ---------------------------------------------------------------------------

def ensure_how_to_directory() -> None:
    """Create the HOW-TO directory when it does not exist."""

    HOW_TO_DIR.mkdir(parents=True, exist_ok=True)


def normalize_topic(value: str) -> str:
    topic = value.strip().lower().replace("_", "-")
    return TOPIC_ALIASES.get(topic, topic)


def normalize_search_text(value: str) -> str:
    """Normalize text for simple cross-language matching."""

    value = unicodedata.normalize("NFKC", value)
    value = value.casefold()
    value = re.sub(r"[_./\\-]+", " ", value)
    value = re.sub(r"\s+", " ", value)

    return value.strip()


def tokenize_query(query: str) -> list[str]:
    """Split a query into useful search tokens."""

    normalized = normalize_search_text(query)

    tokens = re.findall(
        r"[a-z0-9][a-z0-9+#.]{1,}|[\u3400-\u9fff]{2,}|"
        r"[\u3040-\u30ff]{2,}",
        normalized,
    )

    ignored = {
        "how",
        "what",
        "when",
        "where",
        "which",
        "with",
        "from",
        "this",
        "that",
        "git",
        "please",
        "help",
        "如何",
        "怎么",
        "怎样",
        "什么",
        "请问",
        "可以",
    }

    result: list[str] = []

    for token in tokens:
        if token in ignored:
            continue

        if token not in result:
            result.append(token)

    return result


def safe_slug(value: str, fallback: str = "git-how-to") -> str:
    """Convert a title into a safe Markdown filename stem."""

    normalized = unicodedata.normalize("NFKD", value)

    ascii_value = normalized.encode(
        "ascii",
        "ignore",
    ).decode("ascii")

    ascii_value = ascii_value.casefold()
    ascii_value = re.sub(r"[^a-z0-9]+", "-", ascii_value)
    ascii_value = ascii_value.strip("-")
    ascii_value = re.sub(r"-{2,}", "-", ascii_value)

    if not ascii_value:
        ascii_value = fallback

    return ascii_value[:80].strip("-")


def unique_output_path(slug: str) -> Path:
    """Return a non-conflicting HOW-TO Markdown path."""

    ensure_how_to_directory()

    base_slug = safe_slug(slug)
    candidate = HOW_TO_DIR / f"{base_slug}.md"

    if not candidate.exists():
        return candidate

    number = 2

    while True:
        candidate = HOW_TO_DIR / f"{base_slug}-{number}.md"

        if not candidate.exists():
            return candidate

        number += 1


def is_probably_binary(data: bytes) -> bool:
    """Apply a lightweight binary-file heuristic."""

    if not data:
        return False

    if b"\x00" in data[:4096]:
        return True

    sample = data[:4096]
    control_count = sum(
        1
        for byte in sample
        if byte < 9 or 13 < byte < 32
    )

    return control_count > len(sample) * 0.10


def decode_text(data: bytes) -> str | None:
    """Decode text using common documentation encodings."""

    if not data:
        return ""

    if data.startswith(b"\xef\xbb\xbf"):
        return data.decode("utf-8-sig", errors="replace")

    if data.startswith(b"\xff\xfe"):
        return data.decode("utf-16-le", errors="replace")

    if data.startswith(b"\xfe\xff"):
        return data.decode("utf-16-be", errors="replace")

    if is_probably_binary(data):
        return None

    for encoding in (
        "utf-8",
        "cp932",
        "shift_jis",
        "gb18030",
        "latin-1",
    ):
        try:
            return data.decode(encoding)
        except UnicodeDecodeError:
            continue

    return None


def run_capture(
    command: list[str],
    *,
    input_text: str | None = None,
    timeout: int = 30,
) -> subprocess.CompletedProcess[str] | None:
    """Run an external command safely without invoking a shell."""

    try:
        return subprocess.run(
            command,
            input=input_text,
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
            timeout=timeout,
            check=False,
        )
    except (
        OSError,
        subprocess.TimeoutExpired,
    ):
        return None


# ---------------------------------------------------------------------------
# File enumeration
# ---------------------------------------------------------------------------

def list_how_to_files_with_rg() -> list[Path] | None:
    rg = shutil.which("rg")

    if rg is None:
        return None

    completed = run_capture(
        [
            rg,
            "--files",
            "--hidden",
            "--glob",
            "!.*.tmp",
            str(HOW_TO_DIR),
        ]
    )

    if completed is None or completed.returncode not in (0, 1):
        return None

    return [
        Path(line)
        for line in completed.stdout.splitlines()
        if line.strip()
    ]


def list_how_to_files_with_fd() -> list[Path] | None:
    fd = shutil.which("fd") or shutil.which("fdfind")

    if fd is None:
        return None

    completed = run_capture(
        [
            fd,
            "--type",
            "f",
            "--hidden",
            ".",
            str(HOW_TO_DIR),
        ]
    )

    if completed is None or completed.returncode not in (0, 1):
        return None

    return [
        Path(line)
        for line in completed.stdout.splitlines()
        if line.strip()
    ]


def list_how_to_files_with_find() -> list[Path] | None:
    find = shutil.which("find")

    if find is None or os.name == "nt":
        return None

    completed = run_capture(
        [
            find,
            str(HOW_TO_DIR),
            "-type",
            "f",
        ]
    )

    if completed is None or completed.returncode != 0:
        return None

    return [
        Path(line)
        for line in completed.stdout.splitlines()
        if line.strip()
    ]


def list_how_to_files_python() -> list[Path]:
    return [
        path
        for path in HOW_TO_DIR.rglob("*")
        if path.is_file()
    ]


def list_how_to_files() -> list[Path]:
    """Enumerate files using rg, fd, find, then Python."""

    ensure_how_to_directory()

    for finder in (
        list_how_to_files_with_rg,
        list_how_to_files_with_fd,
        list_how_to_files_with_find,
    ):
        paths = finder()

        if paths is not None:
            return sorted(
                {
                    path.resolve()
                    for path in paths
                    if path.is_file()
                }
            )

    return sorted(
        path.resolve()
        for path in list_how_to_files_python()
    )


# ---------------------------------------------------------------------------
# Document extraction
# ---------------------------------------------------------------------------

def read_plain_text(path: Path) -> str | None:
    try:
        if path.stat().st_size > MAX_FILE_SIZE:
            return None

        return decode_text(path.read_bytes())
    except OSError:
        return None


def read_docx(path: Path) -> str | None:
    """Extract visible text from a DOCX file using the standard library."""

    try:
        with zipfile.ZipFile(path) as archive:
            raw_xml = archive.read("word/document.xml")
    except (
        OSError,
        KeyError,
        zipfile.BadZipFile,
    ):
        return None

    try:
        root = ElementTree.fromstring(raw_xml)
    except ElementTree.ParseError:
        return None

    text_parts: list[str] = []

    for element in root.iter():
        local_name = element.tag.rsplit("}", 1)[-1]

        if local_name == "t" and element.text:
            text_parts.append(element.text)

        elif local_name in {"p", "br"}:
            text_parts.append("\n")

    return "".join(text_parts)


def read_odt(path: Path) -> str | None:
    """Extract text from an ODT file."""

    try:
        with zipfile.ZipFile(path) as archive:
            raw_xml = archive.read("content.xml")
    except (
        OSError,
        KeyError,
        zipfile.BadZipFile,
    ):
        return None

    try:
        root = ElementTree.fromstring(raw_xml)
    except ElementTree.ParseError:
        return None

    parts: list[str] = []

    for element in root.iter():
        if element.text:
            parts.append(element.text)

        if element.tail:
            parts.append(element.tail)

        local_name = element.tag.rsplit("}", 1)[-1]

        if local_name in {"p", "h", "line-break"}:
            parts.append("\n")

    return " ".join(parts)


def extract_with_command(
    path: Path,
    command_candidates: Iterable[list[str]],
) -> str | None:
    for command in command_candidates:
        executable = shutil.which(command[0])

        if executable is None:
            continue

        command = [executable, *command[1:]]

        completed = run_capture(
            command,
            timeout=60,
        )

        if (
            completed is not None
            and completed.returncode == 0
            and completed.stdout.strip()
        ):
            return completed.stdout

    return None


def read_pdf(path: Path) -> str | None:
    return extract_with_command(
        path,
        [
            ["pdftotext", "-layout", str(path), "-"],
            ["mutool", "draw", "-F", "txt", str(path)],
        ],
    )


def read_legacy_doc(path: Path) -> str | None:
    return extract_with_command(
        path,
        [
            ["antiword", str(path)],
            ["catdoc", str(path)],
            [
                "pandoc",
                "--from=doc",
                "--to=plain",
                str(path),
            ],
        ],
    )


def read_rtf(path: Path) -> str | None:
    return extract_with_command(
        path,
        [
            ["unrtf", "--text", str(path)],
            [
                "pandoc",
                "--from=rtf",
                "--to=plain",
                str(path),
            ],
        ],
    )


def extract_document_text(path: Path) -> tuple[str | None, bool]:
    """Return extracted document text and whether extraction was used."""

    suffix = path.suffix.casefold()

    if suffix in TEXT_EXTENSIONS or not suffix:
        return read_plain_text(path), False

    if suffix == ".docx":
        return read_docx(path), True

    if suffix == ".odt":
        return read_odt(path), True

    if suffix == ".pdf":
        return read_pdf(path), True

    if suffix == ".doc":
        return read_legacy_doc(path), True

    if suffix == ".rtf":
        return read_rtf(path), True

    return None, False


# ---------------------------------------------------------------------------
# Search
# ---------------------------------------------------------------------------

def filename_match_score(
    path: Path,
    query: str,
    tokens: list[str],
) -> int:
    relative_name = str(
        path.relative_to(HOW_TO_DIR)
    )

    normalized_name = normalize_search_text(relative_name)
    normalized_query = normalize_search_text(query)

    score = 0

    if normalized_query == normalized_name:
        score += 100

    if normalized_query in normalized_name:
        score += 60

    stem = normalize_search_text(path.stem)

    if normalized_query == stem:
        score += 100

    if normalized_query in stem:
        score += 70

    for token in tokens:
        if token == stem:
            score += 30
        elif token in stem:
            score += 20
        elif token in normalized_name:
            score += 10

    return score


def content_match_score(
    text: str,
    query: str,
    tokens: list[str],
) -> int:
    normalized_text = normalize_search_text(text)
    normalized_query = normalize_search_text(query)

    score = 0

    if normalized_query and normalized_query in normalized_text:
        score += 80

    for token in tokens:
        occurrences = normalized_text.count(token)

        if occurrences:
            score += min(occurrences, 5) * 6

    return score


def build_excerpt(
    text: str,
    query: str,
    tokens: list[str],
    *,
    max_chars: int = MAX_EXCERPT_CHARS,
) -> str:
    """Build an excerpt around the first query/token match."""

    if len(text) <= max_chars:
        return text.strip()

    normalized_text = normalize_search_text(text)
    search_terms = [
        normalize_search_text(query),
        *tokens,
    ]

    index = -1

    for term in search_terms:
        if not term:
            continue

        index = normalized_text.find(term)

        if index >= 0:
            break

    if index < 0:
        return text[:max_chars].strip()

    start = max(0, index - max_chars // 3)
    end = min(len(text), start + max_chars)

    excerpt = text[start:end].strip()

    if start > 0:
        excerpt = "...\n" + excerpt

    if end < len(text):
        excerpt += "\n..."

    return excerpt


def search_local_how_to(query: str) -> list[SearchMatch]:
    """Search filenames and supported document contents."""

    tokens = tokenize_query(query)
    matches: list[SearchMatch] = []

    for path in list_how_to_files():
        filename_score = filename_match_score(
            path,
            query,
            tokens,
        )

        text, extracted = extract_document_text(path)

        content_score = 0
        excerpt = ""

        if text:
            content_score = content_match_score(
                text,
                query,
                tokens,
            )

            if filename_score > 0 or content_score > 0:
                excerpt = build_excerpt(
                    text,
                    query,
                    tokens,
                )

        total_score = filename_score + content_score

        if total_score <= 0:
            continue

        matches.append(
            SearchMatch(
                path=path,
                score=total_score,
                filename_score=filename_score,
                content_score=content_score,
                excerpt=excerpt,
                extracted=extracted,
            )
        )

    return sorted(
        matches,
        key=lambda item: (
            item.score,
            item.filename_score,
            item.content_score,
            item.path.name.casefold(),
        ),
        reverse=True,
    )


def is_clear_single_match(matches: list[SearchMatch]) -> bool:
    if not matches:
        return False

    first = matches[0]

    if first.score >= 100 and len(matches) == 1:
        return True

    if first.score < 60:
        return False

    if len(matches) == 1:
        return True

    second = matches[1]

    return first.score >= second.score + 40


def display_local_match(match: SearchMatch) -> int:
    relative_path = match.path.relative_to(HOW_TO_DIR)

    print(f"HOW-TO: {relative_path}")
    print("=" * min(80, len(str(relative_path)) + 8))
    print()

    if match.excerpt:
        print(match.excerpt.rstrip())
    else:
        print(
            "The file matched by filename, but its contents "
            "could not be extracted automatically."
        )
        print()
        print(f"File: {match.path}")

    return 0


# ---------------------------------------------------------------------------
# Codex integration
# ---------------------------------------------------------------------------

CODEX_OUTPUT_SCHEMA: Final[dict] = {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "additionalProperties": False,
    "required": [
        "title",
        "slug",
        "content",
        "save",
    ],
    "properties": {
        "title": {
            "type": "string",
            "minLength": 3,
            "maxLength": 120,
        },
        "slug": {
            "type": "string",
            "minLength": 1,
            "maxLength": 80,
            "pattern": "^[a-z0-9]+(?:-[a-z0-9]+)*$",
        },
        "content": {
            "type": "string",
            "minLength": 20,
        },
        "save": {
            "type": "boolean",
        },
    },
}


def get_codex_path() -> str | None:
    return shutil.which("codex")


def write_codex_schema(directory: Path) -> Path:
    schema_path = directory / "guide-output-schema.json"

    schema_path.write_text(
        json.dumps(
            CODEX_OUTPUT_SCHEMA,
            ensure_ascii=False,
            indent=2,
        ),
        encoding="utf-8",
    )

    return schema_path


def run_codex_structured(
    prompt: str,
) -> tuple[int, CodexAnswer | None, str]:
    """Run Codex and parse its structured final response."""

    codex = get_codex_path()

    if codex is None:
        return (
            127,
            None,
            "Codex CLI was not found in PATH.",
        )

    with tempfile.TemporaryDirectory(
        prefix="aiw-git-guide-"
    ) as temporary_directory:
        temp_dir = Path(temporary_directory)
        schema_path = write_codex_schema(temp_dir)
        output_path = temp_dir / "codex-output.json"

        command = [
            codex,
            "exec",
            "--model",
            CODEX_MODEL,
            "--sandbox",
            "read-only",
            "--output-schema",
            str(schema_path),
            "-o",
            str(output_path),
            "-",
        ]

        try:
            completed = subprocess.run(
                command,
                input=prompt,
                text=True,
                encoding="utf-8",
                errors="replace",
                check=False,
            )
        except KeyboardInterrupt:
            return 130, None, "Codex request cancelled."
        except OSError as exc:
            return 1, None, f"Failed to execute Codex: {exc}"

        if completed.returncode != 0:
            return (
                completed.returncode,
                None,
                f"Codex exited with status "
                f"{completed.returncode}.",
            )

        if not output_path.is_file():
            return (
                1,
                None,
                "Codex did not create its structured output file.",
            )

        try:
            raw = json.loads(
                output_path.read_text(encoding="utf-8")
            )
        except (
            OSError,
            json.JSONDecodeError,
        ) as exc:
            return (
                1,
                None,
                f"Unable to read Codex output: {exc}",
            )

        try:
            answer = CodexAnswer(
                title=str(raw["title"]).strip(),
                slug=safe_slug(str(raw["slug"])),
                content=str(raw["content"]).strip(),
                save=bool(raw["save"]),
            )
        except (
            KeyError,
            TypeError,
            ValueError,
        ) as exc:
            return (
                1,
                None,
                f"Invalid Codex output: {exc}",
            )

        return 0, answer, ""


def build_new_answer_prompt(question: str) -> str:
    return textwrap.dedent(
        f"""
        You are the Git guidance backend for the command:

            aiw git guide

        The local HOW-TO knowledge base has no useful answer for the
        following question:

        <question>
        {question}
        </question>

        Produce a reusable Git HOW-TO document.

        Output requirements:

        - title:
          A concise human-readable title.

        - slug:
          A concise lowercase ASCII filename stem.
          Use only letters, digits, and hyphens.
          Do not include ".md".

        - content:
          A complete Markdown document.
          Start with "# <title>".
          Explain the situation, safe workflow, commands, risks,
          verification, recovery, and relevant alternatives.
          Prefer modern Git commands such as "git switch" and
          "git restore", while mentioning older alternatives when useful.
          Clearly distinguish:
            * local versus pushed history;
            * safe versus destructive operations;
            * commands that rewrite history.
          Do not claim to have run commands.
          Do not use Markdown code fences around the entire document.

        - save:
          Return true because this answer is a newly generated reusable
          HOW-TO.

        Answer in the same language as the user's question unless command
        names or technical terms are clearer in English.
        """
    ).strip()


def build_candidate_answer_prompt(
    question: str,
    matches: list[SearchMatch],
) -> str:
    candidate_sections: list[str] = []

    for index, match in enumerate(
        matches[:MAX_CODEX_CANDIDATES],
        start=1,
    ):
        relative_path = match.path.relative_to(HOW_TO_DIR)

        excerpt = match.excerpt.strip()

        if not excerpt:
            excerpt = (
                "[Content could not be extracted. "
                "Only the filename matched.]"
            )

        candidate_sections.append(
            textwrap.dedent(
                f"""
                ## Candidate {index}

                Path: {relative_path}
                Search score: {match.score}

                <candidate-content>
                {excerpt}
                </candidate-content>
                """
            ).strip()
        )

    candidates = "\n\n".join(candidate_sections)

    return textwrap.dedent(
        f"""
        You are the Git guidance backend for:

            aiw git guide

        Answer the user's question using the most relevant information
        from the local candidate HOW-TO files below.

        User question:

        <question>
        {question}
        </question>

        Local candidates:

        {candidates}

        Instructions:

        - Determine which candidate or combination of candidates best
          answers the question.
        - Do not mention internal search scores.
        - Do not pretend that an unrelated candidate answers the question.
        - If the candidates are insufficient, supplement them using your
          Git knowledge.
        - Return a concise but complete Markdown answer.
        - Preserve important safety warnings.
        - Answer in the same language as the question.
        - Set save to false because this answer is based on existing local
          material and must not create a duplicate HOW-TO.
        - title should describe the answer.
        - slug should be a valid lowercase ASCII slug even though the
          result will not be saved.
        """
    ).strip()


def save_generated_answer(
    answer: CodexAnswer,
    question: str,
) -> Path:
    output_path = unique_output_path(answer.slug)

    content = answer.content.strip()

    if not content.startswith("#"):
        content = f"# {answer.title}\n\n{content}"

    metadata = textwrap.dedent(
        f"""
        <!--
        Generated by: aiw git guide
        Model: {CODEX_MODEL}
        Original question: {question.replace("--", "—")}
        -->

        """
    ).lstrip()

    output_path.write_text(
        metadata + content.rstrip() + "\n",
        encoding="utf-8",
        newline="\n",
    )

    return output_path


def answer_with_codex(
    question: str,
    matches: list[SearchMatch],
) -> int:
    """Use Codex for ambiguity resolution or new content generation."""

    if matches:
        prompt = build_candidate_answer_prompt(
            question,
            matches,
        )
    else:
        prompt = build_new_answer_prompt(question)

    rc, answer, error = run_codex_structured(prompt)

    if rc != 0 or answer is None:
        print(
            f"Error: {error}",
            file=sys.stderr,
        )
        return rc or 1

    print(answer.content.rstrip())

    if not matches and answer.save:
        path = save_generated_answer(
            answer,
            question,
        )

        print()
        print(
            f"Saved new HOW-TO: "
            f"{path.relative_to(HERE)}",
            file=sys.stderr,
        )

    return 0


# ---------------------------------------------------------------------------
# Help and listing
# ---------------------------------------------------------------------------

def print_topics() -> None:
    print("Built-in Git HOW-TO topics:")
    print()

    width = max(
        len(topic)
        for topic in HOW_TO_TOPICS
    )

    for topic, (
        description,
        _,
    ) in HOW_TO_TOPICS.items():
        print(
            f"  {topic:<{width}}  {description}"
        )

    print()
    print("Local HOW-TO directory:")
    print(f"  {HOW_TO_DIR}")
    print()
    print("Supported directly:")
    print("  txt, md, org, rst, json, yaml, source code, etc.")
    print()
    print("Supported with built-in extraction:")
    print("  docx, odt")
    print()
    print("Supported when optional tools are installed:")
    print("  pdf  : pdftotext or mutool")
    print("  doc  : antiword, catdoc, or pandoc")
    print("  rtf  : unrtf or pandoc")
    print()
    print("Search tools, in preference order:")
    print("  rg -> fd/fdfind -> find -> Python pathlib")
    print()
    print("Examples:")
    print("  aiw git guide rollback")
    print("  aiw git guide files")
    print(
        '  aiw git guide '
        '"How do I recover a deleted remote branch?"'
    )


def print_local_files() -> None:
    files = list_how_to_files()

    if not files:
        print(
            f"No local HOW-TO files found in {HOW_TO_DIR}"
        )
        return

    print(f"Local HOW-TO files in {HOW_TO_DIR}:")
    print()

    for path in files:
        relative_path = path.relative_to(HOW_TO_DIR)
        print(f"  {relative_path}")


def print_general_help() -> None:
    core.print_help_meta(META)
    print()
    print_topics()


def run_topic(topic: str) -> int | None:
    normalized = normalize_topic(topic)
    entry = HOW_TO_TOPICS.get(normalized)

    if entry is None:
        return None

    _, handler = entry
    return handler()


# ---------------------------------------------------------------------------
# Query execution
# ---------------------------------------------------------------------------

def handle_query(question: str) -> int:
    """Resolve a question through local search and Codex."""

    matches = search_local_how_to(question)

    if is_clear_single_match(matches):
        return display_local_match(matches[0])

    if matches:
        return answer_with_codex(
            question,
            matches,
        )

    return answer_with_codex(
        question,
        [],
    )


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main(argv: list[str]) -> int:
    ensure_how_to_directory()

    help_flags = {
        "-h",
        "--help",
        "-help",
        "-?",
    }

    list_flags = {
        "list",
        "topics",
        "--list",
    }

    file_flags = {
        "files",
        "--files",
        "documents",
        "docs",
    }

    if not argv:
        print_general_help()
        return 0

    first = argv[0].strip().casefold()

    if first in help_flags:
        print_general_help()
        return 0

    if first in list_flags:
        print_topics()
        return 0

    if first in file_flags:
        print_local_files()
        return 0

    if first == "help":
        if len(argv) == 1:
            print_general_help()
            return 0

        topic = normalize_topic(argv[1])

        if topic in HOW_TO_TOPICS:
            description, _ = HOW_TO_TOPICS[topic]

            print(f"{topic}: {description}")
            print()
            print(f"Run: aiw git guide {topic}")

            return 0

        question = " ".join(argv[1:]).strip()

        if question:
            return handle_query(question)

        return 2

    # A built-in topic must be passed as one argument.
    # Multiple arguments are treated as a natural-language question.
    if len(argv) == 1:
        result = run_topic(argv[0])

        if result is not None:
            return result

    question = " ".join(argv).strip()

    if not question:
        print_general_help()
        return 0

    return handle_query(question)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
