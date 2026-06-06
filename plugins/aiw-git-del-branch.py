#!/usr/bin/env python3
"""aiw-git-del-branch. wrapper

Delete a local and/or remote branch with confirmation.
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
    'name': 'aiw git-del-branch.',
    'short': 'Delete a local and/or remote branch with confirmation.',
    'long': 'Deletes branches locally and optionally from a remote. Includes confirmations to avoid accidental destructive actions.',
    'usage': 'aiw git-del-branch. <branch> [--force] [--remote] [--remote-only] [--remote-name <name>]',
    'args': [
        {'flag': '<branch>', 'description': 'The name of the branch to delete.'},
        {'flag': '--force', 'description': 'Force-delete local branch even if not merged (-D).'},
        {'flag': '--remote', 'description': 'Delete local branch AND remote branch.'},
        {'flag': '--remote-only', 'description': 'Delete only remote branch.'},
        {'flag': '--remote-name', 'description': 'Remote name to use (default: origin).'},
    ],
    'examples': ['aiw git-del-branch. feature/foo']
}


def main(argv):
    """
    Delete a local and/or remote git branch.

    Usage:
        aiw git-del-branch. <branch>
            [--force]
            [--remote]
            [--remote-only]
            [--remote-name <name>]

    Flags:
        --force        Force-delete local branch even if not merged (-D).
        --remote       Delete local branch AND remote branch.
        --remote-only  Delete only remote branch.
        --remote-name  Remote name to use (default: origin).
    """
    if len(args) == 0:
        core.print_help_meta(META)
        return 0
    help_flags = {'-h', '--help', '-help', '-?'}
    if any(f in argv for f in help_flags):
        core.print_help_meta(META)
        return 0
    branch = args[0]
    rest = args[1:]

    force = core.has_flag(rest, "--force")
    delete_remote = (
        core.has_flag(rest, "--remote")
        or core.has_flag(rest, "--remote-only")
    )
    remote_only = core.has_flag(rest, "--remote-only")

    remote_name = "origin"
    i = 0
    while i < len(rest):
        if rest[i] == "--remote-name" and i + 1 < len(rest):
            remote_name = rest[i + 1]
            i += 1
        i += 1

    prompt = f'This will delete branch "{branch}"'

    if delete_remote:
        prompt += " locally AND on remote"

    prompt += ". Add --force to skip this prompt."

    if not core.git_confirm(prompt, rest):
        print("aborted", file=os.sys.stderr)
        return

    # Delete local branch.
    if not remote_only:
        local_flag = "-D" if force else "-d"

        core.run_cmd([
            "git",
            "branch",
            local_flag,
            branch,
        ])

    # Delete remote branch.
    if delete_remote:
        core.run_cmd([
            "git",
            "push",
            remote_name,
            "--delete",
            branch,
        ])


if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)
