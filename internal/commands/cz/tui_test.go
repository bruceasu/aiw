package cz

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFilterTypeOptionsMatchesValueAndName(t *testing.T) {
	cfg := defaultCzConfig()

	got := filterTypeOptions(cfg.Types, "fix")
	if len(got) == 0 || got[0].Value != "fix" {
		t.Fatalf("expected fix to match by value, got %+v", got)
	}

	got = filterTypeOptions(cfg.Types, "文档")
	if len(got) == 0 || got[0].Value != "docs" {
		t.Fatalf("expected docs to match by localized name, got %+v", got)
	}
}

func TestFilterTypeOptionsEmptyQueryReturnsAll(t *testing.T) {
	cfg := defaultCzConfig()

	got := filterTypeOptions(cfg.Types, "")
	if len(got) != len(cfg.Types) {
		t.Fatalf("expected all types for empty query, got %d want %d", len(got), len(cfg.Types))
	}
}

func TestNewDefaultUIFallsBackToLineUIWhenNotInteractive(t *testing.T) {
	ui := newDefaultUI(false)
	if _, ok := ui.(lineUI); !ok {
		t.Fatalf("expected lineUI fallback, got %T", ui)
	}
}

func TestFilterScopeOptionsMatchesValueAndName(t *testing.T) {
	scopes := []czScope{
		{Value: "cli", Name: "CLI commands"},
		{Value: "docs", Name: "文档"},
	}

	got := filterScopeOptions(scopes, "cli")
	if len(got) != 1 || got[0].Value != "cli" {
		t.Fatalf("expected cli match, got %+v", got)
	}

	got = filterScopeOptions(scopes, "文档")
	if len(got) != 1 || got[0].Value != "docs" {
		t.Fatalf("expected docs localized match, got %+v", got)
	}
}

func TestScopeOptionsIncludeEmptyAndCustomEntries(t *testing.T) {
	cfg := defaultCzConfig()
	cfg.Scopes = []czScope{{Value: "cli", Name: "CLI"}}

	got := buildScopeOptions(cfg)
	if len(got) < 3 {
		t.Fatalf("expected empty/custom plus configured scopes, got %+v", got)
	}
	if got[0].Value != "" {
		t.Fatalf("expected first scope to be empty option, got %+v", got[0])
	}
	if got[1].Value != "." {
		t.Fatalf("expected second scope to be custom option, got %+v", got[1])
	}
}

func TestTeaUIDraftFromWizardUsesSearchSelectionForType(t *testing.T) {
	lineInputs := []string{
		"",
		"add searchable type selector",
		"",
	}
	multiInputs := []string{"", "", ""}

	ui := teaUI{
		selectTypeFn: func(cfg czConfig) (string, error) {
			return "feat", nil
		},
		promptLineFn: func(string) string {
			v := lineInputs[0]
			lineInputs = lineInputs[1:]
			return v
		},
		promptMultilineFn: func(string, czConfig, string) (string, error) {
			v := multiInputs[0]
			multiInputs = multiInputs[1:]
			return v, nil
		},
	}

	draft, err := ui.DraftFromWizard(defaultCzConfig())
	if err != nil {
		t.Fatalf("draft from wizard: %v", err)
	}
	if draft.Type != "feat" {
		t.Fatalf("expected selected type feat, got %q", draft.Type)
	}
	if draft.Subject != "add searchable type selector" {
		t.Fatalf("unexpected subject: %q", draft.Subject)
	}
}

func TestTeaUIDraftFromWizardUsesSearchSelectionForScope(t *testing.T) {
	lineInputs := []string{
		"add searchable scope selector",
		"",
	}
	multiInputs := []string{"", "", ""}

	ui := teaUI{
		selectTypeFn: func(cfg czConfig) (string, error) {
			return "feat", nil
		},
		selectScopeFn: func(cfg czConfig) (string, error) {
			return "cli", nil
		},
		promptLineFn: func(string) string {
			v := lineInputs[0]
			lineInputs = lineInputs[1:]
			return v
		},
		promptMultilineFn: func(string, czConfig, string) (string, error) {
			v := multiInputs[0]
			multiInputs = multiInputs[1:]
			return v, nil
		},
	}

	cfg := defaultCzConfig()
	cfg.Scopes = []czScope{{Value: "cli", Name: "CLI"}}

	draft, err := ui.DraftFromWizard(cfg)
	if err != nil {
		t.Fatalf("draft from wizard: %v", err)
	}
	if draft.Scope != "cli" {
		t.Fatalf("expected selected scope cli, got %q", draft.Scope)
	}
}

func TestTeaUIDraftFromWizardUsesCheckboxSelectionForMultipleScopes(t *testing.T) {
	lineInputs := []string{
		"add multiple scope selector",
		"",
	}
	multiInputs := []string{"", "", ""}

	ui := teaUI{
		selectTypeFn: func(cfg czConfig) (string, error) {
			return "feat", nil
		},
		selectScopesFn: func(cfg czConfig) ([]string, error) {
			return []string{"cli", "docs"}, nil
		},
		promptLineFn: func(string) string {
			v := lineInputs[0]
			lineInputs = lineInputs[1:]
			return v
		},
		promptMultilineFn: func(string, czConfig, string) (string, error) {
			v := multiInputs[0]
			multiInputs = multiInputs[1:]
			return v, nil
		},
	}

	cfg := defaultCzConfig()
	cfg.Scopes = []czScope{{Value: "cli", Name: "CLI"}, {Value: "docs", Name: "Docs"}}
	cfg.EnableMultipleScopes = true
	cfg.ScopeEnumSeparator = ","

	draft, err := ui.DraftFromWizard(cfg)
	if err != nil {
		t.Fatalf("draft from wizard: %v", err)
	}
	if draft.Scope != "cli,docs" {
		t.Fatalf("expected joined scopes cli,docs, got %q", draft.Scope)
	}
}

func TestSearchCheckboxModelTogglesAndConfirmsSelection(t *testing.T) {
	model := newSearchCheckboxModel("Scopes", []searchOption{
		{Value: "cli", Label: "CLI"},
		{Value: "docs", Label: "Docs"},
	}, "No matching scopes.")

	updated, _ := model.Update(teaKeySpace())
	model = updated.(searchCheckboxModel)
	updated, _ = model.Update(teaKeyDown())
	model = updated.(searchCheckboxModel)
	updated, _ = model.Update(teaKeySpace())
	model = updated.(searchCheckboxModel)
	updated, _ = model.Update(teaKeyEnter())
	model = updated.(searchCheckboxModel)

	if !model.confirmed {
		t.Fatal("expected checkbox model to confirm selection")
	}
	if len(model.selectedValues()) != 2 {
		t.Fatalf("expected 2 selected values, got %+v", model.selectedValues())
	}
}

func TestBuildCandidateOptionsUsesCommitHeaderLabels(t *testing.T) {
	options := buildCandidateOptions([]czDraft{
		{Type: "feat", Scope: "cli", Subject: "add tui selector"},
		{Type: "fix", Scope: "git", Subject: "handle empty staged"},
	})

	if len(options) != 3 {
		t.Fatalf("expected 2 candidates plus regenerate option, got %+v", options)
	}
	if options[0].Label != "feat(cli): add tui selector" {
		t.Fatalf("unexpected first label: %q", options[0].Label)
	}
	if options[2].Value != "__regen__" {
		t.Fatalf("expected regenerate option, got %+v", options[2])
	}
}

func TestTeaUIDraftFromLLMUsesInjectedCandidateSelector(t *testing.T) {
	ui := teaUI{
		selectCandidateFn: func(cfg czConfig, cands []czDraft) (czDraft, error) {
			if len(cands) != 2 {
				t.Fatalf("expected 2 candidates, got %d", len(cands))
			}
			return cands[1], nil
		},
		draftFromLLMFn: func(cfg czConfig, selectFn func(czConfig, []czDraft) (czDraft, error)) (czDraft, error) {
			return selectFn(cfg, []czDraft{
				{Type: "feat", Subject: "first"},
				{Type: "fix", Subject: "second"},
			})
		},
	}

	draft, err := ui.DraftFromLLM(defaultCzConfig())
	if err != nil {
		t.Fatalf("draft from llm: %v", err)
	}
	if draft.Type != "fix" {
		t.Fatalf("expected selected candidate to be returned, got %+v", draft)
	}
}

func TestSelectDraftCandidateWithSearchSupportsRegenerate(t *testing.T) {
	ui := teaUI{
		runSearchListFn: func(title string, options []searchOption, emptyText string) (string, error) {
			return "__regen__", nil
		},
	}

	_, err := ui.selectDraftCandidateWithSearch(defaultCzConfig(), []czDraft{
		{Type: "feat", Subject: "first"},
	})
	if !errors.Is(err, errRegenerateCandidates) {
		t.Fatalf("expected regenerate error, got %v", err)
	}
}

func TestBuildPreviewOptionsIncludesCommitEditCancel(t *testing.T) {
	options := buildPreviewOptions()
	if len(options) != 3 {
		t.Fatalf("expected 3 preview options, got %+v", options)
	}
	if options[0].Value != "__commit__" || options[1].Value != "__edit__" || options[2].Value != "__cancel__" {
		t.Fatalf("unexpected preview options: %+v", options)
	}
}

func TestTeaUIReviewAndCommitCommitsSelectedPreview(t *testing.T) {
	ui := teaUI{
		runSearchListFn: func(title string, options []searchOption, emptyText string) (string, error) {
			if !strings.Contains(title, "feat(cli): add preview selector") {
				t.Fatalf("expected preview title to contain commit message, got %q", title)
			}
			return "__commit__", nil
		},
	}

	draft := czDraft{Type: "feat", Scope: "cli", Subject: "add preview selector"}
	var committed string
	err := ui.ReviewAndCommit(draft, defaultCzConfig(), func(msg string) error {
		committed = msg
		return nil
	})
	if err != nil {
		t.Fatalf("review and commit: %v", err)
	}
	if !strings.Contains(committed, "feat(cli): add preview selector") {
		t.Fatalf("unexpected committed message: %q", committed)
	}
}

func TestTeaUIReviewAndCommitEditsThenCommits(t *testing.T) {
	choices := []string{"__edit__", "__commit__"}
	ui := teaUI{
		runSearchListFn: func(title string, options []searchOption, emptyText string) (string, error) {
			v := choices[0]
			choices = choices[1:]
			return v, nil
		},
		editDraftFn: func(d czDraft, cfg czConfig) czDraft {
			d.Subject = "edited subject"
			return d
		},
	}

	draft := czDraft{Type: "feat", Scope: "cli", Subject: "original subject"}
	var committed string
	err := ui.ReviewAndCommit(draft, defaultCzConfig(), func(msg string) error {
		committed = msg
		return nil
	})
	if err != nil {
		t.Fatalf("review and commit: %v", err)
	}
	if !strings.Contains(committed, "edited subject") {
		t.Fatalf("expected edited subject in commit, got %q", committed)
	}
}

func TestTeaUIReviewAndCommitCancelsWithoutCommit(t *testing.T) {
	ui := teaUI{
		runSearchListFn: func(title string, options []searchOption, emptyText string) (string, error) {
			return "__cancel__", nil
		},
	}

	called := false
	err := ui.ReviewAndCommit(czDraft{Type: "feat", Subject: "cancel preview"}, defaultCzConfig(), func(msg string) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected cancel to return nil, got %v", err)
	}
	if called {
		t.Fatal("expected cancel path not to commit")
	}
}

func TestBuildSubjectPromptIncludesCurrentLength(t *testing.T) {
	cfg := defaultCzConfig()
	cfg.MaxSubjectLength = 20

	prompt := buildSubjectPrompt(cfg, "add ui")
	if !strings.Contains(prompt, "6/20") {
		t.Fatalf("expected prompt to include current length, got %q", prompt)
	}
}

func TestBuildSubjectPromptMarksOverflow(t *testing.T) {
	cfg := defaultCzConfig()
	cfg.MaxSubjectLength = 10

	prompt := buildSubjectPrompt(cfg, "add tui flow")
	if !strings.Contains(prompt, "over by") {
		t.Fatalf("expected overflow prompt, got %q", prompt)
	}
}

func TestSubjectInputModelUpdatesLengthInView(t *testing.T) {
	cfg := defaultCzConfig()
	cfg.MaxSubjectLength = 10

	model := newSubjectInputModel(cfg, "abc")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	model = updated.(subjectInputModel)

	view := model.View()
	if !strings.Contains(view, "4/10") {
		t.Fatalf("expected live length feedback in view, got %q", view)
	}
	if !strings.Contains(view, "Enter to finish") {
		t.Fatalf("expected subject view to show enter hint, got %q", view)
	}
}

func TestTeaUIDraftFromWizardUsesRealtimeSubjectInput(t *testing.T) {
	lineInputs := []string{"", ""}
	multiInputs := []string{"", "", ""}

	ui := teaUI{
		selectTypeFn:  func(cfg czConfig) (string, error) { return "feat", nil },
		selectScopeFn: func(cfg czConfig) (string, error) { return "cli", nil },
		runSubjectInputFn: func(cfg czConfig, initial string) (string, error) {
			return "live typed subject", nil
		},
		promptLineFn: func(string) string {
			v := lineInputs[0]
			lineInputs = lineInputs[1:]
			return v
		},
		promptMultilineFn: func(string, czConfig, string) (string, error) {
			v := multiInputs[0]
			multiInputs = multiInputs[1:]
			return v, nil
		},
	}

	cfg := defaultCzConfig()
	cfg.Scopes = []czScope{{Value: "cli", Name: "CLI"}}
	draft, err := ui.DraftFromWizard(cfg)
	if err != nil {
		t.Fatalf("draft from wizard: %v", err)
	}
	if draft.Subject != "live typed subject" {
		t.Fatalf("expected subject from live input, got %q", draft.Subject)
	}
}

func TestTeaUIEditDraftUsesFieldPicker(t *testing.T) {
	choices := []string{"__field_subject__", "__done__"}
	ui := teaUI{
		runSearchListFn: func(title string, options []searchOption, emptyText string) (string, error) {
			v := choices[0]
			choices = choices[1:]
			return v, nil
		},
		runSubjectInputFn: func(cfg czConfig, initial string) (string, error) {
			if initial != "old subject" {
				t.Fatalf("expected initial subject, got %q", initial)
			}
			return "new subject", nil
		},
		promptLineFn: func(prompt string) string {
			return ""
		},
		promptMultilineFn: func(prompt string, cfg czConfig, initial string) (string, error) {
			return initial, nil
		},
	}

	edited := ui.editDraftWithTUI(czDraft{Type: "feat", Scope: "cli", Subject: "old subject"}, defaultCzConfig())
	if edited.Subject != "new subject" {
		t.Fatalf("expected edited subject, got %+v", edited)
	}
}

func TestMultilineInputModelShowsCurrentValueInView(t *testing.T) {
	model := newMultilineInputModel("Body", "line1")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	model = updated.(multilineInputModel)

	view := model.View()
	if !strings.Contains(view, "line1a") {
		t.Fatalf("expected multiline view to include updated content, got %q", view)
	}
}

func TestTeaUIEditDraftUsesTUIInputForBody(t *testing.T) {
	choices := []string{"__field_body__", "__done__"}
	ui := teaUI{
		runSearchListFn: func(title string, options []searchOption, emptyText string) (string, error) {
			v := choices[0]
			choices = choices[1:]
			return v, nil
		},
		runMultilineInputFn: func(title, initial string) (string, error) {
			if title != "Body" {
				t.Fatalf("expected Body title, got %q", title)
			}
			if initial != "old body" {
				t.Fatalf("expected initial body, got %q", initial)
			}
			return "new body", nil
		},
		runSubjectInputFn: func(cfg czConfig, initial string) (string, error) {
			return initial, nil
		},
		promptMultilineFn: func(prompt string, cfg czConfig, initial string) (string, error) {
			return initial, nil
		},
	}

	edited := ui.editDraftWithTUI(czDraft{Type: "feat", Subject: "subject", Body: "old body"}, defaultCzConfig())
	if edited.Body != "new body" {
		t.Fatalf("expected edited body, got %+v", edited)
	}
}

func TestTeaUIEditDraftUsesTUIInputForFooter(t *testing.T) {
	choices := []string{"__field_footer__", "__done__"}
	ui := teaUI{
		runSearchListFn: func(title string, options []searchOption, emptyText string) (string, error) {
			v := choices[0]
			choices = choices[1:]
			return v, nil
		},
		runMultilineInputFn: func(title, initial string) (string, error) {
			if title != "Footer" {
				t.Fatalf("expected Footer title, got %q", title)
			}
			return "Refs #12", nil
		},
		runSubjectInputFn: func(cfg czConfig, initial string) (string, error) {
			return initial, nil
		},
		promptMultilineFn: func(prompt string, cfg czConfig, initial string) (string, error) {
			return initial, nil
		},
	}

	edited := ui.editDraftWithTUI(czDraft{Type: "feat", Subject: "subject"}, defaultCzConfig())
	if edited.Footer != "Refs #12" {
		t.Fatalf("expected edited footer, got %+v", edited)
	}
}

func TestTeaUIDraftFromWizardUsesTUIInputForBodyBreakingAndFooter(t *testing.T) {
	lineInputs := []string{"", "#"}
	mlTitles := []string{}

	ui := teaUI{
		selectTypeFn: func(cfg czConfig) (string, error) { return "feat", nil },
		runSubjectInputFn: func(cfg czConfig, initial string) (string, error) {
			return "wizard subject", nil
		},
		runMultilineInputFn: func(title, initial string) (string, error) {
			mlTitles = append(mlTitles, title)
			switch title {
			case "Body":
				return "line1\nline2", nil
			case "Breaking":
				return "breaking api", nil
			case "Footer":
				return "#12", nil
			default:
				return "", nil
			}
		},
		promptLineFn: func(prompt string) string {
			v := lineInputs[0]
			lineInputs = lineInputs[1:]
			return v
		},
	}

	cfg := defaultCzConfig()
	draft, err := ui.DraftFromWizard(cfg)
	if err != nil {
		t.Fatalf("draft from wizard: %v", err)
	}
	if draft.Body != "line1\nline2" {
		t.Fatalf("expected TUI body input, got %q", draft.Body)
	}
	if draft.Breaking != "breaking api" {
		t.Fatalf("expected TUI breaking input, got %q", draft.Breaking)
	}
	if draft.Footer != "#12" {
		t.Fatalf("expected footer prefix to be applied, got %q", draft.Footer)
	}
	if strings.Join(mlTitles, "|") != "Body|Breaking|Footer" {
		t.Fatalf("unexpected multiline title order: %+v", mlTitles)
	}
}

func TestSubjectInputModelSupportsCursorMovement(t *testing.T) {
	model := newSubjectInputModel(defaultCzConfig(), "abcd")

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(subjectInputModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	model = updated.(subjectInputModel)
	if model.value != "abcXd" {
		t.Fatalf("expected insert at moved cursor, got %q", model.value)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyHome})
	model = updated.(subjectInputModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
	model = updated.(subjectInputModel)
	if model.value != "YabcXd" {
		t.Fatalf("expected home insert, got %q", model.value)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnd})
	model = updated.(subjectInputModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Z")})
	model = updated.(subjectInputModel)
	if model.value != "YabcXdZ" {
		t.Fatalf("expected end insert, got %q", model.value)
	}
}

func TestMultilineInputModelSupportsCursorMovement(t *testing.T) {
	model := newMultilineInputModel("Body", "ab\ncd")

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(multilineInputModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	model = updated.(multilineInputModel)
	if model.value != "ab\ncXd" {
		t.Fatalf("expected insert near end after left, got %q", model.value)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyHome})
	model = updated.(multilineInputModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
	model = updated.(multilineInputModel)
	if model.value != "Yab\ncXd" {
		t.Fatalf("expected home insert, got %q", model.value)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnd})
	model = updated.(multilineInputModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Z")})
	model = updated.(multilineInputModel)
	if model.value != "Yab\ncXdZ" {
		t.Fatalf("expected end insert, got %q", model.value)
	}
}

func TestSubjectInputModelDeleteRemovesCurrentCharacter(t *testing.T) {
	model := newSubjectInputModel(defaultCzConfig(), "abcd")

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(subjectInputModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDelete})
	model = updated.(subjectInputModel)

	if model.value != "abc" {
		t.Fatalf("expected delete to remove current character, got %q", model.value)
	}
}

func TestSubjectInputModelSupportsCtrlDeleteAndBackspace(t *testing.T) {
	model := newSubjectInputModel(defaultCzConfig(), "abcd")

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(subjectInputModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	model = updated.(subjectInputModel)
	if model.value != "abc" {
		t.Fatalf("expected Ctrl+D to delete current character, got %q", model.value)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	model = updated.(subjectInputModel)
	if model.value != "ab" {
		t.Fatalf("expected Ctrl+H to backspace, got %q", model.value)
	}
}

func TestMultilineInputModelDeleteAndVerticalMovement(t *testing.T) {
	model := newMultilineInputModel("Body", "abc\ndefg")

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyHome})
	model = updated.(multilineInputModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(multilineInputModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(multilineInputModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(multilineInputModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	model = updated.(multilineInputModel)

	if model.value != "abc\ndeXfg" {
		t.Fatalf("expected down movement to preserve column, got %q", model.value)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(multilineInputModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDelete})
	model = updated.(multilineInputModel)

	if model.value != "ab\ndeXfg" {
		t.Fatalf("expected delete at moved cursor, got %q", model.value)
	}
}

func TestMultilineInputModelSupportsCtrlShortcuts(t *testing.T) {
	model := newMultilineInputModel("Body", "abcd")

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(multilineInputModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	model = updated.(multilineInputModel)
	if model.value != "abc" {
		t.Fatalf("expected Ctrl+D delete, got %q", model.value)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	model = updated.(multilineInputModel)
	if model.value != "ab" {
		t.Fatalf("expected Ctrl+H backspace, got %q", model.value)
	}
}

func TestMultilineInputModelEnterFinishesInput(t *testing.T) {
	model := newMultilineInputModel("Body", "line1")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(multilineInputModel)
	if !model.accepted {
		t.Fatal("expected Enter to finish multiline input")
	}
	if cmd == nil {
		t.Fatal("expected Enter to request quit")
	}
}

func TestMultilineInputModelAltJInsertsNewline(t *testing.T) {
	model := newMultilineInputModel("Body", "line1")

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}, Alt: true})
	model = updated.(multilineInputModel)
	if model.accepted {
		t.Fatal("expected Alt+J not to finish multiline input")
	}
	if model.value != "line1\n" {
		t.Fatalf("expected Alt+J to insert newline, got %q", model.value)
	}
	if model.cursor != len([]rune(model.value)) {
		t.Fatalf("expected cursor to move after newline, got %d", model.cursor)
	}
}

func TestMultilineInputModelViewShowsAltJHint(t *testing.T) {
	model := newMultilineInputModel("Body", "line1")

	view := model.View()
	if !strings.Contains(view, "Enter to finish") {
		t.Fatalf("expected view to show Enter finish hint, got %q", view)
	}
	if !strings.Contains(view, "Alt+J for newline") {
		t.Fatalf("expected view to show Alt+J newline hint, got %q", view)
	}
}

func teaKeySpace() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeySpace}
}

func teaKeyDown() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyDown}
}

func teaKeyEnter() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}
