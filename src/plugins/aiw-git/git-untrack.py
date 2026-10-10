#!/usr/bin/env python3
"""aiw git untrack wrapper

Unset the upstream branch for a local branch.
"""
import sys
import os
import importlib.util

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')
spec = importlib.util.spec_from_file_location('aiw_git_core', CORE_PATH)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    'name': 'aiw git untrack',
    'short': 'Unset the upstream branch for a local branch.',
    'long': 'Removes the configured upstream from the specified local branch, or the current branch by default.',
    'usage': 'aiw git untrack [branch]',
    'args': [
        {'flag': '[branch]', 'description': 'The local branch whose upstream to unset (default: current branch).'}
    ],
    'examples': [
        'aiw git untrack',
        'aiw git untrack feature/login',
    ],
}


def main(argv):
    help_flags = {'-h', '--help', '-help', '-?'}
    if any(flag in argv for flag in help_flags):
        core.print_help_meta(META)
        return 0
    if len(argv) > 1:
        print(f'usage: {META["usage"]}', file=sys.stderr)
        return 2
    return core.run_cmd(['git', 'branch', '--unset-upstream'] + argv)


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
