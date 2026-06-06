#!/usr/bin/env python3
"""aiw-git-get-file-from wrapper

Extract a file version from another branch/commit.
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
	'name': 'aiw git-get-file',
	'short': 'Extract a file version from another branch/commit.',
	'long': 'Retrieves a file from a different branch or commit and writes it into the working tree.',
	'usage': 'aiw git-get-file <commit|branch> <path>',
	'args': [
		{'flag': '<commit|branch>', 'description': 'The commit or branch from which to extract the file.'},
		{'flag': '<path>', 'description': 'The path to the file to extract.'}
	],
	'examples': ['aiw git-get-file origin/main path/to/file']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	if len(argv) < 2:
		print('usage: aiw git-get-file <commit|branch> <path>', file=sys.stderr)
		return 2
	src, path = argv[0], argv[1]
	return core.run_cmd(['git', 'checkout', src, '--', path])


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
