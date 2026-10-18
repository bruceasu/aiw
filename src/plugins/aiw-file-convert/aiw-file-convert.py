#!/usr/bin/env python3
"""AIW file-convert entrypoint. Python 3.11+; conversion imports are lazy."""
import argparse
import json
import os
from pathlib import Path
import sys
import tempfile

from common import ConvertError


class Parser(argparse.ArgumentParser):
    def error(self, message):
        raise ConvertError("arguments", message)


def main(argv):
    for stream in (sys.stdout, sys.stderr):
        if hasattr(stream, "reconfigure"):
            stream.reconfigure(encoding="utf-8")
    if "-h" in argv or "--help" in argv:
        print(Path(__file__).with_name("help.txt").read_text(encoding="utf-8"), end="")
        return 0
    json_mode = "--json" in argv
    arguments = {"input": None, "output": None, "format": "md", "json": json_mode}
    response = {"arguments": arguments, "result-data": None, "content-type": "", "result-code": "ok", "error-message": ""}
    try:
        parser = Parser(add_help=False, allow_abbrev=False)
        parser.add_argument("-i", "--input")
        parser.add_argument("-o", "--output", "--ouput", default="")
        parser.add_argument("--format", default="md")
        parser.add_argument("--json", action="store_true")
        parser.add_argument("--install-plan")
        parser.add_argument("--approve-install")
        args = parser.parse_args(argv)
        arguments.update(vars(args))
        if args.install_plan:
            if args.input or args.output or args.format != "md":
                raise ConvertError("arguments", "Install mode cannot be combined with conversion options")
            from install import install
            try:
                result = install(args.install_plan, args.approve_install)
            except ConvertError:
                raise
            except (ValueError, UnicodeError) as exc:
                raise ConvertError("arguments", f"Invalid install plan: {exc}") from exc
            except Exception as exc:
                raise ConvertError("install", f"Installation failed: {exc}") from exc
            response["result-data"], response["content-type"] = result, "application/json"
            if not json_mode:
                print(json.dumps(result, indent=2, ensure_ascii=False))
        else:
            if args.approve_install:
                raise ConvertError("arguments", "--approve-install requires --install-plan")
            from backends import BINARY, MIMES, convert, route
            if args.format not in MIMES:
                raise ConvertError("arguments", "Unknown target format: " + args.format)
            response["content-type"] = MIMES[args.format]
            if args.format in BINARY and args.output == "-":
                raise ConvertError("arguments", "Binary output must be written to a file; -o - is not allowed")
            if not args.input:
                raise ConvertError("arguments", "-i/--input is required; run --help for examples")
            source = Path(args.input).expanduser().resolve(strict=True)
            if not source.is_file():
                raise ConvertError("arguments", "Input must be a local regular file")
            route(source, args.format)
            destination = None
            if args.output != "-":
                raw_output = args.output or (args.input + "." + args.format)
                destination = Path(raw_output).expanduser().absolute()
                if destination.suffix.lower() not in {"." + args.format, ".markdown" if args.format == "md" else "." + args.format}:
                    raise ConvertError("arguments", "Output extension conflicts with --format")
                if destination.resolve() == source or destination.exists() or destination.is_symlink():
                    raise ConvertError("output", "Output already exists or refers to the input; it will not be overwritten")
                if not destination.parent.is_dir():
                    raise ConvertError("output", "Output directory does not exist")
            with tempfile.TemporaryDirectory(prefix="aiw-convert-", dir=destination.parent if destination else None) as directory:
                temporary = Path(directory)
                output = temporary / ("result." + args.format)
                convert(source, args.format, output, temporary)
                if not output.is_file():
                    raise ConvertError("conversion", "Backend did not create an output file")
                with output.open("rb") as stream:
                    magic = stream.read(5)
                if args.format == "pdf" and magic != b"%PDF-":
                    raise ConvertError("conversion", "Backend did not create a valid PDF header")
                if args.format in {"docx", "odt", "epub"} and not magic.startswith(b"PK"):
                    raise ConvertError("conversion", "Backend did not create a ZIP document")
                if destination:
                    try:
                        os.link(output, destination)
                    except OSError as exc:
                        raise ConvertError("output", f"Cannot publish output without overwriting: {exc}") from exc
                    response["output-file"] = {"path": str(destination), "size": destination.stat().st_size}
                    arguments["output"] = str(destination)
                    if not json_mode:
                        print(str(destination))
                else:
                    content = output.read_text(encoding="utf-8")
                    if json_mode:
                        response["result-data"] = content
                    else:
                        print(content, end="")
    except ConvertError as exc:
        response["result-code"], response["error-message"] = exc.code, str(exc)
        if exc.dependency:
            response["dependency"] = exc.dependency
    except (OSError, UnicodeError, ValueError) as exc:
        response["result-code"], response["error-message"] = "conversion", str(exc)
    except Exception as exc:
        response["result-code"], response["error-message"] = "conversion", f"{type(exc).__name__}: {exc}"
    if json_mode:
        print(json.dumps(response, ensure_ascii=False))
    elif response["result-code"] != "ok":
        print(response["error-message"], file=sys.stderr)
    return {"ok": 0, "arguments": 2, "missing-dependency": 3, "conversion": 4, "output": 5, "install": 6}.get(response["result-code"], 4)


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
