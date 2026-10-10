#!/usr/bin/env python3
"""aiw git unindex wrapper

Remove paths from the index while keeping working-tree copies.
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
    'name': 'aiw git unindex',
    'short': 'Remove paths from the index while keeping working-tree copies.',
    'long': 'Runs git rm --cached so the paths remain in the working tree but are removed from the Git index.',
    'usage': 'aiw git unindex <path> [path...]',
    'args': [
        {'flag': '<path>', 'description': 'A file or directory to remove from the index.'}
    ],
    'examples': ['aiw git unindex path/to/file'],
}


def main(argv):
    help_flags = {'-h', '--help', '-help', '-?'}
    if any(flag in argv for flag in help_flags):
        core.print_help_meta(META)
        return 0
    if not argv:
        print(f'usage: {META["usage"]}', file=sys.stderr)
        return 2
    return core.run_cmd(['git', 'rm', '--cached', '--'] + argv)


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
