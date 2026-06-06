package cz

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	czdata "aiw-cz/internal/cz"
	"aiw-cz/internal/fsx"
	"aiw-cz/internal/llm"
	"aiw-cz/internal/ui"
	"aiw-cz/internal/util"
)

var issueRefRe = regexp.MustCompile(`#\d+`)

type czOptions struct {
	UseLLM     *bool
	Candidates *int
	Retry      *bool
}

func Dispatch(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "-h", "--help", "help":
			fmt.Println(HelpText())
			return nil
		}
	}
	uiFactory := func() czdata.UI {
		return ui.NewPromptUI(DraftFromLLMWithSelector, ui.PromptLine, ui.PromptMultilineWithEditor)
	}
	defaultUI := ui.NewDefaultUI(ui.IsInteractiveTerminal(), ui.PromptLine, ui.PromptMultilineWithEditor, uiFactory)
	return dispatchWithUI(args, defaultUI, stagedChanges, commitWithMessage)
}

func dispatchWithUI(args []string, u czdata.UI, stagedFn func() (string, error), commitFn func(string) error) error {
	opts, err := parseCzOptions(args)
	if err != nil {
		return err
	}

	cfg, err := loadCzConfig(opts)
	if err != nil {
		return err
	}
	if cfg.Candidates <= 0 {
		cfg.Candidates = 3
	}

	staged, err := stagedFn()
	if err != nil {
		return err
	}
	if strings.TrimSpace(staged) == "" {
		return errors.New("no staged changes; run git add first")
	}

	var draft czdata.Draft
	if opts.Retry != nil && *opts.Retry {
		if d, err := draftFromLastCommit(); err == nil {
			draft = czdata.SanitizeDraft(d, cfg)
			return u.ReviewAndCommit(draft, cfg, commitFn)
		} else {
			return fmt.Errorf("retry: %w", err)
		}
	}

	if cfg.UseLLM {
		draft, err = u.DraftFromLLM(cfg)
		if err != nil {
			return err
		}
	} else {
		draft, err = u.DraftFromWizard(cfg)
		if err != nil {
			return err
		}
	}

	return u.ReviewAndCommit(draft, cfg, commitFn)
}

func stagedChanges() (string, error) {
	return util.RunAndWaitForOuput("git", "diff", "--cached", "--name-only")
}

func parseCzOptions(args []string) (czOptions, error) {
	var opts czOptions
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--llm":
			v := true
			opts.UseLLM = &v
		case "--no-llm":
			v := false
			opts.UseLLM = &v
		case "-N", "--candidates":
			if i+1 >= len(args) {
				return opts, errors.New("missing value for --candidates")
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n <= 0 {
				return opts, fmt.Errorf("invalid candidates value: %q", args[i+1])
			}
			opts.Candidates = &n
			i++
		case "-r", "--retry":
			v := true
			opts.Retry = &v
		default:
			return opts, fmt.Errorf("unknown cz option: %s", a)
		}
	}
	return opts, nil
}

func loadCzConfig(opts czOptions) (czdata.Config, error) {
	cfg := czdata.DefaultConfig()

	progCfgPath := ""
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		progCfgPath = firstExistingFile(filepath.Join(exeDir, "aiw.toml"), filepath.Join(exeDir, ".aiw.toml"))
	}
	if progCfgPath != "" {
		if err := mergeCzConfigFromTomlFile(&cfg, progCfgPath); err != nil {
			return cfg, fmt.Errorf("load program config %s: %w", progCfgPath, err)
		}
	}

	projectCfgPath := ""
	if root, rootErr := detectProjectRoot(); rootErr == nil {
		projectCfgPath = firstExistingFile(filepath.Join(root, "aiw.toml"), filepath.Join(root, ".aiw.toml"))
	}
	if projectCfgPath != "" {
		if err := mergeCzConfigFromTomlFile(&cfg, projectCfgPath); err != nil {
			return cfg, fmt.Errorf("load project config %s: %w", projectCfgPath, err)
		}
	}

	if opts.UseLLM != nil {
		cfg.UseLLM = *opts.UseLLM
	}
	if opts.Candidates != nil {
		cfg.Candidates = *opts.Candidates
	}
	return cfg, nil
}

func detectProjectRoot() (string, error) {
	out, err := util.RunAndWaitForOuput("git", "rev-parse", "--show-toplevel")
	if err == nil && strings.TrimSpace(out) != "" {
		return strings.TrimSpace(out), nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return wd, nil
}

func firstExistingFile(paths ...string) string {
	for _, p := range paths {
		if fsx.Exists(p) {
			return p
		}
	}
	return ""
}

func mergeCzConfigFromTomlFile(cfg *czdata.Config, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	section := ""
	curType := map[string]string{}
	hasType := false
	typeSectionSeen := false
	curScope := map[string]string{}
	hasScope := false
	scopeSectionSeen := false

	applyType := func() {
		if !hasType {
			return
		}
		v := strings.TrimSpace(curType["value"])
		n := strings.TrimSpace(curType["name"])
		if v != "" && n != "" {
			cfg.Types = append(cfg.Types, czdata.Type{Value: v, Name: n})
		}
		curType = map[string]string{}
		hasType = false
	}
	applyScope := func() {
		if !hasScope {
			return
		}
		v := strings.TrimSpace(curScope["value"])
		n := strings.TrimSpace(curScope["name"])
		if v != "" {
			if n == "" {
				n = v
			}
			cfg.Scopes = append(cfg.Scopes, czdata.Scope{Value: v, Name: n})
		}
		curScope = map[string]string{}
		hasScope = false
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[[") && strings.HasSuffix(line, "]]") {
			applyType()
			applyScope()
			tag := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "[["), "]]"))
			section = tag
			if section == "cz.types" {
				if !typeSectionSeen {
					cfg.Types = nil
					typeSectionSeen = true
				}
				hasType = true
			} else if section == "cz.scopes" {
				if !scopeSectionSeen {
					cfg.Scopes = nil
					scopeSectionSeen = true
				}
				hasScope = true
			}
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			applyType()
			applyScope()
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valRaw := strings.TrimSpace(parts[1])
		val := parseTomlStringValue(valRaw)

		switch section {
		case "cz":
			switch key {
			case "llm", "use_llm":
				if b, ok := parseTomlBool(valRaw); ok {
					cfg.UseLLM = b
				}
			case "model", "llm_model", "openai_model":
				cfg.LLMModel = val
			case "base_url", "openai_base_url":
				cfg.APIBaseURL = val
			case "api_key", "openai_api_key":
				cfg.APIKey = val
			case "debug_source", "debug":
				if b, ok := parseTomlBool(valRaw); ok {
					cfg.DebugSource = b
				}
			case "candidates":
				if n, ok := parseTomlInt(valRaw); ok && n > 0 {
					cfg.Candidates = n
				}
			case "editor", "EDITOR":
				cfg.Editor = val
			case "enable_multiple_scopes", "enableMultipleScopes":
				if b, ok := parseTomlBool(valRaw); ok {
					cfg.EnableMultipleScopes = b
				}
			case "scope_enum_separator", "scopeEnumSeparator":
				if val != "" {
					cfg.ScopeEnumSeparator = val
				}
			case "max_subject_length", "maxSubjectLength":
				if n, ok := parseTomlInt(valRaw); ok && n > 0 {
					cfg.MaxSubjectLength = n
				}
			}
		case "cz.messages":
			switch key {
			case "type":
				cfg.Messages.Type = val
			case "scope":
				cfg.Messages.Scope = val
			case "customScope":
				cfg.Messages.CustomScope = val
			case "subject":
				cfg.Messages.Subject = val
			case "body":
				cfg.Messages.Body = val
			case "breaking":
				cfg.Messages.Breaking = val
			case "footerPrefixesSelect":
				cfg.Messages.FooterPrefixes = val
			case "customFooterPrefix":
				cfg.Messages.CustomFooterPrefix = val
			case "footer":
				cfg.Messages.Footer = val
			case "confirmCommit":
				cfg.Messages.ConfirmCommit = val
			}
		case "cz.types":
			if !hasType {
				hasType = true
			}
			curType[key] = val
		case "cz.scopes":
			if !hasScope {
				hasScope = true
			}
			curScope[key] = val
		}
	}
	applyType()
	applyScope()
	if len(cfg.Types) == 0 {
		cfg.Types = czdata.DefaultConfig().Types
	}
	return scanner.Err()
}

func parseTomlStringValue(raw string) string {
	v := strings.TrimSpace(raw)
	if idx := strings.Index(v, "#"); idx >= 0 {
		prefix := strings.TrimSpace(v[:idx])
		if strings.Count(prefix, `"`)%2 == 0 {
			v = prefix
		}
	}
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`) && len(v) >= 2 {
		v = strings.Trim(v, `"`)
	}
	return strings.ReplaceAll(v, `\n`, "\n")
}

func parseTomlBool(raw string) (bool, bool) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

func parseTomlInt(raw string) (int, bool) {
	v := strings.TrimSpace(raw)
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

func DraftFromLLMWithSelector(cfg czdata.Config, selector func(czdata.Config, []czdata.Draft) (czdata.Draft, error)) (czdata.Draft, error) {
	for {
		diff, err := util.RunAndWaitForOuput("git", "diff", "--cached", "--")
		if err != nil {
			return czdata.Draft{}, err
		}
		files, err := util.RunAndWaitForOuput("git", "diff", "--cached", "--name-only")
		if err != nil {
			return czdata.Draft{}, err
		}
		hist, _ := util.RunAndWaitForOuput("git", "log", "--oneline", "-n", "5")

		if len(diff) > 12000 {
			diff = diff[:12000] + "\n... (truncated)"
		}
		typeList := make([]string, 0, len(cfg.Types))
		for _, t := range cfg.Types {
			typeList = append(typeList, t.Value)
		}

		prompt := fmt.Sprintf(`You are generating Conventional Commit messages.
Return strict JSON only, no prose.
Schema:
{"candidates":[{"type":"feat","scope":"optional","subject":"short","body":"optional","breaking":"optional","footer":"optional"}]}
Rules:
- candidate count: %d
- type must be one of: %s
- subject concise, imperative.
- output language follows changed code/comments language.

Changed files:
%s

Recent commits:
%s

Staged diff:
%s
`, cfg.Candidates, strings.Join(typeList, ", "), files, hist, diff)

		out, err := llm.RunLLM(prompt, cfg)
		if err != nil {
			return czdata.Draft{}, err
		}

		allowedIssueRefs := detectIssueRefs(diff + "\n" + files + "\n" + hist)

		cands, err := llm.ParseLLMCandidates(out)
		if err != nil {
			return czdata.Draft{}, fmt.Errorf("parse llm output: %w", err)
		}
		if len(cands) == 0 {
			return czdata.Draft{}, errors.New("llm returned no candidates")
		}
		if len(cands) == 1 {
			d := czdata.SanitizeDraft(cands[0], cfg)
			d.Footer = filterLLMIssueFooter(d.Footer, allowedIssueRefs)
			return d, nil
		}

		d, err := selector(cfg, cands)
		if errors.Is(err, ui.ErrRegenerateCandidates) {
			continue
		}
		if err != nil {
			return czdata.Draft{}, err
		}
		d = czdata.SanitizeDraft(d, cfg)
		d.Footer = filterLLMIssueFooter(d.Footer, allowedIssueRefs)
		return d, nil
	}
}

func selectDraftCandidateFromPrompt(cfg czdata.Config, cands []czdata.Draft) (czdata.Draft, error) {
	fmt.Println("\nLLM candidates:")
	for i, c := range cands {
		preview := czdata.BuildHeader(c)
		fmt.Printf("  %d) %s\n", i+1, preview)
	}
	for {
		in := strings.TrimSpace(ui.PromptLine("Select candidate # (or r=regen, q=cancel): "))
		switch strings.ToLower(in) {
		case "q", "quit", "n", "no":
			return czdata.Draft{}, errors.New("aborted")
		case "r", "regen":
			return czdata.Draft{}, ui.ErrRegenerateCandidates
		default:
			idx, convErr := strconv.Atoi(in)
			if convErr == nil && idx >= 1 && idx <= len(cands) {
				return cands[idx-1], nil
			}
			fmt.Fprintln(os.Stderr, "invalid selection")
		}
	}
}

func detectIssueRefs(text string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, m := range issueRefRe.FindAllString(text, -1) {
		set[strings.ToLower(strings.TrimSpace(m))] = struct{}{}
	}
	return set
}

func filterLLMIssueFooter(footer string, allowed map[string]struct{}) string {
	footer = strings.TrimSpace(footer)
	if footer == "" {
		return ""
	}

	refs := issueRefRe.FindAllString(footer, -1)
	if len(refs) == 0 {
		// No issue number in footer, keep as-is.
		return footer
	}
	for _, r := range refs {
		if _, ok := allowed[strings.ToLower(strings.TrimSpace(r))]; !ok {
			return ""
		}
	}
	return footer
}

func draftFromLastCommit() (czdata.Draft, error) {
	msg, err := util.RunAndWaitForOuput("git", "log", "-1", "--pretty=%B")
	if err != nil {
		return czdata.Draft{}, err
	}
	lines := strings.Split(strings.TrimRight(msg, "\n"), "\n")
	if len(lines) == 0 {
		return czdata.Draft{}, errors.New("empty last commit message")
	}
	header := strings.TrimSpace(lines[0])
	d := czdata.Draft{}
	if parsed, ok := llm.ParseConventionalHeader(header); ok {
		d.Type = parsed.Type
		d.Scope = parsed.Scope
		d.Subject = parsed.Subject
	} else {
		// fallback: put whole header into subject
		d.Subject = header
	}
	// join rest as body+footer and try to extract BREAKING and footer refs
	rest := strings.TrimSpace(strings.Join(lines[1:], "\n"))
	if rest == "" {
		return d, nil
	}
	// detect BREAKING CHANGE
	bIdx := strings.Index(strings.ToUpper(rest), "BREAKING CHANGE:")
	if bIdx >= 0 {
		before := strings.TrimSpace(rest[:bIdx])
		after := strings.TrimSpace(rest[bIdx+len("BREAKING CHANGE:"):])
		d.Body = strings.TrimSpace(before)
		d.Breaking = strings.TrimSpace(after)
	} else {
		d.Body = rest
	}
	// try extract footer refs lines starting with # or contains #digit
	footLines := []string{}
	bodyLines := []string{}
	for _, ln := range strings.Split(d.Body, "\n") {
		if issueRefRe.MatchString(ln) || strings.HasPrefix(strings.TrimSpace(strings.ToLower(ln)), "refs") || strings.HasPrefix(strings.TrimSpace(strings.ToLower(ln)), "closes") {
			footLines = append(footLines, strings.TrimSpace(ln))
		} else {
			bodyLines = append(bodyLines, ln)
		}
	}
	d.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))
	if len(footLines) > 0 {
		d.Footer = strings.TrimSpace(strings.Join(footLines, "\n"))
	}
	return d, nil
}

func buildEditorArgs(editor, file string) ([]string, error) {
	parts, err := splitCommandLine(editor)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, errors.New("invalid editor command")
	}
	hasFile := false
	for i, p := range parts {
		if strings.Contains(p, "{file}") {
			parts[i] = strings.ReplaceAll(p, "{file}", file)
			hasFile = true
		}
	}
	if !hasFile {
		parts = append(parts, file)
	}
	return parts, nil
}

func splitCommandLine(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	var args []string
	var cur strings.Builder
	var quote rune

	flush := func() {
		if cur.Len() == 0 {
			return
		}
		args = append(args, cur.String())
		cur.Reset()
	}

	for _, r := range s {
		switch {
		case quote == 0 && (r == '"' || r == '\''):
			quote = r
		case quote != 0 && r == quote:
			quote = 0
		case quote == 0 && (r == ' ' || r == '\t'):
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, errors.New("editor command has unmatched quote")
	}
	flush()
	return args, nil
}

func commitWithMessage(msg string) error {
	tmp, err := os.CreateTemp("", "aiw-cz-*.msg")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(ui.EnsureTrailingNewline(msg)); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return util.Run("git", "commit", "-F", tmp.Name())
}
