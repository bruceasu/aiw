#!/usr/bin/env python3
"""aiw git merge-repo wrapper.

Import the default branch of one repository into another repository under a
new directory, preserving the source history after rewriting its paths.
"""

import importlib.util
import os
import subprocess
import sys
import tempfile
import uuid

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')
FILTER_REPO_PATH = os.path.join(HERE, 'git-filter-repo.py')

spec = importlib.util.spec_from_file_location('aiw_git_core', CORE_PATH)
if spec is None or spec.loader is None:
    raise RuntimeError(f'Cannot load aiw git core module from {CORE_PATH}')
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    'name': 'aiw git merge-repo',
    'short': 'Merge one repository into another under a subdirectory.',
    'long': (
        'Rewrites the source repository history with a new leading directory, '
        'merges it into the target repository on a temporary branch, then '
        'leaves that review branch for manual approval. The selected target '
        'branch is never changed automatically; conflicts are rejected and '
        'the temporary source clone is removed.'
    ),
    'usage': 'aiw git merge-repo <repo1> <repo2> [repo1-to-subdir]',
    'args': [
        {'flag': '<repo1>', 'description': 'Source repository path or URL.'},
        {'flag': '<repo2>', 'description': 'Target repository path.'},
        {
            'flag': '[repo1-to-subdir]',
            'description': 'Directory prefix for source files; prompted when omitted.',
        },
    ],
    'examples': [
        'aiw git merge-repo ../service-a ../monorepo service-a',
        'aiw git merge-repo ../service-a ../monorepo',
    ],
}

HELP_FLAGS = {'-h', '--help', '-help', '-?'}


def _run_git(repo, args):
    return subprocess.run(['git', '-C', repo, *args]).returncode


def _git_output(repo, args):
    result = subprocess.run(
        ['git', '-C', repo, *args],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    if result.returncode != 0:
        return None
    return result.stdout.strip()


def _prompt(prompt):
    print(prompt, end='', file=sys.stderr, flush=True)
    try:
        return sys.stdin.readline().strip()
    except (EOFError, OSError):
        return ''


def _valid_subdir(value):
    normalized = value.replace('\\', '/')
    parts = [part for part in normalized.split('/') if part]
    if not parts or normalized.startswith('/') or ':' in normalized:
        return False
    return all(part not in {'.', '..'} for part in parts)


def _choose_subdir(argv):
    if len(argv) >= 3:
        subdir = argv[2]
        return subdir if _valid_subdir(subdir) else None
    answer = _prompt(
        'Enter repo1-to-subdir, or press Enter to cancel [cancel]: '
    )
    if not answer:
        return None
    return answer if _valid_subdir(answer) else None


def _choose_target_branch(target):
    current = _git_output(target, ['symbolic-ref', '--short', 'HEAD'])
    if not current:
        print('error: target repo must be on a branch', file=sys.stderr)
        return None
    answer = _prompt(
        f'Choose a different target branch from {current}? [y/N]: '
    ).lower()
    if answer not in {'y', 'yes'}:
        return current
    selected = _prompt('Target branch: ')
    if not selected:
        return None
    if _git_output(target, ['rev-parse', '--verify', f'refs/heads/{selected}']) is None:
        print(f'error: target branch not found: {selected}', file=sys.stderr)
        return None
    return selected


def _cleanup_target(target, target_branch, temp_branch, remote):
    _run_git(target, ['merge', '--abort'])
    _run_git(target, ['switch', target_branch])
    _run_git(target, ['remote', 'remove', remote])
    _run_git(target, ['branch', '-D', temp_branch])


def main(argv):
    if not argv or any(flag in argv for flag in HELP_FLAGS):
        core.print_help_meta(META)
        return 0 if argv else 2
    if len(argv) not in {2, 3}:
        core.print_help_meta(META)
        return 2

    source, target = argv[0], os.path.abspath(argv[1])
    subdir = _choose_subdir(argv)
    if not subdir:
        print('cancelled', file=sys.stderr)
        return 0
    if not os.path.isdir(target):
        print(f'error: target repo directory not found: {target}', file=sys.stderr)
        return 2
    if os.path.abspath(source) == target:
        print('error: source and target repos must be different', file=sys.stderr)
        return 2
    if _git_output(target, ['rev-parse', '--show-toplevel']) is None:
        print(f'error: not a Git repository: {target}', file=sys.stderr)
        return 2
    if _git_output(target, ['status', '--porcelain']):
        print('error: target repo has uncommitted changes', file=sys.stderr)
        return 2

    target_branch = _choose_target_branch(target)
    if not target_branch:
        print('cancelled', file=sys.stderr)
        return 0
    if _prompt(
        f'Merge {source} into {target_branch}/{subdir}? [y/N]: '
    ).lower() not in {'y', 'yes'}:
        print('cancelled', file=sys.stderr)
        return 0

    temp_branch = f'aiw-merge-repo-{uuid.uuid4().hex[:12]}'
    remote = f'aiw-merge-source-{uuid.uuid4().hex[:12]}'
    target_changed = False
    preserve_review_branch = False

    with tempfile.TemporaryDirectory(prefix='aiw-merge-repo-') as source_copy:
        clone_args = ['clone']
        if os.path.isdir(source):
            clone_args.append('--no-local')
        clone_args.extend([source, source_copy])
        if subprocess.run(['git', *clone_args]).returncode != 0:
            return 1
        if subprocess.run(
            [sys.executable, FILTER_REPO_PATH, '--to-subdirectory-filter', subdir, '--force'],
            cwd=source_copy,
        ).returncode != 0:
            return 1

        if _run_git(target, ['switch', '-c', temp_branch, target_branch]) != 0:
            return 1
        target_changed = True
        try:
            if _run_git(target, ['remote', 'add', remote, source_copy]) != 0:
                return 1
            if _run_git(target, ['fetch', remote]) != 0:
                return 1
            if _run_git(
                target,
                [
                    'merge', '--no-ff', '--no-commit',
                    '--allow-unrelated-histories', 'FETCH_HEAD',
                ],
            ) != 0:
                print('error: merge conflict or merge failure; target branch unchanged', file=sys.stderr)
                return 1
            if _run_git(
                target,
                ['commit', '-m', f'Merge repository {source} into {subdir}'],
            ) != 0:
                return 1
            if _run_git(target, ['remote', 'remove', remote]) != 0:
                return 1
            preserve_review_branch = True
            print(f'merge prepared on review branch: {temp_branch}')
            print(
                f'After review, run: git switch {target_branch} && '
                f'git merge --no-ff {temp_branch}'
            )
            return 0
        finally:
            if target_changed and not preserve_review_branch:
                _cleanup_target(target, target_branch, temp_branch, remote)


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
