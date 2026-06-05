# aiw git undo

Short: Undo last commit while keeping changes (or discard with --hard).

Description:
Resets HEAD to the previous commit. By default changes are kept in the working tree. Use --hard to discard changes (dangerous).

Usage:
aiw git undo [--hard] [--force]

Arguments:
- --hard — Also discard working-tree changes.
- --force — Skip confirmation prompts.

Examples:
- aiw git undo
- aiw git undo --hard --force

For full help run: generate_git_docs.py -h