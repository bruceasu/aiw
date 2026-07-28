import importlib.util
import sys
import unittest
from pathlib import Path


PLUGINS_DIR = str(Path("plugins").resolve())
if PLUGINS_DIR not in sys.path:
    sys.path.insert(0, PLUGINS_DIR)


def load():
    spec = importlib.util.spec_from_file_location("aiw_patch", "plugins/aiw-patch.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class PatchCodecTests(unittest.TestCase):
    def setUp(self):
        self.patch = load()

    def test_decode_patch_uses_shared_codec(self):
        self.assertEqual(self.patch.decode_patch(b"\xef\xbb\xbfhello"), "hello")
        self.assertEqual(self.patch.decode_patch("abc".encode("utf-16")), "abc")


if __name__ == "__main__":
    unittest.main()
