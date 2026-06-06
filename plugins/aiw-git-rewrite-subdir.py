#!/usr/bin/env python3
"""aiw-git-rewrite-subdir wrapper

Make a subdirectory become the repository root across full history.
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
    'name': 'aiw-git-rewrite-subdir',
    'short': 'Promote a subdirectory to repo root.',
    'long': (
        'Rewrites full git history so that a subdirectory becomes '
        'the repository root using filter-branch. '
        'Commits outside the subdir are dropped.'
    ),
    'usage': 'aiw-git-rewrite-subdir <subdir> [--force]',
    'args': [{ "flag": "subdir", "description": "Path to the subdirectory to promote to root. This directory must exist in the current HEAD." },
             { "flag": "--force", "description": "By default, the command will ask for confirmation before proceeding. Use --force to skip confirmation." }],
    'examples': [
        'aiw-git-rewrite-subdir src',
        'aiw-git-rewrite-subdir project --force',
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

    subdir = argv[0]

    warning = (
        f'subdir-to-root will rewrite ALL history so that "{subdir}" becomes '
        'the repo root.\n'
        'Commits that did not touch this directory will be dropped.\n'
        'This cannot be undone after a force-push.'
    )

    if not core.git_confirm(warning, argv):
        print('aborted', file=sys.stderr)
        return 1

    return core.run_cmd([
        'git',
        'filter-branch',
        '-f',
        '--subdirectory-filter',
        subdir,
        'HEAD',
    ])


if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)