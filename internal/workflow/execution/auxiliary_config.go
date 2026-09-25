package execution

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"aiw/internal/workflow"
)

const auxiliaryAdapter = "aiw-auxiliary-http-v1:"

type auxiliaryHostConfig struct {
	Version int `json:"version"`
	Enabled bool `json:"enabled"`
	Endpoint string `json:"endpoint"`
	APIKeyEnv string `json:"api_key_env"`
	Tasks map[workflow.TaskID][]string `json:"tasks"`
	Capability workflow.ActorReference `json:"capability"`
}

// This record is supplied by the local capability review, never inferred from
// a profile or downloaded. The evidence must establish the UTF-8 byte upper
// bound and ALL provider-added tokens for this exact, tool-free protocol.
type auxiliaryCapabilityRecord struct {
	KnowledgeProtocol string `json:"knowledge_protocol,omitempty"`
	KnowledgeEnvelopeTokens int64 `json:"knowledge_envelope_tokens,omitempty"`
	KnowledgeOutputTokens int64 `json:"knowledge_output_tokens,omitempty"`
	KnowledgeEvidence string `json:"knowledge_evidence,omitempty"`
	Version int `json:"version"`
	Endpoint string `json:"endpoint"`
	Selection workflow.AISelection `json:"selection"`
	ContextTokens int64 `json:"context_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	CounterVersion string `json:"counter_version"`
	EnvelopeTokens int64 `json:"envelope_tokens"`
	Protocol string `json:"protocol"`
	ClosedInput bool `json:"closed_input"`
	EnforcedOutput bool `json:"enforced_output"`
	Evidence string `json:"evidence"`
	ReviewedBy string `json:"reviewed_by"`
}

func auxiliaryDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func readAuxiliaryJSON(path string, target any) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 64*1024+1))
	if err != nil { return nil, err }
	if len(data) > 64*1024 { return nil, errors.New("auxiliary configuration exceeds 64 KiB") }
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil { return nil, err }
	if err := decoder.Decode(new(any)); err != io.EOF { return nil, errors.New("auxiliary configuration contains trailing data") }
	return data, nil
}

func localAuxiliaryReference(root string, ref workflow.ActorReference, target any) error {
	if ref.Kind == "" || len(ref.SHA256) != 64 || filepath.IsAbs(ref.Path) { return errors.New("invalid local auxiliary evidence reference") }
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil { return err }
	path, err := filepath.EvalSymlinks(filepath.Join(canonical, filepath.FromSlash(ref.Path)))
	if err != nil { return err }
	rel, err := filepath.Rel(canonical, path)
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) { return errors.New("auxiliary evidence escapes the project runtime") }
	data, err := readAuxiliaryJSON(path, target)
	if err != nil { return err }
	if auxiliaryDigest(data) != ref.SHA256 { return errors.New("auxiliary evidence version changed") }
	return nil
}

func loadAuxiliaryConfig(store *workflow.Store) (auxiliaryHostConfig, auxiliaryCapabilityRecord, string, error) {
	var config auxiliaryHostConfig
	var record auxiliaryCapabilityRecord
	raw, err := readAuxiliaryJSON(filepath.Join(store.Root, "auxiliary-host.json"), &config)
	if err != nil { return config, record, "", fmt.Errorf("auxiliary configuration unavailable: %w", err) }
	identity := auxiliaryAdapter+auxiliaryDigest(raw)
	if config.Version != 1 || !config.Enabled || len(config.Tasks) == 0 { return config, record, identity, errors.New("auxiliary host disabled or lacks explicit Task authorization") }
	u, err := url.Parse(config.Endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" { return config, record, identity, errors.New("auxiliary endpoint requires an explicit HTTPS URL without credentials or query") }
	if config.APIKeyEnv == "" { return config, record, identity, errors.New("auxiliary credential environment variable is required") }
	if err := localAuxiliaryReference(store.Root, config.Capability, &record); err != nil { return config, record, identity, err }
	if record.Version != 1 || record.Endpoint != config.Endpoint || record.Protocol != "chat-completions-max-tokens-v1" || record.CounterVersion != "utf8-bytes-upper-bound-v1" || record.EnvelopeTokens < 0 || record.EnvelopeTokens > workflow.AuxiliaryInputTokens || record.ContextTokens <= 0 || record.OutputTokens <= 0 || !record.ClosedInput || !record.EnforcedOutput || record.Evidence == "" || record.ReviewedBy == "" || record.Selection.Provider == "" || record.Selection.Model == "" || record.Selection.Digest == "" {
		return config, record, identity, errors.New("capability-unavailable: reviewed closed-input, tokenizer-bound and enforced max_tokens evidence required")
	}
	return config, record, identity, nil
}

func capabilityFromRecord(config auxiliaryHostConfig, record auxiliaryCapabilityRecord) workflow.AuxiliaryCapability {
	return workflow.AuxiliaryCapability{Selection: record.Selection, Reference: config.Capability, CounterVersion: record.CounterVersion, ContextTokens: record.ContextTokens, OutputTokens: record.OutputTokens, EnforcedOutput: record.EnforcedOutput, ClosedInput: record.ClosedInput}
}

// ConfigureProductionAuxiliary is called by every production Store factory,
// including Session and completion paths. It has no I/O or activation effects.
func ConfigureProductionAuxiliary(store *workflow.Store) {
	store.AuxiliaryServices = &workflow.AuxiliaryServices{
		MeasureKnowledge: func(selection *workflow.AISelection, prompt string) (int64, int64, error) {
			_, record, _, err := loadAuxiliaryConfig(store)
			if err != nil { return 0, 0, err }
			if selection == nil || selection.Provider != record.Selection.Provider || selection.Model != record.Selection.Model || selection.Digest != record.Selection.Digest { return 0, 0, errors.New("receiving model lacks a matching reviewed token counter") }
			if record.KnowledgeProtocol != "receiving-fixed-turn-v1" || record.KnowledgeEvidence == "" || record.KnowledgeEnvelopeTokens < 0 || record.KnowledgeOutputTokens <= 0 || record.KnowledgeOutputTokens >= record.ContextTokens { return 0, 0, errors.New("receiving adapter lacks evidence for complete input overhead and output reservation") }
			// The local capability must cover this complete, closed input; no
			// guessed tokenizer or profile-alias capacity is used.
			return int64(len(prompt))+record.KnowledgeEnvelopeTokens, record.ContextTokens-record.KnowledgeOutputTokens, nil
		},
		FreeBytes: workflow.LocalAuxiliaryFreeBytes,
		Authorize: func(state workflow.RuntimeState, job workflow.AuxiliaryJob) error {
			config, _, _, err := loadAuxiliaryConfig(store)
			if err != nil { return err }
			if state.Protocol == nil || state.Protocol.Stop != nil || state.Task.ID != job.Owner { return errors.New("auxiliary source Task is unavailable or stopped") }
			for _, id := range []workflow.TaskID{job.Owner, job.Sponsor} {
				allowed := false
				for _, kind := range config.Tasks[id] { if kind == job.Kind { allowed = true } }
				if !allowed { return errors.New("auxiliary operation lacks explicit source and sponsor authorization") }
			}
			return nil
		},
		VerifyCapability: func(capability workflow.AuxiliaryCapability) error {
			config, record, _, err := loadAuxiliaryConfig(store)
			if err != nil { return err }
			if capability != capabilityFromRecord(config, record) { return errors.New("auxiliary capability no longer matches its reviewed version") }
			return nil
		},
		VerifyObservation: func(_ workflow.RuntimeState, call workflow.AuxiliaryReservation, observation workflow.AuxiliaryObservation) error {
			return verifyAuxiliaryHTTPObservation(store, call, observation)
		},
		VerifyInventory: store.VerifyLocalAuxiliaryInventory,
		ReportGap: func(id workflow.TaskID, cause error) {
			if err := store.RecordAuxiliaryHostGap(id, cause); err != nil { fmt.Fprintln(os.Stderr, "auxiliary gap could not be recorded:", err) }
		},
		StartHost: func(id workflow.TaskID) error {
			// Validate locally before creating a child; missing capability is a
			// visible local gap, never an ordinary LLM fallback.
			_, _, _, err := loadAuxiliaryConfig(store)
			if err != nil {
				// Existing calls must remain reconcilable after disabling/config
				// changes. The helper reads their original journal without sending.
				active, readErr := store.AuxiliaryActiveCall()
				state, stateErr := store.Load(id)
				notices := false
				if stateErr == nil { for _, n := range state.Notifications { if n.Managed != nil && (n.State == workflow.NotificationPending || n.State == "waiting" || n.State == workflow.NotificationFailed || n.State == workflow.NotificationDispatching) { notices = true; break } } }
				if !notices && (readErr != nil || active == nil) { return err }
			}
			executable, err := os.Executable()
			if err != nil { return err }
			launch, err := DetachedAuxiliaryLauncher(executable, nil)
			if err != nil { return err }
			return launch(store, id)
		},
	}
}
