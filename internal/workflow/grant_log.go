package workflow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const grantOpen = "<!-- aiw-grants:v1 -->"
const grantClose = "<!-- /aiw-grants:v1 -->"

type GrantScope struct {
	WorkspaceDigest string `json:"workspace_digest"`
	Paths []string `json:"paths"`
	Actions []string `json:"actions"`
	Targets []string `json:"targets"`
}

type GrantVersion struct {
	ActorReference
	SchemaVersion int `json:"schema_version"`
	Version string `json:"version"`
}

type GrantEntry struct {
	ID string `json:"id"`
	Sequence int `json:"sequence"`
	Date string `json:"date"`
	Permission string `json:"permission"`
	Approved bool `json:"approved"`
	Reason string `json:"reason"`
	AuthorizerType string `json:"authorizer_type"`
	AuthorizerID string `json:"authorizer_id"`
	ApplicantStage string `json:"applicant_stage"`
	DecisionReference ActorReference `json:"decision_reference"`
	PlanReference GrantVersion `json:"plan_reference"`
	PolicyReference GrantVersion `json:"policy_reference"`
	Scope GrantScope `json:"scope"`
	Supersedes []string `json:"supersedes"`
	PreviousDigest string `json:"previous_digest"`
	Digest string `json:"digest,omitempty"`
}

type GrantLog struct {
	SchemaVersion int `json:"schema_version"`
	TaskID TaskID `json:"task_id"`
	Revision int `json:"revision"`
	Entries []GrantEntry `json:"entries"`
}

type GrantAnchor struct {
	Revision int `json:"revision"`
	Digest string `json:"digest"`
}

// strictJSON rejects ambiguity before decoding into the versioned schema.
// In particular, encoding/json alone accepts duplicate keys and null scalars.
func strictJSON(data []byte, target any) error {
	if !utf8.Valid(data) { return errors.New("JSON is not UTF-8") }
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var walk func() error
	walk = func() error {
		t, err := d.Token()
		if err != nil { return err }
		if t == nil { return errors.New("null is not a contract value") }
		switch v := t.(type) {
		case json.Delim:
			switch v {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					key, err := d.Token()
					if err != nil { return err }
					name, ok := key.(string)
					if !ok || seen[name] { return errors.New("duplicate or invalid JSON key") }
					seen[name] = true
					if err := walk(); err != nil { return err }
				}
			case '[':
				for d.More() { if err := walk(); err != nil { return err } }
			default: return errors.New("unexpected JSON delimiter")
			}
			_, err = d.Token()
			return err
		case json.Number:
			if strings.ContainsAny(string(v), ".eE") { return errors.New("floating point contract value") }
		}
		return nil
	}
	if err := walk(); err != nil { return err }
	if _, err := d.Token(); err != io.EOF { return errors.New("trailing JSON value") }
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	return d.Decode(target)
}

func canonicalJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil { return nil, err }
	var object any
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := d.Decode(&object); err != nil { return nil, err }
	// Marshal sorts map keys, preserves arrays and emits UTF-8 without spaces.
	return json.Marshal(object)
}

func grantEntryDigest(entry GrantEntry) string {
	entry.Digest = ""
	data, _ := canonicalJSON(entry)
	return contentDigest(data)
}

func canonicalSet(values []string, required bool) bool {
	if values == nil || (required && len(values) == 0) { return false }
	for i, value := range values {
		if strings.TrimSpace(value) == "" || (i > 0 && values[i-1] >= value) { return false }
	}
	return true
}

func ParseGrantLog(data []byte, id TaskID) (GrantLog, error) {
	var log GrantLog
	text := string(data)
	if !utf8.Valid(data) || strings.Count(text, grantOpen) != 1 || strings.Count(text, grantClose) != 1 { return log, errors.New("grant log requires exactly one versioned block") }
	start, end := strings.Index(text, grantOpen)+len(grantOpen), strings.Index(text, grantClose)
	if end < start { return log, errors.New("invalid grant markers") }
	block := strings.ReplaceAll(strings.TrimSpace(text[start:end]), "\r\n", "\n")
	if !strings.HasPrefix(block, "```json\n") || !strings.HasSuffix(block, "\n```") { return log, errors.New("grant block requires one JSON fence") }
	data = []byte(strings.TrimSuffix(strings.TrimPrefix(block, "```json\n"), "\n```"))
	if err := strictJSON(data, &log); err != nil { return log, err }
	var original any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&original); err != nil { return log, err }
	if !equalJSON(original, log) { return log, errors.New("grant log has missing required fields") }
	if log.SchemaVersion != 1 || log.TaskID != id || log.Entries == nil || log.Revision != len(log.Entries) { return log, errors.New("unsupported grant version, Task or revision") }
	seen := map[string]GrantEntry{}
	previous := ""
	for i, entry := range log.Entries {
		if entry.ID == "" || entry.Sequence != i+1 || entry.PreviousDigest != previous || entry.Digest != grantEntryDigest(entry) { return log, errors.New("grant chain identity or digest mismatch") }
		if _, exists := seen[entry.ID]; exists { return log, errors.New("duplicate grant ID") }
		if _, err := time.Parse(time.RFC3339, entry.Date); err != nil { return log, err }
		if entry.Permission != "test-run" && entry.Permission != "local-delivery" && entry.Permission != "managed-cleanup" { return log, errors.New("unknown grant capability") }
		if (entry.AuthorizerType != "user" && entry.AuthorizerType != "ai") || strings.TrimSpace(entry.AuthorizerID) == "" || strings.TrimSpace(entry.Reason) == "" || entry.ApplicantStage == "" || !validProtocolReference(entry.DecisionReference) { return log, errors.New("grant decision provenance is incomplete") }
		for _, ref := range []GrantVersion{entry.PlanReference, entry.PolicyReference} {
			if !validProtocolReference(ref.ActorReference) || ref.SchemaVersion < 1 || ref.Version == "" || strings.EqualFold(ref.Version, "latest") { return log, errors.New("grant requires immutable plan and policy versions") }
		}
		if len(entry.Scope.WorkspaceDigest) != 64 || !canonicalSet(entry.Scope.Paths, true) || !canonicalSet(entry.Scope.Actions, true) || !canonicalSet(entry.Scope.Targets, true) || !canonicalSet(entry.Supersedes, false) { return log, errors.New("grant scope sets must be sorted, unique and explicit") }
		for _, old := range entry.Supersedes {
			prior, ok := seen[old]
			if !ok || prior.Permission != entry.Permission || prior.PlanReference != entry.PlanReference { return log, errors.New("grant supersedes an unknown or unrelated decision") }
		}
		seen[entry.ID], previous = entry, entry.Digest
	}
	return log, nil
}

func (log GrantLog) anchor() GrantAnchor {
	anchor := GrantAnchor{Revision: log.Revision}
	if log.Revision > 0 { anchor.Digest = log.Entries[log.Revision-1].Digest }
	return anchor
}

func (log GrantLog) authorize(ref ActorReference, plan, policy ActorReference, scope GrantScope) error {
	var selected *GrantEntry
	for i := range log.Entries {
		entry := &log.Entries[i]
		if ref.Path == "grant.md#"+entry.ID && ref.SHA256 == entry.Digest && ref.Kind == "grant" { selected = entry; break }
	}
	if selected == nil || !selected.Approved || selected.Permission != "test-run" || selected.PlanReference.ActorReference != plan || selected.PolicyReference.ActorReference != policy { return errors.New("denied: exact approved grant is absent") }
	a, _ := canonicalJSON(selected.Scope)
	b, _ := canonicalJSON(scope)
	if !bytes.Equal(a, b) { return errors.New("stale: grant scope changed") }
	for _, other := range log.Entries {
		if other.ID == selected.ID || other.Permission != selected.Permission || other.PlanReference != selected.PlanReference { continue }
		if other.Sequence > selected.Sequence { return errors.New("denied: a later decision invalidates the selected grant") }
		index := sort.SearchStrings(selected.Supersedes, other.ID)
		if index == len(selected.Supersedes) || selected.Supersedes[index] != other.ID { return errors.New("denied: approval does not explicitly supersede every conflicting decision") }
	}
	return nil
}

func (s *Store) readGrantLog(state RuntimeState) (GrantLog, error) {
	data, err := os.ReadFile(s.path(state.Task.ID, "grant.md"))
	if err != nil { return GrantLog{}, err }
	log, err := ParseGrantLog(data, state.Task.ID)
	if err != nil { return log, err }
	if state.Protocol == nil || log.anchor() != state.Protocol.Grants { return log, errors.New("unknown: grant file and committed Core anchor require reconciliation") }
	return log, nil
}

// SaveGrantDecision runs under the existing R1 conditional Task commit. An
// orphaned file append can only be reconciled by the exact same decision.
// verify must check the real decision request/output (including AI identity),
// and may not call a model or produce a new decision in this transaction.
func (s *Store) SaveGrantDecision(id TaskID, revision uint64, entry GrantEntry, verify func(RuntimeState, GrantEntry) error) (RuntimeState, error) {
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.grant.saved", Detail: entry.ID}, func(state *RuntimeState) error {
		if state.Protocol == nil || state.SchemaVersion != DurableSchemaVersion || verify == nil { return errors.New("managed grant service or migrated Store is unavailable") }
		if err := verify(*state, entry); err != nil { return err }
		for _, ref := range []ActorReference{entry.DecisionReference, entry.PlanReference.ActorReference, entry.PolicyReference.ActorReference} {
			var artifact json.RawMessage
			if err := s.ReadExecutionArtifact(id, ref, &artifact); err != nil { return err }
		}
		data, err := os.ReadFile(s.path(id, "grant.md"))
		log := GrantLog{SchemaVersion: 1, TaskID: id, Entries: []GrantEntry{}}
		if err == nil { log, err = ParseGrantLog(data, id) }
		if err != nil && !errors.Is(err, os.ErrNotExist) { return err }
		anchor := state.Protocol.Grants
		if log.Revision < anchor.Revision || (anchor.Revision > 0 && log.Entries[anchor.Revision-1].Digest != anchor.Digest) { return errors.New("grant history was truncated or replaced") }
		if log.anchor() == anchor {
			for _, prior := range log.Entries {
				if prior.ID == entry.ID {
					if equalJSON(prior, entry) { return errProtocolNoChange }
					return errors.New("a historical grant decision cannot be replaced")
				}
			}
		}
		if entry.Sequence != anchor.Revision+1 || entry.PreviousDigest != anchor.Digest || entry.Digest != grantEntryDigest(entry) { return errors.New("grant append does not match the expected revision") }
		if log.Revision == anchor.Revision { log.Entries = append(log.Entries, entry); log.Revision++
		} else if log.Revision != anchor.Revision+1 || log.Entries[anchor.Revision].Digest != entry.Digest { return errors.New("unknown grant append; reconcile the original decision") }
		encoded, err := canonicalJSON(log)
		if err != nil { return err }
		block := grantOpen+"\n```json\n"+string(encoded)+"\n```\n"+grantClose
		output := []byte(block+"\n")
		if len(data) != 0 {
			start, end := strings.Index(string(data), grantOpen), strings.Index(string(data), grantClose)+len(grantClose)
			output = []byte(string(data[:start])+block+string(data[end:]))
		}
		if _, err := ParseGrantLog(output, id); err != nil { return fmt.Errorf("invalid decision: %w", err) }
		if err := durableWrite(s.path(id, "grant.md"), output); err != nil { return err }
		state.Protocol.Grants = log.anchor()
		return nil
	})
}

func SealGrantEntry(entry GrantEntry) GrantEntry {
	entry.Digest = grantEntryDigest(entry)
	return entry
}
