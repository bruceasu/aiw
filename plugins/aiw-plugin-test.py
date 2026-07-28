import json
import os
import subprocess
import unittest
from pathlib import Path


class PluginListTests(unittest.TestCase):
    def test_list_json_contains_known_entries(self):
        root = Path(__file__).resolve().parent.parent
        env = os.environ.copy()
        env["PYTHONUTF8"] = "1"
        result = subprocess.run(
            ["python", str(root / "plugins" / "aiw-plugin.py"), "list", "--json"],
            cwd=root,
            env=env,
            capture_output=True,
            text=True,
            check=True,
        )
        payload = json.loads(result.stdout)
        self.assertEqual(sorted(payload.keys()), ["plugins"])
        self.assertIsInstance(payload["plugins"], list)
        names = {item["name"] for item in payload["plugins"]}
        self.assertIn("aiw-file", names)
        self.assertIn("aiw-patch", names)

    def test_list_json_schema(self):
        root = Path(__file__).resolve().parent.parent
        env = os.environ.copy()
        env["PYTHONUTF8"] = "1"
        result = subprocess.run(
            ["python", str(root / "plugins" / "aiw-plugin.py"), "list", "--json"],
            cwd=root,
            env=env,
            capture_output=True,
            text=True,
            check=True,
        )
        payload = json.loads(result.stdout)
        self.assertGreater(len(payload["plugins"]), 0)
        for item in payload["plugins"]:
            self.assertIn("name", item)
            self.assertIn("short", item)
            self.assertIn("description", item)
            self.assertIn("commands", item)
            self.assertIsInstance(item["commands"], list)
            self.assertTrue(all(isinstance(cmd, str) for cmd in item["commands"]))
            self.assertIn("readOnly", item)
            self.assertIn("mutatesFiles", item)
            self.assertIn("requiresConfirmation", item)
            self.assertIn("outputFormat", item)
            self.assertIn(item["readOnly"], [True, False])
            self.assertIn(item["mutatesFiles"], [True, False])
            self.assertIn(item["requiresConfirmation"], [True, False])


if __name__ == "__main__":
    unittest.main()
