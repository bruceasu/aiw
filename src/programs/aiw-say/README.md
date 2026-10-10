# AIW Say

AIW Say translates text supplied as one argument, UTF-8 on stdin, from the system clipboard, or through an optional Zenity dialog. Plain CLI use writes only the completed translation to stdout and leaves the clipboard unchanged unless `--copy` is requested.

## Build and install

On Windows, build both Windows and Linux amd64 plugin binaries and samples
from the repository root with:

```powershell
python build.py say
```

This produces `aiw-say.exe`, `aiw-say`, `aiw.toml.example`, and `profiles/`
under `dist/plugins/aiw-say/`. `python build.py say` only builds;
`python build.py plugins` builds and installs all plugins, including Say, under
`C:\green\aiw\plugins\`. `python build.py all` includes that installation step.
Say installation preserves an existing `aiw.toml` and does not install user profiles.

For a manual Linux/WSL build:

```sh
cd src
go build -o aiw-say ./cmd/aiw-say
```
Put the executable in a directory searched by AIW's plugin discovery, such as
a directory on `PATH`, and invoke it through `aiw say`.

Copy `aiw.toml.example` to `aiw.toml` beside the installed Say executable
(normally `C:\green\aiw\plugins\aiw-say\aiw.toml` on Windows), only if the
destination does not already exist. A configuration beside the main `aiw`
executable is not automatically read by Say. Set `model` to a model verified
for your API account. The program never guesses a model name. You may instead
provide `--config PATH`.

Set `OPENAI_API_KEY` in the environment before using the plugin. Credentials are not accepted in TOML. Requests use the OpenAI Chat Completions API; the source text is sent to the configured API and is not written to application logs. `OPENAI_BASE_URL` can select an HTTPS-compatible endpoint; plain HTTP is accepted only for loopback addresses.

Agent Gateway's Codex backend supports Say's non-streaming text requests through
`/v1/chat/completions`. Set `OPENAI_BASE_URL` to the Gateway URL including `/v1`,
`OPENAI_API_KEY` to a Gateway key, and `model` or `--model` to an allowed logical
model in the Gateway configuration. For the default local listener, the base URL
is `http://127.0.0.1:43127/v1`. Remote endpoints must use HTTPS.

The Gateway's `openai_proxy` backend also accepts this request through the same
`/v1` URL and Gateway key. It forwards the model and request to the configured
upstream; model and API access are controlled by the upstream credential rather
than a local Codex model mapping.

## Profiles

Copy any desired sample from `profiles/` to the user's profile directory. Copy only when the destination does not already exist; AIW Say does not create or overwrite user profile files.

- Windows: `%APPDATA%/aiw/profiles/<name>.toml`
- Linux/WSL: `$XDG_CONFIG_HOME/aiw/profiles/<name>.toml`, or `~/.config/aiw/profiles/<name>.toml` when XDG is unset

For example, copy `profiles/ja-business.toml` to the matching user profile directory, then run:

```text
aiw say --profile ja-business "明日の会議を確認してください"
```

Settings merge in this order: built-in defaults, installation `aiw.toml`, selected profile, then explicitly supplied CLI values. Supported profile fields are `source`, `target`, `mode`, `style`, `polite`, `profanity`, `simplify`, `provider`, `model`, and `timeout`.

## Options

```text
aiw say [options] [text]
```

- `-t, --target`: `zh`, `ja`, or `en` (default `ja`)
- `-s, --source`: `auto`, `zh`, `ja`, or `en` (default `auto`)
- `--mode`: `realtime` or `written`
- `--style`: `spoken`, `teams`, `letter`, `document`, or `article`
- `-p, --polite`: `casual`, `polite`, or `formal`
- `--simple`: simplify the translation
- `--profanity`: `mask`, `soften`, or `preserve`
- `--profile`, `--provider`, `--model`, `--timeout`, `--config`
- `--help`, `--version`

Choose exactly one input source: one positional text value, stdin, `--clipboard`, or `--dialog zenity`. Empty input, source conflicts, malformed TOML, missing profiles, missing API credentials, invalid CLI options, and failed or incomplete API responses return a non-zero exit status and write diagnostics to stderr. Invalid values for recognized configuration settings fall back to the lower-precedence value, then to the built-in default; unknown keys are ignored. The CLI buffers the complete response before writing it.

## Clipboard and dialogs

```text
aiw say --clipboard -t ja --copy
aiw say --dialog zenity -t en
```

`--clipboard` reads plain text. `--copy` writes only a successful translation; ordinary CLI use never changes the clipboard. A translation or provider failure leaves the original clipboard untouched. Clipboard access errors return a non-zero status.

- Windows uses the native Unicode clipboard API. No clipboard tools are needed for ordinary CLI use.
- Wayland Linux uses `wl-paste` and `wl-copy`. X11 uses the read/write pair from `xclip` or `xsel`. The matching tools are required only when clipboard flags are used.
- WSL uses Windows PowerShell (`powershell.exe` or `pwsh.exe`) through enabled Windows interop for clipboard reads and writes. The adapter passes text as UTF-8 through process streams; `clip.exe` is not used for reads.
- `--dialog zenity` requires Zenity and a graphical session. Windows needs a user-installed native `zenity.exe` on `PATH`. Linux and WSL need Linux Zenity; WSL also needs WSLg or a configured X display. Input cancellation ends before a provider request. A successful dialog translation is copied, then shown in a result window.

The dialog result window can be closed after the translation has been copied; that close is reported as a non-zero dialog-cancel result. A missing dialog or display does not affect headless CLI use.

## AutoHotkey v2 shortcuts

Place `aiw-say-hotkeys.ahk` and `aiw-say-hotkeys.ps1` beside `aiw-say.exe`, then run the AHK script with AutoHotkey v2:

| Shortcut | Action |
| --- | --- |
| Ctrl+Alt+J | Translate clipboard to Japanese |
| Ctrl+Alt+E | Translate clipboard to English |
| Ctrl+Alt+B | Use the `ja-business` profile |
| Ctrl+Alt+T | Open `aiw say --dialog zenity` |

J/E/B pass UTF-8 text through process stdin and wait for the CLI status. They copy a result only after success and only if the clipboard still contains the text captured before translation. If the user copies something else while a translation runs, the new clipboard contents are kept. Text is not included in process arguments.

## Troubleshooting

- `model is required`: set `[say.llm].model` or pass `--model`.
- `OPENAI_API_KEY is required`: set the key in the process environment.
- `profile does not exist`: check the platform directory and the profile name; names allow letters, numbers, `_`, and `-` only.
- `option ... is not implemented`: file, output-file, and pair modes are outside this release.
- `clipboard access requires ...`: use the matching Linux clipboard tools in an active display session, or enable Windows interop in WSL.
- `Zenity is unavailable`: install the platform executable and start a supported graphical session; ordinary CLI commands do not require Zenity.
- `OpenAI rate limit reached`: wait and retry after checking API account limits.

The deterministic mock-provider scenarios are in `tests/test_fd036_say_blackbox.py`. They require a compiled executable path in `AIW_SAY_BIN`; their execution is governed by the repository's independent Tester authorization process.
