#!/usr/bin/env python3
"""aiw git del-histories-from wrapper

Remove all history before a given ref and rebuild branch history.
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
    'name': 'aiw git del-histories-from',
    'short': 'Delete history before a ref.',
    'long': (
        'Rewrites history so that only commits after a given ref remain, '
        'creating a new root commit and rebasing the branch onto it.'
    ),
    'usage': (
        'aiw git del-histories-from <ref> '
        '[--branch <name>] [--message <msg>] [--force]'
    ),
    'args': [
        {'flag': '<ref>', 'description': 'The reference before which to delete history.'},
        {'flag': '--branch <name>', 'description': 'The branch to modify (default: current branch).'},
        {'flag': '--message <msg>', 'description': 'The commit message for the new root commit.'},
        {'flag': '--force', 'description': 'Skip confirmation prompt.'}
    ],
    'examples': [
        'aiw git del-histories-from HEAD~10',
        'aiw git del-histories-from abc123 --branch main',
        'aiw git del-histories-from v1.0 --message "new begin"',
    ],
}


def main(argv):
    help_flags = {'-h', '--help', '-help', '-?'}

    if not argv:
        core.print_help_meta(META)
        return 2

    if any(f in argv for f in help_flags):
        core.print_help_meta(META)
        return 0

    ref = argv[0]
    rest = argv[1:]

    branch = None
    msg = "new begin"

    i = 0
    while i < len(rest):
        if rest[i] == "--branch" and i + 1 < len(rest):
            branch = rest[i + 1]
            i += 1

        elif rest[i] == "--message" and i + 1 < len(rest):
            msg = rest[i + 1]
            i += 1

        i += 1

    if not branch:
        try:
            branch = core.current_branch()
        except Exception as exc:
            print(str(exc), file=sys.stderr)
            return 1

    warning = (
        f'detach will permanently erase all history before "{ref}" '
        f'on branch "{branch}".\n'
        'The rewritten branch will be left locally '
        "(use 'aiw git push -f' to publish).\n"
        'This cannot be undone after a force-push.'
    )

    if not core.git_confirm(warning, argv):
        print('aborted', file=sys.stderr)
        return 1

    tmp_branch = "_aiw_detach_tmp_"

    # ------------------------------------------------------------
    # 1. Create orphan branch at ref
    # ------------------------------------------------------------
    core.run_cmd([
        'git',
        'checkout',
        '--orphan',
        tmp_branch,
        ref,
    ])

    # ------------------------------------------------------------
    # 2. Create root commit
    # ------------------------------------------------------------
    try:
        core.run_cmd([
            'git',
            'commit',
            '-m',
            msg,
        ])

    except Exception as exc:
        core.run_cmd(['git', 'checkout', branch])
        raise RuntimeError(f"root commit failed: {exc}")

    # ------------------------------------------------------------
    # 3. Rebase old history onto new root
    # ------------------------------------------------------------
    try:
        core.run_cmd([
            'git',
            'rebase',
            '--onto',
            tmp_branch,
            ref,
            branch,
        ])

    except Exception as exc:
        print(
            "rebase paused (conflicts?). Resolve, then:\n"
            "  git rebase --continue   or\n"
            "  git rebase --abort",
            file=sys.stderr,
        )
        return 1

    # ------------------------------------------------------------
    # 4. Cleanup temp branch
    # ------------------------------------------------------------
    core.run_cmd([
        'git',
        'branch',
        '-D',
        tmp_branch,
    ])

    print(
        f'done: history before "{ref}" removed from branch "{branch}".',
        file=sys.stderr,
    )

    return 0


if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)

