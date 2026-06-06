# aiw git-rebase-onto

Short: Move a branch onto a new base.

Description:
Rebases a branch onto a different base using git rebase --onto.

Usage:
aiw git-rebase-onto <new-base> <old-base> [branch] [--force]

Examples:
- aiw git-rebase-onto develop master feature
- aiw git-rebase-onto main old-main

For full help run: generate_new_plugin_docs.py -h