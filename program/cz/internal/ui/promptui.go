package ui

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"

	"aiw-cz/internal/cz"
)

type PromptUI struct {
	DraftFromLLMFn    func(cfg cz.Config, selectFn func(cz.Config, []cz.Draft) (cz.Draft, error)) (cz.Draft, error)
	PromptLineFn      func(string) string
	PromptMultilineFn func(string, cz.Config, string) (string, error)
}

func NewPromptUI(draftFromLLM func(cz.Config, func(cz.Config, []cz.Draft) (cz.Draft, error)) (cz.Draft, error), promptLine func(string) string, promptMultiline func(string, cz.Config, string) (string, error)) cz.UI {
	return PromptUI{
		DraftFromLLMFn:    draftFromLLM,
		PromptLineFn:      promptLine,
		PromptMultilineFn: promptMultiline,
	}
}

func (ui PromptUI) DraftFromWizard(cfg cz.Config) (cz.Draft, error) {
	d := cz.Draft{}

	t, err := ui.SelectType(cfg)
	if err != nil {
		return d, err
	}
	d.Type = t

	scope, err := ui.selectScope(cfg)
	if err != nil {
		return d, err
	}
	d.Scope = scope

	subject, err := ui.RunSubjectInput(cfg, d.Subject)
	if err != nil {
		return d, err
	}
	d.Subject = subject

	if d.Body, err = ui.PromptMultilineFn(cfg.Messages.Body, cfg, ""); err != nil {
		return d, err
	}
	if d.Breaking, err = ui.PromptMultilineFn(cfg.Messages.Breaking, cfg, ""); err != nil {
		return d, err
	}

	prefix := PromptFooterPrefix(cfg, ui.PromptLineFn)
	footer, err := ui.PromptMultilineFn(cfg.Messages.Footer, cfg, "")
	if err != nil {
		return d, err
	}
	footer = strings.TrimSpace(footer)
	if footer != "" && prefix != "" && !strings.HasPrefix(strings.ToLower(footer), strings.ToLower(prefix)) {
		footer = prefix + " " + footer
	}
	d.Footer = cz.NormalizeMultiline(footer)

	return cz.SanitizeDraft(d, cfg), nil
}

func (ui PromptUI) SelectType(cfg cz.Config) (string, error) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "\U0001F449 {{ .Value | cyan }} ({{ .Name | faint }})",
		Inactive: "   {{ .Value }} ({{ .Name | faint }})",
		Selected: "\U0001F449 {{ .Value | cyan }}",
	}

	searcher := func(input string, index int) bool {
		t := cfg.Types[index]
		name := strings.ToLower(t.Value + t.Name)
		input = strings.ToLower(input)
		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Label:     cfg.Messages.Type,
		Items:     cfg.Types,
		Templates: templates,
		Searcher:  searcher,
		Size:      10,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return cfg.Types[idx].Value, nil
}

func (ui PromptUI) selectScope(cfg cz.Config) (string, error) {
	if len(cfg.Scopes) == 0 {
		scope := strings.TrimSpace(ui.PromptLineFn(cfg.Messages.Scope + " "))
		if scope == "." || strings.EqualFold(scope, "custom") {
			scope = strings.TrimSpace(ui.PromptLineFn(cfg.Messages.CustomScope + " "))
		}
		return scope, nil
	}

	// For promptui, we don't have multi-select, so we use single select.
	// We add "custom" and "none" to the list.
	items := []cz.Scope{
		{Value: "", Name: "(empty) no scope"},
		{Value: ".", Name: "(custom) input custom scope"},
	}
	items = append(items, cfg.Scopes...)

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "\U0001F449 {{ .Value | cyan }} ({{ .Name | faint }})",
		Inactive: "   {{ .Value }} ({{ .Name | faint }})",
		Selected: "\U0001F449 {{ .Value | cyan }}",
	}

	searcher := func(input string, index int) bool {
		s := items[index]
		name := strings.ToLower(s.Value + s.Name)
		input = strings.ToLower(input)
		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Label:     cfg.Messages.Scope,
		Items:     items,
		Templates: templates,
		Searcher:  searcher,
		Size:      10,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return "", err
	}

	val := items[idx].Value
	if val == "." {
		return strings.TrimSpace(ui.PromptLineFn(cfg.Messages.CustomScope + " ")), nil
	}
	return val, nil
}

func (ui PromptUI) RunSubjectInput(cfg cz.Config, initial string) (string, error) {
	validate := func(input string) error {
		if strings.TrimSpace(input) == "" {
			return errors.New("subject is required")
		}
		if cfg.MaxSubjectLength > 0 && len([]rune(input)) > cfg.MaxSubjectLength {
			return fmt.Errorf("subject too long (max %d)", cfg.MaxSubjectLength)
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    BuildSubjectPrompt(cfg, initial),
		Default:  initial,
		Validate: validate,
	}

	return prompt.Run()
}

func (ui PromptUI) DraftFromLLM(cfg cz.Config) (cz.Draft, error) {
	return ui.DraftFromLLMFn(cfg, ui.SelectCandidate)
}

func (ui PromptUI) SelectCandidate(cfg cz.Config, cands []cz.Draft) (cz.Draft, error) {
	type item struct {
		Index int
		Label string
		Value string
	}
	items := make([]item, 0, len(cands)+1)
	for i, c := range cands {
		items = append(items, item{Index: i, Label: cz.BuildHeader(c), Value: strconv.Itoa(i)})
	}
	items = append(items, item{Index: -1, Label: "(regenerate) generate a new candidate list", Value: "__regen__"})

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "\U0001F449 {{ .Label | cyan }}",
		Inactive: "   {{ .Label }}",
		Selected: "\U0001F449 {{ .Label | cyan }}",
	}

	prompt := promptui.Select{
		Label:     "Select AI candidate",
		Items:     items,
		Templates: templates,
		Size:      10,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return cz.Draft{}, err
	}

	if items[idx].Value == "__regen__" {
		return cz.Draft{}, ErrRegenerateCandidates
	}
	return cands[items[idx].Index], nil
}

func (ui PromptUI) ReviewAndCommit(draft cz.Draft, cfg cz.Config, commitFn func(string) error) error {
	for {
		msg := cz.BuildCommitMessage(draft)
		fmt.Printf("\n--- Commit message preview ---\n%s\n--------------------------------\n", msg)

		items := []struct {
			Label string
			Value string
		}{
			{"commit this message", "__commit__"},
			{"edit fields", "__edit__"},
			{"cancel", "__cancel__"},
		}

		templates := &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "\U0001F449 {{ .Label | cyan }}",
			Inactive: "   {{ .Label }}",
			Selected: "\U0001F449 {{ .Label | cyan }}",
		}

		prompt := promptui.Select{
			Label:     "Action",
			Items:     items,
			Templates: templates,
		}

		idx, _, err := prompt.Run()
		if err != nil {
			return err
		}

		switch items[idx].Value {
		case "__commit__":
			return commitFn(msg)
		case "__edit__":
			draft = ui.editDraft(draft, cfg)
		case "__cancel__":
			fmt.Fprintln(os.Stderr, "aborted")
			return nil
		}
	}
}

func (ui PromptUI) editDraft(d cz.Draft, cfg cz.Config) cz.Draft {
	for {
		items := []struct {
			Label string
			Value string
		}{
			{"edit type", "__field_type__"},
			{"edit scope", "__field_scope__"},
			{"edit subject", "__field_subject__"},
			{"edit body", "__field_body__"},
			{"edit breaking", "__field_breaking__"},
			{"edit footer", "__field_footer__"},
			{"done editing", "__done__"},
		}

		templates := &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "\U0001F449 {{ .Label | cyan }}",
			Inactive: "   {{ .Label }}",
			Selected: "\U0001F449 {{ .Label | cyan }}",
		}

		prompt := promptui.Select{
			Label:     fmt.Sprintf("Edit fields (Current: %s)", cz.BuildHeader(d)),
			Items:     items,
			Templates: templates,
		}

		idx, _, err := prompt.Run()
		if err != nil {
			return cz.SanitizeDraft(d, cfg)
		}

		switch items[idx].Value {
		case "__field_type__":
			if t, err := ui.SelectType(cfg); err == nil {
				d.Type = t
			}
		case "__field_scope__":
			if s, err := ui.selectScope(cfg); err == nil {
				d.Scope = s
			}
		case "__field_subject__":
			if s, err := ui.RunSubjectInput(cfg, d.Subject); err == nil {
				d.Subject = s
			}
		case "__field_body__":
			if s, err := ui.PromptMultilineFn(cfg.Messages.Body, cfg, d.Body); err == nil {
				d.Body = s
			}
		case "__field_breaking__":
			if s, err := ui.PromptMultilineFn(cfg.Messages.Breaking, cfg, d.Breaking); err == nil {
				d.Breaking = s
			}
		case "__field_footer__":
			if s, err := ui.PromptMultilineFn(cfg.Messages.Footer, cfg, d.Footer); err == nil {
				d.Footer = s
			}
		case "__done__":
			return cz.SanitizeDraft(d, cfg)
		}
	}
}
