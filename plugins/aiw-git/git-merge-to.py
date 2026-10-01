#!/usr/bin/env python3
"""Merge branches into a target using an isolated temporary worktree."""

import importlib.util
import os
from pathlib import Path
import shlex
import subprocess
import sys
import tempfile


CORE_PATH = Path(__file__).with_name("aiw-git-core.py")
spec = importlib.util.spec_from_file_location("aiw_git_core", CORE_PATH)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    "name": "aiw git merge-to",
    "short": "Merge branches into a target without changing your workspace.",
    "long": (
        "Use a temporary worktree to merge the current or selected branch into "
        "an existing local target. With --new, create the target from the first "
        "source and merge the other sources in order. Sources may be local or "
        "existing remote-tracking branches. No fetch or push is performed. "
        "A target checked out in any worktree is rejected. Your checkout, files, "
        "and index stay unchanged. Success removes the temporary worktree; "
        "failure retains it and prints recovery commands. Earlier successful "
        "merges and a newly created target are retained on failure."
    ),
    "usage": (
        "aiw git merge-to <target_branch> [source_branch]\n"
        "  aiw git merge-to --new <new_branch> <source_branch1> [source_branch2 ...]"
    ),
    "args": [
        {"flag": "<target_branch>", "description": "Existing local target branch."},
        {"flag": "[source_branch]", "description": "Source branch; default: current branch."},
        {"flag": "--new", "description": "Create a new target from the first source."},
    ],
    "examples": [
        "aiw git merge-to main",
        "aiw git merge-to main feature/login",
        "aiw git merge-to --new integration feature/login feature/search",
    ],
}
HELP_FLAGS = {"-h", "--help", "-help", "-?"}


def git(args, cwd, capture=False):
    # Routing variables must not redirect the isolated checkout to the caller's
    # worktree or index. Keep author, signing, hook, and other Git settings.
    environment = os.environ.copy()
    for key in (
        "GIT_DIR", "GIT_COMMON_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE",
        "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES",
        "GIT_NAMESPACE", "GIT_PREFIX",
    ):
        environment.pop(key, None)
    command = ["git", "-C", str(cwd), *args]
    if not capture:
        print(">", " ".join(shlex.quote(arg) for arg in command), file=sys.stderr)
    return subprocess.run(
        command,
        env=environment,
        stdout=subprocess.PIPE if capture else None,
        stderr=subprocess.PIPE if capture else None,
        encoding="utf-8",
        errors="replace",
    )


def output(args, cwd):
    result = git(args, cwd, capture=True)
    if result.returncode:
        raise ValueError(result.stderr.strip() or "Git could not read repository state.")
    return result.stdout.strip()


def branch_ref(name, cwd):
    if name.startswith("-") or git(
        ["check-ref-format", "refs/heads/" + name], cwd, capture=True
    ).returncode:
        raise ValueError(f"Invalid branch name: {name!r}")
    return "refs/heads/" + name


def resolve_source(name, cwd):
    local_ref = branch_ref(name, cwd)
    # Prefer local branches over remote-tracking branches with the same name.
    for ref in (local_ref, "refs/remotes/" + name):
        result = git(["rev-parse", "--verify", ref + "^{commit}"], cwd, capture=True)
        if result.returncode == 0:
            return result.stdout.strip()
    raise ValueError(f"Source branch not found: {name!r}. No fetch was performed.")


def parse_args(argv):
    new = bool(argv and argv[0] == "--new")
    values = argv[1:] if new else argv
    if (
        any(value.startswith("-") for value in values)
        or (new and len(values) < 2)
        or (not new and len(values) not in (1, 2))
    ):
        raise ValueError("Use a target and optional source, or --new with a target and sources.")
    return new, values[0], values[1:]


def preflight(new, target, sources, cwd):
    output(["rev-parse", "--show-toplevel"], cwd)
    target_ref = branch_ref(target, cwd)
    existing = git(["show-ref", "--verify", "--quiet", target_ref], cwd, capture=True)
    if existing.returncode not in (0, 1):
        raise ValueError(existing.stderr.strip() or "Could not check the target branch.")
    if new and existing.returncode == 0:
        raise ValueError(f"Target branch already exists: {target!r}")
    if not new and existing.returncode != 0:
        raise ValueError(f"Target must be an existing local branch: {target!r}")
    if not new:
        records = output(["worktree", "list", "--porcelain", "-z"], cwd).split("\0\0")
        for record in records:
            fields = record.split("\0")
            if "branch " + target_ref in fields:
                path = next(
                    (field[len("worktree "):] for field in fields if field.startswith("worktree ")),
                    "(unknown path)",
                )
                raise ValueError(f"Target branch is already checked out at: {path}")
    if not sources:
        current = git(["symbolic-ref", "--quiet", "--short", "HEAD"], cwd, capture=True)
        if current.returncode != 0:
            raise ValueError("Detached or unborn HEAD: pass an explicit source branch.")
        sources = [current.stdout.strip()]
    return [(source, resolve_source(source, cwd)) for source in sources]


def quote(value):
    value = str(value)
    if os.name == "nt":
        # Recovery commands are copyable in PowerShell on Windows.
        return "'" + value.replace("'", "''") + "'"
    return shlex.quote(value)


def merge_args(target, source, commit):
    return [
        "-c", f"branch.{target}.mergeOptions=",
        "merge", "--no-edit", "--no-squash", "--commit", "--ff",
        "--no-autostash", "-m", f"Merge branch '{source}' into {target}", commit,
    ]


def recovery(path, target, remaining, failed=None):
    print(f"Worktree retained at: {path}", file=sys.stderr)
    print("If a merge is pending, resolve conflicts, stage files, then continue:", file=sys.stderr)
    print(f"  git -C {quote(path)} merge --continue", file=sys.stderr)
    if failed is not None:
        print("If the failed merge did not start, fix its cause and retry:", file=sys.stderr)
        print(
            "  git " + " ".join(quote(arg) for arg in ["-C", str(path), *merge_args(target, *failed)]),
            file=sys.stderr,
        )
    if remaining:
        print("After fixing the failed merge, merge the remaining sources in order:", file=sys.stderr)
        for name, commit in remaining:
            print(
                "  git " + " ".join(quote(arg) for arg in ["-C", str(path), *merge_args(target, name, commit)]),
                file=sys.stderr,
            )
    print("To abort an unfinished merge:", file=sys.stderr)
    print(f"  git -C {quote(path)} merge --abort", file=sys.stderr)
    print("When the worktree is clean and no merge is pending, remove it:", file=sys.stderr)
    print(f"  git worktree remove {quote(path)}", file=sys.stderr)
    print("Earlier successful merges and any newly created target are retained.", file=sys.stderr)


def main(argv):
    if any(arg in HELP_FLAGS for arg in argv):
        core.print_help_meta(META)
        return 0
    try:
        new, target, sources = parse_args(argv)
        cwd = Path.cwd()
        resolved = preflight(new, target, sources, cwd)
    except ValueError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 2
    except OSError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1

    parent = None
    path = None
    pending = resolved[1:] if new else resolved
    remaining = pending
    active = None
    try:
        parent = Path(tempfile.mkdtemp(prefix="aiw-merge-to-"))
        path = parent / "worktree"
        checkout = ["worktree", "add"]
        if new:
            checkout += ["--no-track", "-b", target]
        checkout += ["--", str(path), resolved[0][1] if new else target]
        result = git(checkout, cwd)
        if result.returncode:
            # Checkout can partially succeed (for example, a failing hook).
            # Preserve any nonempty directory and any branch Git created.
            if not path.exists():
                parent.rmdir()
            else:
                recovery(path, target, pending)
            return result.returncode

        for index, (source, commit) in enumerate(pending):
            active = (source, commit)
            remaining = pending[index + 1:]
            # Override per-branch mergeOptions and options that could leave a
            # squash or autostash pending despite a zero exit status.
            result = git(merge_args(target, source, commit), path)
            if result.returncode:
                print(f"error: merge failed for source {source!r}.", file=sys.stderr)
                recovery(path, target, pending[index + 1:], failed=(source, commit))
                return result.returncode
        active = None
        remaining = []
        result = git(["worktree", "remove", "--", str(path)], cwd)
        if result.returncode:
            print("error: merges succeeded, but worktree cleanup failed.", file=sys.stderr)
            recovery(path, target, [])
            return result.returncode
        parent.rmdir()
        print(f"Merged into {target!r}. Your workspace was not switched.")
        return 0
    except (OSError, KeyboardInterrupt) as exc:
        print(f"error: {exc or 'merge-to interrupted'}", file=sys.stderr)
        if path is not None and path.exists():
            recovery(path, target, remaining, failed=active)
        elif parent is not None and parent.exists():
            print(f"Temporary directory remains at: {parent}", file=sys.stderr)
        return 130 if isinstance(exc, KeyboardInterrupt) else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
