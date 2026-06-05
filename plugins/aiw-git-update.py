#!/usr/bin/env python3
"""aiw-git-update wrapper

Update repository: fetch and rebase or pull.
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
	'short': 'Fetch and rebase or pull updates from remote.',
	'long': 'Fetches updates from the remote and rebases the current branch onto its upstream by default. Use --merge to perform a merge pull.',
	'usage': 'aiw git update [--merge]',
	'args': [],
	'examples': ['aiw git update', 'aiw git update --merge']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	merge = '--merge' in argv
	up = core.git_output(['git', 'rev-parse', '--abbrev-ref', '@{u}'])
	if not up:
		print('no upstream configured for current branch', file=sys.stderr)
		return 2
	remote, branch = up.split('/', 1) if '/' in up else ('origin', up)
	if core.run_cmd(['git', 'fetch', remote]) != 0:
		return 1
	if merge:
		return core.run_cmd(['git', 'pull', '--no-rebase', remote, branch])
	return core.run_cmd(['git', 'rebase', f'{remote}/{branch}'])


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
