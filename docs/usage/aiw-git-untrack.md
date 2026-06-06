# aiw-git-untrack

Short: Remove file from HEAD while keeping working tree copy.

Description:
Removes a path from the index but leaves the file in the working tree (safe remove from history/commit).

Usage:
aiw-git-untrack <path>

Arguments:
- <path> — The path to un-track.

Examples:
- aiw-git-untrack path/to/file

For full help run: generate_new_plugin_docs.py -h