#!/usr/bin/env python3
"""
aiw-git-core.py

Central implementation for aiw git subcommands. Thin Python port of the Go
implementation to be used by per-subcommand wrappers aiw-git-<sub>.py.
"""
import sys
import subprocess
import os
import shlex
from datetime import datetime


def run_cmd(cmd, check=True):
    print(">", " ".join(shlex.quote(c) for c in cmd), file=sys.stderr)
    p = subprocess.Popen(cmd)
    p.communicate()
    if check and p.returncode != 0:
        raise SystemExit(p.returncode)
    return p.returncode
#!/usr/bin/env python3
"""aiw-git-core.py

Utility library for aiw git plugin wrappers. This module provides
small helpers for running git commands, printing help from per-plugin
metadata, and generating markdown docs. It intentionally does NOT
implement dispatch for subcommands; each `aiw-git-<sub>.py` wrapper
should be responsible for its own behavior and metadata.
"""
import sys
import subprocess
import os
import shlex
from datetime import datetime


def run_cmd(cmd, check=True):
    """Run an external command and print the invocation to stderr.
    Returns the subprocess return code or raises SystemExit on failure
    when check is True.
    """
    print(">", " ".join(shlex.quote(c) for c in cmd), file=sys.stderr)
    p = subprocess.Popen(cmd)
    p.communicate()
    if check and p.returncode != 0:
        raise SystemExit(p.returncode)
    return p.returncode


def git_output(cmd):
    try:
        out = subprocess.check_output(cmd, stderr=subprocess.DEVNULL)
        return out.decode('utf-8')
    except subprocess.CalledProcessError:
        raise


def has_flag(args, flag):
    return flag in args


def git_confirm(prompt, args):
    if has_flag(args, "--force"):
        return True
    print(f"warning: {prompt}\nProceed? [y/N] ", end="", file=sys.stderr)
    resp = sys.stdin.readline().strip()
    return resp.lower() == 'y'


def current_branch():
    out = git_output(["git", "rev-parse", "--abbrev-ref", "HEAD"])
    return out.strip()


def has_remote(name):
    p = subprocess.Popen(["git", "remote", "get-url", name], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    p.communicate()
    return p.returncode == 0


def print_help_meta(meta):
    """Print help based on a plugin's META dict.

    Expected keys in meta: name, short, long, usage, args (list), examples (list).
    """
    print(meta.get('name', ''))
    print()
    print(meta.get('short', ''))
    print()
    if meta.get('long'):
        print(meta['long'])
        print()
    if meta.get('usage'):
        print('Usage:')
        print(meta['usage'])
        print()
    if meta.get('args'):
        print('Arguments:')
        for a in meta['args']:
            print(f"  {a.get('flag', ''):12}  {a.get('desc','')}")
        print()
    if meta.get('examples'):
        print('Examples:')
        for ex in meta['examples']:
            print(f"  {ex}")


def generate_md_from_meta(meta, outpath):
    """Generate a markdown doc file at outpath from a META dict."""
    lines = []
    lines.append(f"# {meta.get('name','')}")
    lines.append("")
    lines.append(f"Short: {meta.get('short','')}")
    lines.append("")
    if meta.get('long'):
        lines.append("Description:")
        lines.append(meta.get('long',''))
        lines.append("")
    if meta.get('usage'):
        lines.append("Usage:")
        lines.append(meta.get('usage'))
        lines.append("")
    if meta.get('args'):
        lines.append("Arguments:")
        for a in meta['args']:
            lines.append(f"- {a.get('flag','')} — {a.get('desc','')}")
        lines.append("")
    if meta.get('examples'):
        lines.append("Examples:")
        for ex in meta['examples']:
            lines.append(f"- {ex}")
        lines.append("")
    lines.append(f"For full help run: {os.path.basename(sys.argv[0])} -h")
    # Write file
    with open(outpath, 'w', encoding='utf-8') as f:
        f.write('\n'.join(lines))


if __name__ == '__main__':
    print('aiw-git-core is a library; run one of the aiw-git-<sub> wrappers')
    sys.exit(2)
    print(info['short'])
    print()
    print(info['long'])
    print()
    if 'usage' in info:
        print("Usage:")
        print(info['usage'])
        print()
    if 'args' in info:
        print("Arguments:")
        for a in info['args']:
            print(f"  {a['flag']:12}  {a['desc']}")
        print()
    if 'examples' in info:
        print("Examples:")
        for ex in info['examples']:
            print(f"  {ex}")


# Help metadata for subcommands
HELP = {
    'save': {
        'name': 'aiw git save',
        'short': 'Stage all changes and commit (default message: wip).',
        'long': 'Stages all working-tree changes and commits them. If a message is provided, it is used as the commit message; otherwise the message "wip" is used.',
        'usage': 'aiw git save [message]',
        'args': [],
        'examples': ['aiw git save', 'aiw git save "fix tests"']
    },
    'undo': {
        'name': 'aiw git undo',
        'short': 'Undo last commit while keeping changes (or discard with --hard).',
        'long': 'Resets HEAD to the previous commit. By default changes are kept in the working tree. Use --hard to discard changes (dangerous).',
        'usage': 'aiw git undo [--hard] [--force]',
        'args': [
            {'flag': '--hard', 'desc': 'Also discard working-tree changes.'},
            {'flag': '--force', 'desc': 'Skip confirmation prompts.'},
        ],
        'examples': ['aiw git undo', 'aiw git undo --hard --force']
    },
    'st': {
        'name': 'aiw git st',
        'short': 'Show concise working-tree status.',
        'long': 'Shortcut for a concise git status display.',
        'usage': 'aiw git st',
        'args': [],
        'examples': ['aiw git st']
    },
    'status': {
        'name': 'aiw git status',
        'short': 'Show concise working-tree status.',
        'long': 'Alias of `aiw git st`.',
        'usage': 'aiw git status',
        'args': [],
        'examples': ['aiw git status']
    },
    'log': {
        'name': 'aiw git log',
        'short': 'Show a formatted commit log with several styles.',
        'long': 'Displays the commit history. Supports styles: lg (default), l (one-line), hist (absolute dates).',
        'usage': 'aiw git log [lg|l|hist] [-n <count>]',
        'args': [
            {'flag': 'lg|l|hist', 'desc': 'Choose log style.'},
            {'flag': '-n <count>', 'desc': 'Number of commits to show.'},
        ],
        'examples': ['aiw git log lg', 'aiw git log -n 50']
    },
    'gc': {
        'name': 'aiw git gc',
        'short': 'Run git gc aggressively (destructive).',
        'long': 'Runs `git gc --aggressive --prune=now`. This rewrites history objects and may prevent reflog recovery; confirm before running.',
        'usage': 'aiw git gc [--force]',
        'args': [
            {'flag': '--force', 'desc': 'Skip the confirmation prompt.'},
        ],
        'examples': ['aiw git gc', 'aiw git gc --force']
    },
    'unpushed': {
        'name': 'aiw git unpushed',
        'short': 'Show commits not yet pushed to the upstream.',
        'long': 'Lists commits that are present locally but not on the upstream branch.',
        'usage': 'aiw git unpushed',
        'args': [],
        'examples': ['aiw git unpushed']
    },
    'unpulled': {
        'name': 'aiw git unpulled',
        'short': 'Show commits on remote not yet pulled locally.',
        'long': 'Lists commits on the upstream branch that are not present locally.',
        'usage': 'aiw git unpulled',
        'args': [],
        'examples': ['aiw git unpulled']
    },
    'outstanding': {
        'name': 'aiw git outstanding',
        'short': 'Interactive rebase against upstream.',
        'long': 'Runs an interactive rebase against the upstream branch (@{u}).',
        'usage': 'aiw git outstanding',
        'args': [],
        'examples': ['aiw git outstanding']
    },
    'whatchanged': {
        'name': 'aiw git whatchanged',
        'short': 'Show what changed (alias).',
        'long': 'Convenience wrapper to view changes (delegates to git).',
        'usage': 'aiw git whatchanged [args]',
        'args': [],
        'examples': ['aiw git whatchanged']
    },
}


def main():
    if len(sys.argv) < 2:
        print('usage: core <sub> [args...]')
        return 2
    sub = sys.argv[1]
    args = sys.argv[2:]
    return dispatch(sub, args)


if __name__ == '__main__':
    sys.exit(main())
