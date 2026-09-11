package ask

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aiw/internal/ai"

	"github.com/manifoldco/promptui"
)

type options struct {
	chat, resume                   bool
	systemPrompt, systemPromptFile string
	allowPaths                     []string
	prompt                         string
}

type Answer struct {
	SchemaVersion     string           `json:"schema_version"`
	Status            string           `json:"status"`
	Summary           string           `json:"summary"`
	Interpretation    string           `json:"interpretation,omitempty"`
	Steps             []map[string]any `json:"steps,omitempty"`
	Alternatives      []string         `json:"alternatives,omitempty"`
	Warnings          []string         `json:"warnings,omitempty"`
	References        []string         `json:"references,omitempty"`
	FollowUpQuestions []string         `json:"follow_up_questions,omitempty"`
	Capability        map[string]any   `json:"capability"`
	Safety            map[string]any   `json:"safety"`
}

func Dispatch(args []string) error {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			fmt.Println("Usage: aiw ask [--chat|--resume] [--system-prompt TEXT] [--system-prompt-file FILE] [--allow-path PATH] \"PROMPT\"")
			return nil
		}
	}
	opts, err := parse(args)
	if err != nil {
		return err
	}
	if opts.chat {
		return chat(opts)
	}
	if opts.resume {
		return resumeChat(opts)
	}
	if strings.TrimSpace(opts.prompt) == "" {
		return errors.New("usage: aiw ask [--chat|--resume] \"<prompt>\"")
	}
	return turn(opts, opts.prompt, "")
}

func parse(args []string) (options, error) {
	var o options
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--chat":
			o.chat = true
		case "--resume":
			o.resume = true
		case "--system-prompt", "--system-prompt-file":
			if i+1 >= len(args) {
				return o, fmt.Errorf("missing value for %s", args[i])
			}
			i++
			if args[i-1] == "--system-prompt" {
				o.systemPrompt = args[i]
			} else {
				o.systemPromptFile = args[i]
			}
		case "--allow-path":
			if i+1 >= len(args) {
				return o, errors.New("missing value for --allow-path")
			}
			i++
			o.allowPaths = append(o.allowPaths, args[i])
		case "-h", "--help":
			return o, nil
		default:
			if strings.HasPrefix(args[i], "-") {
				return o, fmt.Errorf("unknown ask option: %s", args[i])
			}
			if o.prompt != "" {
				o.prompt += " "
			}
			o.prompt += args[i]
		}
	}
	return o, nil
}

func chat(o options) error {
	policy, err := newReadPolicy(o)
	if err != nil {
		return err
	}
	if o.resume {
		fmt.Println("进入 Chat 模式（/send 发送，/exit 退出；无可恢复会话时将新建）")
	} else {
		fmt.Println("进入 Chat 模式（/send 发送，/exit 退出）")
	}

	var lines []string
	for {
		prompt := promptui.Prompt{Label: "ask"}
		line, err := prompt.Run()
		if err != nil {
			if errors.Is(err, promptui.ErrInterrupt) {
				fmt.Println("已取消当前输入")
				continue
			}
			return nil
		}

		switch line {
		case "/exit":
			return nil
		case "/send":
			if len(lines) == 0 {
				continue
			}
			question := strings.Join(lines, "\n")
			lines = nil
			if _, err := turnWithPolicy(o, question, "", policy, true); err != nil {
				fmt.Println("提示：", err)
			}
		default:
			// Do not trim: an empty line is part of a multiline question.
			lines = append(lines, line)
		}
	}
}

func resumeChat(o options) error {
	policy, err := newReadPolicy(o)
	if err != nil {
		return err
	}
	session, err := latestSession()
	if err != nil {
		return err
	}
	if session == "" {
		fmt.Println("没有可恢复的会话，已开始新的 Chat 会话。")
	}
	fmt.Println("进入 Chat 模式（/send 发送，/exit 退出）")

	var lines []string
	for {
		prompt := promptui.Prompt{Label: "ask"}
		line, err := prompt.Run()
		if err != nil {
			if errors.Is(err, promptui.ErrInterrupt) {
				fmt.Println("已取消当前输入。")
				continue
			}
			return nil
		}

		switch line {
		case "/exit":
			return nil
		case "/send":
			if len(lines) == 0 {
				continue
			}
			question := strings.Join(lines, "\n")
			lines = nil
			session, err = turnWithPolicy(o, question, session, policy, true)
			if err != nil {
				fmt.Println("请求失败：", err)
			}
		default:
			lines = append(lines, line)
		}
	}
}

func turn(o options, prompt, session string) error {
	policy, err := newReadPolicy(o)
	if err != nil {
		return err
	}
	_, err = turnWithPolicy(o, prompt, session, policy, false)
	return err
}

func turnWithSession(o options, prompt, session string) (string, error) {
	policy, err := newReadPolicy(o)
	if err != nil {
		return session, err
	}
	return turnWithPolicy(o, prompt, session, policy, false)
}

func turnWithPolicy(o options, prompt, session string, policy *readPolicy, interactive bool) (string, error) {
	system, err := resolveSystemPrompt(o, policy, interactive)
	if err != nil {
		return session, err
	}
	cfg, err := ai.LoadConfig()
	if err != nil {
		return session, err
	}
	if isCodexProvider(cfg.Name) && len(policy.allowedDirs()) > 0 {
		fmt.Fprintln(os.Stderr, "aiw ask: Codex --add-dir grants an additional directory but is not a strict read allowlist; its sandbox behavior depends on the installed Codex version.")
	}
	out, err := ai.RunLLMWithSystemPrompt(prompt, ai.LLMConfig{Provider: cfg.Name, Model: cfg.Model, APIBaseURL: cfg.BaseURL, APIKey: cfg.APIKey, CodexCommand: cfg.CodexCommand, CopilotCommand: cfg.CopilotCommand, Workspace: policy.workspace, AdditionalDirs: policy.allowedDirs(), ReadOnly: true}, answerSchema(), system)
	if err != nil {
		return persist(prompt, "error", err.Error(), "", session), err
	}
	var a Answer
	if err := json.Unmarshal([]byte(out), &a); err != nil {
		message := invalidStructuredResponseError("invalid JSON", err, out)
		return persist(prompt, "error", message, "", session), errors.New(message)
	}
	if err := validateAnswer(a); err != nil {
		message := invalidStructuredResponseError("invalid answer", err, out)
		return persist(prompt, "error", message, "", session), errors.New(message)
	}
	b, _ := json.MarshalIndent(a, "", "  ")
	fmt.Println(a.Summary)
	if len(a.Steps) > 0 {
		for i, s := range a.Steps {
			fmt.Printf("%d. %v\n", i+1, s["title"])
		}
	}
	return persist(prompt, a.Status, "", string(b), session), nil
}

func isCodexProvider(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "codex", "codex-cli", "codex_cli", "codexcli":
		return true
	default:
		return false
	}
}

func answerSchema() map[string]any {
	return map[string]any{"type": "object", "additionalProperties": false, "required": []string{"schema_version", "status", "summary", "capability", "safety"}, "properties": map[string]any{"schema_version": map[string]any{"type": "string", "const": "1.0"}, "status": map[string]any{"type": "string", "enum": []string{"ok", "needs_clarification", "unsupported", "error", "interrupted"}}, "summary": map[string]any{"type": "string"}, "capability": map[string]any{"type": "object", "additionalProperties": false, "required": []string{}, "properties": map[string]any{}}, "safety": map[string]any{"type": "object", "additionalProperties": false, "required": []string{}, "properties": map[string]any{}}}}
}

func invalidStructuredResponseError(kind string, err error, output string) string {
	const maxOutputLength = 4096
	output = strings.TrimSpace(output)
	if len(output) > maxOutputLength {
		output = output[:maxOutputLength] + "..."
	}
	if output == "" {
		return fmt.Sprintf("LLM returned %s: %v", kind, err)
	}
	return fmt.Sprintf("LLM returned %s: %v; response: %s", kind, err, output)
}

func validateAnswer(a Answer) error {
	if a.SchemaVersion != "1.0" {
		return errors.New("unsupported schema version")
	}
	switch a.Status {
	case "ok", "needs_clarification", "unsupported", "error", "interrupted":
	default:
		return fmt.Errorf("invalid answer status %q", a.Status)
	}
	if a.Capability == nil || a.Safety == nil {
		return errors.New("missing required answer object")
	}
	return nil
}

func persist(q, status, errMsg, data, session string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return session
	}
	now := time.Now().UTC()
	path := session
	if path == "" {
		sum := sha256.Sum256([]byte(q))
		dir := filepath.Join(home, ".aiw", "ask", now.Format("2006-01-02"))
		if err := os.MkdirAll(dir, 0700); err != nil {
			return session
		}
		path = filepath.Join(dir, now.Format("20060102T150405Z")+"-"+hex.EncodeToString(sum[:])+".md")
	}

	_, statErr := os.Stat(path)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return session
	}
	defer f.Close()
	if os.IsNotExist(statErr) {
		fmt.Fprint(f, "# session\n")
	}
	fmt.Fprintf(f, "\n# %s\n## question\n%s\n", now.Format(time.RFC3339), q)
	if status == "ok" || status == "needs_clarification" || status == "unsupported" {
		fmt.Fprintf(f, "\n## answer\n%s\n\n## data\n```json\n%s\n```\n", data, data)
	} else {
		fmt.Fprintf(f, "\n## status\n%s\n## error\n%s\n", status, errMsg)
	}
	return path
}

func latestSession() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory for ask sessions: %w", err)
	}
	root := filepath.Join(home, ".aiw", "ask")
	var latest string
	var latestMod time.Time
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if latest == "" || info.ModTime().After(latestMod) || (info.ModTime().Equal(latestMod) && path > latest) {
			latest, latestMod = path, info.ModTime()
		}
		return nil
	})
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("find latest ask session: %w", err)
	}
	return latest, nil
}
