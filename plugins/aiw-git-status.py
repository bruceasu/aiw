#!/usr/bin/env python3
"""aiw-git-status wrapper

Alias for st: show concise working-tree status.
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
	'name': 'aiw git status',
	'short': 'Show concise working-tree status (alias of st).',
	'long': 'Alias of aiw git st.',
	'usage': 'aiw git status',
	'args': [],
	'examples': ['aiw git status']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	return core.run_cmd(['git', 'status', '-sb'])


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
