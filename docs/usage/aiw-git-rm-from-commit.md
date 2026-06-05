# aiw git rm-from-commit

Short: Remove a file from a historical commit (rewrite history assistance).

Description:
Helper to remove a file introduced in a particular commit. Typically used to strip accidental files from history.

Usage:
aiw git rm-from-commit <commit> <path>

Examples:
- aiw git rm-from-commit abc123 path/to/secret

For full help run: generate_git_docs.py -h