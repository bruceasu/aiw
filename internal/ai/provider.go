package ai

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"encoding/hex"
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
	ReasoningIntensity string
	APIKey         string
	BaseURL        string
	Command        string
	ForceNewThread bool
	OutputSchema   map[string]any
	Timeout        time.Duration
	AdditionalDirs []string
	// Environment contains process-local environment entries for a CLI-backed
	// provider invocation. It is never persisted in Session state.
	Environment    []string
	ReadOnly       bool
	InvocationObserver InvocationObserver
}

// InvocationObserver persists one-shot CLI process and output evidence.
type InvocationObserver interface {
	ProcessKey() string
	Started(pid int) error
	Write([]byte) (int, error)
	Finished(exitCode int) error
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
	Usage       *UsageEnvelope `json:"usage,omitempty"`
}

const UsageEnvelopeVersion = 1

const (
	UsageRawEvidenceRetention = 90 * 24 * time.Hour
	UsageRawEvidenceMaxBytes  = 4 * 1024
)

type UsageFieldState string

const (
	UsageFieldKnown   UsageFieldState = "known"
	UsageFieldUnknown UsageFieldState = "unknown"
	UsageFieldInvalid UsageFieldState = "invalid"
)

// UsageField preserves availability separately from the value. Value is set
// only when State is UsageFieldKnown; unknown and invalid values are never
// represented as zero.
type UsageField[T any] struct {
	State UsageFieldState `json:"state"`
	Value *T              `json:"value,omitempty"`
}

// UsageEnvelope is the versioned, Provider-reported usage evidence for one
// call. Token totals are independent Provider fields and are not derived from
// input/output counts. Cost amount and currency are independent fields so
// partial Provider responses remain representable.
type UsageEnvelope struct {
	Version      int                      `json:"version"`
	Availability UsageFieldState          `json:"availability"`
	Provider     string                   `json:"provider,omitempty"`
	Model        string                   `json:"model,omitempty"`
	StartedAt    time.Time                `json:"started_at,omitempty"`
	CompletedAt  time.Time                `json:"completed_at,omitempty"`
	InputTokens  UsageField[int64]        `json:"input_tokens"`
	CachedInputTokens UsageField[int64]   `json:"cached_input_tokens,omitempty"`
	OutputTokens UsageField[int64]        `json:"output_tokens"`
	ReasoningOutputTokens UsageField[int64] `json:"reasoning_output_tokens,omitempty"`
	TotalTokens  UsageField[int64]        `json:"total_tokens"`
	CostAmount   UsageField[json.Number]  `json:"cost_amount"`
	CostCurrency UsageField[string]       `json:"cost_currency"`
	RawResponse  json.RawMessage          `json:"raw_response,omitempty"`
	RawResponseDigest string               `json:"raw_response_digest,omitempty"`
	RawResponseExpiresAt time.Time          `json:"raw_response_expires_at,omitempty"`
}

// PrepareUsageEvidence bounds the retained fragment and records its stable
// digest and expiry. Only usage-related fields are retained in RawResponse.
func PrepareUsageEvidence(usage *UsageEnvelope, now time.Time) {
	if usage == nil || len(usage.RawResponse) == 0 { return }
	if now.IsZero() { now = time.Now().UTC() }
	usage.RawResponse = sanitizeRawUsage(usage.RawResponse)
	if usage.RawResponseDigest == "" {
		digest := sha256.Sum256(usage.RawResponse)
		usage.RawResponseDigest = hex.EncodeToString(digest[:])
	}
	if len(usage.RawResponse) > UsageRawEvidenceMaxBytes { usage.RawResponse = nil }
	if usage.RawResponseExpiresAt.IsZero() {
		base := usage.CompletedAt
		if base.IsZero() { base = now }
		usage.RawResponseExpiresAt = base.Add(UsageRawEvidenceRetention).UTC()
	}
}

// ExpireUsageEvidence removes only the raw fragment once its retention
// boundary passes. Normalized values and the evidence digest remain intact.
func ExpireUsageEvidence(usage *UsageEnvelope, now time.Time) bool {
	if usage == nil || len(usage.RawResponse) == 0 { return false }
	original := append(json.RawMessage(nil), usage.RawResponse...)
	PrepareUsageEvidence(usage, now)
	if len(usage.RawResponse) == 0 { return true }
	if now.Before(usage.RawResponseExpiresAt) { return string(original) != string(usage.RawResponse) }
	usage.RawResponse = nil
	return true
}

// UsageWithoutRawEvidence returns normalized evidence suitable for the
// immutable accounting ledger, which retains the digest rather than fragment.
func UsageWithoutRawEvidence(usage *UsageEnvelope) *UsageEnvelope {
	if usage == nil { return nil }
	copy := *usage
	copy.RawResponse = nil
	return &copy
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
