#!/usr/bin/env python3
"""aiw git move-commits wrapper

Move accidental commits from the current branch to a new branch.
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
    'name': 'aiw git move-commits',
    'short': 'Move commits after a chosen point onto a new branch.',
    'long': (
        'Creates <new-branch> at the current HEAD, hard-resets the original branch '
        'to [reset-to] (default: HEAD^), then switches to <new-branch>. The moved '
        'commits remain on the new branch. Uncommitted working-tree changes are '
        'discarded; --force skips the confirmation prompt.'
    ),
    'usage': (
        'aiw git move-commits <new-branch> '
        '[reset-to] [--force]'
    ),
    'args': [
        {'flag': '<new-branch>', 'description': 'Name of the branch to create at the current HEAD.'},
        {'flag': '[reset-to]', 'description': 'Where to reset the original branch (default: HEAD^).'},
        {'flag': '--force', 'description': 'Skip confirmation prompt.'}
    ],
    'examples': [
        'aiw git move-commits feature/login',
        'aiw git move-commits hotfix HEAD~3',
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

    new_branch = argv[0]

    reset_to = 'HEAD^'

    if len(argv) >= 2 and argv[1] != '--force':
        reset_to = argv[1]

    if not core.git_confirm(
        (
            f'This will reset the current branch to "{reset_to}". '
            'Uncommitted changes will be lost. '
            'Add --force to skip.'
        ),
        argv,
    ):
        print('aborted', file=sys.stderr)
        return 1

    # ------------------------------------------------------------
    # Step 1: create new branch at current HEAD.
    # ------------------------------------------------------------
    core.run_cmd([
        'git',
        'branch',
        new_branch,
    ])

    # ------------------------------------------------------------
    # Step 2: reset current branch back.
    # ------------------------------------------------------------
    core.run_cmd([
        'git',
        'reset',
        '--hard',
        reset_to,
    ])

    # ------------------------------------------------------------
    # Step 3: switch to the new branch.
    # ------------------------------------------------------------
    core.run_cmd([
        'git',
        'checkout',
        new_branch,
    ])

    return 0


if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)

