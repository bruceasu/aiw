#!/usr/bin/env python3
"""aiw-git-mv-branch wrapper

Rename a local branch and optionally update the remote.
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
    'name': 'aiw-git-mv-branch',
    'short': 'Rename a git branch.',
    'long': (
        'Renames a local branch and optionally updates '
        'the remote branch name.'
    ),
    'usage': (
        'aiw-git-mv-branch <new-name> '
        '[--remote] [--remote-name <name>]\n'
        'aiw-git-mv-branch <old-name> <new-name> '
        '[--remote] [--remote-name <name>]'
    ),
    'args': [],
    'examples': [
        'aiw-git-mv-branch feature-v2',
        'aiw-git-mv-branch old-name new-name',
        'aiw-git-mv-branch main stable --remote',
        'aiw-git-mv-branch dev release '
        '--remote --remote-name upstream',
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

    delete_remote = core.has_flag(argv, '--remote')

    remote_name = 'origin'
    positional = []

    i = 0
    while i < len(argv):
        arg = argv[i]

        if arg == '--remote':
            pass

        elif arg == '--remote-name':
            if i + 1 >= len(argv):
                print(
                    'error: --remote-name requires a value',
                    file=sys.stderr,
                )
                return 2

            remote_name = argv[i + 1]
            i += 1

        else:
            positional.append(arg)

        i += 1

    old_name = None
    new_name = None

    # ------------------------------------------------------------
    # Rename current branch.
    # ------------------------------------------------------------
    if len(positional) == 1:
        new_name = positional[0]

        try:
            old_name = core.current_branch()

        except Exception as exc:
            print(str(exc), file=sys.stderr)
            return 1

        core.run_cmd([
            'git',
            'branch',
            '-m',
            new_name,
        ])

    # ------------------------------------------------------------
    # Rename explicit branch.
    # ------------------------------------------------------------
    elif len(positional) == 2:
        old_name = positional[0]
        new_name = positional[1]

        core.run_cmd([
            'git',
            'branch',
            '-m',
            old_name,
            new_name,
        ])

    else:
        print(
            'usage: aiw-git-mv-branch <new-name> '
            '[--remote] [--remote-name <name>]\n'
            '       aiw-git-mv-branch <old-name> <new-name> '
            '[--remote] [--remote-name <name>]',
            file=sys.stderr,
        )
        return 2

    # ------------------------------------------------------------
    # Local rename only.
    # ------------------------------------------------------------
    if not delete_remote:
        return 0

    # ------------------------------------------------------------
    # Delete old remote branch.
    # ------------------------------------------------------------
    try:
        core.run_cmd([
            'git',
            'push',
            remote_name,
            '--delete',
            old_name,
        ])

    except Exception as exc:
        print(
            f'warning: could not delete remote branch '
            f'"{old_name}": {exc}',
            file=sys.stderr,
        )

    # ------------------------------------------------------------
    # Push new branch name.
    # ------------------------------------------------------------
    core.run_cmd([
        'git',
        'push',
        remote_name,
        new_name,
    ])

    return 0


if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)