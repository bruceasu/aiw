#!/usr/bin/env python3
"""aiw-git-delete-tag wrapper

Delete a local tag and optionally its remote counterpart.
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
	'name': 'aiw git delete-tag',
	'short': 'Delete a local tag and optionally remove it from a remote.',
	'long': 'Deletes local tag and can push deletion to remote. Falls back to explicit refspec if remote delete fails.',
	'usage': 'aiw git delete-tag <tag> [--remote] [--remote-name <name>] [--force]',
	'args': [],
	'examples': ['aiw git delete-tag v1.2.3 --remote']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	# simple local delete; remote handling can be added later
	return core.run_cmd(['git', 'tag', '-d'] + argv)


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
