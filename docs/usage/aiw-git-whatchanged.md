# aiw git whatchanged

Short: Show changes between commits (git whatchanged).

Description:
Forwards arguments to `git whatchanged` to show commit-level diffs.

Usage:
aiw git whatchanged [<rev-range>] [-p]

Examples:
- aiw git whatchanged HEAD~5..HEAD -p

For full help run: generate_git_docs.py -h