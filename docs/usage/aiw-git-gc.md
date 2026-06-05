# aiw git gc

Short: Run git gc aggressively (destructive).

Description:
Runs git gc --aggressive --prune=now. This rewrites history objects and may prevent reflog recovery; confirm before running.

Usage:
aiw git gc [--force]

Arguments:
- --force — Skip confirmation prompt.

Examples:
- aiw git gc
- aiw git gc --force

For full help run: generate_git_docs.py -h