#!/usr/bin/env python3
import contextlib
import importlib.util
import io
import os
import unittest
from unittest import mock


HERE = os.path.dirname(__file__)
MODULE_PATH = os.path.join(HERE, 'git-merge-repo.py')


def load_module():
    spec = importlib.util.spec_from_file_location('aiw_git_merge_repo_test_target', MODULE_PATH)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class MergeRepoTests(unittest.TestCase):
    def test_subdir_must_be_relative_and_non_empty(self):
        module = load_module()
        self.assertTrue(module._valid_subdir('service-a'))
        self.assertTrue(module._valid_subdir('services/service-a'))
        self.assertFalse(module._valid_subdir(''))
        self.assertFalse(module._valid_subdir('/service-a'))
        self.assertFalse(module._valid_subdir('../service-a'))
        self.assertFalse(module._valid_subdir('C:/service-a'))

    def test_missing_subdir_defaults_to_cancel(self):
        module = load_module()
        with mock.patch.object(module.sys, 'stdin', io.StringIO('\n')):
            self.assertIsNone(module._choose_subdir(['repo1', 'repo2']))

    def test_help_describes_history_and_target_branch_safety(self):
        module = load_module()
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            module.core.print_help_meta(module.META)
        rendered = output.getvalue()
        self.assertIn('temporary branch', module.META['long'])
        self.assertIn('conflicts are rejected', module.META['long'])
        self.assertIn('repo1-to-subdir', rendered)


if __name__ == '__main__':
    unittest.main()
