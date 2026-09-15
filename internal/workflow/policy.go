package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// CapabilityState is deliberately separate from ordinary configuration.
// Higher-precedence configuration can restrict a capability, but cannot turn a
// restriction from a lower-precedence source into a wider permission.
type CapabilityState string

const (
	CapabilityForbidden            CapabilityState = "forbidden"
	CapabilityRequiresConfirmation CapabilityState = "requires-confirmation"
	CapabilityAllowed              CapabilityState = "allowed"
)

// CapabilityAuthorization records the effective capability state and, where
// needed, the explicit human confirmation that satisfied it.
type CapabilityAuthorization struct {
	State       CapabilityState `json:"state"`
	ConfirmedBy string          `json:"confirmed_by,omitempty"`
}

// PolicyLayer is one named configuration source. Values use ordinary
// last-writer-wins precedence; capabilities are intersected separately.
type PolicyLayer struct {
	Ordinary     map[string]string
	Capabilities map[string]CapabilityAuthorization
}

// PolicyLayers must be supplied from lowest to highest precedence: defaults,
// user, repository, Task request, and command-line overrides.
type PolicyLayers []PolicyLayer

// PolicySnapshot is the immutable, secret-free policy used for a Task run.
// Only environment references (for example env:GITHUB_TOKEN) may be recorded.
type PolicySnapshot struct {
	Ordinary     map[string]string                    `json:"ordinary"`
	Capabilities map[string]CapabilityAuthorization `json:"capabilities"`
	Digest       string                               `json:"digest"`
	ResolvedAt   string                               `json:"resolved_at"`
}

// ResolvePolicy applies ordinary precedence and capability intersection. A
// confirmation can satisfy only requires-confirmation; it can never override
// a forbidden value from any source.
func ResolvePolicy(layers PolicyLayers) (PolicySnapshot, error) {
	snapshot := PolicySnapshot{Ordinary: map[string]string{}, Capabilities: map[string]CapabilityAuthorization{}}
	for _, layer := range layers {
		for key, value := range layer.Ordinary {
			key, value = strings.TrimSpace(key), strings.TrimSpace(value)
			if key == "" || value == "" {
				continue
			}
			if isSecretPolicyKey(key) && !strings.HasPrefix(value, "env:") {
				return PolicySnapshot{}, fmt.Errorf("ordinary setting %q contains a secret; persist an env: reference instead", key)
			}
			snapshot.Ordinary[key] = value
		}
		for capability, requested := range layer.Capabilities {
			capability = strings.TrimSpace(capability)
			if capability == "" {
				return PolicySnapshot{}, errors.New("capability name is required")
			}
			if err := validateCapabilityAuthorization(requested); err != nil {
				return PolicySnapshot{}, fmt.Errorf("capability %q: %w", capability, err)
			}
			current, exists := snapshot.Capabilities[capability]
			if !exists || capabilityRank(requested.State) < capabilityRank(current.State) {
				snapshot.Capabilities[capability] = requested
				continue
			}
			if requested.State == CapabilityRequiresConfirmation && current.State == CapabilityRequiresConfirmation && requested.ConfirmedBy != "" {
				current.ConfirmedBy = requested.ConfirmedBy
				snapshot.Capabilities[capability] = current
			}
		}
	}
	for capability, authorization := range snapshot.Capabilities {
		if authorization.State == CapabilityRequiresConfirmation && authorization.ConfirmedBy != "" {
			authorization.State = CapabilityAllowed
			snapshot.Capabilities[capability] = authorization
		}
	}
	snapshot.ResolvedAt = time.Now().UTC().Format(time.RFC3339)
	if err := snapshot.setDigest(); err != nil {
		return PolicySnapshot{}, err
	}
	return snapshot, nil
}

// Validate confirms a persisted snapshot has not been modified outside the
// event protocol. ResolvedAt is intentionally excluded from the digest.
func (snapshot PolicySnapshot) Validate() error {
	if snapshot.ResolvedAt == "" {
		return errors.New("resolved time is required")
	}
	if _, err := time.Parse(time.RFC3339, snapshot.ResolvedAt); err != nil {
		return fmt.Errorf("resolved time: %w", err)
	}
	if len(snapshot.Digest) != sha256.Size*2 {
		return errors.New("digest must be a SHA-256 hex value")
	}
	copy := snapshot
	if err := copy.setDigest(); err != nil {
		return err
	}
	if copy.Digest != snapshot.Digest {
		return errors.New("digest does not match snapshot")
	}
	for key, value := range snapshot.Ordinary {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return errors.New("ordinary setting key and value are required")
		}
		if isSecretPolicyKey(key) && !strings.HasPrefix(value, "env:") {
			return fmt.Errorf("ordinary setting %q contains a secret", key)
		}
	}
	for capability, authorization := range snapshot.Capabilities {
		if strings.TrimSpace(capability) == "" {
			return errors.New("capability name is required")
		}
		if err := validateCapabilityAuthorization(authorization); err != nil {
			return fmt.Errorf("capability %q: %w", capability, err)
		}
	}
	return nil
}

func (snapshot *PolicySnapshot) setDigest() error {
	if snapshot.Ordinary == nil { snapshot.Ordinary = map[string]string{} }
	if snapshot.Capabilities == nil { snapshot.Capabilities = map[string]CapabilityAuthorization{} }
	type canonical struct { Ordinary [][2]string `json:"ordinary"`; Capabilities [][3]string `json:"capabilities"` }
	value := canonical{}
	for key, setting := range snapshot.Ordinary { value.Ordinary = append(value.Ordinary, [2]string{key, setting}) }
	for key, authorization := range snapshot.Capabilities { value.Capabilities = append(value.Capabilities, [3]string{key, string(authorization.State), authorization.ConfirmedBy}) }
	sort.Slice(value.Ordinary, func(i, j int) bool { return value.Ordinary[i][0] < value.Ordinary[j][0] })
	sort.Slice(value.Capabilities, func(i, j int) bool { return value.Capabilities[i][0] < value.Capabilities[j][0] })
	b, err := json.Marshal(value)
	if err != nil { return err }
	digest := sha256.Sum256(b)
	snapshot.Digest = hex.EncodeToString(digest[:])
	return nil
}

func capabilityRank(state CapabilityState) int {
	switch state { case CapabilityForbidden: return 0; case CapabilityRequiresConfirmation: return 1; case CapabilityAllowed: return 2; default: return -1 }
}

func validateCapabilityAuthorization(authorization CapabilityAuthorization) error {
	if capabilityRank(authorization.State) < 0 { return fmt.Errorf("invalid state %q", authorization.State) }
	return nil
}

func isSecretPolicyKey(key string) bool {
	key = strings.ToLower(key)
	return strings.Contains(key, "secret") || strings.Contains(key, "token") || strings.Contains(key, "password") || strings.Contains(key, "api_key")
}

// SnapshotPolicy persists the first effective policy before a mutating Actor
// starts. Resumes return the original snapshot instead of resolving mutable
// files or environment variables again.
func (s *Store) SnapshotPolicy(id TaskID, snapshot PolicySnapshot) (RuntimeState, error) {
	if err := snapshot.Validate(); err != nil { return RuntimeState{}, err }
	current, err := s.Load(id)
	if err != nil { return RuntimeState{}, err }
	if current.Policy != nil {
		if current.Policy.Digest != snapshot.Digest { return RuntimeState{}, errors.New("policy snapshot already exists; use UpdatePolicy") }
		return current, nil
	}
	return s.UpdateWithEvent(id, Event{Type: "policy.resolved", Detail: snapshot.Digest}, func(state *RuntimeState) error {
		if state.Policy != nil {
			return errors.New("policy snapshot already exists; retry loading it")
		}
		state.Policy = &snapshot
		return nil
	})
}

// UpdatePolicy is the only explicit policy-change transition. It preserves
// both digests, the responsible actor, and the reason in the event detail.
func (s *Store) UpdatePolicy(id TaskID, snapshot PolicySnapshot, actor, reason string) (RuntimeState, error) {
	if err := snapshot.Validate(); err != nil { return RuntimeState{}, err }
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(reason) == "" { return RuntimeState{}, errors.New("policy update actor and reason are required") }
	current, err := s.Load(id)
	if err != nil { return RuntimeState{}, err }
	if current.Policy == nil { return RuntimeState{}, errors.New("policy snapshot is required before update") }
	if current.Policy.Digest == snapshot.Digest { return current, nil }
	detail := current.Policy.Digest + "->" + snapshot.Digest + ";actor=" + actor + ";reason=" + reason
	return s.UpdateWithEvent(id, Event{Type: "policy.updated", Detail: detail}, func(state *RuntimeState) error {
		if state.Policy == nil { return errors.New("policy snapshot is required before update") }
		if state.Policy.Digest == snapshot.Digest { return nil }
		state.Policy = &snapshot
		return nil
	})
}
