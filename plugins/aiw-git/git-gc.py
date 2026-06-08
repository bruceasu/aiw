#!/usr/bin/env python3
"""aiw git gc wrapper

Run git gc aggressively with confirmation.
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
	'name': 'aiw git gc',
	'short': 'Run git gc aggressively (destructive).',
	'long': 'Runs git gc --aggressive --prune=now. This rewrites history objects and may prevent reflog recovery; confirm before running.',
	'usage': 'aiw git gc [--force]',
	'args': [
		{'flag': '--force', 'description': 'Skip confirmation prompt.'}
	],
	'examples': ['aiw git gc', '--force']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	if not core.git_confirm('gc --aggressive rewrites history objects and cannot be undone. Ensure no reflog recovery is needed first.', argv):
		print('aborted', file=sys.stderr)
		return 0
	return core.run_cmd(['git', 'gc', '--prune=now', '--aggressive'])


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)


