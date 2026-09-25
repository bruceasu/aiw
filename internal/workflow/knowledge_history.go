package workflow

import "errors"

// Called only after the consumer has charged a relevant, fixed historical
// source. Reuse the first source for each acceptance, including its retry
// ledger, even when another Task is the first sponsor after an upgrade.
func (s *Store) backfillKnowledgeSource(consumer TaskID, historical KnowledgeHistorySource) error {
	state, err := s.Load(historical.Owner); if err != nil { return err }
	if state.Protocol == nil || state.Protocol.Auxiliary == nil { return errors.New("historical source ledger is unavailable") }
	selected, err := auxiliarySource(state, historical.SourceID); if err != nil { return err }
	if err := s.prepareKnowledgeStorage(historical.Owner); err != nil { return err }
	for _, item := range selected.Items {
		if item.Accepted == nil { continue }
		key := extractionIdentity(item)
		current, err := s.Load(historical.Owner); if err != nil { return err }
		k := taskKnowledge(current.Protocol.Auxiliary)
		if k.Extractions[key] != "" { continue }
		first := ""
		for _, source := range current.Protocol.Auxiliary.Sources {
			for _, candidate := range source.Items { if candidate.Accepted != nil && extractionIdentity(candidate) == key { first = source.ID; break } }
			if first != "" { break }
		}
		if first == "" { return errors.New("historical acceptance source is missing") }
		ref, err := s.EnqueueAuxiliary(historical.Owner, consumer, first, "knowledge-extraction", knowledgeInstruction("knowledge-extraction", knowledgeJobContext{WorkItem: item.ID, Acceptance: item.Accepted}))
		if err != nil { return err }
		_, err = s.updateKnowledge(historical.Owner, Event{Type: "protocol.knowledge.history-linked", Detail: ref.Key}, func(current *RuntimeState) error {
			k := taskKnowledge(current.Protocol.Auxiliary)
			if old := k.Extractions[key]; old != "" && old != ref.Key { return errors.New("historical acceptance has another fixed extraction") }
			k.Extractions[key] = ref.Key
			return nil
		})
		if err != nil { return err }
	}
	return nil
}
