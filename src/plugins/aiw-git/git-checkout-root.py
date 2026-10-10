#!/usr/bin/env python3
"""Print the root directory of the current Git checkout."""
import sys
import os
import importlib.util

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')

spec = importlib.util.spec_from_file_location('aiw_git_core', CORE_PATH)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
	'name': 'aiw git checkout-root',
	'short': 'Find the root directory of the current checkout.',
	'long': 'Prints the top-level directory of the current Git working tree, including when running inside a linked worktree.',
	'usage': 'aiw git checkout-root',
	'args': [],
	'examples': ['aiw git checkout-root']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	return core.run_cmd(['git', 'rev-parse', '--show-toplevel'])


if __name__ == '__main__':
	try:
		rc = main(sys.argv[1:])
		sys.exit(rc)
	except SystemExit:
		raise


