# AI-assisted `aiw git` commands

AI commands use the provider settings and fallback order configured for `aiw cz`.

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
bounded branch context described above for `aib`. Provider setup follows the `[cz]`,
`[cz.codex]`, `[cz.copilot]`, and `[cz.openai]` configuration documented for
`aiw cz`.
