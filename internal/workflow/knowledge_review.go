package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type KnowledgeDecisionPointer struct {
	Owner TaskID `json:"owner"`
	Version string `json:"version"`
}

type KnowledgeReviewRequest struct {
	Version string `json:"version"`
	ExpectedRevision int `json:"expected_revision"`
	Action string `json:"action"`
	Reason string `json:"reason"`
	Replacement string `json:"replacement,omitempty"`
	Edit *KnowledgeContent `json:"edit,omitempty"`
}

type KnowledgeReviewer struct {
	Actor string
	IdentityVerified bool
}

func knowledgeFind(state RuntimeState, version string) (*KnowledgeVersion, error) {
	if state.Protocol != nil && state.Protocol.Auxiliary != nil && state.Protocol.Auxiliary.Knowledge != nil {
		for i := range state.Protocol.Auxiliary.Knowledge.Versions {
			v := &state.Protocol.Auxiliary.Knowledge.Versions[i]
			if v.Version == version { return v, nil }
		}
	}
	return nil, errors.New("knowledge version is unavailable")
}

// Reads recheck original evidence, current acceptance, and exact bound files.
func (s *Store) knowledgeApplicability(v KnowledgeVersion, root string) string {
	state, err := s.Load(v.Origin.Owner)
	if err != nil || state.PendingEvent != nil { return "source Task is unavailable or unreconciled" }
	accepted := false
	if v.Origin.WorkItem == "" && v.Origin.Acceptance.Kind == "knowledge-human-source" { accepted = true }
	for _, item := range state.WorkItems { if item.ID == v.Origin.WorkItem && item.AcceptedReference != nil && *item.AcceptedReference == v.Origin.Acceptance { accepted = true } }
	if !accepted { return "accepted source version changed or cannot be checked" }
	for _, ref := range []ActorReference{v.Origin.Acceptance, v.Origin.Output} {
		var raw json.RawMessage
		if err := s.ReadExecutionArtifact(v.Origin.Owner, ref, &raw); err != nil { return "bound source evidence is unreadable: "+err.Error() }
	}
	if len(v.Content.Checks) == 0 { return "applicability cannot be detected: no exact workspace checks" }
	if root == "" { return "applicability cannot be detected: workspace is unavailable" }
	for _, expected := range v.Content.Checks {
		actual := ReadInputSource(root, InputSource{Kind: "knowledge-check", Path: expected.Path, ExpectedSHA256: expected.SHA256})
		if actual.Status != "loaded" || actual.SHA256 != expected.SHA256 { return "bound content changed or unreadable: "+expected.Path }
	}
	return ""
}

func (s *Store) knowledgeDecision(r auxiliaryResources, fingerprint string) (string, error) {
	pointer, ok := r.KnowledgeDecisions[fingerprint]
	if !ok { return "", nil }
	state, err := s.Load(pointer.Owner); if err != nil { return "", err }
	v, err := knowledgeFind(state, pointer.Version); if err != nil { return "", err }
	if v.Fingerprint != fingerprint { return "", errors.New("knowledge decision pointer conflicts with source") }
	return v.State, nil
}

// Only the interactive human adapter supplies a reviewer. Model output never
// enters this method. The project lock orders cross-Task replay protection.
func (s *Store) ReviewKnowledge(id TaskID, root string, request KnowledgeReviewRequest, reviewer KnowledgeReviewer) (KnowledgeVersion, error) {
	var result KnowledgeVersion
	if strings.TrimSpace(reviewer.Actor) == "" || strings.TrimSpace(request.Reason) == "" { return result, errors.New("human reviewer and reason are required") }
	lock, err := s.auxiliaryProjectLock(); if err != nil { return result, err }; defer systemTaskUnlock(lock)
	r, err := s.readAuxiliaryResources(); if err != nil { return result, err }
	if err := s.reserveKnowledgeStorage(&r, id); err != nil { return result, err }
	state, err := s.Load(id); if err != nil { return result, err }
	old, err := knowledgeFind(state, request.Version); if err != nil { return result, err }
	if old.Revision != request.ExpectedRevision { return *old, fmt.Errorf("knowledge review conflict: current revision is %d", old.Revision) }
	if len(old.Reviews) >= 64 { return *old, errors.New("knowledge review history is full; preserve evidence") }
	if request.Action == "confirm" || request.Action == "recheck" {
		decision, err := s.knowledgeDecision(r, old.Fingerprint); if err != nil { return *old, err }
		if decision == "rejected" || decision == "retired" { return *old, errors.New("identical input has a terminal human review; it cannot be confirmed under another version") }
		if reason := s.knowledgeApplicability(*old, root); reason != "" { return *old, errors.New(reason) }
	}
	switch request.Action {
	case "confirm", "recheck", "reject", "retire", "replace", "edit":
	default: return *old, errors.New("unsupported human knowledge review action")
	}
	if request.Action == "replace" {
		replacement, err := knowledgeFind(state, request.Replacement)
		if err != nil || replacement.Version == old.Version || replacement.State != "confirmed" { return *old, errors.New("replacement must name another confirmed entry version") }
	}
	if old.State == "rejected" || old.State == "retired" {
		if request.Action != "edit" { return *old, errors.New("terminal review is preserved; edit creates a new candidate") }
	}
	if request.Action == "recheck" && old.State != "needs-review" { return *old, errors.New("recheck requires a needs-review version") }
	var edited KnowledgeContent
	if request.Action == "edit" {
		if request.Edit == nil { return *old, errors.New("edit requires the complete new entry") }
		edited, err = normalizeKnowledge(*request.Edit); if err != nil { return *old, err }
		if knowledgeDigest(edited) == knowledgeDigest(old.Content) { return *old, errors.New("unchanged content is not a new candidate") }
		decision, err := s.knowledgeDecision(r, knowledgeFingerprint(edited)); if err != nil { return *old, err }
		if decision == "rejected" || decision == "retired" { return *old, errors.New("identical input already has a terminal human review") }
	}
	// Consumers resolve this pointer against the authoritative version, so a
	// crash before the review commit never invents a decision.
	if r.KnowledgeDecisions == nil { r.KnowledgeDecisions = map[string]KnowledgeDecisionPointer{} }
	if request.Action == "reject" || request.Action == "retire" || request.Action == "replace" {
		r.KnowledgeDecisions[old.Fingerprint] = KnowledgeDecisionPointer{id, old.Version}
		if err := s.saveAuxiliaryResources(r); err != nil { return *old, err }
	}
	_, err = s.updateWithEvent(id, &state.StateRevision, Event{Type: "protocol.knowledge.human-review", Detail: request.Version}, func(current *RuntimeState) error {
		v, err := knowledgeFind(*current, request.Version); if err != nil { return err }
		if v.Revision != request.ExpectedRevision { return errors.New("knowledge review revision changed") }
		review := KnowledgeReview{Action: request.Action, Actor: reviewer.Actor, IdentityVerified: reviewer.IdentityVerified, At: time.Now().UTC().Format(time.RFC3339Nano), Reason: request.Reason, Replacement: request.Replacement}
		if request.Action == "confirm" || request.Action == "recheck" { review.SuggestedSync = "Review whether this exact version and its sources should be reflected in CONTEXT, ADR, or requirements. No formal document was modified." }
		if request.Action == "edit" {
			version := knowledgeVersionDigest(edited, v.Origin)
			if _, err := knowledgeFind(*current, version); err == nil { return errors.New("edited version already exists; review that exact version") }
			result = KnowledgeVersion{ID: v.ID, Version: version, Fingerprint: knowledgeFingerprint(edited), Content: edited, Origin: v.Origin, State: "candidate", Revision: 1, Reviews: []KnowledgeReview{review}}
			taskKnowledge(current.Protocol.Auxiliary).Versions = append(taskKnowledge(current.Protocol.Auxiliary).Versions, result)
		} else {
			switch request.Action {
			case "confirm", "recheck": v.State = "confirmed"
			case "reject": v.State = "rejected"
			case "retire", "replace": v.State = "retired"
			}
			v.Reviews = append(v.Reviews, review); v.Revision++; v.Reason = request.Reason; result = *v
		}
		return validateTaskKnowledge(current.Protocol.Auxiliary)
	})
	return result, err
}

// Recovery of identical bytes still needs an explicit human recheck.
func (s *Store) RecheckKnowledge(id TaskID, root string) error {
	state, err := s.Load(id); if err != nil { return err }
	if state.Protocol == nil || state.Protocol.Auxiliary == nil || state.Protocol.Auxiliary.Knowledge == nil { return nil }
	reasons := map[string]string{}
	for _, v := range state.Protocol.Auxiliary.Knowledge.Versions {
		if v.State == "confirmed" { if reason := s.knowledgeApplicability(v, root); reason != "" { reasons[v.Version] = reason } }
	}
	if len(reasons) == 0 { return nil }
	_, err = s.updateWithEvent(id, &state.StateRevision, Event{Type: "protocol.knowledge.needs-review"}, func(current *RuntimeState) error {
		for i := range current.Protocol.Auxiliary.Knowledge.Versions {
			v := &current.Protocol.Auxiliary.Knowledge.Versions[i]
			if reason := reasons[v.Version]; reason != "" && v.State == "confirmed" { v.State, v.Reason = "needs-review", reason; v.Revision++ }
		}
		return nil
	})
	return err
}
