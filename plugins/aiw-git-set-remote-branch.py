#!/usr/bin/env python3
"""aiw-git-set-remote-branch wrapper

Set the remote branch for the current branch.
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
	'name': 'aiw git set-remote-branch',
	'short': 'Set the remote branch for the current branch.',
	'long': 'Sets or adjusts the remote tracking branch for the current local branch.',
	'usage': 'aiw git set-remote-branch <remote-branch>',
	'args': [],
	'examples': ['aiw git set-remote-branch origin/main']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	if not argv:
		print('usage: aiw git set-remote-branch <remote-branch>', file=sys.stderr)
		return 2
	return core.run_cmd(['git', 'branch', '--set-upstream-to', argv[0]])


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
