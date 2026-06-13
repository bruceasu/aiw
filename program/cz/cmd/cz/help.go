package cz

func HelpText() string {
	return `aiw cz [options]

Options:
  --llm / --no-llm        Use LLM (OpenAI/Gemini) to generate candidates
  -N, --candidates N      Number of LLM candidates
  -r, --retry             Retry using the last commit message as draft

Configuration:
  Set via aiw.toml [cz] section:
    provider = "openai" or "gemini"
    api_key = "..."
    model = "..."
    base_url = "..."

Interactive: supports issue-prefix selection and external editor for multiline fields.`
}
