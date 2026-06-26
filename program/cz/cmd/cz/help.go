package cz

func HelpText() string {
	return `aiw cz [options]

Options:
  --llm / --no-llm        Use LLM to generate candidates
  -N, --candidates N      Number of LLM candidates
  -r, --retry             Retry using the last commit message as draft

Configuration:
  Set via aiw.toml [cz] section:
    provider = "openai" | "gemini" | "ollama" | "codex-cli" | "copilot-cli" | "auto"
    api_key = "..."
    model = "..."
    base_url = "..."
    codex_command = "codex"      # optional, default: codex
    copilot_command = "gh"       # optional, default: gh

Provider fallback:
  If provider is empty or auto, cz tries available providers in order:
    codex-cli -> copilot-cli -> openai -> gemini -> ollama

Interactive: supports issue-prefix selection and external editor for multiline fields.`
}
