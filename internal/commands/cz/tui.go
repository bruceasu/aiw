package cz

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type teaUI struct {
	selectTypeFn        func(cfg czConfig) (string, error)
	selectScopeFn       func(cfg czConfig) (string, error)
	selectScopesFn      func(cfg czConfig) ([]string, error)
	selectCandidateFn   func(cfg czConfig, cands []czDraft) (czDraft, error)
	draftFromLLMFn      func(cfg czConfig, selectFn func(czConfig, []czDraft) (czDraft, error)) (czDraft, error)
	runSearchListFn     func(title string, options []searchOption, emptyText string) (string, error)
	runSubjectInputFn   func(cfg czConfig, initial string) (string, error)
	runMultilineInputFn func(title, initial string) (string, error)
	editDraftFn         func(d czDraft, cfg czConfig) czDraft
	promptLineFn        func(string) string
	promptMultilineFn   func(string, czConfig, string) (string, error)
}

type searchOption struct {
	Value string
	Label string
}

type searchOptionModel struct {
	title       string
	emptyText   string
	instruction string
	all         []searchOption
	filtered    []searchOption
	query       string
	index       int
	selected    string
	aborted     bool
}

type searchCheckboxModel struct {
	title       string
	emptyText   string
	instruction string
	all         []searchOption
	filtered    []searchOption
	query       string
	index       int
	selected    map[string]struct{}
	confirmed   bool
	aborted     bool
}

type subjectInputModel struct {
	cfg      czConfig
	value    string
	cursor   int
	aborted  bool
	accepted bool
}

type multilineInputModel struct {
	title    string
	value    string
	cursor   int
	aborted  bool
	accepted bool
}

var errRegenerateCandidates = errors.New("regenerate llm candidates")

func newDefaultUI(interactive bool) UI {
	if !interactive {
		return lineUI{}
	}
	return newTeaUI()
}

func newTeaUI() UI {
	return teaUI{
		selectTypeFn:        selectTypeWithSearch,
		selectScopeFn:       selectScopeWithSearch,
		selectScopesFn:      selectScopesWithSearch,
		selectCandidateFn:   selectDraftCandidateWithSearch,
		draftFromLLMFn:      czDraftFromLLMWithSelector,
		runSearchListFn:     runSearchList,
		runSubjectInputFn:   runSubjectInput,
		runMultilineInputFn: runMultilineInput,
		promptLineFn:        promptLine,
		promptMultilineFn:   promptMultilineWithEditor,
	}
}

func (ui teaUI) DraftFromWizard(cfg czConfig) (czDraft, error) {
	d := czDraft{}

	t, err := ui.selectTypeFn(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "interactive selector unavailable, falling back to line input: %v\n", err)
		t, err = promptType(cfg)
		if err != nil {
			return d, err
		}
	}
	d.Type = t

	scope, err := ui.selectScope(cfg)
	if err != nil {
		return d, err
	}
	d.Scope = scope

	if ui.runSubjectInputFn != nil {
		for {
			subject, err := ui.runSubjectInputFn(cfg, d.Subject)
			if err != nil {
				return d, err
			}
			d.Subject = strings.TrimSpace(subject)
			if d.Subject != "" {
				break
			}
			fmt.Fprintln(os.Stderr, "subject is required")
		}
	} else {
		for {
			d.Subject = strings.TrimSpace(ui.promptLineFn(buildSubjectPrompt(cfg, d.Subject)))
			if d.Subject != "" {
				break
			}
			fmt.Fprintln(os.Stderr, "subject is required")
		}
	}

	multilineFn := ui.runMultilineInputFn
	if multilineFn == nil {
		multilineFn = func(title, initial string) (string, error) {
			switch title {
			case "Body":
				return ui.promptMultilineFn(cfg.Messages.Body, cfg, initial)
			case "Breaking":
				return ui.promptMultilineFn(cfg.Messages.Breaking, cfg, initial)
			case "Footer":
				return ui.promptMultilineFn(cfg.Messages.Footer, cfg, initial)
			default:
				return ui.promptMultilineFn(title, cfg, initial)
			}
		}
	}

	if d.Body, err = multilineFn("Body", ""); err != nil {
		return d, err
	}
	if d.Breaking, err = multilineFn("Breaking", ""); err != nil {
		return d, err
	}
	prefix := strings.TrimSpace(ui.promptLineFn(cfg.Messages.FooterPrefixes + " [#, refs, closes, custom, skip] "))
	if strings.EqualFold(prefix, "custom") {
		prefix = strings.TrimSpace(ui.promptLineFn(cfg.Messages.CustomFooterPrefix + " "))
	}
	footer, err := multilineFn("Footer", "")
	if err != nil {
		return d, err
	}
	footer = strings.TrimSpace(footer)
	if footer != "" && prefix != "" && !strings.HasPrefix(strings.ToLower(footer), strings.ToLower(prefix)) {
		footer = prefix + " " + footer
	}
	d.Footer = normalizeMultiline(footer)
	return d, nil
}

func (ui teaUI) selectScope(cfg czConfig) (string, error) {
	if cfg.EnableMultipleScopes && ui.selectScopesFn != nil && len(cfg.Scopes) > 0 {
		scopes, err := ui.selectScopesFn(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "interactive scope checkbox unavailable, falling back to line input: %v\n", err)
		} else {
			return strings.Join(scopes, scopeSeparator(cfg)), nil
		}
	}

	if len(cfg.Scopes) == 0 || ui.selectScopeFn == nil {
		scope := strings.TrimSpace(ui.promptLineFn(cfg.Messages.Scope + " "))
		if scope == "." || strings.EqualFold(scope, "custom") {
			scope = strings.TrimSpace(ui.promptLineFn(cfg.Messages.CustomScope + " "))
		}
		return scope, nil
	}

	scope, err := ui.selectScopeFn(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "interactive scope selector unavailable, falling back to line input: %v\n", err)
		scope := strings.TrimSpace(ui.promptLineFn(cfg.Messages.Scope + " "))
		if scope == "." || strings.EqualFold(scope, "custom") {
			scope = strings.TrimSpace(ui.promptLineFn(cfg.Messages.CustomScope + " "))
		}
		return scope, nil
	}
	if scope == "." {
		return strings.TrimSpace(ui.promptLineFn(cfg.Messages.CustomScope + " ")), nil
	}
	return scope, nil
}

func (ui teaUI) DraftFromLLM(cfg czConfig) (czDraft, error) {
	selectFn := ui.selectCandidateFn
	if selectFn == nil {
		selectFn = selectDraftCandidateWithSearch
	}
	draftFn := ui.draftFromLLMFn
	if draftFn == nil {
		draftFn = czDraftFromLLMWithSelector
	}
	return draftFn(cfg, selectFn)
}

func (ui teaUI) ReviewAndCommit(draft czDraft, cfg czConfig, commitFn func(string) error) error {
	runFn := ui.runSearchListFn
	if runFn == nil {
		runFn = runSearchList
	}
	editFn := ui.editDraftFn
	if editFn == nil {
		editFn = ui.editDraftWithTUI
	}

	for {
		msg := buildCommitMessage(draft)
		title := "\n--- Commit message preview ---\n" + msg + "\n--------------------------------"
		choice, err := runFn(title, buildPreviewOptions(), "No preview actions.")
		if err != nil {
			return err
		}
		switch choice {
		case "__commit__":
			return commitFn(msg)
		case "__edit__":
			draft = editFn(draft, cfg)
		case "__cancel__":
			fmt.Fprintln(os.Stderr, "aborted")
			return nil
		default:
			return fmt.Errorf("invalid preview action: %s", choice)
		}
	}
}

func filterTypeOptions(types []czType, query string) []czType {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return append([]czType(nil), types...)
	}

	filtered := make([]czType, 0, len(types))
	for _, t := range types {
		value := strings.ToLower(t.Value)
		name := strings.ToLower(t.Name)
		if strings.Contains(value, query) || strings.Contains(name, query) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func filterScopeOptions(scopes []czScope, query string) []czScope {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return append([]czScope(nil), scopes...)
	}

	filtered := make([]czScope, 0, len(scopes))
	for _, s := range scopes {
		value := strings.ToLower(s.Value)
		name := strings.ToLower(s.Name)
		if strings.Contains(value, query) || strings.Contains(name, query) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func buildScopeOptions(cfg czConfig) []czScope {
	options := []czScope{
		{Value: "", Name: "(empty) no scope"},
		{Value: ".", Name: "(custom) input custom scope"},
	}
	options = append(options, cfg.Scopes...)
	return options
}

func scopeSeparator(cfg czConfig) string {
	if strings.TrimSpace(cfg.ScopeEnumSeparator) != "" {
		return cfg.ScopeEnumSeparator
	}
	return ","
}

func selectTypeWithSearch(cfg czConfig) (string, error) {
	options := make([]searchOption, 0, len(cfg.Types))
	for _, item := range cfg.Types {
		options = append(options, searchOption{Value: item.Value, Label: item.Name})
	}
	return runSearchList(cfg.Messages.Type, options, "No matching commit types.")
}

func selectScopeWithSearch(cfg czConfig) (string, error) {
	scopes := buildScopeOptions(cfg)
	options := make([]searchOption, 0, len(scopes))
	for _, item := range scopes {
		options = append(options, searchOption{Value: item.Value, Label: item.Name})
	}
	return runSearchList(cfg.Messages.Scope, options, "No matching scopes.")
}

func selectScopesWithSearch(cfg czConfig) ([]string, error) {
	scopes := buildScopeOptions(cfg)
	options := make([]searchOption, 0, len(scopes))
	for _, item := range scopes {
		options = append(options, searchOption{Value: item.Value, Label: item.Name})
	}
	return runSearchCheckbox(cfg.Messages.Scope, options, "No matching scopes.")
}

func buildCandidateOptions(cands []czDraft) []searchOption {
	options := make([]searchOption, 0, len(cands)+1)
	for i, cand := range cands {
		options = append(options, searchOption{
			Value: strconv.Itoa(i),
			Label: buildHeader(cand),
		})
	}
	options = append(options, searchOption{
		Value: "__regen__",
		Label: "(regenerate) generate a new candidate list",
	})
	return options
}

func buildPreviewOptions() []searchOption {
	return []searchOption{
		{Value: "__commit__", Label: "commit this message"},
		{Value: "__edit__", Label: "edit fields"},
		{Value: "__cancel__", Label: "cancel"},
	}
}

func buildEditFieldOptions() []searchOption {
	return []searchOption{
		{Value: "__field_type__", Label: "edit type"},
		{Value: "__field_scope__", Label: "edit scope"},
		{Value: "__field_subject__", Label: "edit subject"},
		{Value: "__field_body__", Label: "edit body"},
		{Value: "__field_breaking__", Label: "edit breaking"},
		{Value: "__field_footer__", Label: "edit footer"},
		{Value: "__done__", Label: "done editing"},
	}
}

func buildSubjectPrompt(cfg czConfig, current string) string {
	base := cfg.Messages.Subject
	limit := cfg.MaxSubjectLength
	if limit <= 0 {
		return base
	}
	length := len([]rune(strings.TrimSpace(current)))
	if current == "" {
		return fmt.Sprintf("%s (%d/%d)", strings.TrimRight(base, "\n"), 0, limit)
	}
	if length > limit {
		return fmt.Sprintf("%s (%d/%d, over by %d)", strings.TrimRight(base, "\n"), length, limit, length-limit)
	}
	return fmt.Sprintf("%s (%d/%d)", strings.TrimRight(base, "\n"), length, limit)
}

func runSubjectInput(cfg czConfig, initial string) (string, error) {
	model := newSubjectInputModel(cfg, initial)
	finalModel, err := tea.NewProgram(model).Run()
	if err != nil {
		return "", err
	}
	m, ok := finalModel.(subjectInputModel)
	if !ok {
		return "", errors.New("unexpected subject input model type")
	}
	if m.aborted {
		return "", errors.New("aborted")
	}
	return strings.TrimSpace(m.value), nil
}

func runMultilineInput(title, initial string) (string, error) {
	model := newMultilineInputModel(title, initial)
	finalModel, err := tea.NewProgram(model).Run()
	if err != nil {
		return "", err
	}
	m, ok := finalModel.(multilineInputModel)
	if !ok {
		return "", errors.New("unexpected multiline input model type")
	}
	if m.aborted {
		return "", errors.New("aborted")
	}
	return strings.TrimSpace(m.value), nil
}

func newSubjectInputModel(cfg czConfig, initial string) subjectInputModel {
	return subjectInputModel{
		cfg:    cfg,
		value:  initial,
		cursor: len([]rune(initial)),
	}
}

func newMultilineInputModel(title, initial string) multilineInputModel {
	return multilineInputModel{
		title:  title,
		value:  initial,
		cursor: len([]rune(initial)),
	}
}

func lineColumnForCursor(runes []rune, cursor int) (line int, col int) {
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	for i := 0; i < cursor; i++ {
		if runes[i] == '\n' {
			line++
			col = 0
			continue
		}
		col++
	}
	return line, col
}

func cursorForLineColumn(runes []rune, targetLine int, targetCol int) int {
	if targetLine < 0 {
		targetLine = 0
	}
	if targetCol < 0 {
		targetCol = 0
	}
	line := 0
	lineStart := 0
	for i, r := range runes {
		if line == targetLine {
			if r == '\n' {
				lineLen := i - lineStart
				if lineLen == 0 {
					return i
				}
				if targetCol >= lineLen {
					return i - 1
				}
				return lineStart + targetCol
			}
		}
		if r == '\n' {
			line++
			lineStart = i + 1
		}
	}
	if line == targetLine {
		lineLen := len(runes) - lineStart
		if lineLen == 0 {
			return len(runes)
		}
		if targetCol >= lineLen {
			return len(runes) - 1
		}
		return lineStart + targetCol
	}
	return len(runes)
}

func selectDraftCandidateWithSearch(cfg czConfig, cands []czDraft) (czDraft, error) {
	ui := newTeaUI().(teaUI)
	return ui.selectDraftCandidateWithSearch(cfg, cands)
}

func (ui teaUI) selectDraftCandidateWithSearch(cfg czConfig, cands []czDraft) (czDraft, error) {
	runFn := ui.runSearchListFn
	if runFn == nil {
		runFn = runSearchList
	}
	value, err := runFn("Select AI candidate", buildCandidateOptions(cands), "No matching candidates.")
	if err != nil {
		return czDraft{}, err
	}
	if value == "__regen__" {
		return czDraft{}, errRegenerateCandidates
	}
	idx, convErr := strconv.Atoi(value)
	if convErr != nil || idx < 0 || idx >= len(cands) {
		return czDraft{}, errors.New("invalid llm candidate selection")
	}
	return cands[idx], nil
}

func runSearchList(title string, options []searchOption, emptyText string) (string, error) {
	model := newSearchOptionModel(title, options, emptyText)
	finalModel, err := tea.NewProgram(model).Run()
	if err != nil {
		return "", err
	}
	m, ok := finalModel.(searchOptionModel)
	if !ok {
		return "", errors.New("unexpected search model type")
	}
	if m.aborted {
		return "", errors.New("aborted")
	}
	if m.selected == "" && !containsEmptyOption(options) {
		return "", errors.New("selection is required")
	}
	return m.selected, nil
}

func containsEmptyOption(options []searchOption) bool {
	for _, opt := range options {
		if opt.Value == "" {
			return true
		}
	}
	return false
}

func newSearchOptionModel(title string, options []searchOption, emptyText string) searchOptionModel {
	return searchOptionModel{
		title:       title,
		emptyText:   emptyText,
		instruction: "Use arrows to move, type to filter, enter to confirm.",
		all:         append([]searchOption(nil), options...),
		filtered:    append([]searchOption(nil), options...),
	}
}

func newSearchCheckboxModel(title string, options []searchOption, emptyText string) searchCheckboxModel {
	return searchCheckboxModel{
		title:       title,
		emptyText:   emptyText,
		instruction: "Use arrows to move, type to filter, space to toggle, enter to confirm.",
		all:         append([]searchOption(nil), options...),
		filtered:    append([]searchOption(nil), options...),
		selected:    map[string]struct{}{},
	}
}

func (m searchOptionModel) Init() tea.Cmd {
	return nil
}

func (m searchOptionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.aborted = true
			return m, tea.Quit
		case tea.KeyEnter:
			if len(m.filtered) == 0 {
				return m, nil
			}
			m.selected = m.filtered[m.index].Value
			return m, tea.Quit
		case tea.KeyUp:
			if len(m.filtered) == 0 {
				return m, nil
			}
			if m.index > 0 {
				m.index--
			}
			return m, nil
		case tea.KeyDown, tea.KeyTab:
			if len(m.filtered) == 0 {
				return m, nil
			}
			if m.index < len(m.filtered)-1 {
				m.index++
			}
			return m, nil
		case tea.KeyBackspace, tea.KeyDelete:
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.applyFilter()
			}
			return m, nil
		default:
			if msg.Type == tea.KeyRunes {
				m.query += msg.String()
				m.applyFilter()
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *searchOptionModel) applyFilter() {
	m.filtered = filterSearchOptions(m.all, m.query)
	if len(m.filtered) == 0 {
		m.index = 0
		return
	}
	if m.index >= len(m.filtered) {
		m.index = len(m.filtered) - 1
	}
	if m.index < 0 {
		m.index = 0
	}
}

func filterSearchOptions(options []searchOption, query string) []searchOption {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return append([]searchOption(nil), options...)
	}

	filtered := make([]searchOption, 0, len(options))
	for _, opt := range options {
		value := strings.ToLower(opt.Value)
		label := strings.ToLower(opt.Label)
		if strings.Contains(value, query) || strings.Contains(label, query) {
			filtered = append(filtered, opt)
		}
	}
	return filtered
}

func runSearchCheckbox(title string, options []searchOption, emptyText string) ([]string, error) {
	model := newSearchCheckboxModel(title, options, emptyText)
	finalModel, err := tea.NewProgram(model).Run()
	if err != nil {
		return nil, err
	}
	m, ok := finalModel.(searchCheckboxModel)
	if !ok {
		return nil, errors.New("unexpected checkbox model type")
	}
	if m.aborted {
		return nil, errors.New("aborted")
	}
	return m.selectedValues(), nil
}

func (ui teaUI) editDraftWithTUI(d czDraft, cfg czConfig) czDraft {
	runFn := ui.runSearchListFn
	if runFn == nil {
		runFn = runSearchList
	}
	subjectFn := ui.runSubjectInputFn
	if subjectFn == nil {
		subjectFn = runSubjectInput
	}
	multilineFn := ui.runMultilineInputFn
	if multilineFn == nil {
		multilineFn = runMultilineInput
	}

	for {
		title := fmt.Sprintf("Edit commit fields\nCurrent: %s", buildHeader(d))
		choice, err := runFn(title, buildEditFieldOptions(), "No edit actions.")
		if err != nil {
			fmt.Fprintf(os.Stderr, "edit selector failed: %v\n", err)
			return sanitizeDraft(d, cfg)
		}
		switch choice {
		case "__field_type__":
			if t, err := ui.selectTypeFn(cfg); err == nil && strings.TrimSpace(t) != "" {
				d.Type = strings.TrimSpace(t)
			}
		case "__field_scope__":
			if scope, err := ui.selectScope(cfg); err == nil {
				d.Scope = strings.TrimSpace(scope)
			}
		case "__field_subject__":
			if subject, err := subjectFn(cfg, d.Subject); err == nil {
				d.Subject = strings.TrimSpace(subject)
			}
		case "__field_body__":
			if body, err := multilineFn("Body", d.Body); err == nil {
				d.Body = strings.TrimSpace(body)
			}
		case "__field_breaking__":
			if breaking, err := multilineFn("Breaking", d.Breaking); err == nil {
				d.Breaking = strings.TrimSpace(breaking)
			}
		case "__field_footer__":
			if footer, err := multilineFn("Footer", d.Footer); err == nil {
				d.Footer = strings.TrimSpace(footer)
			}
		case "__done__":
			return sanitizeDraft(d, cfg)
		default:
			return sanitizeDraft(d, cfg)
		}
	}
}

func (m searchOptionModel) View() string {
	var b strings.Builder
	b.WriteString(m.title)
	b.WriteString("\n")
	b.WriteString("Search: ")
	b.WriteString(m.query)
	b.WriteString("\n")
	b.WriteString(m.instruction)
	b.WriteString("\n\n")

	if len(m.filtered) == 0 {
		b.WriteString("  ")
		b.WriteString(m.emptyText)
		b.WriteString("\n")
		return b.String()
	}

	for i, item := range m.filtered {
		cursor := "  "
		if i == m.index {
			cursor = "> "
		}
		b.WriteString(cursor)
		b.WriteString(item.Label)
		b.WriteString("\n")
	}
	return b.String()
}

func (m subjectInputModel) Init() tea.Cmd {
	return nil
}

func (m subjectInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.aborted = true
			return m, tea.Quit
		case tea.KeyEnter:
			m.accepted = true
			return m, tea.Quit
		case tea.KeyLeft:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case tea.KeyRight:
			if m.cursor < len([]rune(m.value)) {
				m.cursor++
			}
			return m, nil
		case tea.KeyHome:
			m.cursor = 0
			return m, nil
		case tea.KeyEnd:
			m.cursor = len([]rune(m.value))
			return m, nil
		case tea.KeyBackspace, tea.KeyDelete, tea.KeyCtrlH, tea.KeyCtrlD:
			runes := []rune(m.value)
			if (msg.Type == tea.KeyBackspace || msg.Type == tea.KeyCtrlH) && len(runes) > 0 && m.cursor > 0 {
				runes = append(runes[:m.cursor-1], runes[m.cursor:]...)
				m.value = string(runes)
				m.cursor--
			}
			if (msg.Type == tea.KeyDelete || msg.Type == tea.KeyCtrlD) && len(runes) > 0 && m.cursor < len(runes) {
				runes = append(runes[:m.cursor], runes[m.cursor+1:]...)
				m.value = string(runes)
			}
			return m, nil
		default:
			if msg.Type == tea.KeyRunes {
				runes := []rune(m.value)
				insert := msg.Runes
				runes = append(runes[:m.cursor], append(insert, runes[m.cursor:]...)...)
				m.value = string(runes)
				m.cursor += len(insert)
			}
			return m, nil
		}
	}
	return m, nil
}

func (m subjectInputModel) View() string {
	return buildSubjectPrompt(m.cfg, m.value) +
		"\n\n" + m.value +
		"\n\nEnter to finish, Ctrl+H backspace, Ctrl+D delete"
}

func (m multilineInputModel) Init() tea.Cmd {
	return nil
}

func (m multilineInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyRunes && msg.Alt && len(msg.Runes) == 1 && (msg.Runes[0] == 'j' || msg.Runes[0] == 'J') {
			runes := []rune(m.value)
			runes = append(runes[:m.cursor], append([]rune{'\n'}, runes[m.cursor:]...)...)
			m.value = string(runes)
			m.cursor++
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.aborted = true
			return m, tea.Quit
		case tea.KeyEnter:
			m.accepted = true
			return m, tea.Quit
		case tea.KeyLeft:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case tea.KeyRight:
			if m.cursor < len([]rune(m.value)) {
				m.cursor++
			}
			return m, nil
		case tea.KeyHome:
			m.cursor = 0
			return m, nil
		case tea.KeyEnd:
			m.cursor = len([]rune(m.value))
			return m, nil
		case tea.KeyBackspace, tea.KeyDelete, tea.KeyCtrlH, tea.KeyCtrlD:
			runes := []rune(m.value)
			if (msg.Type == tea.KeyBackspace || msg.Type == tea.KeyCtrlH) && len(runes) > 0 && m.cursor > 0 {
				runes = append(runes[:m.cursor-1], runes[m.cursor:]...)
				m.value = string(runes)
				m.cursor--
			}
			if (msg.Type == tea.KeyDelete || msg.Type == tea.KeyCtrlD) && len(runes) > 0 && m.cursor < len(runes) {
				runes = append(runes[:m.cursor], runes[m.cursor+1:]...)
				m.value = string(runes)
			}
			return m, nil
		case tea.KeyUp:
			runes := []rune(m.value)
			line, col := lineColumnForCursor(runes, m.cursor)
			if line > 0 {
				m.cursor = cursorForLineColumn(runes, line-1, col)
			}
			return m, nil
		case tea.KeyDown:
			runes := []rune(m.value)
			line, col := lineColumnForCursor(runes, m.cursor)
			m.cursor = cursorForLineColumn(runes, line+1, col)
			return m, nil
		default:
			if msg.Type == tea.KeyRunes {
				runes := []rune(m.value)
				insert := msg.Runes
				runes = append(runes[:m.cursor], append(insert, runes[m.cursor:]...)...)
				m.value = string(runes)
				m.cursor += len(insert)
			}
			return m, nil
		}
	}
	return m, nil
}

func (m multilineInputModel) View() string {
	return m.title + "\n\n" + m.value + "\n\nEnter to finish, Alt+J for newline, Ctrl+H backspace, Ctrl+D delete"
}

func (m searchCheckboxModel) Init() tea.Cmd {
	return nil
}

func (m searchCheckboxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.aborted = true
			return m, tea.Quit
		case tea.KeyEnter:
			m.confirmed = true
			return m, tea.Quit
		case tea.KeySpace:
			if len(m.filtered) == 0 {
				return m, nil
			}
			value := m.filtered[m.index].Value
			if _, ok := m.selected[value]; ok {
				delete(m.selected, value)
			} else {
				m.selected[value] = struct{}{}
			}
			return m, nil
		case tea.KeyUp:
			if len(m.filtered) == 0 {
				return m, nil
			}
			if m.index > 0 {
				m.index--
			}
			return m, nil
		case tea.KeyDown, tea.KeyTab:
			if len(m.filtered) == 0 {
				return m, nil
			}
			if m.index < len(m.filtered)-1 {
				m.index++
			}
			return m, nil
		case tea.KeyBackspace, tea.KeyDelete:
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.applyFilter()
			}
			return m, nil
		default:
			if msg.Type == tea.KeyRunes {
				m.query += msg.String()
				m.applyFilter()
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *searchCheckboxModel) applyFilter() {
	m.filtered = filterSearchOptions(m.all, m.query)
	if len(m.filtered) == 0 {
		m.index = 0
		return
	}
	if m.index >= len(m.filtered) {
		m.index = len(m.filtered) - 1
	}
	if m.index < 0 {
		m.index = 0
	}
}

func (m searchCheckboxModel) selectedValues() []string {
	values := make([]string, 0, len(m.selected))
	for _, opt := range m.all {
		if _, ok := m.selected[opt.Value]; ok {
			if opt.Value != "" && opt.Value != "." {
				values = append(values, opt.Value)
			}
		}
	}
	return values
}

func (m searchCheckboxModel) View() string {
	var b strings.Builder
	b.WriteString(m.title)
	b.WriteString("\n")
	b.WriteString("Search: ")
	b.WriteString(m.query)
	b.WriteString("\n")
	b.WriteString(m.instruction)
	b.WriteString("\n\n")

	if len(m.filtered) == 0 {
		b.WriteString("  ")
		b.WriteString(m.emptyText)
		b.WriteString("\n")
		return b.String()
	}

	for i, item := range m.filtered {
		cursor := "  "
		if i == m.index {
			cursor = "> "
		}
		checked := "[ ]"
		if _, ok := m.selected[item.Value]; ok {
			checked = "[x]"
		}
		b.WriteString(cursor)
		b.WriteString(checked)
		b.WriteString(" ")
		b.WriteString(item.Label)
		b.WriteString("\n")
	}
	return b.String()
}
