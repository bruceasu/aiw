package cz

import "strings"

type Messages struct {
	Type, Scope, CustomScope, Subject, Body, Breaking, FooterPrefixes, CustomFooterPrefix, Footer, ConfirmCommit string
	SelectType, SelectPrefix, SubjectRequired, SubjectTooLong, InvalidType, InvalidSelection string
	Preview, Action, Candidate, Regenerate, Commit, Edit, Cancel, Aborted, Editing, DoneEditing string
	EmptyScope, CustomScopeOption, FooterEdit, EditorHint string
}
type Type struct { Value string; Name string }
type Scope struct { Value string; Name string }
type Config struct {
	Language string
	DefaultLanguage string
	Locales map[string]LocaleOverride
	UseLLM bool; Candidates int; Editor string; LLMProvider string; LLMModel string
	OpenAIModel, OpenAIBaseURL, OpenAIKey string
	GeminiModel, GeminiBaseURL, GeminiKey string
	OllamaModel, OllamaBaseURL, CodexCommand, CopilotCommand string
	APIBaseURL, APIKey string; DebugSource, EnableMultipleScopes bool
	ScopeEnumSeparator string; MaxSubjectLength int
	Messages Messages; Types []Type; Scopes []Scope
}
type LocaleOverride struct {
	Messages map[string]string
	Types []Type
	Scopes []Scope
	HasTypes bool
	HasScopes bool
}
type Draft struct { Type string `json:"type"`; Scope string `json:"scope"`; Subject string `json:"subject"`; Body string `json:"body"`; Breaking string `json:"breaking"`; Footer string `json:"footer"` }
type LLMResponse struct { Candidates []Draft `json:"candidates"` }
var DryRun bool
type UI interface { DraftFromWizard(Config) (Draft,error); DraftFromLLM(Config) (Draft,error); ReviewAndCommit(Draft, Config, func(string) error) error }

func DefaultConfig() Config {
	cfg := Config{UseLLM:false, Candidates:3, ScopeEnumSeparator:",", MaxSubjectLength:72, Scopes:[]Scope{{"core","core"},{"api","api"},{"docs","docs"},{"ci","ci"},{"tests","tests"}}}
	ApplyLocale(&cfg, "en")
	return cfg
}
func SanitizeDraft(d Draft, cfg Config) Draft {
	d.Type=SanitizeCommitText(strings.TrimSpace(d.Type)); d.Scope=SanitizeCommitText(strings.TrimSpace(d.Scope)); d.Subject=SanitizeCommitText(strings.TrimSpace(d.Subject)); d.Body=SanitizeCommitText(strings.TrimSpace(d.Body)); d.Breaking=SanitizeCommitText(strings.TrimSpace(d.Breaking)); d.Footer=SanitizeCommitText(strings.TrimSpace(d.Footer))
	if d.Type=="" { d.Type=cfg.Types[0].Value }; if d.Subject=="" { d.Subject="update" }
	for _,t:=range cfg.Types { if t.Value==d.Type { return d } }; d.Type="chore"; return d
}
func SanitizeCommitText(s string) string { return strings.ReplaceAll(s,"\x00","") }
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
func BuildCommitMessage(d Draft) string { d=Draft{SanitizeCommitText(d.Type),SanitizeCommitText(d.Scope),SanitizeCommitText(d.Subject),SanitizeCommitText(d.Body),SanitizeCommitText(d.Breaking),SanitizeCommitText(d.Footer)}; parts:=[]string{BuildHeader(d)}; if d.Body!="" {parts=append(parts,"",d.Body)}; if d.Breaking!="" {parts=append(parts,"","BREAKING CHANGE: "+d.Breaking)}; if d.Footer!="" {parts=append(parts,"",d.Footer)}; return strings.Join(parts,"\n") }
func BuildHeader(d Draft) string { header:=d.Type; if d.Scope!="" {header+="("+d.Scope+")"}; return header+": "+d.Subject }
