#!/usr/bin/env python3
"""aiw git subdir-to-root wrapper

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
    'name': 'aiw git subdir-to-root',
    'short': 'Make a subdirectory the repository root by rewriting the current branch.',
    'long': (
        'Run this inside the source repository. Rewrites history reachable from the '
        'current branch so files under <subdir> become the repository root. Files '
        'outside that directory and commits that do not affect it are dropped; '
        'commit IDs change. This is a standalone repository conversion, not a '
        'preparation step for aiw git merge-repo. Confirmation is required unless '
        '--force is supplied.'
    ),
    'usage': 'aiw git subdir-to-root <subdir> [--force]',
    'args': [
        {'flag': '<subdir>', 'description': 'Path relative to the repository root; must exist in the current commit.'},
        {'flag': '--force', 'description': 'Skip the confirmation prompt.'},
    ],
    'examples': [
        'aiw git subdir-to-root packages/service-a',
        'aiw git subdir-to-root packages/service-a --force',
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

