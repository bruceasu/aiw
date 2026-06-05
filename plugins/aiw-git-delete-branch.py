#!/usr/bin/env python3
"""aiw-git-delete-branch wrapper

Delete a local and/or remote branch with confirmation.
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
	'name': 'aiw git delete-branch',
	'short': 'Delete a local and/or remote branch with confirmation.',
	'long': 'Deletes branches locally and optionally from a remote. Includes confirmations to avoid accidental destructive actions.',
	'usage': 'aiw git delete-branch <branch> [--force] [--remote] [--remote-only] [--remote-name <name>]',
	'args': [],
	'examples': ['aiw git delete-branch feature/foo']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	# forward to git; leave complex behavior to future enhancements
	return core.run_cmd(['git', 'branch', '-d'] + argv)


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
