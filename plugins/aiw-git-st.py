#!/usr/bin/env python3
"""aiw-git-st wrapper

Shows concise working-tree status.
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
	'name': 'aiw git-st',
	'short': 'Show concise working-tree status.',
	'long': 'Shortcut for a concise git status display (git status -sb).',
	'usage': 'aiw git-st',
	'args': [],
	'examples': ['aiw git-st']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	return core.run_cmd(['git', 'status', '-sb'])


if __name__ == '__main__':
	try:
		rc = main(sys.argv[1:])
		sys.exit(rc)
	except SystemExit:
		raise
