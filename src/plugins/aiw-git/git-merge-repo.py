#!/usr/bin/env python3
"""Import one selected Git branch into a subdirectory of a target repository.

Requires git-filter-repo.py in the same directory (or git-filter-repo in PATH).
The source repository is never rewritten in place. The target's chosen branch
is not changed: the result is left on a new review branch.
"""

import argparse
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import uuid


def log(message):
    print(message, flush=True)


def info(message):
    print(f"  {message}", flush=True)


def fail(message):
    print(f"ERROR: {message}", file=sys.stderr, flush=True)
    return 1


def git(repo, *args, capture=False, quiet=False):
    return subprocess.run(
        ['git', '-C', str(repo), *args],
        text=True,
        stdout=subprocess.PIPE if capture else (subprocess.DEVNULL if quiet else None),
        stderr=subprocess.PIPE if capture or quiet else None,
    )


def output(repo, *args):
    result = git(repo, *args, capture=True)
    return result.stdout.strip() if result.returncode == 0 else None


def prompt(question, default=''):
    try:
        reply = input(question).strip()
    except (EOFError, KeyboardInterrupt):
        return None
    return reply or default


def normalized_subdir(value):
    value = value.replace('\\', '/')
    parts = value.split('/')
    if (not value or value.startswith('/') or ':' in value
            or any(part in ('', '.', '..') for part in parts)):
        return None
    return '/'.join(parts)


def source_is_local(source):
    return os.path.isdir(os.path.expanduser(source))


def available_source_branches(clone):
    refs = output(clone, 'for-each-ref', '--format=%(refname:short)', 'refs/remotes/origin')
    branches = set()
    if refs:
        for ref in refs.splitlines():
            if ref.startswith('origin/') and ref != 'origin/HEAD':
                branches.add(ref[len('origin/'):])
    current = output(clone, 'symbolic-ref', '--short', 'HEAD')
    if current:
        branches.add(current)
    return sorted(branches), current


def choose_source_branch(clone, requested):
    branches, default = available_source_branches(clone)
    if not default:
        fail('Source clone has no checked-out branch (unborn or detached HEAD). Specify a valid source repository.')
        return None
    if requested:
        selected = requested
    elif len(branches) <= 1:
        selected = default
    else:
        log('\nSource branches:')
        for name in branches:
            info(f"{'*' if name == default else ' '} {name}{' (default)' if name == default else ''}")
        selected = prompt(f'Source branch [{default}]: ', default)
        if selected is None:
            return None
    if selected not in branches:
        fail(f"Unknown source branch {selected!r}. Available: {', '.join(branches)}")
        return None
    if selected != default:
        if git(clone, 'switch', '--track', f'origin/{selected}').returncode != 0:
            fail(f'Could not check out source branch {selected!r}')
            return None
    return selected


def cleanup(target, initial_branch, review_branch, temp_remote):
    # Only abort if a merge really is active; otherwise no noisy 'MERGE_HEAD missing'.
    if (Path(target) / '.git' / 'MERGE_HEAD').exists() or output(target, 'rev-parse', '-q', '--verify', 'MERGE_HEAD'):
        git(target, 'merge', '--abort', quiet=True)
    # The review branch is created only after checking the original working tree is clean.
    if output(target, 'symbolic-ref', '--short', 'HEAD') == review_branch:
        git(target, 'reset', '--hard', quiet=True)
        git(target, 'switch', initial_branch, quiet=True)
    if temp_remote:
        git(target, 'remote', 'remove', temp_remote, quiet=True)
    if output(target, 'show-ref', '--verify', f'refs/heads/{review_branch}') is not None:
        git(target, 'branch', '-D', review_branch, quiet=True)


def parser():
    p = argparse.ArgumentParser(
        description='Import one source Git branch under a target subdirectory, preserving history. '
                    'Create a review branch; do not change the target branch.',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog='Examples:\n'
               '  %(prog)s /work/customer-profile /work/fxtf-daily-stat customer-profile\n'
               '  %(prog)s /work/customer-profile /work/fxtf-daily-stat customer-profile --source-branch master\n'
               '  %(prog)s /work/customer-profile /work/fxtf-daily-stat customer-profile --source-branch fix/duckdb-aliases\n',
    )
    p.add_argument('source', help='Local Git repository path or clone URL')
    p.add_argument('target', help='Existing, clean target Git repository path')
    p.add_argument('subdir', nargs='?', help='Target-relative directory to hold source files')
    p.add_argument('--source-branch', help='Source branch to import (otherwise prompt when multiple exist)')
    p.add_argument('--target-branch', help='Target base branch (default: current target branch)')
    return p


def main(argv):
    args = parser().parse_args(argv)
    source = os.path.abspath(os.path.expanduser(args.source)) if source_is_local(args.source) else args.source
    target = os.path.abspath(os.path.expanduser(args.target))
    subdir = normalized_subdir(args.subdir) if args.subdir else None
    if not args.subdir:
        requested = prompt('Target subdirectory (blank to cancel): ')
        if requested is None or not requested:
            return fail('Cancelled: no target subdirectory specified.')
        subdir = normalized_subdir(requested)
    if subdir is None:
        return fail('Invalid subdirectory: use a relative path without . or .. components.')
    if not os.path.isdir(target):
        return fail(f'Target directory not found: {target}')
    root = output(target, 'rev-parse', '--show-toplevel')
    if root is None:
        return fail(f'Target is not a Git repository: {target}')
    target = root
    if source_is_local(source):
        source_root = output(source, 'rev-parse', '--show-toplevel')
        if source_root is None:
            return fail(f'Source is not a Git repository: {source}')
        if os.path.realpath(source_root) == os.path.realpath(target):
            return fail('Source and target must be different Git repositories.')
    initial_branch = output(target, 'symbolic-ref', '--short', 'HEAD')
    if not initial_branch:
        return fail('Target must be on a named branch.')
    if output(target, 'status', '--porcelain'):
        return fail('Target working tree is not clean. Commit or stash changes first.')
    target_branch = args.target_branch or initial_branch
    if output(target, 'show-ref', '--verify', f'refs/heads/{target_branch}') is None:
        return fail(f'Target branch does not exist: {target_branch}')

    filter_script = Path(__file__).resolve().parent / 'git-filter-repo.py'
    filter_cmd = [sys.executable, str(filter_script)] if filter_script.is_file() else None
    if filter_cmd is None:
        executable = shutil.which('git-filter-repo')
        if not executable:
            return fail('git-filter-repo.py is missing next to this script and git-filter-repo is not in PATH.')
        filter_cmd = [executable]

    log('== Repository import ==')
    info(f'Source: {source}')
    info(f'Target: {target}')
    info(f'Target base branch: {target_branch}')
    info(f'Destination directory: {subdir}/')

    with tempfile.TemporaryDirectory(prefix='aiw-merge-repo-') as clone:
        log('\n[1/4] Clone source (temporary copy)')
        clone_cmd = ['git', 'clone', '--quiet']
        if source_is_local(source):
            clone_cmd.append('--no-local')
        clone_cmd += ['--', source, clone]
        if subprocess.run(clone_cmd).returncode:
            return fail('Could not clone source repository.')

        branch = choose_source_branch(clone, args.source_branch)
        if branch is None:
            return 1
        source_sha = output(clone, 'rev-parse', '--short', 'HEAD')
        log('\nSelected import:')
        info(f'Source branch: {branch} ({source_sha})')
        info(f'Target base branch: {target_branch}')
        info(f'Path mapping: <source-root>/* -> {subdir}/*')
        info('History: only the selected source branch will be merged')
        info('Target base branch: will NOT be modified')
        answer = prompt('Proceed and create a review branch? [y/N]: ', 'n')
        if answer is None or answer.lower() not in ('y', 'yes'):
            log('Cancelled; target repository unchanged.')
            return 0

        log('\n[2/4] Rewrite paths in temporary source clone')
        if subprocess.run(filter_cmd + ['--to-subdirectory-filter', subdir, '--force'], cwd=clone).returncode:
            return fail('History rewrite failed; target repository unchanged.')
        # Resolve a concrete ref after rewrite. The chosen branch must remain available.
        rewritten_ref = f'refs/heads/{branch}'
        if output(clone, 'show-ref', '--verify', rewritten_ref) is None:
            return fail(f'Rewritten source branch is missing: {branch}')
        rewritten_sha = output(clone, 'rev-parse', '--short', rewritten_ref)
        info(f'Rewritten source ref: {branch} ({rewritten_sha})')

        review_branch = f'aiw-merge-repo-{uuid.uuid4().hex[:12]}'
        temp_remote = f'aiw-merge-source-{uuid.uuid4().hex[:12]}'
        created = False
        keep = False
        try:
            log('\n[3/4] Create review branch and fetch selected source ref')
            if git(target, 'switch', '-c', review_branch, target_branch).returncode:
                return fail('Could not create review branch.')
            created = True
            if git(target, 'remote', 'add', temp_remote, clone).returncode:
                return fail('Could not add temporary source remote.')
            # Explicit refspec ensures there is only one merge input, independent of FETCH_HEAD.
            fetched_ref = f'refs/remotes/{temp_remote}/import'
            refspec = f'{rewritten_ref}:{fetched_ref}'
            if git(target, 'fetch', '--no-tags', temp_remote, refspec).returncode:
                return fail('Could not fetch selected rewritten source branch.')
            if output(target, 'show-ref', '--verify', fetched_ref) is None:
                return fail('Expected fetched reference is absent.')
            fetched_sha = output(target, 'rev-parse', '--short', fetched_ref)
            info(f'Fetched: {branch} -> {fetched_ref} ({fetched_sha})')

            if git(target, 'merge-base', '--is-ancestor', fetched_ref, 'HEAD', quiet=True).returncode == 0:
                log('\nALREADY IMPORTED: selected source history is an ancestor of the target branch.')
                info('No new merge commit is necessary; review branch will be removed.')
                return 0

            log('\n[4/4] Merge on review branch')
            if git(target, 'merge', '--no-ff', '--no-edit', '--allow-unrelated-histories', fetched_ref).returncode:
                return fail('Merge failed (possibly conflicts). Original target branch will remain unchanged.')

            if git(target, 'remote', 'remove', temp_remote, quiet=True).returncode:
                return fail('Merge finished but temporary remote cleanup failed.')
            temp_remote = None
            keep = True
            log('\nSUCCESS: source imported on a new review branch')
            info(f'Review branch: {review_branch}')
            info(f'Current branch: {review_branch}')
            info(f'Unchanged target branch: {target_branch}')
            info('Review:')
            info(f'  git -C "{target}" diff --stat {target_branch}..{review_branch}')
            info(f'  git -C "{target}" log --oneline --graph --max-count=15')
            info('After review, integrate into target branch:')
            info(f'  git -C "{target}" switch {target_branch}')
            info(f'  git -C "{target}" merge --no-ff {review_branch}')
            return 0
        finally:
            if created and not keep:
                log('\nCleaning up incomplete review branch...')
                cleanup(target, initial_branch, review_branch, temp_remote)
                info(f'Original branch restored: {initial_branch}')


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
