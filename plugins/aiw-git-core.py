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
    answer = sys.stdin.readline().strip()
    return answer.lower() in ("y", "yes")



def current_branch():
    """Return the current checked-out branch."""
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
            print(f"  {a.get('flag', ''):12}  {a.get('description','')}")
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
            lines.append(f"- {a.get('flag','')} — {a.get('description','')}")
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

def detect_default_branch(remote):
    """
    Detect the default branch for a remote.

    Tries:
        refs/remotes/<remote>/HEAD

    Falls back to:
        main
        master
    """

    try:
        ref = git_output(
            "git",
            "symbolic-ref",
            f"refs/remotes/{remote}/HEAD",
            "--short",
        )

        prefix = f"{remote}/"

        if ref.startswith(prefix):
            return ref[len(prefix):]

    except RuntimeError:
        pass

    # Fallback candidates.
    for candidate in ("main", "master"):
        if ref_exists(f"refs/heads/{candidate}"):
            return candidate

    raise RuntimeError(
        "cannot detect default branch; "
        "pass one explicitly, e.g.: aiw git update main"
    )



def ref_exists(ref):
    """Check whether a git ref exists."""
    result = subprocess.run(
        ["git", "show-ref", "--verify", "--quiet", ref]
    )
    return result.returncode == 0


def git_confirm(message, args):
    """
    Ask for confirmation unless --force is present.
    """
    if "--force" in args:
        return True

    answer = input(f"{message} [y/N]: ").strip().lower()
    return answer in ("y", "yes")

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
            print(f"  {a['flag']:12}  {a['description']}")
        print()
    if 'examples' in info:
        print("Examples:")
        for ex in info['examples']:
            print(f"  {ex}")


def main():
    if len(sys.argv) < 2:
        print('usage: core <sub> [args...]')
        return 2
    sub = sys.argv[1]
    args = sys.argv[2:]
    return dispatch(sub, args)


if __name__ == '__main__':
    sys.exit(main())
