"""Shared text codec policy for aiw file and aiw patch."""


class CodecError(Exception):
    pass


UTF8_BOM = b"\xef\xbb\xbf"
UTF16_LE_BOM = b"\xff\xfe"
UTF16_BE_BOM = b"\xfe\xff"


def detect(data, requested=None):
    if requested:
        return requested, data.decode(requested), "explicit"
    if data.startswith(UTF8_BOM):
        return "utf-8", data[len(UTF8_BOM) :].decode("utf-8"), "high"
    if data.startswith((UTF16_LE_BOM, UTF16_BE_BOM)):
        return "utf-16", data.decode("utf-16"), "high"
    try:
        return "utf-8", data.decode("utf-8"), "high"
    except UnicodeDecodeError:
        pass
    candidates = []
    for name in ("gb18030", "shift_jis"):
        try:
            candidates.append((name, data.decode(name)))
        except UnicodeDecodeError:
            pass
    if len(candidates) == 1:
        return candidates[0][0], candidates[0][1], "medium"
    if candidates:
        raise CodecError("encoding is ambiguous; use --encoding gb18030 or --encoding windows-31j")
    raise CodecError("unable to decode text; use --encoding explicitly")


def newline_style(text):
    if "\r\n" in text:
        return "crlf"
    if "\r" in text:
        return "cr"
    return "lf"


def encode_text(text, encoding, bom):
    raw = text.encode("shift_jis" if encoding == "windows-31j" else encoding)
    if bom and encoding == "utf-8":
        return UTF8_BOM + raw
    if bom and encoding == "utf-16":
        return UTF16_LE_BOM + raw
    return raw


def normalize_newlines(text, newline):
    if newline == "crlf":
        return text.replace("\r\n", "\n").replace("\r", "\n").replace("\n", "\r\n")
    if newline == "cr":
        return text.replace("\r\n", "\n").replace("\n", "\r")
    return text.replace("\r\n", "\n").replace("\r", "\n")
