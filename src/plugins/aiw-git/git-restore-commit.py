#!/usr/bin/env python3
"""Interactively reset HEAD to a commit from the reflog."""

import subprocess
import sys
import os
import importlib.util

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')
spec = importlib.util.spec_from_file_location('aiw_git_core', CORE_PATH)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    'name': 'aiw git restore-commit',
    'short': 'Select a reflog commit and move the current branch to it.',
    'long': (
        'Lists entries from the HEAD reflog. Selecting one moves the current branch '
        'to that commit and hard-resets the index and working tree, discarding '
        'uncommitted changes. With --keep/-k, the working-tree files are preserved '
        'and their changes become unstaged. Confirmation is required.'
    ),
    'usage': 'aiw git restore-commit [--keep|-k]',
    'args': [
        {'flag': '--keep, -k', 'description': 'Preserve working-tree files; changes become unstaged.'},
    ],
    'examples': [
        'aiw git restore-commit',
        'aiw git restore-commit --keep',
    ],
}


def main(argv):
    help_flags = {'-h', '--help', '-help', '-?'}
    if any(flag in argv for flag in help_flags):
        core.print_help_meta(META)
        return 0
    if any(arg not in {'--keep', '-k'} for arg in argv):
        print(f'usage: {META["usage"]}', file=sys.stderr)
        return 2
    keep = any(arg in {'--keep', '-k'} for arg in argv)

    try:
        subprocess.check_output(
            ['git', 'symbolic-ref', '--quiet', '--short', 'HEAD'],
            stderr=subprocess.DEVNULL,
        )
    except subprocess.CalledProcessError:
        print('restore-commit requires a checked-out branch; HEAD is detached.', file=sys.stderr)
        return 1

    try:
        output = subprocess.check_output(
            ['git', 'reflog', '--date=local', '--format=%H%x09%gD%x09%gs'],
            stderr=subprocess.DEVNULL,
        )
    except subprocess.CalledProcessError:
        print('Unable to read the HEAD reflog.', file=sys.stderr)
        return 1

    entries = []
    for line in output.decode('utf-8', errors='replace').splitlines():
        parts = line.split('\t', 2)
        if len(parts) == 3:
            entries.append(parts)
    if not entries:
        print('The HEAD reflog is empty.', file=sys.stderr)
        return 1

    print('Select a reflog entry (newest first):')
    for index, (commit, selector, subject) in enumerate(entries, start=1):
        print(f'{index:>3}. {selector} {commit[:12]} {subject}')
    try:
        answer = input('Entry number, or q to cancel: ').strip()
    except EOFError:
        print('restore-commit cancelled', file=sys.stderr)
        return 1
    if answer.lower() in {'q', 'quit', 'cancel'}:
        print('restore-commit cancelled', file=sys.stderr)
        return 1
    try:
        selected = int(answer)
    except ValueError:
        print('Invalid entry number.', file=sys.stderr)
        return 2
    if selected < 1 or selected > len(entries):
        print('Entry number is out of range.', file=sys.stderr)
        return 2

    commit, selector, subject = entries[selected - 1]
    mode = '--mixed' if keep else '--hard'
    prompt = (
        f'Reset to {selector} ({commit[:12]}: {subject}) with {mode}? '
        + ('Working-tree files will be kept, but changes will become unstaged.' if keep
           else 'This discards all uncommitted index and working-tree changes.')
    )
    try:
        confirmed = core.git_confirm(prompt, [])
    except EOFError:
        confirmed = False
    if not confirmed:
        print('restore-commit cancelled', file=sys.stderr)
        return 1

    return core.run_cmd(['git', 'reset', mode, commit])


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
