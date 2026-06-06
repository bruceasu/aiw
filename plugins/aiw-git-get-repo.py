#!/usr/bin/env python3
"""aiw-git-get-repo (clone) wrapper

Perform a shallow or full git clone with optional branch and directory.
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
    'name': 'aiw git-get-repo',
    'short': 'Shallow or full git clone.',
    'long': (
        'Clones a repository with optional branch selection, '
        'target directory, and shallow depth control.'
    ),
    'usage': (
        'aiw git-get-repo <url> '
        '[-b <branch>] [-d <dir>] [--depth <n>] [--full]'
    ),
    'args': [
        {'flag': '<url>', 'description': 'The URL of the repository to clone.'},
        {'flag': '-b <branch>', 'description': 'The branch to clone.'},
        {'flag': '-d <dir>', 'description': 'The directory into which to clone the repository.'},
        {'flag': '--depth <n>', 'description': 'The depth of the shallow clone.'},
        {'flag': '--full', 'description': 'Perform a full clone.'}
    ],
    'examples': [
        'aiw git-get-repo https://github.com/user/repo.git',
        'aiw git-get-repo repo.git -b main -d myrepo',
        'aiw git-get-repo repo.git --depth 5',
        'aiw git-get-repo repo.git --full',
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

    url = argv[0]
    rest = argv[1:]

    branch = None
    directory = None
    depth = "1"
    full = core.has_flag(rest, "--full")

    i = 0
    while i < len(rest):
        arg = rest[i]

        if arg in ("-b", "--branch") and i + 1 < len(rest):
            branch = rest[i + 1]
            i += 1

        elif arg in ("-d", "--dir") and i + 1 < len(rest):
            directory = rest[i + 1]
            i += 1

        elif arg == "--depth" and i + 1 < len(rest):
            depth = rest[i + 1]
            i += 1

        i += 1

    cmd = ["git", "clone"]

    if branch:
        cmd.extend(["--branch", branch, "--single-branch"])

    if not full:
        cmd.extend(["--depth", depth])

    cmd.append(url)

    if directory:
        cmd.append(directory)

    return core.run_cmd(cmd)


if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)