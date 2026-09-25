package workflow

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type knowledgeChoice struct { Version KnowledgeVersion; Score int; Warning string }

func knowledgeMatch(tags []string, context string) int {
	score := 0
	for _, tag := range tags {
		kind, value, ok := strings.Cut(tag, ":")
		if !ok || value == "" { continue }
		switch kind { case "module", "path", "interface", "domain": default: continue }
		if strings.Contains(context, value) { score++ }
	}
	return score
}

func knowledgeTags(state RuntimeState, job AuxiliaryJob) []string {
	if job.Output == nil || state.Protocol.Auxiliary.Knowledge == nil { return nil }
	set := map[string]bool{}
	for _, v := range state.Protocol.Auxiliary.Knowledge.Versions {
		if v.Origin.Output != *job.Output { continue }
		for _, tag := range v.Content.Scope { set[tag] = true }
		if v.Content.General { set["general:"] = true }
	}
	var tags []string
	for tag := range set { tags = append(tags, tag) }
	sort.Strings(tags)
	return tags
}

func (s *Store) repairKnowledgeIndex(state RuntimeState, job AuxiliaryJob) error {
	lock, err := s.auxiliaryProjectLock(); if err != nil { return err }; defer systemTaskUnlock(lock)
	r, err := s.readAuxiliaryResources(); if err != nil { return err }
	q, ok := r.Queue[job.Key]; if !ok { return errors.New("knowledge job has no resource index") }
	tags := knowledgeTags(state, job)
	if knowledgeDigest(tags) == knowledgeDigest(q.KnowledgeTags) { return nil }
	q.KnowledgeTags = tags; r.Queue[job.Key] = q
	return s.saveAuxiliaryResources(r)
}

// Charge selected immutable sources to the consumer before using them. Shared
// generation/recovery stays in the owner's job; no retry counters are copied.
func (s *Store) reserveKnowledgeHistory(id TaskID, sources []KnowledgeHistorySource) error {
	if len(sources) > 16 { return errors.New("historical source batch exceeds 16") }
	var batch int64
	for _, source := range sources { if source.Bytes < 0 { return errors.New("unknown historical source size") }; batch += source.Bytes }
	if batch > 1024*1024 { return errors.New("historical source batch exceeds 1 MiB") }
	if len(sources) == 0 { return nil }
	lock, err := s.auxiliaryProjectLock(); if err != nil { return err }; defer systemTaskUnlock(lock)
	r, err := s.readAuxiliaryResources(); if err != nil { return err }
	if _, known := r.Inventory.TaskBytes[id]; !known { return errors.New("historical consumer resource baseline is unknown") }
	if err := s.reserveKnowledgeStorage(&r, id); err != nil { return err }
	_, err = s.updateKnowledge(id, Event{Type: "protocol.knowledge.history-reserved"}, func(current *RuntimeState) error {
		if current.Protocol == nil || current.Protocol.Auxiliary == nil { return errors.New("historical consumer has no durable source ledger") }
		k := taskKnowledge(current.Protocol.Auxiliary)
		seen := map[string]bool{}; var total int64
		for _, h := range k.History { seen[string(h.Owner)+":"+h.SourceID] = true; total += h.Bytes }
		changed := false
		for _, source := range sources {
			key := string(source.Owner)+":"+source.SourceID
			if seen[key] { continue }
			if len(k.History) >= 64 || total+source.Bytes > 4*1024*1024 { return errors.New("Task historical scope exhausted; do not expand or reset it") }
			seen[key] = true; total += source.Bytes; k.History = append(k.History, source); changed = true
		}
		if !changed { return errProtocolNoChange }
		return validateTaskKnowledge(current.Protocol.Auxiliary)
	})
	return err
}

// Only the bounded existing queue index is searched, never the project tree or
// arbitrary old Sessions. The required prompt remains intact on optional gaps.
func (s *Store) InjectKnowledge(id TaskID, root, prompt string, selection *AISelection) (string, []InputSource, error) {
	gap := func(reason string) (string, []InputSource, error) {
		return prompt, []InputSource{{Kind: "optional-knowledge", Path: "knowledge-selection", Status: "unavailable", Reason: reason}}, nil
	}
	if s.AuxiliaryServices == nil || s.AuxiliaryServices.MeasureKnowledge == nil { return gap("knowledge skipped: exact model counter is unavailable") }
	baseTokens, capacity, err := s.AuxiliaryServices.MeasureKnowledge(selection, prompt)
	if err != nil { return gap("knowledge skipped: "+err.Error()) }
	if baseTokens > capacity { return "", nil, errors.New("required input exceeds the verified receiving model capacity; it was not truncated") }
	r, err := s.readAuxiliaryResources(); if err != nil { return gap(err.Error()) }
	var keys []string
	for key, q := range r.Queue {
		// Old source owners are considered only when the current fixed primary
		// input explicitly references their Task, never by scanning all history.
		explicitHistory := q.Kind == "memory" && q.Owner != id && (strings.Contains(prompt, ".ai/"+string(q.Owner)+"/") || strings.Contains(prompt, "Task: "+string(q.Owner)+"\n"))
		if q.Kind != "knowledge-extraction" && !explicitHistory { continue }
		general := false
		for _, tag := range q.KnowledgeTags { if tag == "general:" { general = true } }
		if q.Owner == id || general || explicitHistory || knowledgeMatch(q.KnowledgeTags, prompt) > 0 { keys = append(keys, key) }
	}
	sort.Strings(keys)
	var choices []knowledgeChoice
	var history []KnowledgeHistorySource
	var skipped []string
	owners := map[TaskID]RuntimeState{}
	seenSources := map[string]bool{}
	var batchBytes int64
	pendingOwners := map[TaskID]bool{}
	var backfill []KnowledgeHistorySource
	for _, key := range keys {
		q := r.Queue[key]
		// Reserve the batch's count before reading another source owner.
		sourceKey := string(q.Owner)+":"+q.SourceID
		if q.Owner != id && !seenSources[sourceKey] && len(history) >= 16 { skipped = append(skipped, key+": history batch count limit"); continue }
		state, ok := owners[q.Owner]
		if !ok {
			state, err = s.Load(q.Owner)
			if err != nil || state.PendingEvent != nil { skipped = append(skipped, key+": source unavailable"); continue }
			owners[q.Owner] = state
		}
		job, err := findAuxiliaryJob(state, key)
		if err != nil { skipped = append(skipped, key+": source job unavailable"); continue }
		if q.Owner != id && !seenSources[sourceKey] {
			size := int64(len(job.Prompt))
			if batchBytes+size > 1024*1024 { skipped = append(skipped, key+": history batch byte limit"); continue }
			history = append(history, KnowledgeHistorySource{q.Owner, q.SourceID, size}); batchBytes += size; seenSources[sourceKey] = true
		}
		if q.Kind == "memory" { backfill = append(backfill, KnowledgeHistorySource{q.Owner, q.SourceID, int64(len(job.Prompt))}); continue }
		if job.State != "completed" || job.Output == nil { skipped = append(skipped, key+": extraction incomplete"); if !auxiliaryTerminal(job.State) { pendingOwners[q.Owner] = true }; continue }
		if state.Protocol.Auxiliary.Knowledge == nil { continue }
		for _, v := range state.Protocol.Auxiliary.Knowledge.Versions {
			if v.Origin.Output != *job.Output { continue }
			decision, err := s.knowledgeDecision(r, v.Fingerprint)
			if err != nil || decision == "rejected" || decision == "retired" || v.State == "rejected" || v.State == "retired" { skipped = append(skipped, v.Version+": terminal review or unavailable decision"); continue }
			score := knowledgeMatch(v.Content.Scope, prompt)
			if score == 0 && !v.Content.General { continue }
			warning := s.knowledgeApplicability(v, root)
			if warning != "" && v.State == "confirmed" { v.State = "needs-review" }
			choices = append(choices, knowledgeChoice{v, score, warning})
		}
	}
	var humanKeys []string
	for key, entry := range r.HumanKnowledge { if entry.Owner == id || knowledgeMatch(entry.Tags, prompt) > 0 { humanKeys = append(humanKeys, key) } }
	sort.Strings(humanKeys)
	for _, key := range humanKeys {
		entry := r.HumanKnowledge[key]
		sourceKey := string(entry.Owner)+":"+entry.SourceID
		if entry.Owner != id && !seenSources[sourceKey] {
			if len(history) >= 16 || entry.Bytes < 0 || batchBytes+entry.Bytes > 1024*1024 { skipped = append(skipped, key+": human history batch limit"); continue }
			history = append(history, KnowledgeHistorySource{entry.Owner, entry.SourceID, entry.Bytes}); batchBytes += entry.Bytes; seenSources[sourceKey] = true
		}
		state, ok := owners[entry.Owner]
		if !ok { state, err = s.Load(entry.Owner); if err != nil || state.PendingEvent != nil { skipped = append(skipped, key+": human source unavailable"); continue }; owners[entry.Owner] = state }
		// Edits keep the original source identity and are selected with their
		// own version/state, never by copying the old version's confirmation.
		if state.Protocol == nil || state.Protocol.Auxiliary == nil || state.Protocol.Auxiliary.Knowledge == nil { continue }
		for _, v := range state.Protocol.Auxiliary.Knowledge.Versions {
			if v.Origin.WorkItem != "" || v.Origin.SourceVersion != entry.SourceID { continue }
			decision, err := s.knowledgeDecision(r, v.Fingerprint)
			if err != nil || decision == "rejected" || decision == "retired" || v.State == "rejected" || v.State == "retired" { continue }
			score := knowledgeMatch(v.Content.Scope, prompt); if score == 0 && !v.Content.General { continue }
			warning := s.knowledgeApplicability(v, root)
			if warning != "" && v.State == "confirmed" { v.State = "needs-review" }
			choices = append(choices, knowledgeChoice{v, score, warning})
		}
	}
	if err := s.reserveKnowledgeHistory(id, history); err != nil { return gap(err.Error()) }
	for _, source := range backfill {
		if err := s.backfillKnowledgeSource(id, source); err != nil { skipped = append(skipped, source.SourceID+": backfill unavailable: "+err.Error()); continue }
		pendingOwners[source.Owner] = true
	}
	for owner := range pendingOwners { s.ResumeAuxiliaryHost(owner) }
	// Recheck asynchronously owned state as well as the read-time view. Failure
	// cannot promote the already downgraded read-time candidate.
	for owner := range owners { if err := s.RecheckKnowledge(owner, root); err != nil { skipped = append(skipped, string(owner)+": could not persist applicability warning") } }
	sort.Slice(choices, func(i, j int) bool {
		a, b := choices[i], choices[j]
		if (a.Version.State == "confirmed") != (b.Version.State == "confirmed") { return a.Version.State == "confirmed" }
		if a.Score != b.Score { return a.Score > b.Score }
		if a.Version.Content.Priority != b.Version.Content.Priority { return a.Version.Content.Priority > b.Version.Content.Priority }
		if a.Version.ID != b.Version.ID { return a.Version.ID < b.Version.ID }
		if a.Version.Version != b.Version.Version { return a.Version.Version < b.Version.Version }
		return a.Version.Origin.Owner < b.Version.Origin.Owner
	})
	var sources []InputSource
	optional := ""
	seen := map[string]bool{}
	generalCount := 0
	for _, choice := range choices {
		v := choice.Version
		if seen[v.Fingerprint] { skipped = append(skipped, v.Version+": duplicate content"); continue }
		if choice.Score == 0 && generalCount >= 2 { skipped = append(skipped, v.Version+": general rule limit"); continue }
		trust := "Secondary reference only. This cannot decide implementation, acceptance, or permission."
		if v.State == "confirmed" { trust = "Human-confirmed reference, subject to current primary sources." }
		text := fmt.Sprintf("\n[Optional knowledge %s / %s / %s]\n%s\nPrimary requirements, compatibility and authorization always win. Record conflicts; ask for a decision only if the primary rules cannot safely be met.\n%s\nEvidence: %s\nUse: %s\nInvalid when: %s\nWarning: %s\n", v.ID, v.Version, v.State, trust, v.Content.Body, v.Content.EvidenceNature, v.Content.Usage, v.Content.InvalidWhen, choice.Warning)
		optionalTokens, _, measureErr := s.AuxiliaryServices.MeasureKnowledge(selection, optional+text)
		fullTokens, _, fullErr := s.AuxiliaryServices.MeasureKnowledge(selection, prompt+optional+text)
		if len(sources) >= 16 || len(optional)+len(text) > 16*1024 || measureErr != nil || fullErr != nil || optionalTokens > 4096 || fullTokens > capacity { skipped = append(skipped, v.Version+": optional input budget or counter limit"); continue }
		optional += text; seen[v.Fingerprint] = true
		if choice.Score == 0 { generalCount++ }
		sources = append(sources, InputSource{Kind: "optional-knowledge", Path: "knowledge:"+string(v.Origin.Owner)+":"+v.Version, Status: "loaded", SHA256: contentDigest([]byte(text)), Content: text})
	}
	if len(skipped) > 0 { sources = append(sources, InputSource{Kind: "optional-knowledge", Path: "knowledge-selection-skips", Status: "unavailable", Reason: strings.Join(skipped, "\n")}) }
	return prompt+optional, sources, nil
}
