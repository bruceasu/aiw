#!/usr/bin/env python3
"""aiw git del-tag wrapper

Delete a local tag and optionally its remote counterpart.
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
    'name': 'del-tag',
    'short': 'Delete a local tag and optionally remove it from a remote.',
    'long': 'Deletes local tag and can push deletion to remote. Falls back to explicit refspec if remote delete fails.',
    'usage': 'aiw git del-tag <tag> [--remote] [--remote-name <name>] [--force]',
    'args': [
        {'flag': '<tag>', 'description': 'The name of the tag to delete.'},
        {'flag': '--remote', 'description': 'Also delete the tag from the remote (default: origin).'},
        {'flag': '--remote-name <name>', 'description': 'Remote name to use (default: origin).'},
        {'flag': '--force', 'description': 'Skip confirmation prompt.'}
    ],
    'examples': ['aiw git del-tag v1.2.3 --remote']
}


def main(argv):
    """
    Delete a local git tag and optionally its remote counterpart.

    Usage:
        aiw git del-tag <tag> [--remote] [--remote-name <name>] [--force]

    Options:
        --remote       Also delete the tag from the remote (default: origin).
        --remote-name  Remote name to use (default: origin).
        --force        Skip confirmation prompt.

    If the standard remote delete fails, retry with:
        refs/tags/<tag>
    """
    help_flags = {'-h', '--help', '-help', '-?'}
    if any(f in argv for f in help_flags):
        core.print_help_meta(META)
        return 0
    
    if len(argv) == 0:
        core.print_help_meta(META)
        return 0

    tag = argv[0]
    rest = argv[1:]

    if not core.git_confirm(
        f'This will delete tag "{tag}". Add --force to skip this prompt.',
        rest,
    ):
        print("aborted", file=os.sys.stderr)
        return 1
    
    # simple local delete; remote handling can be added later
    core.run_cmd(['git', 'tag', '-d'] + [tag])
    # Skip remote delete if --remote is not set.
    if not core.has_flag(rest, "--remote"):
        return

    remote_name = "origin"

    i = 0
    while i < len(rest):
        if rest[i] == "--remote-name" and i + 1 < len(rest):
            remote_name = rest[i + 1]
            i += 1
        i += 1

    # Try standard delete first.
    try:
        core.run_cmd(["git", "push", "--delete", remote_name, tag])

    except RuntimeError:
        # Fallback: explicit tag refspec avoids branch-name conflicts.
        print(
            f"standard remote delete failed; "
            f"retrying with explicit refspec refs/tags/{tag}",
            file=os.sys.stderr,
        )

        core.run_cmd([
            "git",
            "push",
            remote_name,
            f":refs/tags/{tag}",
        ])

if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)


