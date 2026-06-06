#!/usr/bin/env python3
"""aiw-git-rename wrapper

Rename files tracked by git.
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
	'name': 'aiw git-mv',
	'short': 'Move or rename files or paths tracked by git with helper semantics.',
	'long': 'Convenience wrapper for moving or renaming tracked files and updating the index and history appropriately.',
	'usage': 'aiw git-mv <old> <new>',
	'args': [],
	'examples': ['aiw git-mv oldname newname']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	if len(argv) < 2:
		print('usage: aiw git-mv <old> <new>', file=sys.stderr)
		return 2
	return core.run_cmd(['git', 'mv', '--force', argv[0], argv[1]])


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
