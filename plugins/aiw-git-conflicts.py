#!/usr/bin/env python3
"""aiw-git-conflicts wrapper

Identify and assist with merge conflicts.
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
	'name': 'aiw git conflicts',
	'short': 'Show and assist with merge conflicts.',
	'long': 'Helper to identify conflicting files and provide steps to resolve merge conflicts.',
	'usage': 'aiw git conflicts',
	'args': [],
	'examples': ['aiw git conflicts']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	return core.run_cmd(['git', 'diff', '--name-only', '--diff-filter=U'])


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
