#!/usr/bin/env python3
"""aiw git purge wrapper

Remove a file or directory from repository history.
"""

import sys
import os
import importlib.util
import shutil
import shlex

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')

spec = importlib.util.spec_from_file_location('aiw_git_core', CORE_PATH)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    'name': 'aiw git purge',
    'short': 'Remove a file or directory from repository history.',
    'long': (
        'Rewrites all repository history to remove the path using git-filter-repo '
        '(if available) or filter-branch. This changes commit IDs.'
    ),
    'usage': 'aiw git purge <path> [--from <commit>] [--force]',
    'args': [
        {'flag': '<path>', 'description': 'The file or directory to remove from history.'},
        {'flag': '--from <commit>', 'description': 'Limit the rewrite to commits after this commit up to HEAD.'},
        {'flag': '--force', 'description': 'Skip confirmation and force the history-rewrite tool.'}
    ],
    'examples': [
        'aiw git purge secrets.txt',
        'aiw git purge generated/ --from abc123',
        'aiw git purge secrets.txt --force',
    ],
}


def main(argv):
    help_flags = {'-h', '--help', '-help', '-?'}
    if not argv:
        core.print_help_meta(META)
        return 2
    if any(flag in argv for flag in help_flags):
        core.print_help_meta(META)
        return 0

    path = argv[0]
    rest = argv[1:]
    from_commit = None
    i = 0
    while i < len(rest):
        arg = rest[i]
        if arg == '--force':
            i += 1
            continue
        if arg == '--from' and i + 1 < len(rest):
            from_commit = rest[i + 1]
            i += 2
            continue
        print(f'usage: {META["usage"]}', file=sys.stderr)
        return 2

    filter_repo_script = os.path.join(HERE, 'git-filter-repo.py')
    filter_repo_exe = shutil.which('git-filter-repo')
    if os.path.exists(filter_repo_script):
        filter_repo = [sys.executable, filter_repo_script]
    elif filter_repo_exe:
        filter_repo = [filter_repo_exe]
    else:
        filter_repo = None

    tool_name = 'git-filter-repo' if filter_repo else 'git filter-branch'
    scope = f'commits after "{from_commit}" up to HEAD' if from_commit else 'all refs and commits'
    warning = (
        f'purge will permanently delete "{path}" from {scope} using {tool_name}. '
        'This rewrites history and changes commit IDs. Collaborators may need to '
        're-clone or reset after a force-push.'
    )
    if not core.git_confirm(warning, rest):
        print('aborted', file=sys.stderr)
        return 1

    if filter_repo:
        cmd = filter_repo + ['--path', path, '--invert-paths', '--force']
        if from_commit:
            cmd.extend(['--refs', f'{from_commit}..HEAD'])
    else:
        tree_filter = f'git rm -fr --ignore-unmatch -- {shlex.quote(path)}'
        cmd = ['git', 'filter-branch', '-f', '--tree-filter', tree_filter, '--']
        cmd.append(f'{from_commit}..HEAD' if from_commit else '--all')
    core.run_cmd(cmd)
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
