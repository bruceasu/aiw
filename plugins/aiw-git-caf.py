#!/usr/bin/env python3
"""aiw-git-caf wrapper

Amend last commit without changing the message.
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
	'name': 'aiw git caf',
	'short': 'Amend the last commit without changing the message.',
	'long': 'Runs git commit -a --amend --no-edit to amend the previous commit while preserving its message.',
	'usage': 'aiw git caf',
	'args': [],
	'examples': ['aiw git caf']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	return core.run_cmd(['git', 'commit', '-a', '--amend', '--no-edit'])


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
