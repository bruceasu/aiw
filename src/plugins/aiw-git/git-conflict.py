#!/usr/bin/env python3
"""Inspect unresolved conflicts and conflict-resolution state."""

import importlib.util
import os
import sys

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')
spec = importlib.util.spec_from_file_location('aiw_git_core', CORE_PATH)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    'name': 'aiw git conflict',
    'short': 'List and inspect unresolved conflicts.',
    'long': 'Shows unresolved files by default. Use an option to inspect conflict diffs, check the worktree, or view staged changes.',
    'usage': 'aiw git conflict [--diff|--check|--staged]',
    'args': [
        {'flag': '--diff', 'description': 'Show unmerged hunks.'},
        {'flag': '--check', 'description': 'Check for whitespace errors and conflict markers.'},
        {'flag': '--staged', 'description': 'Show staged changes.'},
    ],
    'examples': [
        'aiw git conflict',
        'aiw git conflict --diff',
        'aiw git conflict --check',
    ],
}


def main(argv):
    if any(flag in argv for flag in {'-h', '--help', '-help', '-?'}):
        core.print_help_meta(META)
        return 0
    modes = [flag for flag in ('--diff', '--check', '--staged') if flag in argv]
    unknown = [arg for arg in argv if arg not in {'--diff', '--check', '--staged'}]
    if unknown or len(modes) > 1:
        print('error: choose at most one supported conflict option', file=sys.stderr)
        core.print_help_meta(META)
        return 2
    if modes == ['--diff']:
        return core.run_cmd(['git', 'diff', '--diff-filter=U'])
    if modes == ['--check']:
        return core.run_cmd(['git', 'diff', '--check'])
    if modes == ['--staged']:
        return core.run_cmd(['git', 'diff', '--staged'])

    out = core.git_output(['git', 'diff', '--name-only', '--diff-filter=U'])
    if not out.strip():
        print('No unresolved conflicts found.')
        return 0
    files = out.strip().splitlines()
    print(f'{len(files)} conflicted file(s):\n')
    for path in files:
        print(' ', path)
    print(
        '\nNext steps:\n'
        '  1. Edit each file above and resolve markers.\n'
        '  2. aiw git conflict --check\n'
        '  3. git add <file>\n'
        '  4. git commit'
    )
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
