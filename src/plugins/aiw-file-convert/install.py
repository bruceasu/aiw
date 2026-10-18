"""Explicit, digest-authorized portable ZIP installation. Never called by conversion."""
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import platform
import re
import shutil
import stat
import tempfile
from urllib.parse import urlsplit
import urllib.request
import zipfile

from common import ConvertError

HOSTS = {"github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com",
         "downloads.sourceforge.net", "storage.googleapis.com"}
FIELDS = {"dependency", "version", "url", "sha256", "system", "machine", "target", "entrypoint"}
VARIABLES = {"pandoc": "AIW_CONVERT_PANDOC", "antiword": "AIW_CONVERT_ANTIWORD",
             "browser": "AIW_CONVERT_BROWSER", "md-converter": "AIW_MD_CONVERTER"}


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise ConvertError("install", "Redirect rejected. Supply and approve a trusted direct ZIP URL.")


def install(plan_file, approval):
    def unique(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ConvertError("arguments", "Duplicate installation plan key")
            result[key] = value
        return result
    plan = json.loads(Path(plan_file).read_text(encoding="utf-8"), object_pairs_hook=unique)
    if not isinstance(plan, dict) or set(plan) != FIELDS or not all(isinstance(x, str) and x for x in plan.values()):
        raise ConvertError("arguments", "Install plan must contain exactly these string fields: " + ", ".join(sorted(FIELDS)))
    if plan["dependency"] not in VARIABLES or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._+-]{0,80}", plan["version"]):
        raise ConvertError("arguments", "Unknown dependency or invalid fixed version")
    if plan["system"].lower() != platform.system().lower() or plan["machine"].lower() != platform.machine().lower():
        raise ConvertError("arguments", "Install plan OS/CPU does not match this machine")
    url = urlsplit(plan["url"])
    if url.scheme != "https" or url.hostname not in HOSTS or url.username or url.password or url.port not in {None, 443} or url.fragment:
        raise ConvertError("arguments", "Install URL must be a trusted HTTPS direct ZIP URL")
    if not re.fullmatch(r"[0-9a-f]{64}", plan["sha256"]):
        raise ConvertError("arguments", "A lowercase SHA256 release checksum is required")
    entry = PurePosixPath(plan["entrypoint"])
    if entry.is_absolute() or ".." in entry.parts or "\\" in str(entry) or ":" in str(entry):
        raise ConvertError("arguments", "Unsafe installation entrypoint")
    requested_target = Path(plan["target"])
    if not requested_target.is_absolute():
        raise ConvertError("arguments", "Install target must be an explicit absolute path; relative and ~ paths are not allowed")
    target = requested_target.resolve()
    # The displayed, approved and executed snapshot must name the same actual
    # destination, including resolution of existing ancestor links.
    plan = dict(plan, target=str(target))
    if target.exists() or target.is_symlink() or not target.parent.is_dir():
        raise ConvertError("install", "Target must be a new directory with an existing parent")
    digest = hashlib.sha256(json.dumps(plan, sort_keys=True, separators=(",", ":"), ensure_ascii=True).encode("utf-8")).hexdigest()
    result = {"plan": plan, "plan-sha256": digest, "approved": approval == digest,
              "instructions": "Review this exact plan. To install, repeat with --approve-install " + digest}
    if not approval:
        return result
    if approval != digest:
        raise ConvertError("arguments", "Installation approval does not match this plan snapshot")
    # No download happens before every authorization and destination check above.
    with tempfile.TemporaryDirectory(prefix="aiw-convert-install-") as temporary:
        package, extracted = Path(temporary) / "package.zip", Path(temporary) / "contents"
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
        hasher, total = hashlib.sha256(), 0
        with opener.open(plan["url"], timeout=30) as response, package.open("xb") as output:
            while chunk := response.read(1024 * 1024):
                total += len(chunk)
                if total > 512 * 1024 * 1024:
                    raise ConvertError("install", "ZIP download exceeds 512 MiB")
                hasher.update(chunk)
                output.write(chunk)
        if hasher.hexdigest() != plan["sha256"]:
            raise ConvertError("install", "Release checksum mismatch; nothing installed")
        extracted.mkdir()
        with zipfile.ZipFile(package) as archive:
            infos = archive.infolist()
            if len(infos) > 20000 or sum(x.file_size for x in infos) > 1024 * 1024 * 1024:
                raise ConvertError("install", "Expanded ZIP exceeds its size or entry limit")
            names = set()
            for item in infos:
                path = PurePosixPath(item.filename)
                mode = item.external_attr >> 16
                if (path.is_absolute() or ".." in path.parts or "\\" in item.filename or ":" in item.filename
                        or stat.S_ISLNK(mode) or (stat.S_IFMT(mode) not in {0, stat.S_IFREG, stat.S_IFDIR})
                        or any(part.rstrip(" .") != part or part.upper().split(".")[0] in {"CON", "PRN", "AUX", "NUL", *("COM" + str(x) for x in range(1, 10)), *("LPT" + str(x) for x in range(1, 10))} for part in path.parts)):
                    raise ConvertError("install", "Unsafe ZIP entry: " + item.filename)
                key = str(path).casefold()
                if key in names:
                    raise ConvertError("install", "Duplicate ZIP entry")
                names.add(key)
                dest = extracted.joinpath(*path.parts)
                if item.is_dir():
                    dest.mkdir(parents=True, exist_ok=True)
                else:
                    dest.parent.mkdir(parents=True, exist_ok=True)
                    with archive.open(item) as source, dest.open("xb") as output:
                        shutil.copyfileobj(source, output)
                    dest.chmod(0o755 if mode & 0o111 else 0o644)
        binary = extracted.joinpath(*entry.parts)
        if not binary.is_file() or binary.stat().st_size == 0 or (os.name != "nt" and not os.access(binary, os.X_OK)):
            raise ConvertError("install", "Entrypoint is missing, empty or not executable")
        # Atomic directory reservation: never merge into an existing installation.
        target.mkdir()
        try:
            for source in extracted.rglob("*"):
                dest = target / source.relative_to(extracted)
                if source.is_dir():
                    dest.mkdir(exist_ok=True)
                else:
                    dest.parent.mkdir(parents=True, exist_ok=True)
                    with source.open("rb") as inp, dest.open("xb") as out:
                        shutil.copyfileobj(inp, out)
                    dest.chmod(source.stat().st_mode & 0o777)
        except Exception as exc:
            raise ConvertError("install", f"Installation incomplete; inspect owned recovery directory {target}: {exc}") from exc
    result["installed"] = str(target / Path(*entry.parts))
    result["instructions"] = f"Set {VARIABLES[plan['dependency']]} to {result['installed']}. Then explicitly retry your conversion."
    return result
