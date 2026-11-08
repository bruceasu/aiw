# AI-assisted `aiw git` commands

AI commands use the shared `[ai]` provider by default. Explicit `[cz]` settings
remain command-specific overrides for `aiw cz`, `aic`, `air`, and `aib`.

```text
aiw git aic                 # stage all changes, generate a Conventional Commit message, and commit
aiw git air                 # review the staged diff
aiw git aib                 # summarize main..HEAD history and bounded changes
aiw git aib --base develop  # summarize develop..HEAD history and bounded changes
```

`aic` runs `git add -A` before generating the message. If generation or commit
fails, the command returns an error; generated text is passed to Git through
stdin. `air` and `aib` only print the AI response and do not modify the
repository. `air` sends only the staged diff. `aib` sends one-line commit
history from the requested base to `HEAD`, plus changed file statuses, a short
change summary, and a branch diff from the merge base to `HEAD`. The file list
is limited to 8,000 characters and the diff to 12,000 characters. Truncated
content is marked, and the model is told not to infer omitted changes. The
default base is `main`.

The configured provider receives the staged diff for `aic`/`air` or the
bounded branch context described above for `aib`. These commands use shared
`[ai]` settings by default. Explicit `[cz]` settings and CZ environment values
override them. DeepSeek and OpenAI credentials use `DEEPSEEK_API_KEY` and
`OPENAI_API_KEY`.

`aiw git guide` uses Codex CLI by default when no shared provider is configured.
It supports Codex CLI, OpenAI Responses, and DeepSeek Responses. Install its
optional SDK dependency with `python -m pip install -r <AIW_ROOT>/plugins/aiw-git/requirements.txt`
to use OpenAI or DeepSeek. The selected provider must be supported by this
command; it does not silently switch providers.
