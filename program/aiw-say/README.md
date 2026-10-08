# AIW Say Phase 1

AIW Say is a text-only translation plugin. The command accepts one text argument or UTF-8 on stdin and writes only the completed translation to stdout. GUI, clipboard, file, output-file, pair, and glossary modes are not included in Phase 1.

## Build and install

From the repository root, build the plugin with:

```powershell
go build -o aiw-say.exe ./cmd/aiw-say
```

On Linux or WSL, use `-o aiw-say`. Put the executable in a directory searched by AIW's plugin discovery, such as a directory on `PATH`, and invoke it through `aiw say`.

Copy `aiw.toml.example` to `aiw.toml` beside the installed plugin or AIW executable. Set `model` to a model verified for your API account. The program never guesses a model name. You may instead provide `--config PATH`.

Set `OPENAI_API_KEY` in the environment before using the plugin. Credentials are not accepted in TOML. Requests use the OpenAI Chat Completions API; the source text is sent to the configured API and is not written to application logs. `OPENAI_BASE_URL` can select an HTTPS-compatible endpoint; plain HTTP is accepted only for loopback addresses.

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

Input may be one positional text value or stdin, but not both. Empty input, malformed TOML, missing profiles, missing API credentials, invalid CLI options, and failed or incomplete API responses return a non-zero exit status, write diagnostics to stderr, and leave stdout empty. Invalid values for recognized configuration settings fall back to the lower-precedence value, then to the built-in default; unknown keys are ignored. The CLI buffers the complete response before writing it.

## Troubleshooting

- `model is required`: set `[say.llm].model` or pass `--model`.
- `OPENAI_API_KEY is required`: set the key in the process environment.
- `profile does not exist`: check the platform directory and the profile name; names allow letters, numbers, `_`, and `-` only.
- `option ... is not implemented`: clipboard, copy, file, output, dialog, and pair modes belong to later phases.
- `OpenAI rate limit reached`: wait and retry after checking API account limits.

The deterministic mock-provider scenarios are in `tests/test_fd036_say_blackbox.py`. They require a compiled executable path in `AIW_SAY_BIN`; their execution is governed by the repository's independent Tester authorization process.
