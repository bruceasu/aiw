# aiw git get

Short: Shallow single-branch clone helper.

Description:
Performs a shallow single-branch clone with optional branch, directory, and depth flags.

Usage:
aiw git get <url> [-b <branch>] [-d <dir>] [--depth <n>] [--full]

Examples:
- aiw git get https://example.com/repo.git -b main -d repo

For full help run: generate_git_docs.py -h