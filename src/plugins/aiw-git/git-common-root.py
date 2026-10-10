#!/usr/bin/env python3
"""Find the main repository root shared by linked worktrees."""
import os
import subprocess
import sys

import importlib.util

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')

spec = importlib.util.spec_from_file_location('aiw_git_core', CORE_PATH)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    'name': 'aiw git common-root',
    'short': 'Find the main repository root shared by linked worktrees.',
    'long': (
        'Prints the root directory of the main working tree shared by this checkout '
        'and its linked worktrees. It returns the same path when run in the main '
        'working tree.'
    ),
    'usage': 'aiw git common-root',
    'args': [],
    'examples': ['aiw git common-root']
}


def main(argv):
    help_flags = {'-h', '--help', '-help', '-?'}
    if any(flag in argv for flag in help_flags):
        core.print_help_meta(META)
        return 0

    common_dir = subprocess.check_output(
        ['git', 'rev-parse', '--git-common-dir'],
        text=True,
        stderr=subprocess.DEVNULL,
    ).strip()
    if not os.path.isabs(common_dir):
        common_dir = os.path.join(os.getcwd(), common_dir)
    print(os.path.normpath(os.path.dirname(common_dir)))
    return 0


if __name__ == '__main__':
    try:
        sys.exit(main(sys.argv[1:]))
    except subprocess.CalledProcessError as error:
        sys.exit(error.returncode)
