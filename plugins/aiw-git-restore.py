#!/usr/bin/env python3
"""aiw-git-restore wrapper

Restore a file to its last committed version.
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
    'name': 'aiw-git-restore',
    'short': 'Restore files from the working tree or a commit.',
    'long': (
        'Restores one or more files either from the current working tree '
        'or from a specific commit using git restore.'
    ),
    'usage': (
        'aiw-git-restore <file...>\n'
        'aiw-git-restore <file...> --from <commit>'
    ),
    'args': [
        {'flag': '<file...>', 'description': 'The file(s) to restore.'},
        {'flag': '--from', 'description': 'The commit from which to restore the file(s).'}
    ],
    'examples': [
        'aiw-git-restore main.py',
        'aiw-git-restore a.txt b.txt',
        'aiw-git-restore main.py --from HEAD~1',
        'aiw-git-restore app.py --from abc1234^',
    ],
}


def main(argv):
    help_flags = {'-h', '--help', '-help', '-?'}

    if not argv or any(f in argv for f in help_flags):
        core.print_help_meta(META)
        return 0 if argv else 2

    source = None
    files = []

    i = 0
    while i < len(argv):
        arg = argv[i]

        if arg == '--from':
            if i + 1 >= len(argv):
                print(
                    'error: --from requires a commit',
                    file=sys.stderr,
                )
                return 2

            source = argv[i + 1]
            i += 1

        else:
            files.append(arg)

        i += 1

    if not files:
        print(
            'usage: aiw-git-restore <file...> [--from <commit>]',
            file=sys.stderr,
        )
        return 2

    cmd = ['git', 'restore']

    if source:
        cmd.append(f'--source={source}')

    cmd.extend(files)

    core.run_cmd(cmd)

    return 0

if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)
