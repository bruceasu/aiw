#!/usr/bin/env python3
"""aiw-git-track wrapper

Fetch all remotes and checkout a remote branch as a tracking branch,
or set upstream / push for an existing local branch.
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
    'name': 'aiw git-track',
    'short': 'Track or set upstream for a branch.',
    'long': (
        'Fetches all remotes and checks out a remote branch as a local tracking branch '
        'if --track is used, otherwise sets upstream for an existing local branch '
        'or pushes with -u.'
    ),
    'usage': 'aiw git-track <branch> [remote] [--push] [--track]',
    'args': [
        {'flag': '<branch>', 'description': 'The branch to track or set upstream for.'},
        {'flag': '[remote]', 'description': 'The remote to track the branch from (default: origin).'},
        {'flag': '--push', 'description': 'Push the branch to the remote and set upstream.'},
        {'flag': '--track', 'description': 'Create a new local tracking branch from the remote branch.'}
    ],
    'examples': [
        'aiw git-track feature/login',
        'aiw git-track main origin --push',
        'aiw git-track develop origin --track',
    ],
}


def main(argv):
    help_flags = {'-h', '--help', '-help', '-?'}

    if not argv or any(f in argv for f in help_flags):
        core.print_help_meta(META)
        return 0

    branch = argv[0]
    remote = 'origin'
    push = core.has_flag(argv, '--push')
    track = core.has_flag(argv, '--track')

    if len(argv) >= 2:
        # Skip known flags
        for i, arg in enumerate(argv[1:], start=1):
            if arg in ('--push', '--track'):
                continue
            remote = arg
            break

    if track:
        # Track remote branch
        core.run_cmd(['git', 'fetch', '--all'])
        core.run_cmd(['git', 'checkout', '--track', f'{remote}/{branch}'])
        return 0

    # Default: local branch exists, set upstream or push
    if not core.branch_exists(branch):
        print(f'Error: Local branch "{branch}" does not exist. Use --track to create it.', file=sys.stderr)
        return 1

    if push:
        return core.run_cmd(['git', 'push', '-u', remote, branch])

    return core.run_cmd(['git', 'branch', f'--set-upstream-to={remote}/{branch}', branch])


if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)