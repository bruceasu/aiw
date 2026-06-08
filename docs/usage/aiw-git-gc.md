# aiw git gc

Short: Run git gc aggressively (destructive).

Description:
Runs git gc --aggressive --prune=now. This rewrites history objects and may prevent reflog recovery; confirm before running.

Usage:
aiw git gc [--force]

Arguments:
- --force 鈥?Skip confirmation prompt.

Examples:
- aiw git gc
- --force

For full help run: generate_new_plugin_docs.py -h
