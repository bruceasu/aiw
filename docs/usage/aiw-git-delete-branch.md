# aiw git delete-branch

Short: Delete a local and/or remote branch with confirmation.

Description:
Deletes branches locally and optionally from a remote. Includes confirmations to avoid accidental destructive actions.

Usage:
aiw git delete-branch <branch> [--force] [--remote] [--remote-only] [--remote-name <name>]

Examples:
- aiw git delete-branch feature/foo

For full help run: generate_git_docs.py -h