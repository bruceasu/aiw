"""Capture read-only Git evidence. No tests, Git writes or trust changes."""
import subprocess
import json
import sys
from pathlib import Path
from datetime import datetime, timezone

base = Path(__file__).resolve().parent
root = base.parents[2]
output = base / 'verification-results' / ('closeout-git-' + datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
output.mkdir(parents=True, exist_ok=False)
commands = {
    'branch': ['branch', '--show-current'],
    'head': ['rev-parse', 'HEAD'],
    'base': ['rev-parse', 'develop'],
    'merge-base': ['merge-base', 'develop', 'HEAD'],
    'branch-history': ['log', '--oneline', '--left-right', 'develop...HEAD'],
    'branch-diff': ['diff', '--no-ext-diff', '--no-textconv', 'develop...HEAD', '--'],
    'working-diff': ['diff', '--no-ext-diff', '--no-textconv', 'HEAD', '--'],
    'status': ['status', '--short', '--untracked-files=all'],
    'diff': ['diff', '--no-ext-diff', '--no-textconv', 'develop', '--'],
    'untracked': ['ls-files', '--others', '--exclude-standard'],
}
records = []
for name, args in commands.items():
    command = ['git', '--no-optional-locks', '-C', str(root)] + args
    result = subprocess.run(command, capture_output=True)
    (output / (name + '.txt')).write_bytes(result.stdout)
    (output / (name + '.stderr.txt')).write_bytes(result.stderr)
    records.append({'command': command, 'exit_code': result.returncode})
    if result.returncode:
        break
(output / 'commands.json').write_text(json.dumps(records, indent=2), encoding='utf-8')
print('Evidence:', output)
print('Exit code:', records[-1]['exit_code'])
sys.exit(records[-1]['exit_code'])
