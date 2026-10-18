"""Shared conversion errors, dependency guidance and subprocess boundary."""
import os
from pathlib import Path
import shutil
import subprocess


class ConvertError(Exception):
    def __init__(self, code, message, dependency=None):
        super().__init__(message)
        self.code = code
        self.dependency = dependency


GUIDES = {
    "markitdown": ("https://github.com/microsoft/markitdown", 'Create a dedicated venv, then run: python -m pip install "markitdown[pdf,docx,pptx,xlsx,xls]==0.1.3". Run this plugin with that Python interpreter.'),
    "antiword": ("https://github.com/grobian/antiword", "Use antiword 0.37 and its UTF-8.txt mapping files. Linux: apt install antiword; macOS: brew install antiword. Windows: provide a trusted portable build and mapping files. Set AIW_CONVERT_ANTIWORD and ANTIWORDHOME. No verified Windows download is bundled."),
    "pandoc": ("https://pandoc.org/installing.html", "Use Pandoc 3.6.4 or a compatible version with --sandbox. Prefer the official OS/CPU ZIP from https://github.com/jgm/pandoc/releases. Unzip to a user directory and set AIW_CONVERT_PANDOC. Windows alternative: winget install --source winget --exact --id JohnMacFarlane.Pandoc"),
    "browser": ("https://www.microsoft.com/edge/download", "Install Edge, Chrome (https://www.google.com/chrome/) or Chromium (https://www.chromium.org/getting-involved/download-chromium/), or set AIW_CONVERT_BROWSER to an installed executable. No verified portable browser download is bundled. Playwright is not required."),
    "md-converter": ("https://github.com/yuin/goldmark", "Install the matching AIW md-converter executable next to this plugin or set AIW_MD_CONVERTER. For source users: cd src/programs/md-converter; go build -o bin/md-converter . (Windows: bin/md-converter.exe). This is an explicit build, never run automatically."),
}


def missing(name, detail=""):
    url, command = GUIDES[name]
    guidance = {"name": name, "website": url, "instructions": command,
                "platform": os.name, "detail": detail}
    raise ConvertError("missing-dependency", f"Missing or unusable {name}. {detail}\n{url}\n{command}", guidance)


def executable(name):
    variable = "AIW_MD_CONVERTER" if name == "md-converter" else "AIW_CONVERT_" + name.upper()
    explicit = os.environ.get(variable)
    candidates = [explicit] if explicit else [name]
    if name == "md-converter" and not explicit:
        base = Path(__file__).resolve().parent
        suffix = ".exe" if os.name == "nt" else ""
        candidates = [str(base / (name + suffix)),
                      str(base.parents[1] / "programs" / name / "bin" / (name + suffix)), name]
    for candidate in candidates:
        found = shutil.which(candidate)
        if found:
            return found
    missing(name)


def run(command, timeout=120):
    try:
        result = subprocess.run(command, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                                stderr=subprocess.PIPE, timeout=timeout, check=False)
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise ConvertError("conversion", f"Backend failed: {exc}") from exc
    if result.returncode:
        detail = result.stderr.decode("utf-8", errors="replace")[-6000:]
        if "browser-not-found" in detail:
            missing("browser", detail)
        raise ConvertError("conversion", f"Backend exit {result.returncode}: {detail}")
    return result.stdout
