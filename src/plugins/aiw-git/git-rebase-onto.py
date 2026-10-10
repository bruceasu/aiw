#!/usr/bin/env python3
"""aiw git rebase-onto wrapper

Move a branch onto a new base using git rebase --onto.
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
    'name': 'aiw git rebase-onto',
    'short': 'Replay a branch’s commits onto a different base.',
    'long': (
        'Takes commits reachable from [branch] after <old-base> and replays them '
        'on top of <new-base>. If [branch] is omitted, the current branch is used. '
        'This rewrites commit IDs. The command asks for confirmation by default; '
        '--force only skips that prompt and does not force-push.'
    ),
    'usage': (
        'aiw git rebase-onto '
        '<new-base> <old-base> [branch] [--force]'
    ),
    'args': [
        {'flag': '<new-base>', 'description': 'The commit or branch to place the rebased commits on.'},
        {'flag': '<old-base>', 'description': 'The old upstream/base; commits after it are replayed.'},
        {'flag': '[branch]', 'description': 'The branch to rebase (default: current branch).'},
        {'flag': '--force', 'description': 'Skip the confirmation prompt; does not force-push.'},
    ],
    'examples': [
        'aiw git rebase-onto main feature/base feature/work',
        'aiw git rebase-onto main old-main',
        'aiw git rebase-onto main old-main feature/work --force',
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

    if len(argv) < 2:
        print(
            'usage: aiw git rebase-onto '
            '<new-base> <old-base> [branch] [--force]',
            file=sys.stderr,
        )
        return 2

    new_base = argv[0]
    old_base = argv[1]

    if not core.git_confirm(
        (
            f'This will rebase --onto "{new_base}" '
            f'from "{old_base}". '
            'Add --force to skip this prompt.'
        ),
        argv,
    ):
        print('aborted', file=sys.stderr)
        return 1

    cmd = [
        'git',
        'rebase',
        '--onto',
        new_base,
        old_base,
    ]

    # Optional branch argument.
    if len(argv) >= 3 and argv[2] != '--force':
        cmd.append(argv[2])

    core.run_cmd(cmd)

    return 0


if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)

