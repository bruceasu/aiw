#!/usr/bin/env python3
"""aiw git undo wrapper

Implements undo behavior and help metadata.
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
    'name': 'aiw git undo',
    'short': 'Undo the last commit while keeping changes staged (or discard with --hard).',
    'long': 'Moves HEAD to the previous commit. By default changes remain staged with --soft. Use --hard to discard working-tree and staged changes (dangerous).',
    'usage': 'aiw git undo [--hard] [--force]',
    'args': [
        {'flag': '--hard', 'description': 'Discard working-tree and staged changes.'},
        {'flag': '--force', 'description': 'Skip the confirmation prompt for --hard.'},
    ],
    'examples': ['aiw git undo', 'aiw git undo --hard --force']
}


def main(argv):
    help_flags = {'-h', '--help', '-help', '-?'}
    if any(f in argv for f in help_flags):
        core.print_help_meta(META)
        return 0
    if any(arg not in {'--hard', '--force'} for arg in argv):
        print('error: undo accepts only --hard and --force', file=sys.stderr)
        core.print_help_meta(META)
        return 2
    if '--hard' in argv:
        if not core.git_confirm('--hard will permanently discard all working-tree changes.', argv):
            print('aborted', file=sys.stderr)
            return 0
        return core.run_cmd(['git', 'reset', '--hard', 'HEAD~1'])
    return core.run_cmd(['git', 'reset', '--soft', 'HEAD~1'])


if __name__ == '__main__':
    try:
        rc = main(sys.argv[1:])
        sys.exit(rc)
    except SystemExit:
        raise


