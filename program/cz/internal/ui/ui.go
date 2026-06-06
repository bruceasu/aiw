package ui

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"aiw-cz/internal/cz"
)

var ErrRegenerateCandidates = errors.New("regenerate llm candidates")

type LineUI struct {
	PromptLineFn            func(string) string
	PromptMultilineEditorFn func(string, cz.Config, string) (string, error)
}

func (ui LineUI) DraftFromWizard(cfg cz.Config) (cz.Draft, error) {
	d := cz.Draft{}
	t, err := PromptType(cfg, ui.PromptLineFn)
	if err != nil {
		return d, err
	}
	d.Type = t

	scope := strings.TrimSpace(ui.PromptLineFn(cfg.Messages.Scope + " "))
	if scope == "." || strings.EqualFold(scope, "custom") {
		scope = strings.TrimSpace(ui.PromptLineFn(cfg.Messages.CustomScope + " "))
	}
	d.Scope = scope

	for {
		d.Subject = strings.TrimSpace(ui.PromptLineFn(BuildSubjectPrompt(cfg, d.Subject)))
		if d.Subject != "" {
			break
		}
		fmt.Fprintln(os.Stderr, "subject is required")
	}

	if d.Body, err = ui.PromptMultilineEditorFn(cfg.Messages.Body, cfg, ""); err != nil {
		return d, err
	}
	if d.Breaking, err = ui.PromptMultilineEditorFn(cfg.Messages.Breaking, cfg, ""); err != nil {
		return d, err
	}

	prefix := PromptFooterPrefix(cfg, ui.PromptLineFn)
	footer, err := ui.PromptMultilineEditorFn(cfg.Messages.Footer, cfg, "")
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

func (ui LineUI) DraftFromLLM(cfg cz.Config) (cz.Draft, error) {
	return cz.Draft{}, errors.New("DraftFromLLM not implemented for LineUI")
}

func (ui LineUI) ReviewAndCommit(draft cz.Draft, cfg cz.Config, commitFn func(string) error) error {
	for {
		msg := cz.BuildCommitMessage(draft)
		fmt.Printf("\n--- Commit message preview ---\n%s\n--------------------------------\n", msg)
		in := strings.ToLower(strings.TrimSpace(ui.PromptLineFn(cfg.Messages.ConfirmCommit + " [y(commit), e(edit), n(cancel)] ")))
		switch in {
		case "y", "yes", "":
			return commitFn(msg)
		case "e", "edit":
			draft = ui.editDraftWithLineUI(draft, cfg)
		case "n", "no":
			fmt.Fprintln(os.Stderr, "aborted")
			return nil
		default:
			fmt.Fprintln(os.Stderr, "invalid input")
		}
	}
}

func (ui LineUI) editDraftWithLineUI(d cz.Draft, cfg cz.Config) cz.Draft {
	fmt.Println("Editing fields (leave empty to keep current):")
	if t, err := PromptType(cfg, ui.PromptLineFn); err == nil && t != "" {
		d.Type = t
	}
	scope := strings.TrimSpace(ui.PromptLineFn(fmt.Sprintf("Scope [%s]: ", d.Scope)))
	if scope != "" {
		if scope == "." || strings.EqualFold(scope, "custom") {
			scope = strings.TrimSpace(ui.PromptLineFn(cfg.Messages.CustomScope + " "))
		}
		d.Scope = scope
	}
	subject := strings.TrimSpace(ui.PromptLineFn(BuildSubjectPrompt(cfg, d.Subject)))
	if subject != "" {
		d.Subject = subject
	}
	if s, err := ui.PromptMultilineEditorFn("Body: ", cfg, d.Body); err == nil {
		d.Body = cz.NormalizeMultiline(s)
	}
	if s, err := ui.PromptMultilineEditorFn("Breaking: ", cfg, d.Breaking); err == nil {
		d.Breaking = cz.NormalizeMultiline(s)
	}
	if in := strings.TrimSpace(ui.PromptLineFn("Edit footer? [y/N] ")); strings.EqualFold(in, "y") {
		prefix := PromptFooterPrefix(cfg, ui.PromptLineFn)
		if s, err := ui.PromptMultilineEditorFn("Footer: ", cfg, d.Footer); err == nil {
			footer := strings.TrimSpace(s)
			if footer != "" && prefix != "" && !strings.HasPrefix(strings.ToLower(footer), strings.ToLower(prefix)) {
				footer = prefix + " " + footer
			}
			d.Footer = cz.NormalizeMultiline(footer)
		}
	}
	return cz.SanitizeDraft(d, cfg)
}

func PromptType(cfg cz.Config, promptFn func(string) string) (string, error) {
	fmt.Println(cfg.Messages.Type)
	for i, t := range cfg.Types {
		fmt.Printf("  %2d) %s\n", i+1, t.Name)
	}
	for {
		in := strings.TrimSpace(promptFn("Type # or value: "))
		if in == "" {
			return "", errors.New("type is required")
		}
		if idx, err := strconv.Atoi(in); err == nil {
			if idx >= 1 && idx <= len(cfg.Types) {
				return cfg.Types[idx-1].Value, nil
			}
		}
		for _, t := range cfg.Types {
			if t.Value == in {
				return in, nil
			}
		}
		fmt.Fprintln(os.Stderr, "invalid type, try again")
	}
}

func PromptLine(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimRight(line, "\r\n")
}

func PromptMultilineWithEditor(prompt string, cfg cz.Config, current string) (string, error) {
	in := strings.TrimSpace(PromptLine(prompt + "(输入 /edit 或 Ctrl+E 文本快捷触发编辑器) "))
	if !IsEditorShortcut(in) {
		return cz.NormalizeMultiline(in), nil
	}
	edited, err := EditInExternalEditor(cfg, current)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(edited), nil
}

func IsEditorShortcut(s string) bool {
	v := strings.ToLower(strings.TrimSpace(s))
	switch v {
	case "/edit", "^e", "ctrl+e", "control+e":
		return true
	}
	return strings.ContainsRune(s, rune(5))
}

func EditInExternalEditor(cfg cz.Config, initial string) (string, error) {
	editor := ResolveEditor(cfg)
	if strings.TrimSpace(editor) == "" {
		return "", errors.New("no editor available")
	}
	tmpDir, err := os.MkdirTemp("", "aiw-cz-edit-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	tmpPath := filepath.Join(tmpDir, "message.txt")
	tmp, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return "", err
	}

	seed := EnsureTrailingNewline(initial)
	if initial == "" {
		seed = "# Write your text below. Lines starting with # are ignored.\n"
	}
	if _, err := tmp.WriteString(seed); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}

	if cz.DryRun {
		fmt.Fprintf(os.Stderr, "+ %s %s\n", editor, tmpPath)
		fmt.Fprintln(os.Stderr, "  (dry-run: skipped editor launch)")
		return initial, nil
	}

	if err := RunEditorCommand(editor, tmpPath); err != nil {
		return "", err
	}
	b, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ResolveEditor(cfg cz.Config) string {
	if cfg.Editor != "" {
		return cfg.Editor
	}
	for _, key := range []string{"GIT_EDITOR", "VISUAL", "EDITOR"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	if out, err := GitOutput("git", "config", "--get", "core.editor"); err == nil && strings.TrimSpace(out) != "" {
		return strings.TrimSpace(out)
	}
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}

func RunEditorCommand(editor, path string) error {
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return errors.New("empty editor command")
	}
	args := append(parts[1:], path)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func EnsureTrailingNewline(s string) string {
	if s == "" || strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

func GitOutput(args ...string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func IsInteractiveTerminal() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func BuildSubjectPrompt(cfg cz.Config, current string) string {
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

func PromptFooterPrefix(cfg cz.Config, promptFn func(string) string) string {
	fmt.Println(cfg.Messages.FooterPrefixes)
	options := []string{"#", "refs", "closes", "custom", "skip"}
	for i, o := range options {
		fmt.Printf("  %2d) %s\n", i+1, o)
	}
	for {
		in := strings.TrimSpace(promptFn("Choose prefix # or value: "))
		if in == "" {
			return ""
		}
		if idx, err := strconv.Atoi(in); err == nil {
			if idx >= 1 && idx <= len(options) {
				sel := options[idx-1]
				if sel == "custom" {
					return strings.TrimSpace(promptFn(cfg.Messages.CustomFooterPrefix + " "))
				}
				if sel == "skip" {
					return ""
				}
				return sel
			}
		}
		for _, o := range options {
			if strings.EqualFold(o, in) || o == in {
				if o == "custom" {
					return strings.TrimSpace(promptFn(cfg.Messages.CustomFooterPrefix + " "))
				}
				if o == "skip" {
					return ""
				}
				return o
			}
		}
		fmt.Fprintln(os.Stderr, "invalid selection; try again")
	}
}

func NewDefaultUI(interactive bool, promptLineFn func(string) string, promptMultilineFn func(string, cz.Config, string) (string, error), teaUIFactory func() cz.UI) cz.UI {
	if !interactive {
		return LineUI{PromptLineFn: promptLineFn, PromptMultilineEditorFn: promptMultilineFn}
	}
	return teaUIFactory()
}
