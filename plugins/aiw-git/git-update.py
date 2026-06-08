#!/usr/bin/env python3
"""aiw git update wrapper

Fetch updates from remote, optionally rebase onto remote branch, and optionally push.

Modes:
- Default: fetch + pull --ff-only (update local branch)
- --rebase: fetch + rebase current branch onto remote
- --push: push current branch after rebase or pull
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
    'name': 'aiw git update',
    'short': 'Fetch updates, optionally rebase and push.',
    'long': (
        'Fetches remote updates and either pulls or rebases the current branch. '
        'Optionally pushes after rebase or pull. Defaults: remote=origin, branch=current.'
    ),
    'usage': 'aiw git update [branch] [remote] [--rebase] [--push] [--merge]',
    'args': [],
    'examples': [
        'aiw git update',                        # update current branch
        'aiw git update main origin --rebase',   # fetch + rebase main
        'aiw git update develop origin --push',  # fetch + pull + push
    ],
}


def main(argv):
    help_flags = {'-h', '--help', '-help', '-?'}
    if not argv or any(f in argv for f in help_flags):
        core.print_help_meta(META)
        return 0

    branch = None
    remote = 'origin'
    rebase = core.has_flag(argv, '--rebase')
    push = core.has_flag(argv, '--push')
    merge = core.has_flag(argv, '--merge')  # optional pull merge mode

    # positional arguments: branch [remote]
    pos_args = [arg for arg in argv if not arg.startswith('--')]
    if len(pos_args) >= 1:
        branch = pos_args[0]
    if len(pos_args) >= 2:
        remote = pos_args[1]

    if branch is None:
        branch = core.current_branch()

    if not core.has_remote(remote):
        print(f'remote {remote!r} not found', file=sys.stderr)
        return 2

    # fetch first
    if core.run_cmd(['git', 'fetch', '--prune', remote]) != 0:
        return 1

    cur = core.current_branch()

    if rebase:
        # rebase current branch onto remote
        if core.run_cmd(['git', 'rebase', f'{remote}/{branch}']) != 0:
            return 1
    else:
        # update mode: pull or fetch only
        if branch == cur:
            cmd = ['git', 'pull']
            if not merge:
                cmd.append('--ff-only')
            cmd += [remote, branch]
            if core.run_cmd(cmd) != 0:
                return 1
        else:
            # off-target branch: update via fetch refspec
            refspec = f'refs/heads/{branch}:refs/heads/{branch}'
            if core.run_cmd(['git', 'fetch', remote, refspec, '--recurse-submodules=no', '--progress', '--prune']) != 0:
                return 1

    if push:
        return core.run_cmd(['git', 'push', remote, branch])

    return 0


if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)

