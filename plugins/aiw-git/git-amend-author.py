#!/usr/bin/env python3
"""aiw git change-author wrapper

Change the author on the last commit or advise history rewrite.
"""
import sys
import os
import importlib.util

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')
spec = importlib.util.spec_from_file_location('aiw_git_core', CORE_PATH)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    'name': 'aiw git amend-author',
    'short': 'Change the author on the last commit (or rewrite history).',
    'long': 'Convenience helper to amend the last commit author with --author "Name <email>". For wide rewrites use proper history-rewrite tools.',
    'usage': 'aiw git amend-author "Name <email>" [--all --filter-name <old> ...] [--force]',
    'args': [],
    'examples': ['aiw git amend-author "Alice <alice@example.com>"']
}


def main(argv):
    """
    Amend the last commit author or rewrite history.

    Usage:
        aiw git amend-author <Name> <email>
            [--all --filter-name <old> ...]
            [--force]

    Without --all:
        Amend only the last commit author.

    With --all:
        Rewrite all commits whose author/committer name matches
        one of the --filter-name values.

    鈿? --all rewrites history.
    """
    if len(argv) < 2:
        core.print_help_meta(META)
        return 0
    help_flags = {'-h', '--help', '-help', '-?'}
    if any(f in argv for f in help_flags):
        core.print_help_meta(META)
        return 0
    name = argv[0]
    email = argv[1]
    rest = argv[2:]

    # ------------------------------------------------------------------
    # Amend only the last commit.
    # ------------------------------------------------------------------
    if not core.has_flag(rest, "--all"):

        if not core.git_confirm(
            f'This will amend the last commit author to "{name} <{email}>". '
            f"Add --force to skip this prompt.",
            rest,
        ):
            print("aborted", file=os.sys.stderr)
            return

        author = f"{name} <{email}>"

        core.run_cmd([
            "git",
            "commit",
            "--amend",
            "--author",
            author,
            "--no-edit",
        ])
        return 0

    # ------------------------------------------------------------------
    # Rewrite full history.
    # ------------------------------------------------------------------

    filter_names = []

    i = 0
    while i < len(rest):
        if rest[i] == "--filter-name" and i + 1 < len(rest):
            filter_names.append(rest[i + 1])
            i += 1
        i += 1

    if not filter_names:
        raise ValueError(
            "--all requires at least one --filter-name <old-name>"
        )

    warning = (
        "change-author --all will rewrite ALL commits whose "
        f"author/committer matches {filter_names}\n"
        f'and replace them with "{name} <{email}>".\n'
        "This permanently rewrites history. "
        "Collaborators must re-clone or reset."
    )

    if not core.git_confirm(warning, rest):
        print("aborted", file=os.sys.stderr)
        return 1

    # Build shell condition.
    conditions = []

    for old_name in filter_names:
        conditions.append(f'[ "$GIT_AUTHOR_NAME" = "{old_name}" ]')
        conditions.append(f'[ "$GIT_COMMITTER_NAME" = "{old_name}" ]')

    if_expr = " || ".join(conditions)

    # Build env-filter script.
    script = f'''
an="$GIT_AUTHOR_NAME"
am="$GIT_AUTHOR_EMAIL"
cn="$GIT_COMMITTER_NAME"
cm="$GIT_COMMITTER_EMAIL"

if {if_expr}
then
    an="{name}"
    am="{email}"
    cn="{name}"
    cm="{email}"
fi

export GIT_AUTHOR_NAME="$an"
export GIT_AUTHOR_EMAIL="$am"
export GIT_COMMITTER_NAME="$cn"
export GIT_COMMITTER_EMAIL="$cm"
'''.strip()

    core.run_cmd([
        "git",
        "filter-branch",
        "-f",
        "--env-filter",
        script,
        "HEAD",
    ])
    return 0

if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)


