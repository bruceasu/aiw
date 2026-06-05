#!/usr/bin/env python3
"""aiw-git-sync wrapper

Fetch, rebase onto remote branch, and push.
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
	'name': 'aiw git sync',
	'short': 'Fetch, rebase the current branch onto the remote, then push.',
	'long': 'Fetches remote, rebases the current branch onto the remote branch, and pushes. Defaults: remote=origin, branch=current.',
	'usage': 'aiw git sync [branch] [remote]',
	'args': [],
	'examples': ['aiw git sync', 'aiw git sync main origin']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	remote = 'origin'
	branch = None
	if len(argv) >= 2:
		branch = argv[0]
		remote = argv[1]
	elif len(argv) == 1:
		branch = argv[0]
	# fetch
	if not core.has_remote(remote):
		print(f'remote {remote!r} not found', file=sys.stderr)
		return 2
	if core.run_cmd(['git', 'fetch', '-p', remote]) != 0:
		return 1
	if branch is None:
		branch = core.current_branch()
	if core.run_cmd(['git', 'rebase', f'{remote}/{branch}']) != 0:
		return 1
	return core.run_cmd(['git', 'push', remote, branch])


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
