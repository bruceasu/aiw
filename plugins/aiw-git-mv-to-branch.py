#!/usr/bin/env python3
"""aiw-git-mv-to-branch wrapper

Move files to a branch.
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
	'name': 'aiw git mv-to-branch',
	'short': 'Move files to a branch (helper).',
	'long': 'Assists moving changes or files onto another branch, creating or switching branches as needed.',
	'usage': 'aiw git mv-to-branch <branch> [paths...]',
	'args': [],
	'examples': ['aiw git mv-to-branch feature/foo path1 path2']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	if not argv:
		print('usage: aiw git mv-to-branch <branch> [paths...]', file=sys.stderr)
		return 2
	branch = argv[0]
	paths = argv[1:]
	# simple implementation: create branch and move files via commit
	return core.run_cmd(['git', 'checkout', '-b', branch] + (paths or []))


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
