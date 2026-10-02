#!/usr/bin/env python3
"""Move existing flat FD archives and their evidence into per-FD folders."""

from pathlib import Path
import os
import re
import uuid


FEATURES = Path(__file__).resolve().parents[1] / "docs" / "features"
ARCHIVE = FEATURES / "archive"
REPOSITORY = FEATURES.parents[1]
FD_FILE = re.compile(r"^(FD-\d{3,})_[A-Z0-9_]+\.md$", re.IGNORECASE)
EVIDENCE_FILE = re.compile(r"^(FD-\d{3,})(?:-.*)?\.md$", re.IGNORECASE)


def atomic_write(path: Path, content: str) -> None:
    temporary = path.with_name(path.name + f".tmp-{os.getpid()}-{uuid.uuid4().hex}")
    try:
        with temporary.open("x", encoding="utf-8", newline="\n") as stream:
            stream.write(content)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)


def main() -> None:
    designs = sorted(path for path in ARCHIVE.glob("FD-*.md")
                     if FD_FILE.fullmatch(path.name))
    ids = {FD_FILE.fullmatch(path.name).group(1).upper() for path in designs}
    ids.update(path.parent.name.upper()
               for path in ARCHIVE.glob("FD-*/FD-*.md")
               if path.parent.is_dir() and FD_FILE.fullmatch(path.name)
               and path.name.upper().startswith(path.parent.name.upper() + "_"))
    moves = []
    replacements = {}
    for source in designs:
        name = FD_FILE.fullmatch(source.name).group(1).upper()
        target = ARCHIVE / name / source.name
        moves.append((source, target))
        replacements[f"docs/features/archive/{source.name}"] = (
            f"docs/features/archive/{name}/{source.name}"
        )
        replacements[f"archive/{source.name}"] = f"archive/{name}/{source.name}"

    for folder in ("reports", "reviews"):
        for source_dir in (ARCHIVE / folder, FEATURES / folder):
            for source in sorted(source_dir.glob("FD-*.md")):
                match = EVIDENCE_FILE.fullmatch(source.name)
                if not match or match.group(1).upper() not in ids:
                    continue
                name = match.group(1).upper()
                target = ARCHIVE / name / folder / source.name
                moves.append((source, target))
                new_path = f"docs/features/archive/{name}/{folder}/{source.name}"
                replacements[f"docs/features/{folder}/{source.name}"] = new_path
                if source_dir == ARCHIVE / folder:
                    replacements[f"docs/features/archive/{folder}/{source.name}"] = new_path
                    replacements[f"archive/{folder}/{source.name}"] = (
                        f"archive/{name}/{folder}/{source.name}"
                    )

    targets = set()
    source_targets = {}
    for source, target in moves:
        if not source.is_file() or source.is_symlink():
            raise RuntimeError(f"archive source is not a regular file: {source}")
        if target in targets or target.exists() or target.is_symlink():
            raise RuntimeError(f"archive destination already exists: {target}")
        parent = target.parent
        while parent != ARCHIVE.parent:
            if parent.is_symlink() or (parent.exists() and not parent.is_dir()):
                raise RuntimeError(f"archive destination parent is unsafe: {parent}")
            parent = parent.parent
        targets.add(target)
        source_targets[source] = target

    rewrites = {}
    search_roots = (FEATURES, REPOSITORY / "openspec", REPOSITORY / "skills")
    markdown_files = {path for directory in search_roots if directory.is_dir()
                      for path in directory.rglob("*.md")}
    markdown_files.update(path for path in (REPOSITORY / "README.md",
                                            REPOSITORY / "AGENTS.md",
                                            REPOSITORY / "CODEX.md") if path.is_file())
    for path in sorted(markdown_files):
        if path.is_symlink():
            continue
        original = path.read_text(encoding="utf-8")
        updated = original
        for old, new in replacements.items():
            updated = updated.replace(old, new)
        if updated != original:
            rewrites[source_targets.get(path, path)] = (original, updated)

    moved = []
    written = []
    try:
        for source, target in moves:
            target.parent.mkdir(parents=True, exist_ok=True)
            source.replace(target)
            moved.append((source, target))
        for path, (original, updated) in rewrites.items():
            atomic_write(path, updated)
            written.append((path, original))
    except OSError as exc:
        rollback_errors = []
        for path, original in reversed(written):
            try:
                atomic_write(path, original)
            except OSError as rollback_error:
                rollback_errors.append(str(rollback_error))
        for source, target in reversed(moved):
            try:
                source.parent.mkdir(parents=True, exist_ok=True)
                target.replace(source)
            except OSError as rollback_error:
                rollback_errors.append(str(rollback_error))
        if rollback_errors:
            raise RuntimeError("archive migration rollback incomplete: "
                               + "; ".join(rollback_errors)) from exc
        raise

    print(f"migrated {len(designs)} FD designs and {len(moves) - len(designs)} evidence files")


if __name__ == "__main__":
    main()
