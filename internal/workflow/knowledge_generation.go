package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type knowledgeJobContext struct {
	WorkItem WorkItemID `json:"work_item,omitempty"`
	Acceptance *ActorReference `json:"acceptance,omitempty"`
	Coverage []KnowledgeCoverage `json:"coverage,omitempty"`
	Entries []string `json:"entries,omitempty"`
}

const extractionInstructions = `Extract reusable rules, decisions, and lessons for this accepted Work Item only. Use existing failure evidence only. Do not claim a new human decision. Return one JSON object with entries (an explicit array) and no_new_reason (a string). Each entry has body, evidence_nature, scope (module:, path:, interface:, or domain: tags), usage, invalid_when, checks (an array of {kind,path,required,sha256,content} with workspace-relative paths and exact observed file hashes, or [] if applicability cannot be checked), priority (integer -100 to 100), and general (boolean). Use evidence_nature to distinguish observed facts, human decisions, and suggestions. Keep each entry below 16 KiB and return at most 16 entries. If there is no new knowledge, return entries=[] with an evidence-based no_new_reason. Never treat a failed extraction as no new knowledge.`

func knowledgeInstruction(kind string, context knowledgeJobContext) string {
	data, _ := json.Marshal(context)
	if kind == "knowledge-extraction" { return kind+"-v1 "+string(data)+"\n"+extractionInstructions }
	return kind+"-v1 "+string(data)+"\nSummarize the fixed extraction coverage below as a partial project knowledge draft. Show all failed, running, and missing Work Items. Preserve entry version references. Do not confirm, reject, retire, replace, or edit entries. Distinguish no new knowledge from missing or failed extraction. Return short plain text.\n"
}

func knowledgeContext(job AuxiliaryJob) (knowledgeJobContext, error) {
	var result knowledgeJobContext
	line, _, _ := strings.Cut(job.Prompt, "\n")
	prefix := job.Kind+"-v1 "
	if !strings.HasPrefix(line, prefix) { return result, errors.New("knowledge job has no fixed semantic context") }
	if err := strictJSON([]byte(strings.TrimPrefix(line, prefix)), &result); err != nil { return result, err }
	if job.Kind == "knowledge-extraction" && (result.WorkItem == "" || result.Acceptance == nil || !validProtocolReference(*result.Acceptance)) { return result, errors.New("knowledge extraction lacks accepted source") }
	return result, nil
}

// Called before the controlled adapter journals its terminal observation, so
// semantic errors consume the same one-recovery allowance as transport errors.
func ValidateKnowledgeOutput(job AuxiliaryJob, text string) error {
	if job.Kind != "knowledge-extraction" && job.Kind != "knowledge-summary" { return nil }
	if _, err := knowledgeContext(job); err != nil { return err }
	if job.Kind == "knowledge-summary" {
		if strings.TrimSpace(text) == "" { return errors.New("empty knowledge summary") }; return nil
	}
	var result knowledgeExtraction
	if err := strictJSON([]byte(text), &result); err != nil { return err }
	if result.Entries == nil || len(result.Entries) > 16 || len(result.Entries) == 0 && strings.TrimSpace(result.NoNewReason) == "" || len(result.Entries) > 0 && result.NoNewReason != "" { return errors.New("extraction needs explicit entries or an explained no-new result") }
	for _, c := range result.Entries { if _, err := normalizeKnowledge(c); err != nil { return err } }
	return nil
}

func extractionIdentity(item AuxiliarySourceItem) string { return string(item.ID)+":"+knowledgeDigest(item.Accepted) }

// Persisted source facts are the outbox. No failure/Session event starts an
// extraction; an acceptance version is registered once and reused thereafter.
func (s *Store) RegisterTaskKnowledge(id TaskID) error {
	state, err := s.Load(id)
	if err != nil { return err }
	if state.Protocol == nil || state.Protocol.Auxiliary == nil { return nil }
	if err := s.prepareKnowledgeStorage(id); err != nil { return err }
	root := state.Task.Workspace
	if state.Workspace != nil && state.Workspace.WorktreePath != "" { root = state.Workspace.WorktreePath }
	if !filepath.IsAbs(root) { root = filepath.Join(filepath.Dir(s.Root), filepath.FromSlash(root)) }
	if err := s.RecheckKnowledge(id, root); err != nil { return err }
	a := state.Protocol.Auxiliary
	cursor := 0
	if a.Knowledge != nil { cursor = a.Knowledge.Cursor }
	for i := cursor; i < len(a.Sources); i++ {
		source := a.Sources[i]
		for _, item := range source.Items {
			if item.Accepted == nil { continue }
			current, err := s.Load(id); if err != nil { return err }
			k := taskKnowledge(current.Protocol.Auxiliary)
			key := extractionIdentity(item)
			if k.Extractions[key] != "" { continue }
			ref, err := s.EnqueueAuxiliary(id, id, source.ID, "knowledge-extraction", knowledgeInstruction("knowledge-extraction", knowledgeJobContext{WorkItem: item.ID, Acceptance: item.Accepted}))
			if err != nil { return err }
			_, err = s.updateKnowledge(id, Event{Type: "protocol.knowledge.extraction-linked", Detail: ref.Key}, func(current *RuntimeState) error {
				k := taskKnowledge(current.Protocol.Auxiliary)
				if old := k.Extractions[key]; old != "" && old != ref.Key { return errors.New("acceptance extraction identity conflict") }
				k.Extractions[key] = ref.Key
				return nil
			})
			if err != nil { return err }
		}
		_, err := s.updateKnowledge(id, Event{Type: "protocol.knowledge.source-consumed", Detail: source.ID}, func(current *RuntimeState) error {
			k := taskKnowledge(current.Protocol.Auxiliary)
			if k.Cursor > i { return errProtocolNoChange }
			if k.Cursor != i { return errors.New("knowledge cursor changed") }
			k.Cursor++
			return nil
		})
		if err != nil { return err }
	}
	// Repair an interrupted output-to-index projection without re-generation.
	current, err := s.Load(id); if err != nil { return err }
	for _, job := range current.Protocol.Auxiliary.Jobs {
		if job.Kind == "knowledge-extraction" && job.State == "completed" {
			if err := s.repairKnowledgeIndex(current, job); err != nil { return err }
		}
	}
	return s.registerKnowledgeSummary(id)
}

func knowledgeCoverage(state RuntimeState, source AuxiliarySource) ([]KnowledgeCoverage, []string) {
	k := taskKnowledge(state.Protocol.Auxiliary)
	coverage := make([]KnowledgeCoverage, 0, len(source.Items))
	for _, item := range source.Items {
		c := KnowledgeCoverage{WorkItem: item.ID, State: "missing", Reason: "no extraction registered"}
		if key := k.Extractions[extractionIdentity(item)]; key != "" {
			c.JobKey = key
			if job, err := findAuxiliaryJob(state, key); err == nil {
				c.Reason = job.Reason
				switch job.State {
				case "completed": c.State, c.Output = "success", job.Output
				case "unavailable": c.State = "failed"
				default: c.State = "running"; c.Reason = "extraction is not terminal"
				}
			}
		}
		coverage = append(coverage, c)
	}
	sort.Slice(coverage, func(i, j int) bool { return coverage[i].WorkItem < coverage[j].WorkItem })
	var entries []string
	for _, v := range k.Versions { entries = append(entries, v.Version) }
	sort.Strings(entries)
	return coverage, entries
}

func (s *Store) registerKnowledgeSummary(id TaskID) error {
	state, err := s.Load(id); if err != nil { return err }
	a := state.Protocol.Auxiliary
	if len(a.Sources) == 0 { return nil }
	source := a.Sources[len(a.Sources)-1]
	if source.Kind != "completed" { return nil }
	coverage, entries := knowledgeCoverage(state, source)
	context := knowledgeJobContext{Coverage: coverage, Entries: entries}
	instruction := knowledgeInstruction("knowledge-summary", context)
	for _, c := range coverage {
		if c.Output == nil { continue }
		var output AuxiliaryOutput
		if err := s.ReadExecutionArtifact(id, *c.Output, &output); err != nil { return err }
		instruction += fmt.Sprintf("\n[Extraction %s %s]\n%s\n", c.WorkItem, c.Output.SHA256, output.Text)
	}
	_, err = s.EnqueueAuxiliary(id, id, source.ID, "knowledge-summary", instruction)
	return err
}

// Publication happens in the existing output/Task commit. A retry reuses the
// exact source and cannot overwrite a human edit or a previously reviewed body.
func (s *Store) publishKnowledge(state *RuntimeState, job AuxiliaryJob, output ActorReference, resources auxiliaryResources) error {
	if job.Kind != "knowledge-extraction" && job.Kind != "knowledge-summary" { return nil }
	if err := ValidateKnowledgeOutput(job, job.Result.Text); err != nil { return err }
	k := taskKnowledge(state.Protocol.Auxiliary)
	if k.Published[job.Key] { return nil }
	context, err := knowledgeContext(job); if err != nil { return err }
	if job.Kind == "knowledge-summary" {
		draft := KnowledgeDraft{SourceVersion: job.SourceVersion, Coverage: context.Coverage, Entries: context.Entries, Output: output}
		draft.Version = knowledgeDigest(draft)
		k.Drafts = append(k.Drafts, draft)
	} else {
		var result knowledgeExtraction
		if err := strictJSON([]byte(job.Result.Text), &result); err != nil { return err }
		origin := KnowledgeOrigin{job.Owner, context.WorkItem, job.SourceVersion, *context.Acceptance, output}
		for _, content := range result.Entries {
			content, err = normalizeKnowledge(content); if err != nil { return err }
			fingerprint := knowledgeFingerprint(content)
			decision, err := s.knowledgeDecision(resources, fingerprint); if err != nil { return err }
			if decision == "rejected" || decision == "retired" { continue }
			version := knowledgeVersionDigest(content, origin)
			duplicate := false
			for _, prior := range k.Versions {
				if prior.Version == version || prior.Fingerprint == fingerprint && (prior.State == "rejected" || prior.State == "retired" || prior.Origin.Acceptance == origin.Acceptance) { duplicate = true; break }
			}
			if duplicate { continue }
			k.Versions = append(k.Versions, KnowledgeVersion{ID: fingerprint, Version: version, Fingerprint: fingerprint, Content: content, Origin: origin, State: "candidate", Revision: 1})
		}
	}
	k.Published[job.Key] = true
	return validateTaskKnowledge(state.Protocol.Auxiliary)
}
