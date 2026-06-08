#!/usr/bin/env python3
"""aiw git un-add wrapper

Unstage files (reverse git add).
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
    'name': 'aiw git un-add',
    'short': 'Unstage files from git index.',
    'long': (
        'Reverses git add by restoring files from the index. '
        'If no files are provided, unstages everything.'
    ),
    'usage': 'aiw git un-add [file...]',
    'args': [],
    'examples': [
        'aiw git un-add',
        'aiw git un-add file1.py file2.py',
    ],
}


def main(argv):
    help_flags = {'-h', '--help', '-help', '-?'}

    if any(f in argv for f in help_flags):
        core.print_help_meta(META)
        return 0

    # ------------------------------------------------------------
    # No args: unstage everything
    # ------------------------------------------------------------
    if not argv:
        return core.run_cmd([
            'git',
            'restore',
            '--staged',
            '.',
        ])

    # ------------------------------------------------------------
    # Unstage specific files
    # ------------------------------------------------------------
    return core.run_cmd([
        'git',
        'restore',
        '--staged',
        *argv,
    ])


if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)

