package cz

func HelpText() string {
	return `aiw cz [options]

Options:
  --llm / --no-llm        Use LLM to generate candidates
  -N, --candidates N      Number of LLM candidates
  -r, --retry             Retry using the last commit message as draft
  --provider NAME         Override the LLM provider for this invocation
  --model MODEL           Override the LLM model for this invocation
  --lang LANGUAGE         Set a built-in (zh, en, ja) or configured language.
                          Priority: --lang > [i18n].default_language >
                          LC_ALL > LC_MESSAGES > LANGUAGE > LANG >
                          embedded en.
                          Terminal candidates normalize to a base language
                          by trimming, lowercasing, removing encoding, then
                          removing region/variant suffixes.
                          LANGUAGE is checked left-to-right by colon-separated
                          candidates. Unsupported detected locales fall back
                          to embedded en. Blank/unsupported --lang and invalid
                          configured defaults return errors.

Configuration:
  Set the global default language in aiw.toml:
    [i18n]
    default_language = "fr"
  Configure non-built-in languages under [cz.locales.<language>].
  A valid --lang always wins over lower-priority settings.

  Set globally via aiw.toml [ai] section:
    provider = "openai" | "gemini" | "ollama" | "codex-cli" | "copilot-cli" | "auto"
    api_key = "..."
    model = "..."
    base_url = "..."
    codex_command = "codex"      # optional, default: codex
    copilot_command = "gh"       # optional, default: gh

  Provider settings under [cz] remain supported as a legacy fallback.

Provider fallback:
  If provider is empty or auto, cz tries available providers in order:
    codex-cli -> copilot-cli -> openai -> gemini -> ollama

Interactive: supports issue-prefix selection and external editor for multiline fields.`
}
