# aiw git-export

Short: Export the current working directory to a tar archive.

Description:
Creates a tar archive of the current working directory, including all tracked files.

Usage:
aiw git-export <ref> [file.zip]

Arguments:
- <ref> — The reference to export.
- [file.zip] — The output zip file (default: <ref>.zip).

Examples:
- aiw git-export myproject.zip

For full help run: generate_new_plugin_docs.py -h