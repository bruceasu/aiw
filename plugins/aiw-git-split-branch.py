#!/usr/bin/env python3
"""aiw-git-split-branchwrapper

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
    'name': 'aiw git-mv-current-commit-to-new-branch',
    'short': 'Move the current commit to a new branch.',
    'long': (
        'Creates a new branch at the current HEAD, resets the '
        'current branch backward, then switches to the new branch.'
    ),
    'usage': (
        'aiw git-split-branch<new-branch> '
        '[reset-to] [--force]'
    ),
    'args': [
        {'flag': '<new-branch>', 'description': 'The name of the new branch to create.'},
        {'flag': '[reset-to]', 'description': 'The reference to reset the current branch to (default: HEAD^).'},
        {'flag': '--force', 'description': 'Skip confirmation prompt.'}
    ],
    'examples': [
        'aiw git-split-branchfeature/login',
        'aiw git-split-branchhotfix HEAD~3',
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