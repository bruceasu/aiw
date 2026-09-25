package workflow

import (
	"encoding/json"
	"errors"
	"strings"
)

type KnowledgeHumanIndex struct {
	Owner TaskID `json:"owner"`
	Version string `json:"version"`
	Tags []string `json:"tags"`
	SourceID string `json:"source_id"`
	Bytes int64 `json:"bytes"`
}

// Import an explicitly selected original human source; never scan Sessions or
// infer formal approval. The original wording is retained in its own artifact.
func (s *Store) ImportHumanKnowledge(id TaskID, source InputSource, content KnowledgeContent) (KnowledgeVersion, error) {
	var result KnowledgeVersion
	if source.Status != "loaded" || source.Path == "" || source.SHA256 != contentDigest([]byte(source.Content)) || strings.TrimSpace(source.Content) == "" || len(source.Content) > 64*1024 { return result, errors.New("human import requires a bounded exact original source") }
	content.EvidenceNature = "original human text; identity unverified; not a formal approval"
	content, err := normalizeKnowledge(content); if err != nil { return result, err }
	lock, err := s.auxiliaryProjectLock(); if err != nil { return result, err }; defer systemTaskUnlock(lock)
	r, err := s.readAuxiliaryResources(); if err != nil { return result, err }
	if err := s.reserveKnowledgeStorage(&r, id); err != nil { return result, err }
	decision, err := s.knowledgeDecision(r, knowledgeFingerprint(content)); if err != nil { return result, err }
	if decision == "rejected" || decision == "retired" { return result, errors.New("identical human input already has a terminal review") }
	state, err := s.Load(id); if err != nil { return result, err }
	if state.Protocol == nil || state.Protocol.Auxiliary == nil { return result, errors.New("human import requires a durable Task source") }
	_, err = s.updateWithEvent(id, &state.StateRevision, Event{Type: "protocol.knowledge.human-import", Detail: source.SHA256}, func(current *RuntimeState) error {
		k := taskKnowledge(current.Protocol.Auxiliary)
		imports := 0
		for _, v := range k.Versions { if v.Origin.WorkItem == "" { imports++ } }
		// Check the bound before storing another unique original artifact.
		if imports >= 16 {
			for _, v := range k.Versions { if v.Origin.SourceVersion == source.SHA256 && knowledgeDigest(v.Content) == knowledgeDigest(content) { result = v; return errProtocolNoChange } }
			return errors.New("human source import limit reached; preserve originals")
		}
		data, err := json.MarshalIndent(source, "", "  "); if err != nil { return err }
		data = append(data, '\n')
		ref := ActorReference{Kind: "knowledge-human-source", Path: "reports/protocol/"+contentDigest(data)+".json", SHA256: contentDigest(data)}
		origin := KnowledgeOrigin{Owner: id, SourceVersion: source.SHA256, Acceptance: ref, Output: ref}
		version := knowledgeVersionDigest(content, origin)
		for _, v := range k.Versions { if v.Version == version { result = v; return errProtocolNoChange } }
		result = KnowledgeVersion{ID: knowledgeFingerprint(content), Version: version, Fingerprint: knowledgeFingerprint(content), Content: content, Origin: origin, State: "candidate", Revision: 1}
		k.Versions = append(k.Versions, result)
		if err := validateTaskKnowledge(current.Protocol.Auxiliary); err != nil { return err }
		stored, err := s.persistProtocolArtifactLocked(id, "knowledge-human-source", source)
		if err != nil { return err }
		if stored != ref { return errors.New("human source storage identity mismatch") }
		return nil
	})
	if err != nil { return result, err }
	if r.HumanKnowledge == nil { r.HumanKnowledge = map[string]KnowledgeHumanIndex{} }
	r.HumanKnowledge[result.Version] = KnowledgeHumanIndex{id, result.Version, content.Scope, source.SHA256, int64(len(source.Content))}
	return result, s.saveAuxiliaryResources(r)
}
