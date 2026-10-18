"""Source-tree shim; packaging replaces this with the complete codec module."""
import importlib.util
from pathlib import Path

_spec = importlib.util.spec_from_file_location(
    "_aiw_source_codec", Path(__file__).resolve().parents[1] / "aiw_codec.py"
)
_codec = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_codec)

CodecError = _codec.CodecError
detect = _codec.detect
