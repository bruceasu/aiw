"""Explicit format matrix and lazy backend adapters. No shell commands."""
import contextlib
import io
from pathlib import Path
import re
import tempfile
import zipfile
import xml.etree.ElementTree as ET

from common import ConvertError, executable, missing, run
from resources import prepare_html

EXTRACT = {"pdf", "docx", "pptx", "xlsx", "xls", "html", "htm"}
NATIVE = {"txt", "csv", "json", "xml", "md", "markdown"}
PANDOC_INPUT = {"odt", "epub", "rst"}
PANDOC_OUTPUT = {"docx", "odt", "epub", "rst"}
BINARY = {"pdf", "docx", "odt", "epub"}
MIMES = {"md": "text/markdown", "txt": "text/plain", "html": "text/html", "pdf": "application/pdf",
         "docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
         "odt": "application/vnd.oasis.opendocument.text", "epub": "application/epub+zip", "rst": "text/x-rst"}


def route(source, target):
    ext = source.suffix.lower().lstrip(".")
    if ext == "markdown":
        ext = "md"
    if target in {"md", "txt"} and ext in EXTRACT | NATIVE | {"doc"} | PANDOC_INPUT:
        return ext
    if ext == "md" and target in {"html", "pdf"} | PANDOC_OUTPUT:
        return ext
    if ext in {"html", "htm"} and target == "pdf":
        return ext
    raise ConvertError("arguments", f"Unsupported conversion: {ext or '(no extension)'} -> {target}")


def helper(source, target, output):
    run([executable("md-converter"), "--input", str(source), "--output", str(output), "--format", target])


def plain(markdown, directory):
    source, output = directory / "text.md", directory / "text.txt"
    source.write_text(markdown, encoding="utf-8")
    helper(source, "txt", output)
    return output.read_text(encoding="utf-8")


def checked_pandoc_source(source, directory):
    """Read once, validate, then convert the same byte snapshot."""
    data = source.read_bytes()
    if source.suffix.lower() == ".rst":
        text = data.decode("utf-8-sig")
        if re.search(r"^\s*\.\.\s+(?:include|raw|csv-table|image|figure)::", text, re.I | re.M):
            raise ConvertError("conversion", "RST resource/include/raw directives are not supported")
    else:
        with zipfile.ZipFile(io.BytesIO(data)) as archive:
            if sum(item.file_size for item in archive.infolist()) > 128 * 1024 * 1024:
                raise ConvertError("conversion", "Document ZIP exceeds the inspection limit")
            for item in archive.infolist():
                if item.filename.lower().endswith((".xml", ".rels", ".xhtml", ".html", ".opf", ".css", ".svg")):
                    text = archive.read(item).decode("utf-8", errors="strict")
                    if re.search(r"<!ENTITY|<!DOCTYPE[^>]*(?:SYSTEM|PUBLIC)|@import|url\s*\(", text, re.I):
                        raise ConvertError("conversion", "External or uninspectable ZIP document resources are not allowed")
                    if not item.filename.lower().endswith(".css"):
                        for node in ET.fromstring(text).iter():
                            for name, value in node.attrib.items():
                                key = name.rsplit("}", 1)[-1].lower()
                                if key in {"href", "src", "target", "data", "base"} and re.match(r"\s*(?:[a-z][a-z0-9+.-]*:|//|/|\\)", value, re.I):
                                    raise ConvertError("conversion", "External ZIP document reference is not allowed")
                                if key == "targetmode" and value.lower() == "external":
                                    raise ConvertError("conversion", "External ZIP document relationship is not allowed")
    snapshot = directory / ("checked" + source.suffix.lower())
    snapshot.write_bytes(data)
    return snapshot


def convert(source, target, output, directory):
    ext = route(source, target)
    if target in {"html", "pdf"}:
        if ext in {"html", "htm"}:
            safe = directory / "passive.html"
            safe.write_text(prepare_html(source), encoding="utf-8")
            source = safe
        helper(source, target, output)
        return
    if ext == "md" and target in PANDOC_OUTPUT:
        pandoc = executable("pandoc")
        safe = directory / "pandoc.html"
        helper(source, "html", safe)
        # Mermaid is deliberately kept as source code in Pandoc exports.
        text = safe.read_text(encoding="utf-8")
        text = re.sub(r"<script\b[^>]*>.*?</script>", "", text, flags=re.I | re.S)
        safe.write_text(text, encoding="utf-8")
        run([pandoc, "--sandbox", "--from=html", "--to=" + target, "--output", str(output), str(safe)])
        return
    if ext in PANDOC_INPUT:
        checked = checked_pandoc_source(source, directory)
        data = run([executable("pandoc"), "--sandbox", "--from=" + ext, "--to=gfm", str(checked)])
        markdown = data.decode("utf-8")
    elif ext == "doc":
        data = run([executable("antiword"), "-m", "UTF-8.txt", str(source)])
        text = data.decode("utf-8", errors="strict")
        if "\ufffd" in text:
            raise ConvertError("conversion", "antiword returned replacement characters; check the document encoding")
        output.write_text(text, encoding="utf-8")
        return
    elif ext in NATIVE:
        markdown = source.read_text(encoding="utf-8-sig")
        if ext in {"csv", "json", "xml"}:
            fence = "`" * max(3, max((len(x) + 1 for x in re.findall(r"`+", markdown)), default=3))
            markdown = f"{fence}{ext}\n{markdown}\n{fence}\n"
        elif ext == "txt" and target == "txt":
            output.write_text(markdown, encoding="utf-8")
            return
    else:
        try:
            with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
                from markitdown import MarkItDown
                result = MarkItDown(enable_plugins=False).convert(str(source))
                markdown = result.text_content
        except ImportError as exc:
            missing("markitdown", str(exc))
        except Exception as exc:
            # Missing format extras may be surfaced by the converter itself.
            if "MissingDependency" in type(exc).__name__ or "dependenc" in str(exc).lower():
                missing("markitdown", str(exc))
            raise ConvertError("conversion", f"MarkItDown failed: {exc}") from exc
    output.write_text(plain(markdown, directory) if target == "txt" else markdown, encoding="utf-8")
