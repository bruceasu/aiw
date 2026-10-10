#!/usr/bin/env python3
"""aiw git change-remote wrapper

Change the remote for the current branch.
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
    'name': 'aiw git change-remote',
    'short': 'Change the remote for the current branch.',
    'long': 'Changes the remote for the current branch.',
    'usage': 'aiw git change-remote <url> [remote]',
    'args': [
        {'flag': '<url>', 'description': 'The URL of the repository to change the remote to.'},
        {'flag': '[remote]', 'description': 'The name of the remote (default: origin).'}
    ],
    'examples': ['aiw git change-remote https://github.com/user/repo.git origin']
}


def main(argv):
    help_flags = {'-h', '--help', '-help', '-?'}
    if any(f in argv for f in help_flags):
        core.print_help_meta(META)
        return 0
    if not argv:
        print('usage: aiw git change-remote <url> [remote]', file=sys.stderr)
        return 2
    url = argv[0]
    remote = argv[1] if len(argv) > 1 else 'origin'
    core.run_cmd(['git', 'remote', 'set-url', remote, url])
    return 0

if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)


