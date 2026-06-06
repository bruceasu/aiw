package cz

import (
	"strings"
)

type Messages struct {
	Type               string
	Scope              string
	CustomScope        string
	Subject            string
	Body               string
	Breaking           string
	FooterPrefixes     string
	CustomFooterPrefix string
	Footer             string
	ConfirmCommit      string
}

type Type struct {
	Value string
	Name  string
}

type Scope struct {
	Value string
	Name  string
}

type Config struct {
	UseLLM               bool
	Candidates           int
	Editor               string
	LLMModel             string
	APIBaseURL           string
	APIKey               string
	DebugSource          bool
	EnableMultipleScopes bool
	ScopeEnumSeparator   string
	MaxSubjectLength     int
	Messages             Messages
	Types                []Type
	Scopes               []Scope
}

type Draft struct {
	Type     string `json:"type"`
	Scope    string `json:"scope"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	Breaking string `json:"breaking"`
	Footer   string `json:"footer"`
}

type LLMResponse struct {
	Candidates []Draft `json:"candidates"`
}

var DryRun bool

type UI interface {
	DraftFromWizard(cfg Config) (Draft, error)
	DraftFromLLM(cfg Config) (Draft, error)
	ReviewAndCommit(draft Draft, cfg Config, commitFn func(string) error) error
}

func DefaultConfig() Config {
	return Config{
		UseLLM:     false,
		Candidates: 3,
		Messages: Messages{
			Type:               "选择你要提交的类型 / Select commit type:",
			Scope:              "选择一个提交范围（可选）/ Scope (optional):",
			CustomScope:        "请输入自定义的提交范围 / Input custom scope:",
			Subject:            "填写简短精炼的变更描述 / Subject:\n",
			Body:               "填写更加详细的变更描述（可选） / Body, use | for new line:\n",
			Breaking:           "列举非兼容性重大的变更（可选） / Breaking changes, use | for new line:\n",
			FooterPrefixes:     "选择关联 issue 前缀（可选） / Select issue prefix (optional):",
			CustomFooterPrefix: "输入自定义 issue 前缀 / Input custom issue prefix:",
			Footer:             "列举关联 issue（可选），例如: #31, #I3244:\n",
			ConfirmCommit:      "是否提交或修改 commit? / Commit or modify?",
		},
		ScopeEnumSeparator: ",",
		MaxSubjectLength:   72,
		Types: []Type{
			{Value: "feat", Name: "feat:     新增功能 | A new feature"},
			{Value: "fix", Name: "fix:      修复缺陷 | A bug fix"},
			{Value: "docs", Name: "docs:     文档更新 | Documentation only changes"},
			{Value: "style", Name: "style:    代码格式 | Changes that do not affect the meaning of the code"},
			{Value: "refactor", Name: "refactor: 代码重构 | A code change that neither fixes a bug nor adds a feature"},
			{Value: "perf", Name: "perf:     性能提升 | A code change that improves performance"},
			{Value: "test", Name: "test:     测试相关 | Adding missing tests or correcting existing tests"},
			{Value: "build", Name: "build:    构建相关 | Changes that affect the build system or external dependencies"},
			{Value: "ci", Name: "ci:       持续集成 | Changes to our CI configuration files and scripts"},
			{Value: "revert", Name: "revert:   回退代码 | Revert to a commit"},
			{Value: "chore", Name: "chore:    其他修改 | Other changes that do not modify src or test files"},
		},
		Scopes: []Scope{
			{Value: "core", Name: "core"},
			{Value: "api", Name: "api"},
			{Value: "docs", Name: "docs"},
			{Value: "ci", Name: "ci"},
			{Value: "tests", Name: "tests"},
		},
	}
}

func SanitizeDraft(d Draft, cfg Config) Draft {
	d.Type = SanitizeCommitText(strings.TrimSpace(d.Type))
	d.Scope = SanitizeCommitText(strings.TrimSpace(d.Scope))
	d.Subject = SanitizeCommitText(strings.TrimSpace(d.Subject))
	d.Body = SanitizeCommitText(strings.TrimSpace(d.Body))
	d.Breaking = SanitizeCommitText(strings.TrimSpace(d.Breaking))
	d.Footer = SanitizeCommitText(strings.TrimSpace(d.Footer))
	if d.Type == "" {
		d.Type = cfg.Types[0].Value
	}
	if d.Subject == "" {
		d.Subject = "update"
	}
	valid := false
	for _, t := range cfg.Types {
		if t.Value == d.Type {
			valid = true
			break
		}
	}
	if !valid {
		d.Type = "chore"
	}
	return d
}

func SanitizeCommitText(s string) string {
	return strings.ReplaceAll(s, "\x00", "")
}

func BuildCommitMessage(d Draft) string {
	d = Draft{
		Type:     SanitizeCommitText(d.Type),
		Scope:    SanitizeCommitText(d.Scope),
		Subject:  SanitizeCommitText(d.Subject),
		Body:     SanitizeCommitText(d.Body),
		Breaking: SanitizeCommitText(d.Breaking),
		Footer:   SanitizeCommitText(d.Footer),
	}
	header := BuildHeader(d)
	parts := []string{header}
	if d.Body != "" {
		parts = append(parts, "", d.Body)
	}
	if d.Breaking != "" {
		parts = append(parts, "", "BREAKING CHANGE: "+d.Breaking)
	}
	if d.Footer != "" {
		parts = append(parts, "", d.Footer)
	}
	return strings.Join(parts, "\n")
}

func BuildHeader(d Draft) string {
	header := d.Type
	if d.Scope != "" {
		header += "(" + d.Scope + ")"
	}
	header += ": " + d.Subject
	return header
}

func NormalizeMultiline(s string) string {
	if s == "" {
		return s
	}
	parts := strings.Split(s, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return strings.Join(parts, "\n")
}
