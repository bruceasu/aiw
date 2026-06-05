package cz

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

type fakeUI struct {
	draft czDraft
	err   error
	msg   string
}

func (f *fakeUI) DraftFromWizard(cfg czConfig) (czDraft, error) {
	return f.draft, f.err
}

func (f *fakeUI) DraftFromLLM(cfg czConfig) (czDraft, error) {
	return f.draft, f.err
}

func (f *fakeUI) ReviewAndCommit(draft czDraft, cfg czConfig, commitFn func(string) error) error {
	f.msg = buildCommitMessage(draft)
	return commitFn(f.msg)
}

func TestDispatchWithUICommitsBuiltMessage(t *testing.T) {
	ui := &fakeUI{
		draft: czDraft{Type: "feat", Scope: "cli", Subject: "add dispatch flow"},
	}
	var committed string
	err := dispatchWithUI(nil, ui,
		func() (string, error) { return "cmd/cz/command.go", nil },
		func(msg string) error {
			committed = msg
			return nil
		},
	)
	if err != nil {
		t.Fatalf("dispatch with ui: %v", err)
	}
	if !strings.Contains(committed, "feat(cli): add dispatch flow") {
		t.Fatalf("unexpected committed message: %q", committed)
	}
}

func TestDispatchWithUINoStagedChangesFails(t *testing.T) {
	ui := &fakeUI{}
	err := dispatchWithUI(nil, ui,
		func() (string, error) { return "", nil },
		func(msg string) error { return errors.New("should not commit") },
	)
	if err == nil {
		t.Fatal("expected no staged changes to fail")
	}
}

func TestParseCzOptions(t *testing.T) {
	opts, err := parseCzOptions([]string{"--llm", "-N", "5"})
	if err != nil {
		t.Fatalf("parse options: %v", err)
	}
	if opts.UseLLM == nil || !*opts.UseLLM {
		t.Fatalf("expected UseLLM=true, got %+v", opts)
	}
	if opts.Candidates == nil || *opts.Candidates != 5 {
		t.Fatalf("expected candidates=5, got %+v", opts)
	}

}

func TestParseCzOptionsRejectsEmojiFlags(t *testing.T) {
	if _, err := parseCzOptions([]string{"--emoji"}); err == nil {
		t.Fatal("expected --emoji to be rejected")
	}
	if _, err := parseCzOptions([]string{"--no-emoji"}); err == nil {
		t.Fatal("expected --no-emoji to be rejected")
	}
}

func TestBuildCommitMessage(t *testing.T) {
	d := czDraft{
		Type:     "feat",
		Scope:    "git",
		Subject:  "add cz wizard",
		Body:     "line1\nline2",
		Breaking: "old behavior removed",
		Footer:   "Refs #12",
	}
	msg := buildCommitMessage(d)
	if msg == "" {
		t.Fatal("expected non-empty commit message")
	}
	if msg[:15] != "feat(git): add " {
		t.Fatalf("unexpected header: %q", msg)
	}
}

func TestBuildCommitMessageContainsNoNUL(t *testing.T) {
	d := czDraft{
		Type:    "feat",
		Scope:   "git",
		Subject: "add cz wizard",
	}

	msg := buildCommitMessage(d)
	if strings.ContainsRune(msg, rune(0)) {
		t.Fatalf("expected no NUL bytes in commit message, got %q", msg)
	}
	if !utf8.ValidString(msg) {
		t.Fatalf("expected valid utf-8 commit message, got %q", msg)
	}
}

func TestBuildCommitMessageStripsNULFromFields(t *testing.T) {
	d := czDraft{
		Type:     "feat",
		Scope:    "g\x00it",
		Subject:  "add\x00 wizard",
		Body:     "line\x001",
		Breaking: "break\x00ing",
		Footer:   "Refs \x00#12",
	}

	msg := buildCommitMessage(d)
	if strings.ContainsRune(msg, rune(0)) {
		t.Fatalf("expected NUL bytes to be stripped, got %q", msg)
	}
	if !strings.Contains(msg, "feat(git): add wizard") {
		t.Fatalf("expected cleaned header, got %q", msg)
	}
	if !strings.Contains(msg, "line1") {
		t.Fatalf("expected cleaned body, got %q", msg)
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

func TestMergeCzConfigFromTomlFileEditor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aiw.toml")
	content := "[cz]\nEDITOR = \"code --wait\"\nmodel = \"gpt-4.1-mini\"\nbase_url = \"https://api.openai.com/v1\"\napi_key = \"test-key\"\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := defaultCzConfig()
	if err := mergeCzConfigFromTomlFile(&cfg, path); err != nil {
		t.Fatalf("merge config: %v", err)
	}
	if cfg.Editor != "code --wait" {
		t.Fatalf("expected editor to be parsed, got %q", cfg.Editor)
	}
	if cfg.LLMModel != "gpt-4.1-mini" {
		t.Fatalf("expected model to be parsed, got %q", cfg.LLMModel)
	}
	if cfg.APIBaseURL != "https://api.openai.com/v1" {
		t.Fatalf("expected base_url to be parsed, got %q", cfg.APIBaseURL)
	}
	if cfg.APIKey != "test-key" {
		t.Fatalf("expected api_key to be parsed, got %q", cfg.APIKey)
	}
}

func TestMergeCzConfigFromTomlFileScopes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aiw.toml")
	content := strings.Join([]string{
		"[[cz.scopes]]",
		`value = "cli"`,
		`name = "CLI"`,
		"[[cz.scopes]]",
		`value = "docs"`,
		`name = "Documentation"`,
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := defaultCzConfig()
	if err := mergeCzConfigFromTomlFile(&cfg, path); err != nil {
		t.Fatalf("merge config: %v", err)
	}
	if len(cfg.Scopes) != 2 {
		t.Fatalf("expected 2 scopes, got %+v", cfg.Scopes)
	}
	if cfg.Scopes[0].Value != "cli" || cfg.Scopes[1].Value != "docs" {
		t.Fatalf("unexpected scopes: %+v", cfg.Scopes)
	}
}

func TestMergeCzConfigFromTomlFileMultipleScopeFlags(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aiw.toml")
	content := strings.Join([]string{
		"[cz]",
		"enable_multiple_scopes = true",
		`scope_enum_separator = "/"`,
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := defaultCzConfig()
	if err := mergeCzConfigFromTomlFile(&cfg, path); err != nil {
		t.Fatalf("merge config: %v", err)
	}
	if !cfg.EnableMultipleScopes {
		t.Fatal("expected multiple scopes to be enabled")
	}
	if cfg.ScopeEnumSeparator != "/" {
		t.Fatalf("expected custom separator, got %q", cfg.ScopeEnumSeparator)
	}
}

func TestIsEditorShortcut(t *testing.T) {
	cases := []string{"/edit", "^e", "ctrl+e", "control+e", string(rune(5))}
	for _, c := range cases {
		if !isEditorShortcut(c) {
			t.Fatalf("expected %q to be editor shortcut", c)
		}
	}
	if isEditorShortcut("e") || isEditorShortcut("edit") {
		t.Fatal("single-letter or bare edit should not trigger editor")
	}
	if isEditorShortcut("hello") {
		t.Fatal("unexpected editor shortcut match for normal text")
	}
}

func TestResolveEditorPrefersConfig(t *testing.T) {
	t.Setenv("GIT_EDITOR", "env-editor")
	cfg := defaultCzConfig()
	cfg.Editor = "code --wait"
	got := resolveEditor(cfg)
	if got != "code --wait" {
		t.Fatalf("expected config editor to win, got %q", got)
	}
}

func TestBuildEditorArgs(t *testing.T) {
	file := `C:\tmp\msg file.txt`
	got, err := buildEditorArgs("code --wait", file)
	if err != nil {
		t.Fatalf("build args: %v", err)
	}
	want := []string{"code", "--wait", `C:\tmp\msg file.txt`}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("unexpected args, got %+v want %+v", got, want)
	}

	got, err = buildEditorArgs("nvim {file}", file)
	if err != nil {
		t.Fatalf("build args with placeholder: %v", err)
	}
	want = []string{"nvim", `C:\tmp\msg file.txt`}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("unexpected placeholder args, got %+v want %+v", got, want)
	}

	got, err = buildEditorArgs(`"C:\Program Files\Vim\gvim.exe" --nofork`, file)
	if err != nil {
		t.Fatalf("build args with quoted binary: %v", err)
	}
	want = []string{`C:\Program Files\Vim\gvim.exe`, "--nofork", `C:\tmp\msg file.txt`}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("unexpected quoted binary args, got %+v want %+v", got, want)
	}
}

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

func TestDetectIssueRefs(t *testing.T) {
	refs := detectIssueRefs("fix #12 and docs #34")
	if _, ok := refs["#12"]; !ok {
		t.Fatal("expected #12 to be detected")
	}
	if _, ok := refs["#34"]; !ok {
		t.Fatal("expected #34 to be detected")
	}
}

func TestFilterLLMIssueFooter(t *testing.T) {
	allowed := map[string]struct{}{"#12": {}}

	if got := filterLLMIssueFooter("Related to #12", allowed); got == "" {
		t.Fatal("expected footer with allowed ref to be kept")
	}
	if got := filterLLMIssueFooter("Related to #42", allowed); got != "" {
		t.Fatalf("expected footer with unknown ref to be dropped, got %q", got)
	}
	if got := filterLLMIssueFooter("Reviewed-by: someone", allowed); got == "" {
		t.Fatal("expected non-issue footer to be kept")
	}
}
