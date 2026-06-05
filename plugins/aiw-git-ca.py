#!/usr/bin/env python3
"""aiw-git-ca wrapper

Amend last commit including all changes.
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
	'name': 'aiw git ca',
	'short': 'Amend the last commit, include all staged changes.',
	'long': 'Runs git commit -a --amend to amend the previous commit with staged and unstaged changes.',
	'usage': 'aiw git ca',
	'args': [],
	'examples': ['aiw git ca']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	return core.run_cmd(['git', 'commit', '-a', '--amend'])


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
