package llm

import (
	"strings"
	"testing"
)

func TestExtractOpenAIContent(t *testing.T) {
	raw := []byte(`{"choices":[{"message":{"content":"{\"candidates\":[{\"type\":\"feat\",\"scope\":\"git\",\"subject\":\"add wizard\",\"body\":\"\",\"breaking\":\"\",\"footer\":\"\"}] }"}}]}`)
	content, err := extractOpenAIContent(raw)
	if err != nil {
		t.Fatalf("extract content: %v", err)
	}
	if !strings.Contains(content, `"candidates"`) {
		t.Fatalf("unexpected content: %s", content)
	}
}

func TestExtractOpenAIContentEmptyChoices(t *testing.T) {
	raw := []byte(`{"choices":[]}`)
	_, err := extractOpenAIContent(raw)
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestResolveOpenAIValuePrecedence(t *testing.T) {
	t.Setenv("OPENAI_MODEL", "env-model")
	cwdEnv := map[string]string{"OPENAI_MODEL": "cwd-model"}
	exeEnv := map[string]string{"OPENAI_MODEL": "exe-model"}

	v, src := resolveOpenAIValue("cfg-model", "OPENAI_MODEL", cwdEnv, exeEnv, "default-model")
	if v != "cfg-model" || src != "config" {
		t.Fatalf("expected config to win, got value=%q source=%q", v, src)
	}

	v, src = resolveOpenAIValue("", "OPENAI_MODEL", cwdEnv, exeEnv, "default-model")
	if v != "env-model" || src != "env" {
		t.Fatalf("expected env to win, got value=%q source=%q", v, src)
	}

	t.Setenv("OPENAI_MODEL", "")
	v, src = resolveOpenAIValue("", "OPENAI_MODEL", cwdEnv, exeEnv, "default-model")
	if v != "cwd-model" || src != "cwd .env" {
		t.Fatalf("expected cwd .env to win, got value=%q source=%q", v, src)
	}

	v, src = resolveOpenAIValue("", "OPENAI_BASE_URL", map[string]string{}, map[string]string{}, "https://api.openai.com/v1")
	if src != "default" {
		t.Fatalf("expected default source, got %q", src)
	}
}

func TestMaskSecret(t *testing.T) {
	if got := maskSecret(""); got != "<empty>" {
		t.Fatalf("expected <empty>, got %q", got)
	}
	if got := maskSecret("abcd"); got != "****" {
		t.Fatalf("expected masked short secret, got %q", got)
	}
	if got := maskSecret("abcdefgh"); got != "****efgh" {
		t.Fatalf("unexpected masked value: %q", got)
	}
}

func TestParseLLMCandidatesFallback(t *testing.T) {
	out := "feat: add cz command\nfix: handle empty staged"
	cands, err := parseLLMCandidates(out)
	if err != nil {
		t.Fatalf("parse fallback candidates: %v", err)
	}
	if len(cands) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(cands))
	}
	if cands[0].Type != "feat" {
		t.Fatalf("unexpected first type: %+v", cands[0])
	}
}

func TestParseLLMCandidatesRejectsNoisyLogs(t *testing.T) {
	out := strings.Join([]string{
		"1) workdir: C:/repo",
		"2) model: gpt-5.5-mini",
		"`feat(cli): add batch import command`",
		"type(scope): summary",
		"Get-Content: path not found",
	}, "\n")

	_, err := parseLLMCandidates(out)
	if err == nil {
		t.Fatal("expected parse to fail for noisy logs")
	}
}

func TestParseLLMCandidatesParsesConventionalListWithNumbering(t *testing.T) {
	out := strings.Join([]string{
		"1) feat(cli): add commit wizard",
		"2) fix(parser): reject noisy llm output",
	}, "\n")

	cands, err := parseLLMCandidates(out)
	if err != nil {
		t.Fatalf("parse fallback candidates: %v", err)
	}
	if len(cands) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(cands))
	}
	if cands[0].Type != "feat" || cands[0].Scope != "cli" {
		t.Fatalf("unexpected first candidate: %+v", cands[0])
	}
}
