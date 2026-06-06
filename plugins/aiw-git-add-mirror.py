#!/usr/bin/env python3
"""aiw-git-ca wrapper

Amend last commit including all changes.
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
    'name': 'aiw git-add-mirror',
    'short': 'Add a new mirror for the current repository.',
    'long': 'Adds a new mirror to the current repository.',
    'usage': 'aiw git-add-mirror <url> [mirror] [--push [branch]]',
    'args': [
        {'flag': '<url>', 'description': 'The URL of the repository to add as a mirror.'},
        {'flag': '[mirror]', 'description': 'The name of the mirror (default: mirror).'},
        {'flag': '--push [branch]', 'description': 'Push to the mirror after adding it.'}
    ],
    'examples': ['aiw git-add-mirror https://github.com/user/repo.git origin']
}


def main(argv):
    if not argv:
        print('usage: aiw git-add-mirror <url> [mirror]', file=sys.stderr)
        return 2
    if len(argv) == 0:
        core.print_help_meta(META)
        return 2

    help_flags = {'-h', '--help', '-help', '-?'}
    if any(f in argv for f in help_flags):
        core.print_help_meta(META)
        return 0
    
    push = core.has_flag(argv, '--push')
    positional = []
    for arg in argv:
        if arg.startswith('--'):
            continue
        positional.append(arg)
    
    url = positional[0] if positional else None
    mirror = positional[1] if len(positional) > 1 else 'mirror'
    core.run_cmd(['git', 'remote', 'add', mirror, url])
    if not push:
        return 0
    
    branch = positional[2] if len(positional) > 2 else core.current_branch()
    core.run_cmd(['git', 'push', '--set-upstream', mirror, branch])
    return 0

if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)
