# aiw git change-author

Short: Change the author on the last commit (or rewrite history).

Description:
Convenience helper to amend the last commit author with --author "Name <email>". For wide rewrites use proper history-rewrite tools.

Usage:
aiw git change-author "Name <email>" [--all --filter-name <old> ...] [--force]

Examples:
- aiw git change-author "Alice <alice@example.com>"

For full help run: generate_new_plugin_docs.py -h
