#!/usr/bin/env python3
"""Validate a RELEASE_REVIEW.md (or release.md) against the release-review
Output Format.

Checks that:
  - Every required (heading-level, heading-title-substring) pair appears at
    least once as an actual Markdown heading (not just body text or text
    inside a fenced code block).
  - With --strict, headings appear in the required order.

Usage:
  python scripts/validate_release_review.py <path-to-md-file>
  python scripts/validate_release_review.py <path-to-md-file> --strict
  python scripts/validate_release_review.py <path-to-md-file> --json

Exit codes:
  0 - all checks passed
  1 - one or more checks failed
  2 - invalid arguments / file missing
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, List, Optional, Tuple

# Required (level, title-substring) in the intended order.
# Substring match is case-insensitive so hosts may append qualifiers such as
# "## 3. Schema and Migration (postgres)".
REQUIRED: "list[tuple[int, str]]" = [
    (1, "release review"),
    (2, "decision"),
    (2, "scope"),
    (2, "release checklist"),
    (2, "schema and migration"),
    (2, "data impact"),
    (2, "metrics and reporting"),
    (2, "permission impact"),
    (2, "audit impact"),
    (2, "rollback plan"),
    (2, "observability"),
    (2, "open risks"),
    (2, "final recommendation"),
]

HEADING_RE = re.compile(r"^(#{1,6})\s+(.+?)\s*#*\s*$")


@dataclass(frozen=True)
class Heading:
    level: int
    title: str
    line: int


def parse_headings(text: str) -> List[Heading]:
    """Extract ATX headings from Markdown, skipping fenced code blocks."""
    headings: List[Heading] = []
    in_code = False
    for idx, raw in enumerate(text.splitlines(), start=1):
        line = raw.rstrip()
        stripped = line.lstrip()
        if stripped.startswith("```") or stripped.startswith("~~~"):
            in_code = not in_code
            continue
        if in_code:
            continue
        m = HEADING_RE.match(line)
        if not m:
            continue
        level = len(m.group(1))
        title = m.group(2).strip().lower()
        headings.append(Heading(level, title, idx))
    return headings


def find_index(
    headings: List[Heading], level: int, needle: str, start: int = 0
) -> int:
    for i in range(start, len(headings)):
        h = headings[i]
        if h.level == level and needle in h.title:
            return i
    return -1


def validate_text(
    text: str, required: List[Tuple[int, str]], strict: bool
) -> List[str]:
    """Validate raw markdown text against the required heading list."""
    errors: List[str] = []
    headings = parse_headings(text)

    if strict:
        cursor = 0
        for level, needle in required:
            idx = find_index(headings, level, needle, cursor)
            if idx < 0:
                any_idx = find_index(headings, level, needle, 0)
                if any_idx >= 0:
                    errors.append(
                        f"heading (h{level}) '{needle}' is out of order"
                        f" (found at line {headings[any_idx].line}, expected"
                        f" after cursor index {cursor})"
                    )
                else:
                    errors.append(f"missing heading (h{level}) '{needle}'")
                continue
            cursor = idx + 1
    else:
        for level, needle in required:
            idx = find_index(headings, level, needle, 0)
            if idx < 0:
                errors.append(f"missing heading (h{level}) '{needle}'")
    return errors


def validate_file(path: Path, strict: bool = False) -> List[str]:
    if not path.exists() or not path.is_file():
        return [f"file not found: {path}"]
    # utf-8-sig transparently strips a leading BOM if present. Files authored
    # on Windows (e.g., via PowerShell's Set-Content -Encoding utf8) commonly
    # carry a BOM which would otherwise break the ATX heading regex on the
    # first line.
    text = path.read_text(encoding="utf-8-sig")
    return validate_text(text, REQUIRED, strict)


def main(argv: Optional[Iterable[str]] = None) -> int:
    parser = argparse.ArgumentParser(
        description="Validate a RELEASE_REVIEW.md against release-review."
    )
    parser.add_argument("path", help="path to the markdown file")
    parser.add_argument(
        "--strict", action="store_true", help="also enforce heading order"
    )
    parser.add_argument(
        "--json", action="store_true", help="emit JSON report on stdout"
    )
    args = parser.parse_args(list(argv) if argv is not None else None)

    path = Path(args.path)
    if not path.exists() or not path.is_file():
        msg = f"file not found: {path}"
        if args.json:
            print(json.dumps({"ok": False, "errors": [msg], "strict": args.strict}))
        else:
            print(msg, file=sys.stderr)
        return 2

    errors = validate_file(path, strict=args.strict)
    ok = not errors
    if args.json:
        print(json.dumps({"ok": ok, "errors": errors, "strict": args.strict}))
    else:
        if ok:
            print("Release Review validation passed.")
        else:
            print("Release Review validation failed:")
            for e in errors:
                print(f"- {e}")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())