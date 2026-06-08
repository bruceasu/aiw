#!/usr/bin/env python3
"""aiw git ca wrapper

Amend last commit including all changes.
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
    'name': 'aiw git revert',
    'short': 'Revert the last commit.',
    'long': 'Runs git revert to revert the last commit.',
    'usage': 'aiw git revert [options]',
    'args': [],
    'examples': ['aiw git revert']
}


def main(argv):
    if len(argv) == 0:
        core.print_help_meta(META)
        return 0
    
    help_flags = {'-h', '--help', '-help', '-?'}
    if any(f in argv for f in help_flags):
        core.print_help_meta(META)
        return 0
    sha = ""
    for  a  in argv:
        if a != "--no-commit" and a != "--force":
            sha = a
            break
        
    
    if not core.git_confirm(f"This will create a new commit that undoes $sha. Add --force to skip this prompt.", argv) :
        print("aborted")
        return 1
    
    cmdArgs = []
    if core.has_flag(argv, "--no-commit"):
        cmdArgs.append("--no-commit")
    
    for a in argv :
        if a != "--no-commit" and a != "--force":
            cmdArgs.append(a)
    
    core.run_cmd(['git', 'revert'] + cmdArgs)
    return 0

if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)


