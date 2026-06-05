#!/usr/bin/env python3
"""aiw-git-how-to-split wrapper

Guidance helper for splitting commits into smaller ones.
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
	'name': 'aiw git how-to-split',
	'short': 'Guidance helper for splitting commits into smaller ones.',
	'long': 'Prints a short how-to for splitting large commits into logically separated smaller commits.',
	'usage': 'aiw git how-to-split',
	'args': [],
	'examples': ['aiw git how-to-split']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	print('To split a commit: use git reset --soft HEAD~1; git add -p; git commit -m "..."')
	return 0


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
