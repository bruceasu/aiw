"""Offline regression cases for preserving parent edits; execution needs authorization."""
from pathlib import Path
import os
import subprocess
import sys
import tempfile
import unittest

PLUGIN = Path(__file__).resolve().parents[1] / 'src/plugins/aiw-git/git-wt.py'
FD = 'FD-107'
PLAN = 'docs/features/FD-107/FD-107_TEST.md'


class ParentEditTests(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(prefix='fd107-parent-')
        self.addCleanup(self.scratch.cleanup)
        self.root = Path(self.scratch.name) / 'repo'
        self.root.mkdir()
        hooks = Path(self.scratch.name) / 'hooks'
        hooks.mkdir()
        self.env = os.environ.copy()
        self.env.update(GIT_CONFIG_NOSYSTEM='1', GIT_CONFIG_GLOBAL=os.devnull,
                        GIT_CONFIG_COUNT='1', GIT_CONFIG_KEY_0='core.hooksPath',
                        GIT_CONFIG_VALUE_0=str(hooks), GIT_TERMINAL_PROMPT='0')
        self.env.pop('GIT_INDEX_FILE', None)
        self.git('init', '-b', 'main')
        self.git('config', 'user.name', 'FD Test')
        self.git('config', 'user.email', 'fd@example.invalid')
        (self.root / PLAN).parent.mkdir(parents=True)
        (self.root / PLAN).write_text('# FD-107\n**Status:** Open\n')
        (self.root / '.gitignore').write_text('.ai/\n.wt/\n')
        for name in ('staged.txt', 'unstaged.txt', 'shared.txt'):
            (self.root / name).write_text('base\n')
        self.git('add', '.')
        self.git('commit', '-m', 'base')

    def command(self, *args, cwd=None):
        return subprocess.run(args, cwd=cwd or self.root, env=self.env, text=True,
                              capture_output=True, check=False, timeout=30)

    def git(self, *args, cwd=None):
        result = self.command('git', *args, cwd=cwd)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        return result.stdout

    def wt(self, *args):
        return self.command(sys.executable, str(PLUGIN), *args)

    def add(self):
        result = self.wt('add', FD)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        return self.root / '.wt' / FD

    def test_creation_keeps_unrelated_parent_edits(self):
        (self.root / 'unstaged.txt').write_text('local\n')
        tree = self.add()
        self.assertEqual((tree / 'unstaged.txt').read_text(), 'base\n')
        self.assertEqual((self.root / 'unstaged.txt').read_text(), 'local\n')

    def test_related_edit_requires_explicit_head_choice(self):
        (self.root / PLAN).write_text('changed FD plan\n')
        self.assertNotEqual(self.wt('add', FD).returncode, 0)
        result = self.wt('add', FD, '--from-head')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual((self.root / '.wt' / FD / PLAN).read_text(), '# FD-107\n**Status:** Open\n')
        self.assertEqual((self.root / PLAN).read_text(), 'changed FD plan\n')

    def test_delivery_excludes_staged_parent_content_and_preserves_staging(self):
        tree = self.add()
        (tree / 'feature.txt').write_text('feature\n')
        self.git('add', 'feature.txt', cwd=tree)
        self.git('commit', '-m', 'feature', cwd=tree)
        source = self.git('rev-parse', 'HEAD', cwd=tree).strip()
        (self.root / 'staged.txt').write_text('staged local\n')
        self.git('add', 'staged.txt')
        (self.root / 'unstaged.txt').write_text('unstaged local\n')
        (self.root / 'untracked.txt').write_text('untracked local\n')
        status = self.git('status', '--porcelain')
        staged = self.git('diff', '--cached', '--binary')
        result = self.wt('local-merge', FD)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(self.git('show', 'HEAD:staged.txt'), 'base\n')
        self.assertEqual(self.git('show', 'HEAD:feature.txt'), 'feature\n')
        self.assertEqual(self.git('status', '--porcelain'), status)
        self.assertEqual(self.git('diff', '--cached', '--binary'), staged)
        self.assertEqual((self.root / 'unstaged.txt').read_text(), 'unstaged local\n')
        self.assertEqual((self.root / 'untracked.txt').read_text(), 'untracked local\n')
        self.assertIn('FD-Source: ' + source, self.git('show', '-s', '--format=%B'))
        self.assertEqual(len(self.git('rev-list', '--parents', '-n', '1', 'HEAD').split()), 2)
        self.assertFalse(tree.exists())

    def test_branch_conflict_with_dirty_parent_stops_without_mutation(self):
        tree = self.add()
        (tree / 'shared.txt').write_text('feature\n')
        self.git('add', 'shared.txt', cwd=tree)
        self.git('commit', '-m', 'feature', cwd=tree)
        (self.root / 'shared.txt').write_text('parent\n')
        self.git('add', 'shared.txt')
        self.git('commit', '-m', 'parent')
        (self.root / 'untracked.txt').write_text('local\n')
        head = self.git('rev-parse', 'HEAD')
        status = self.git('status', '--porcelain')
        result = self.wt('local-merge', FD)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('make the parent clean', result.stderr)
        self.assertEqual(self.git('rev-parse', 'HEAD'), head)
        self.assertEqual(self.git('status', '--porcelain'), status)
        self.assertEqual((self.root / 'shared.txt').read_text(), 'parent\n')
        self.assertTrue(tree.exists())
