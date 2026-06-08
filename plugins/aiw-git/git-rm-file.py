#!/usr/bin/env python3
"""aiw git rm-file wrapper

Remove a file from the last commit or scrub it from history.
"""

import sys
import os
import importlib.util
import shutil

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')

spec = importlib.util.spec_from_file_location(
    'aiw_git_core',
    CORE_PATH,
)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    'name': 'aiw git rm-file',
    'short': 'Remove a file from commit or full history.',
    'long': (
        'Removes a file from the last commit, or rewrites history '
        'to remove it from commits using git-filter-repo (if available) '
        'or filter-branch.'
    ),
    'usage': (
        'aiw git rm-file <file> '
        '[--history] [--from <commit>] [--force]'
    ),
    'args': [
        {'flag': '<file>', 'description': 'The file to remove.'},
        {'flag': '--history', 'description': 'Remove the file from commits in history.'},
        {'flag': '--from <commit>', 'description': 'Only remove the file from commits after the specified commit.'},
        {'flag': '--force', 'description': 'Skip confirmation prompt and pass --force to the rewrite tool.'}
    ],
    'examples': [
        'aiw git rm-file secrets.txt',
        'aiw git rm-file secrets.txt --history',
        'aiw git rm-file secrets.txt --history --from abc123',
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

    file = argv[0]
    rest = argv[1:]

    # ------------------------------------------------------------
    # Non-history mode: amend last commit only
    # ------------------------------------------------------------
    if not core.has_flag(rest, '--history'):

        core.run_cmd([
            'git',
            'checkout',
            'HEAD^',
            '--',
            file,
        ])

        core.run_cmd([
            'git',
            'add',
            '-A',
        ])

        core.run_cmd([
            'git',
            'commit',
            '--amend',
            '--no-edit',
        ])

        return 0

    # ------------------------------------------------------------
    # History mode
    # ------------------------------------------------------------
    from_commit = None
    i = 0
    while i < len(rest):
        if rest[i] == '--from' and i + 1 < len(rest):
            from_commit = rest[i + 1]
            i += 1
        i += 1

    # Check for git-filter-repo
    filter_repo_script = os.path.join(HERE, 'git-filter-repo.py')
    filter_repo_exe = shutil.which('git-filter-repo')
    
    use_filter_repo = None
    if os.path.exists(filter_repo_script):
        use_filter_repo = [sys.executable, filter_repo_script]
    elif filter_repo_exe:
        use_filter_repo = [filter_repo_exe]

    tool_name = "git-filter-repo" if use_filter_repo else "git filter-branch"

    if from_commit:
        range_desc = f'commits after "{from_commit}" up to HEAD'
    else:
        range_desc = 'ALL commits in history'

    warning = (
        f'rm-file --history will permanently delete "{file}" from '
        f'{range_desc}\n'
        f'using {tool_name}. This rewrites history.\n'
        'Collaborators must re-clone or reset after a force-push.'
    )

    if not core.git_confirm(warning, rest):
        print('aborted', file=sys.stderr)
        return 1

    if use_filter_repo:
        # Use git-filter-repo
        cmd = use_filter_repo + ['--path', file, '--invert-paths', '--force']
        if from_commit:
            cmd.extend(['--refs', f'{from_commit}..HEAD'])
        core.run_cmd(cmd)
    else:
        # Fallback to filter-branch
        tree_filter = f'git rm -fr --ignore-unmatch {file}'
        cmd = [
            'git',
            'filter-branch',
            '-f',
            '--tree-filter',
            tree_filter,
        ]
        if from_commit:
            cmd.append('--')
            cmd.append(f'{from_commit}..HEAD')
        else:
            cmd.append('HEAD')
        core.run_cmd(cmd)

    return 0


if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)


