import importlib.util
import sys
import tempfile
import unittest
from pathlib import Path


PLUGINS_DIR = str(Path("plugins").resolve())
if PLUGINS_DIR not in sys.path:
    sys.path.insert(0, PLUGINS_DIR)


def load():
    spec = importlib.util.spec_from_file_location("aiw_file", "plugins/aiw-file.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class FileCodecTests(unittest.TestCase):
    def setUp(self):
        self.file = load()

    def test_detect_utf8_bom_utf16_and_gb18030(self):
        codec = self.file.read_file.__globals__["detect"]
        self.assertEqual(codec(b"\xef\xbb\xbf" + b"abc")[0], "utf-8")
        self.assertEqual(codec("\xe4\xb8\xad\xe6\x96\x87".encode("utf-16"))[0], "utf-16")
        self.assertEqual(codec("\xe4\xb8\xad\xe6\x96\x87".encode("gb18030"), "gb18030")[0], "gb18030")

    def test_write_preserves_existing_style(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "notes.txt"
            path.write_bytes(b"\xef\xbb\xbfone\r\ntwo\r\n")
            self.file.atomic_write(str(path), "three\nfour\n", "utf-8", True, "crlf")
            self.assertEqual(path.read_bytes(), b"\xef\xbb\xbfthree\r\nfour\r\n")


if __name__ == "__main__":
    unittest.main()
