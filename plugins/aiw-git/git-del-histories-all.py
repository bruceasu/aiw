#!/usr/bin/env python3
"""aiw git del-histories-all wrapper

Squash entire repository history into a single root commit,
optionally force-push to remote and re-link branch.
"""

import sys
import os
import importlib.util

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')

spec = importlib.util.spec_from_file_location(
    'aiw_git_core',
    CORE_PATH,
)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    'name': 'aiw git del-histories-all',
    'short': 'Delete all git history from the repository.',
    'long': (
        'Permanently erases all commit history from the repository, '
        'force-pushes to remote, and re-links the local branch.'
    ),
    'usage': (
        'aiw git del-histories-all '
        '[--branch <name>] [--remote <name>] '
        '[--message <msg>] [--force]'
    ),
    'args': [
        {'flag': '[--branch <name>]', 'description': 'The branch to delete history from (default: current branch).'},
        {'flag': '[--remote <name>]', 'description': 'The remote to force-push to (default: origin).'},
        {'flag': '[--message <msg>]', 'description': 'The commit message for the new initial commit.'},
        {'flag': '[--force]', 'description': 'Force the operation without confirmation.'}
    ],
    'examples': [
        'aiw git del-histories-all',
        'aiw git del-histories-all --branch main',
        'aiw git del-histories-all --remote origin --message "init"',
    ],
}


def main(argv):
    help_flags = {'-h', '--help', '-help', '-?'}

    if any(f in argv for f in help_flags):
        core.print_help_meta(META)
        return 0

    branch = None
    remote = "origin"
    msg = "initial commit"

    i = 0
    while i < len(argv):
        arg = argv[i]

        if arg == "--branch" and i + 1 < len(argv):
            branch = argv[i + 1]
            i += 1

        elif arg == "--remote" and i + 1 < len(argv):
            remote = argv[i + 1]
            i += 1

        elif arg == "--message" and i + 1 < len(argv):
            msg = argv[i + 1]
            i += 1

        i += 1

    # ------------------------------------------------------------
    # Auto-detect branch if not provided
    # ------------------------------------------------------------
    if not branch:
        try:
            out = core.git_output([
                "git",
                "rev-parse",
                "--abbrev-ref",
                "HEAD",
            ])
            branch = out.strip()
        except Exception:
            branch = "master"

        if not branch or branch == "HEAD":
            branch = "master"

    warning = (
        f'delete-all-histories will PERMANENTLY erase all commit history '
        f'on branch "{branch}"\n'
        f'and force-push to remote "{remote}".\n'
        'All collaborators must re-clone or reset their local copies.\n'
        'This cannot be undone once pushed.'
    )

    if not core.git_confirm(warning, argv):
        print('aborted', file=sys.stderr)
        return 1

    orphan_branch = "_aiw_delete_histories_tmp_"

    # ------------------------------------------------------------
    # 1. Create orphan branch
    # ------------------------------------------------------------
    core.run_cmd([
        "git",
        "checkout",
        "--orphan",
        orphan_branch,
    ])

    # ------------------------------------------------------------
    # 2. Stage all files
    # ------------------------------------------------------------
    try:
        core.run_cmd(["git", "add", "-A"])
    except Exception as exc:
        core.run_cmd(["git", "checkout", branch])
        print(f"git add failed: {exc}", file=sys.stderr)
        return 1

    # ------------------------------------------------------------
    # 3. Create initial commit
    # ------------------------------------------------------------
    try:
        core.run_cmd([
            "git",
            "commit",
            "-m",
            msg,
        ])
    except Exception as exc:
        core.run_cmd(["git", "checkout", branch])
        print(f"git commit failed: {exc}", file=sys.stderr)
        return 1

    # ------------------------------------------------------------
    # 4. Delete original branch
    # ------------------------------------------------------------
    core.run_cmd([
        "git",
        "branch",
        "-D",
        branch,
    ])

    # ------------------------------------------------------------
    # 5. Rename orphan branch to original
    # ------------------------------------------------------------
    core.run_cmd([
        "git",
        "branch",
        "-m",
        branch,
    ])

    # ------------------------------------------------------------
    # 6. Force push to remote
    # ------------------------------------------------------------
    if not core.has_remote(remote):
        print(
            f'note: remote "{remote}" not found; skipping push.',
            file=sys.stderr,
        )
        return 0

    core.run_cmd([
        "git",
        "push",
        "-f",
        remote,
        branch,
    ])

    # ------------------------------------------------------------
    # 7. Re-link upstream
    # ------------------------------------------------------------
    core.run_cmd([
        "git",
        "branch",
        f"--set-upstream-to={remote}/{branch}",
    ])

    return 0


if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)

