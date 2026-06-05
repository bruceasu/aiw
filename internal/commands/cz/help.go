package cz

func HelpText() string {
	return `aiw cz [options]

Options:
  --llm / --no-llm        Use LLM to generate candidates
  -N, --candidates N      Number of LLM candidates
  -r, --retry             Retry using the last commit message as draft

Interactive: supports issue-prefix selection and external editor for multiline fields.`
}
