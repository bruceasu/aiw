#!/usr/bin/env python3
"""Find the primary repository root shared by linked worktrees."""
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
    'name': 'aiw git root',
    'short': 'Find the primary Git repository root.',
    'long': 'Find the repository root shared by the current checkout and linked worktrees.',
    'usage': 'aiw git root',
    'args': [],
    'examples': ['aiw git root']
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
