# aiw git-get-file

Short: Extract a file version from another branch/commit.

Description:
Retrieves a file from a different branch or commit and writes it into the working tree.

Usage:
aiw git-get-file <commit|branch> <path>

Arguments:
- <commit|branch> — The commit or branch from which to extract the file.
- <path> — The path to the file to extract.

Examples:
- aiw git-get-file origin/main path/to/file

For full help run: generate_new_plugin_docs.py -h