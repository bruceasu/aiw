#!/usr/bin/env python3
"""aiw git amend wrapper

Amend the last commit, optionally including all working-tree changes.
"""
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
    'name': 'aiw git amend',
    'short': 'Amend the last commit after confirmation.',
    'long': (
        'By default, amends using staged changes after confirmation. --all/-a also '
        'includes tracked, untracked, and deleted working-tree paths; you can remove '
        'paths from that list before confirming. --force/-f skips confirmation.'
    ),
    'usage': 'aiw git amend [--all|-a] [--force|-f] [git commit options...]',
    'args': [
        {'flag': '--all, -a', 'description': 'Include all tracked and untracked changes.'},
        {'flag': '--force, -f', 'description': 'Skip confirmation and include all selected changes.'},
        {'flag': '[git commit options...]', 'description': 'Additional options passed to git commit --amend.'},
    ],
    'examples': [
        'aiw git amend',
        'aiw git amend --all',
        'aiw git amend -a --force --no-edit',
    ],
}


def changed_paths(all_changes):
    if all_changes:
        tracked = subprocess.check_output(
            ['git', 'diff', '--name-only', '--no-renames', '-z', 'HEAD'],
            stderr=subprocess.DEVNULL,
        )
        untracked = subprocess.check_output(
            ['git', 'ls-files', '--others', '--exclude-standard', '-z'],
            stderr=subprocess.DEVNULL,
        )
        raw_paths = tracked.split(b'\0') + untracked.split(b'\0')
    else:
        staged = subprocess.check_output(
            ['git', 'diff', '--cached', '--name-only', '--no-renames', '-z'],
            stderr=subprocess.DEVNULL,
        )
        raw_paths = staged.split(b'\0')
    return sorted({os.fsdecode(path) for path in raw_paths if path})


def choose_paths(paths):
    if not paths:
        return paths
    print('Files to include:')
    for index, path in enumerate(paths, start=1):
        print(f'  {index}. {path!r}')
    answer = input(
        'Enter file numbers to remove (comma-separated), Enter keeps all, or q cancels: '
    ).strip()
    if answer.lower() in {'q', 'quit', 'cancel'}:
        return None
    if not answer:
        return paths
    try:
        remove = {int(part.strip()) for part in answer.split(',')}
    except ValueError:
        print('Invalid selection; amend cancelled.', file=sys.stderr)
        return None
    if any(number < 1 or number > len(paths) for number in remove):
        print('Selection out of range; amend cancelled.', file=sys.stderr)
        return None
    return [path for index, path in enumerate(paths, start=1) if index not in remove]


def main(argv):
    help_flags = {'-h', '--help', '-help', '-?'}
    if any(flag in argv for flag in help_flags):
        core.print_help_meta(META)
        return 0

    all_changes = False
    force = False
    commit_args = []
    for arg in argv:
        if arg in {'--all', '-a'}:
            all_changes = True
        elif arg in {'--force', '-f'}:
            force = True
        else:
            commit_args.append(arg)

    try:
        paths = changed_paths(all_changes)
    except subprocess.CalledProcessError:
        print('Unable to list changed files; amend cancelled.', file=sys.stderr)
        return 1

    if all_changes and not force:
        paths = choose_paths(paths)
        if paths is None:
            print('amend cancelled', file=sys.stderr)
            return 1

    if paths:
        print('Selected files:')
        for path in paths:
            print(f'  {path!r}')
    else:
        print('No files selected; this will amend the commit without adding file changes.')

    if not force and not core.git_confirm(
        'Amend the last commit with the selected changes?', []
    ):
        print('amend cancelled', file=sys.stderr)
        return 1

    if all_changes:
        if paths:
            core.run_cmd(['git', 'add', '-A', '--'] + paths)
        cmd = ['git', 'commit', '--amend', '--only']
        cmd.extend(commit_args)
        if paths:
            cmd.extend(['--'] + paths)
        return core.run_cmd(cmd)

    return core.run_cmd(['git', 'commit', '--amend'] + commit_args)


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
