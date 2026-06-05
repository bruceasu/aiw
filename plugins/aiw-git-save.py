#!/usr/bin/env python3
"""aiw-git-save wrapper

Defines META and implements the save subcommand using core utilities.
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
	'name': 'aiw git save',
	'short': 'Stage all changes and commit (default message: wip).',
	'long': 'Stages all working-tree changes and commits them. If a message is provided, it is used as the commit message; otherwise the message "wip" is used.',
	'usage': 'aiw git save [message]',
	'args': [],
	'examples': ['aiw git save', 'aiw git save "fix tests"']
}


def main(argv):
	# help
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0

	msg = 'wip' if not argv else ' '.join(argv)
	core.run_cmd(['git', 'add', '-A'])
	return core.run_cmd(['git', 'commit', '-m', msg])


if __name__ == '__main__':
	try:
		rc = main(sys.argv[1:])
		sys.exit(rc)
	except SystemExit as e:
		raise
