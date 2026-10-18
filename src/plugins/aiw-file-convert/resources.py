"""Turn local HTML into a self-contained, passive document before rendering."""
import base64
from html import escape
from html.parser import HTMLParser
from pathlib import Path
import re
from urllib.parse import unquote, urlsplit

from common import ConvertError

MIMES = {".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
         ".gif": "image/gif", ".webp": "image/webp", ".svg": "image/svg+xml",
         ".woff": "font/woff", ".woff2": "font/woff2", ".ttf": "font/ttf", ".otf": "font/otf"}


def resource(root, value):
    parsed = urlsplit(value.strip())
    if parsed.scheme or parsed.netloc or parsed.query or not parsed.path or "\\" in parsed.path:
        raise ConvertError("conversion", f"Only local document resources are allowed: {value}")
    path = (root / unquote(parsed.path)).resolve()
    if not path.is_relative_to(root) or not path.is_file():
        raise ConvertError("conversion", f"Resource is missing or outside the input directory: {value}")
    if path.stat().st_size > 32 * 1024 * 1024:
        raise ConvertError("conversion", "Resource exceeds 32 MiB")
    return path


def data_url(root, value):
    if value.startswith("data:"):
        if not re.match(r"^data:(?:image/(?:png|jpeg|gif|webp)|font/(?:woff2?|ttf|otf));base64,[A-Za-z0-9+/=\s]+$", value):
            raise ConvertError("conversion", "Unsupported data resource")
        return value
    path = resource(root, value)
    mime = MIMES.get(path.suffix.lower())
    if not mime:
        raise ConvertError("conversion", f"Unsupported resource type: {path.suffix}")
    # SVG may contain external resources or active content; keep this boundary strict.
    if mime == "image/svg+xml":
        raise ConvertError("conversion", "SVG resources are not allowed; use PNG, JPEG, GIF or WebP")
    return "data:" + mime + ";base64," + base64.b64encode(path.read_bytes()).decode("ascii")


def css(text, root):
    # Escapes/comments make CSS URLs ambiguous. Reject them instead of guessing.
    if "\\" in text or "/*" in text or re.search(r"@import|expression\s*\(|</style", text, re.I):
        raise ConvertError("conversion", "Unsupported CSS escape, comment, import or active content")
    return re.sub(r"url\(\s*(['\"]?)(.*?)\1\s*\)",
                  lambda m: 'url("' + data_url(root, m[2]) + '")', text, flags=re.I)


class PassiveHTML(HTMLParser):
    def __init__(self, root):
        super().__init__(convert_charrefs=True)
        self.root, self.parts, self.in_style = root, [], False

    def handle_starttag(self, tag, attrs):
        if tag in {"script", "iframe", "frame", "frameset", "object", "embed", "base", "form", "portal", "svg", "math", "video", "audio", "source"}:
            raise ConvertError("conversion", f"Active HTML element is not supported: {tag}")
        values = dict(attrs)
        if tag == "meta" and values.get("http-equiv", "").lower() != "":
            raise ConvertError("conversion", "HTML http-equiv is not allowed")
        if tag == "link":
            if values.get("rel", "").lower() != "stylesheet" or not values.get("href"):
                raise ConvertError("conversion", "Only local stylesheet links are supported")
            path = resource(self.root, values["href"])
            self.parts.append("<style>" + css(path.read_text(encoding="utf-8"), path.parent) + "</style>")
            return
        rendered = []
        for key, value in attrs:
            value = value or ""
            if key.startswith("on") or key in {"srcdoc", "srcset", "action", "formaction", "background", "ping", "manifest"}:
                raise ConvertError("conversion", f"Active HTML attribute is not supported: {key}")
            if key == "src":
                if tag != "img":
                    raise ConvertError("conversion", "Only image src is supported")
                value = data_url(self.root, value)
            elif key == "style":
                value = css(value, self.root)
            elif key == "href":
                if tag != "a" or urlsplit(value).scheme.lower() not in {"", "http", "https", "mailto"}:
                    raise ConvertError("conversion", "Unsupported HTML link")
            rendered.append(f' {key}="{escape(value, quote=True)}"')
        self.parts.append("<" + tag + "".join(rendered) + ">")
        if tag == "style":
            self.in_style = True

    def handle_startendtag(self, tag, attrs):
        self.handle_starttag(tag, attrs)

    def handle_endtag(self, tag):
        self.parts.append("</" + tag + ">")
        if tag == "style":
            self.in_style = False

    def handle_data(self, data):
        self.parts.append(css(data, self.root) if self.in_style else escape(data))


def prepare_html(source):
    parser = PassiveHTML(source.parent.resolve())
    parser.feed(source.read_text(encoding="utf-8-sig"))
    parser.close()
    return "".join(parser.parts)


class PlainText(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.parts, self.skip = [], 0

    def handle_starttag(self, tag, attrs):
        if tag in {"script", "style", "head"}:
            self.skip += 1
        if not self.skip:
            if tag in {"p", "div", "br", "tr", "li", "pre", "h1", "h2", "h3", "h4", "h5", "h6"}:
                self.parts.append("\n")
            elif tag in {"td", "th"}:
                self.parts.append("\t")
            elif tag == "img":
                self.parts.append(dict(attrs).get("alt", ""))

    def handle_endtag(self, tag):
        if tag in {"script", "style", "head"}:
            self.skip = max(0, self.skip - 1)
        elif not self.skip and tag in {"p", "div", "tr", "li", "pre"}:
            self.parts.append("\n")

    def handle_data(self, data):
        if not self.skip:
            self.parts.append(data)

    def text(self):
        return re.sub(r"\n[ \t]*\n(?:[ \t]*\n)+", "\n\n", "".join(self.parts)).strip() + "\n"
