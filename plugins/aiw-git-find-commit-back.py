#!/usr/bin/env python3
"""aiw-git-find-commit-back wrapper

Find a commit a number of steps back.
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
	'name': 'aiw git find-commit-back',
	'short': 'Find a commit a number of steps back (helper).',
	'long': 'Searches the commit history backward to locate a commit matching criteria or a specific offset.',
	'usage': 'aiw git find-commit-back [options]',
	'args': [],
	'examples': ['aiw git find-commit-back']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	return core.run_cmd(['git', 'rev-list', '--all'] + argv)


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
