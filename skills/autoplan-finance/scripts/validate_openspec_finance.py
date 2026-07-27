#!/usr/bin/env python3
"""Validate OpenSpec Finance Profile structural coverage.

Checks that:
  - Every required file exists in the specs directory.
  - Every required (heading-level, heading-title) pair appears at least once
    as an actual Markdown heading (not just body text or code-fenced text).
  - With --strict, headings appear in the required order.

Usage:
  python scripts/validate_openspec_finance.py <specs-dir>
  python scripts/validate_openspec_finance.py <specs-dir> --strict
  python scripts/validate_openspec_finance.py <specs-dir> --json

Exit codes:
  0 - all checks passed
  1 - one or more checks failed
  2 - invalid arguments / specs dir missing
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, List, Optional, Tuple

# Required (level, title-substring). Titles are matched case-insensitively as
# substrings of the heading text so projects may append qualifiers such as
# "## Metric Registry (Golden Definitions)".
REQUIRED: "dict[str, list[tuple[int, str]]]" = {
    "requirements.md": [
        (1, "requirements"),
        (2, "decision flow"),
    ],
    "design.md": [
        (1, "design"),
    ],
    "tasks.md": [
        (1, "tasks"),
    ],
    "metrics.md": [
        (1, "metrics"),
        (2, "metric registry"),
        (2, "source mapping"),
        (2, "financial correctness"),
        (2, "consistency review"),
    ],
    "permissions.md": [
        (1, "permissions"),
        (2, "roles"),
        (2, "permission matrix"),
        (2, "field-level restrictions"),
        (2, "data scope rules"),
    ],
    "audit.md": [
        (1, "audit"),
        (2, "audited actions"),
        (2, "retention policy"),
        (2, "audit query requirements"),
    ],
    "release.md": [
        (1, "release"),
        (2, "decision"),
        (2, "release checklist"),
        (2, "rollback plan"),
        (2, "open release risks"),
    ],
}

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


def validate_file(
    path: Path, required: List[Tuple[int, str]], strict: bool
) -> List[str]:
    errors: List[str] = []
    # utf-8-sig transparently strips a leading BOM if present. Files authored
    # on Windows (e.g., via PowerShell's Set-Content -Encoding utf8) commonly
    # carry a BOM which would otherwise break the ATX heading regex on the
    # first line.
    text = path.read_text(encoding="utf-8-sig")
    headings = parse_headings(text)

    if strict:
        cursor = 0
        for level, needle in required:
            idx = find_index(headings, level, needle, cursor)
            if idx < 0:
                any_idx = find_index(headings, level, needle, 0)
                if any_idx >= 0:
                    errors.append(
                        f"{path.name}: heading (h{level}) '{needle}' is out of order"
                        f" (found at line {headings[any_idx].line}, expected after"
                        f" cursor index {cursor})"
                    )
                else:
                    errors.append(
                        f"{path.name}: missing heading (h{level}) '{needle}'"
                    )
                continue
            cursor = idx + 1
    else:
        for level, needle in required:
            idx = find_index(headings, level, needle, 0)
            if idx < 0:
                errors.append(
                    f"{path.name}: missing heading (h{level}) '{needle}'"
                )
    return errors


def validate(specs_dir: Path, strict: bool = False) -> List[str]:
    if not specs_dir.exists() or not specs_dir.is_dir():
        return [f"specs directory not found: {specs_dir}"]

    errors: List[str] = []
    for filename, required in REQUIRED.items():
        path = specs_dir / filename
        if not path.exists():
            errors.append(f"missing required file: {filename}")
            continue
        errors.extend(validate_file(path, required, strict))
    return errors


def main(argv: Optional[Iterable[str]] = None) -> int:
    parser = argparse.ArgumentParser(
        description="Validate OpenSpec Finance Profile structural coverage."
    )
    parser.add_argument("specs_dir", help="path to specs directory")
    parser.add_argument(
        "--strict", action="store_true", help="also enforce heading order"
    )
    parser.add_argument(
        "--json", action="store_true", help="emit JSON report on stdout"
    )
    args = parser.parse_args(list(argv) if argv is not None else None)

    specs_dir = Path(args.specs_dir)
    if not specs_dir.exists():
        msg = f"specs directory not found: {specs_dir}"
        if args.json:
            print(json.dumps({"ok": False, "errors": [msg], "strict": args.strict}))
        else:
            print(msg, file=sys.stderr)
        return 2

    errors = validate(specs_dir, strict=args.strict)
    ok = not errors
    if args.json:
        print(json.dumps({"ok": ok, "errors": errors, "strict": args.strict}))
    else:
        if ok:
            print("OpenSpec Finance Profile validation passed.")
        else:
            print("OpenSpec Finance Profile validation failed:")
            for e in errors:
                print(f"- {e}")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())