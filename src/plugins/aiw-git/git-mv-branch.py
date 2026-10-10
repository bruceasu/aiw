#!/usr/bin/env python3
"""Rename a local branch and its matching branch on a remote."""

import importlib.util
import os
import subprocess
import sys

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')
spec = importlib.util.spec_from_file_location('aiw_git_core', CORE_PATH)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    'name': 'aiw git mv-branch',
    'short': 'Rename a local branch and its remote branch together.',
    'long': (
        'Renames an existing local branch and its branch on a remote. The new '
        'name must not already exist locally or on the selected remote. The '
        'command first pushes the old local branch under the new remote name '
        'and sets that branch as upstream, then renames the local branch, and '
        'finally deletes the old remote branch. The remote defaults to origin. '
        'These steps are not atomic: if local renaming fails, both remote names '
        'remain; if deleting the old remote branch fails, the local branch and '
        'new remote branch use the new name while the old remote branch remains.'
    ),
    'usage': 'aiw git mv-branch <old-name> <new-name> [--remote-name <remote>]',
    'args': [
        {'flag': '<old-name>', 'description': 'Existing local and remote branch name to rename.'},
        {'flag': '<new-name>', 'description': 'New name; must not already exist locally or remotely.'},
        {'flag': '--remote-name <remote>', 'description': 'Remote to update (default: origin).'},
    ],
    'examples': [
        'aiw git mv-branch old-name new-name',
        'aiw git mv-branch dev release --remote-name upstream',
    ],
}


def usage_error(message):
    print(f'error: {message}', file=sys.stderr)
    core.print_help_meta(META)
    return 2


def main(argv):
    if any(flag in argv for flag in {'-h', '--help', '-help', '-?'}):
        core.print_help_meta(META)
        return 0

    remote_name = 'origin'
    positional = []
    i = 0
    while i < len(argv):
        arg = argv[i]
        if arg == '--remote-name':
            i += 1
            if i >= len(argv):
                return usage_error('--remote-name requires a value')
            remote_name = argv[i]
        elif arg.startswith('-'):
            return usage_error(f'unknown option: {arg}')
        else:
            positional.append(arg)
        i += 1

    if len(positional) != 2:
        return usage_error('provide both the old and new branch names')

    old_name, new_name = positional
    if old_name == new_name:
        return usage_error('old and new branch names must be different')
    if not core.has_remote(remote_name):
        print(f'error: remote "{remote_name}" does not exist', file=sys.stderr)
        return 2

    try:
        local_match = core.git_output(['git', 'branch', '--list', new_name]).strip()
    except subprocess.CalledProcessError:
        print('error: could not inspect local branches', file=sys.stderr)
        return 1
    if local_match:
        print(f'error: local branch "{new_name}" already exists', file=sys.stderr)
        return 1

    new_ref = f'refs/heads/{new_name}'
    try:
        remote_refs = core.git_output(['git', 'ls-remote', '--heads', remote_name, new_ref])
    except subprocess.CalledProcessError:
        print(f'error: could not inspect remote "{remote_name}"', file=sys.stderr)
        return 1
    if any(line.split()[-1] == new_ref for line in remote_refs.splitlines() if line.split()):
        print(f'error: remote branch "{new_name}" already exists', file=sys.stderr)
        return 1

    # Publish the destination first and point the local branch's upstream at it.
    core.run_cmd([
        'git', 'push', '--set-upstream', remote_name,
        f'{old_name}:{new_name}',
    ])

    try:
        core.run_cmd(['git', 'branch', '-m', old_name, new_name])
    except SystemExit as exc:
        print(
            f'warning: remote branch "{new_name}" was created, but local '
            f'branch "{old_name}" could not be renamed. Remote branch '
            f'"{old_name}" was left intact.',
            file=sys.stderr,
        )
        return exc.code if isinstance(exc.code, int) else 1

    try:
        core.run_cmd(['git', 'push', remote_name, '--delete', old_name])
    except SystemExit as exc:
        print(
            f'warning: local and new remote branches are named "{new_name}", '
            f'but old remote branch "{old_name}" could not be deleted.',
            file=sys.stderr,
        )
        return exc.code if isinstance(exc.code, int) else 1

    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
