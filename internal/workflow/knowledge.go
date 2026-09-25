package workflow

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

// Knowledge is a source-owned projection, never an acceptance or grant.
type KnowledgeContent struct {
	Body string `json:"body"`
	EvidenceNature string `json:"evidence_nature"`
	Scope []string `json:"scope"`
	Usage string `json:"usage"`
	InvalidWhen string `json:"invalid_when"`
	Checks []InputSource `json:"checks"`
	Priority int `json:"priority"`
	General bool `json:"general"`
}

type KnowledgeOrigin struct {
	Owner TaskID `json:"owner"`
	WorkItem WorkItemID `json:"work_item"`
	SourceVersion string `json:"source_version"`
	Acceptance ActorReference `json:"acceptance"`
	Output ActorReference `json:"output"`
}

type KnowledgeReview struct {
	Action string `json:"action"`
	Actor string `json:"actor"`
	IdentityVerified bool `json:"identity_verified"`
	At string `json:"at"`
	Reason string `json:"reason"`
	Replacement string `json:"replacement,omitempty"`
	SuggestedSync string `json:"suggested_sync,omitempty"`
}

type KnowledgeVersion struct {
	ID string `json:"id"`
	Version string `json:"version"`
	Fingerprint string `json:"fingerprint"`
	Content KnowledgeContent `json:"content"`
	Origin KnowledgeOrigin `json:"origin"`
	State string `json:"state"`
	Revision int `json:"revision"`
	Reviews []KnowledgeReview `json:"reviews"`
	Reason string `json:"reason,omitempty"`
}

type KnowledgeCoverage struct {
	WorkItem WorkItemID `json:"work_item"`
	State string `json:"state"`
	JobKey string `json:"job_key,omitempty"`
	Output *ActorReference `json:"output,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type KnowledgeDraft struct {
	Version string `json:"version"`
	SourceVersion string `json:"source_version"`
	Coverage []KnowledgeCoverage `json:"coverage"`
	Entries []string `json:"entries"`
	Output ActorReference `json:"output"`
}

type KnowledgeHistorySource struct {
	Owner TaskID `json:"owner"`
	SourceID string `json:"source_id"`
	Bytes int64 `json:"bytes"`
}

type TaskKnowledge struct {
	Cursor int `json:"cursor"`
	Extractions map[string]string `json:"extractions"`
	Published map[string]bool `json:"published"`
	Versions []KnowledgeVersion `json:"versions"`
	Drafts []KnowledgeDraft `json:"drafts"`
	History []KnowledgeHistorySource `json:"history"`
	Gap string `json:"gap,omitempty"`
}

type knowledgeExtraction struct {
	Entries []KnowledgeContent `json:"entries"`
	NoNewReason string `json:"no_new_reason"`
}

func taskKnowledge(a *TaskAuxiliary) *TaskKnowledge {
	if a.Knowledge == nil { a.Knowledge = &TaskKnowledge{} }
	k := a.Knowledge
	if k.Extractions == nil { k.Extractions = map[string]string{} }
	if k.Published == nil { k.Published = map[string]bool{} }
	return k
}

func knowledgeDigest(value any) string { data, _ := json.Marshal(value); return contentDigest(data) }

func normalizeKnowledge(c KnowledgeContent) (KnowledgeContent, error) {
	c.Body = strings.TrimSpace(c.Body)
	if c.Body == "" || c.EvidenceNature == "" || c.Usage == "" || c.InvalidWhen == "" || len(c.Scope) == 0 || len(c.Scope) > 16 || len(c.Checks) > 32 || c.Priority < -100 || c.Priority > 100 { return c, errors.New("knowledge requires body, evidence nature, scope, usage and invalidation conditions") }
	c.Scope = append([]string(nil), c.Scope...)
	sort.Strings(c.Scope)
	unique := c.Scope[:0]
	for _, tag := range c.Scope { if tag == "" { return c, errors.New("empty knowledge scope") }; if len(unique) == 0 || unique[len(unique)-1] != tag { unique = append(unique, tag) } }
	c.Scope = unique
	c.Checks = append([]InputSource(nil), c.Checks...)
	sort.Slice(c.Checks, func(i, j int) bool { return c.Checks[i].Path < c.Checks[j].Path })
	for i := range c.Checks {
		check := &c.Checks[i]
		if check.Path == "" || len(check.SHA256) != 64 || i > 0 && c.Checks[i-1].Path == check.Path { return c, errors.New("knowledge applicability checks need unique paths and exact hashes") }
		*check = InputSource{Kind: "knowledge-check", Path: check.Path, SHA256: check.SHA256, Content: ""}
	}
	if raw, _ := json.Marshal(c); len(raw) > 16*1024 { return c, errors.New("knowledge entry exceeds 16 KiB") }
	return c, nil
}

// Generated IDs and source identities cannot bypass rejection of the same input.
func knowledgeFingerprint(c KnowledgeContent) string { c.Priority = 0; return knowledgeDigest(c) }

func knowledgeVersionDigest(c KnowledgeContent, origin KnowledgeOrigin) string {
	return knowledgeDigest(struct { Content KnowledgeContent; Origin KnowledgeOrigin }{c, origin})
}

func validateTaskKnowledge(a *TaskAuxiliary) error {
	if a.Knowledge == nil { return nil }
	k := a.Knowledge
	if k.Cursor < 0 || k.Cursor > len(a.Sources) || len(k.History) > 64 { return errors.New("invalid knowledge cursor or historical scope") }
	var total int64
	seenHistory := map[string]bool{}
	for _, h := range k.History {
		key := string(h.Owner)+":"+h.SourceID
		if h.Owner == "" || h.SourceID == "" || h.Bytes < 0 || seenHistory[key] { return errors.New("invalid knowledge history ledger") }
		seenHistory[key] = true; total += h.Bytes
	}
	if total > 4*1024*1024 { return errors.New("knowledge history exceeds cumulative limit") }
	seen := map[string]bool{}
	for _, v := range k.Versions {
		c, err := normalizeKnowledge(v.Content)
		if err != nil || knowledgeVersionDigest(c, v.Origin) != v.Version || knowledgeFingerprint(c) != v.Fingerprint || v.ID == "" || seen[v.Version] || v.Revision < 1 { return errors.New("knowledge version identity is invalid") }
		seen[v.Version] = true
		switch v.State { case "candidate", "confirmed", "needs-review", "rejected", "retired": default: return errors.New("invalid knowledge review state") }
	}
	// Exhaustion preserves old evidence; no pruning or automatic deletion.
	if data, _ := json.Marshal(k); len(data) > 512*1024 { return errors.New("Task knowledge projection is full; preserve history") }
	return nil
}
