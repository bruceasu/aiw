# aiw git-del-branch.

Short: Delete a local and/or remote branch with confirmation.

Description:
Deletes branches locally and optionally from a remote. Includes confirmations to avoid accidental destructive actions.

Usage:
aiw git-del-branch. <branch> [--force] [--remote] [--remote-only] [--remote-name <name>]

Arguments:
- <branch> — The name of the branch to delete.
- --force — Force-delete local branch even if not merged (-D).
- --remote — Delete local branch AND remote branch.
- --remote-only — Delete only remote branch.
- --remote-name — Remote name to use (default: origin).

Examples:
- aiw git-del-branch. feature/foo

For full help run: generate_new_plugin_docs.py -h