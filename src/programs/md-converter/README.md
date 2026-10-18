# md-converter

Private rendering helper for `aiw file-convert`. Go 1.25.1+, Goldmark 1.7.13
(source included), Mermaid 11.12.1 (embedded). No Node or Playwright.

Build explicitly, when authorized:

```text
go build -o bin/md-converter .
# Windows: go build -o bin/md-converter.exe .
```

Set `AIW_MD_CONVERTER` to the resulting executable, or keep it in `bin/`.
The plugin never compiles it at runtime. A source checkout is not a binary release.

Internal interface:

```text
md-converter --input local.md --output temporary.html --format html
md-converter --input local.md --output temporary.pdf --format pdf
md-converter --input local.md --output temporary.txt --format txt
```

The Python plugin owns final output naming, collision protection and JSON.
The helper writes a caller-owned temporary file and can replace that file.
HTML input is reserved for passive, self-contained HTML prepared by the plugin.
PDF uses installed Edge/Chrome/Chromium with a fresh temporary profile and local
CDP. `AIW_CONVERT_BROWSER` overrides discovery. There is a 90-second overall
deadline and 45-second rendering deadline. Fonts must already exist locally.

Mermaid, images and fonts must finish before print. HTTP/HTTPS/file/FTP/WebSocket
document requests are blocked. This is a document resource boundary, not an OS
sandbox for a compromised browser. No `--no-sandbox` flag is used.

Porting reference: `D:/03_projects/mermaid-tools/go` (Markdown renderer and image
embedding, default CSS). PDF transport, readiness and plugin contract are new.
SVG images are rejected; use PNG/JPEG/GIF/WebP. Images cannot escape the input
directory, including through symbolic links. Markdown raw HTML is disabled.
Third-party notices are in `THIRD_PARTY_LICENSES.md` and
`third_party/goldmark/LICENSE`. No antiword, Pandoc or browser binaries are bundled.

Compile-only (no retained artifact):

```text
python scripts/compile.py
```

This compiles Python plugin syntax without importing it and compiles this Go
module offline. It does not run conversions, tests, a browser or network requests.
