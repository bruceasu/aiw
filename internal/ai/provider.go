package ai

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Request is the provider-neutral input for one model turn.
type Request struct {
	SessionID      string
	Prompt         string
	SystemPrompt   string
	Instructions   string
	Memory         string
	Phase          string
	TurnNumber     int
	OutputDir      string
	Workspace      string
	ThreadID       string
	Model          string
	APIKey         string
	BaseURL        string
	Command        string
	ForceNewThread bool
	OutputSchema   map[string]any
	Timeout        time.Duration
	AdditionalDirs []string
	ReadOnly       bool
}

type Response struct {
	ThreadID    string
	FinalOutput string
	Events      []byte
	Stderr      []byte
	ExitCode    int
	Metadata    map[string]string
	StartedAt   time.Time
	CompletedAt time.Time
}

type Provider interface {
	Name() string
	Generate(context.Context, Request) (Response, error)
	Interactive(context.Context, Request) (Response, error)
}

func unsupportedInteractiveProvider(name string) (Response, error) {
	return Response{}, fmt.Errorf("AI provider %s does not support interactive execution", name)
}

type Config struct {
	Name           string
	Model          string
	APIKey         string
	BaseURL        string
	Command        string
	CodexCommand   string
	CopilotCommand string
	HTTPClient     *http.Client
}

// ResolveConfig applies the documented execution precedence without changing
// persisted Session state. Callers pass the stored Session values separately
// from one-call command-line overrides.
func ResolveConfig(sessionName, sessionModel, providerOverride, modelOverride string) (Config, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return Config{}, err
	}
	if strings.TrimSpace(sessionName) != "" {
		cfg.Name = normalize(sessionName)
	}
	if strings.TrimSpace(sessionModel) != "" {
		cfg.Model = sessionModel
	}
	if strings.TrimSpace(providerOverride) != "" {
		cfg.Name = normalize(providerOverride)
	}
	if strings.TrimSpace(modelOverride) != "" {
		cfg.Model = modelOverride
	}
	cfg.Command = commandForProvider(cfg)
	if cfg.Name != "" && cfg.Name != "auto" {
		applyProviderDefaults(&cfg)
	}
	return cfg, nil
}

func commandForProvider(cfg Config) string {
	switch normalize(cfg.Name) {
	case "codex", "codex-cli", "codex_cli", "codexcli":
		return cfg.CodexCommand
	case "copilot", "copilot-cli", "copilot_cli", "copilotcli":
		return cfg.CopilotCommand
	default:
		return cfg.Command
	}
}

func ConfigFromEnv(name, model string) Config {
	cfg := Config{Name: name, Model: model, APIKey: os.Getenv("OPENAI_API_KEY"), BaseURL: os.Getenv("OPENAI_BASE_URL")}
	switch normalize(name) {
	case "gemini":
		cfg.APIKey = os.Getenv("GEMINI_API_KEY")
		cfg.BaseURL = os.Getenv("GEMINI_BASE_URL")
	case "ollama":
		cfg.APIKey = os.Getenv("OLLAMA_API_KEY")
		cfg.BaseURL = os.Getenv("OLLAMA_BASE_URL")
	case "llama.cpp", "llamacpp":
		cfg.APIKey = os.Getenv("LLAMACPP_API_KEY")
		cfg.BaseURL = os.Getenv("LLAMACPP_BASE_URL")
	}
	return cfg
}

func NewProvider(cfg Config) (Provider, error) {
	switch normalize(cfg.Name) {
	case "codex", "codex-cli", "codex_cli", "codexcli", "exec":
		return NewCLIProvider("codex", cfg), nil
	case "copilot", "copilot-cli", "copilot_cli", "copilotcli":
		return NewCLIProvider("copilot", cfg), nil
	case "openai":
		return NewOpenAIResponsesProvider(cfg), nil
	case "openai-compatible", "ollama", "llama.cpp", "llamacpp":
		return NewOpenAICompatibleProvider(normalize(cfg.Name), cfg), nil
	case "gemini":
		return NewGeminiProvider(cfg), nil
	default:
		return nil, unknownProvider(cfg.Name)
	}
}

func normalize(name string) string {
	if name == "" {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(name))
}

func unknownProvider(name string) error {
	return fmt.Errorf("unknown AI provider %q", name)
}
