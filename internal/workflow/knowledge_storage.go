package workflow

import "errors"

// Keep every knowledge transition inside the schema-10 conditional protocol.
func (s *Store) updateKnowledge(id TaskID, event Event, change func(*RuntimeState) error) (RuntimeState, error) {
	state, err := s.Load(id)
	if err != nil { return RuntimeState{}, err }
	if state.SchemaVersion != DurableSchemaVersion || state.Protocol == nil { return RuntimeState{}, errors.New("knowledge requires the durable execution protocol") }
	return s.updateWithEvent(id, &state.StateRevision, event, change)
}

// Fixed conservative headroom covers the bounded 512 KiB projection, copies,
// human-source artifacts and review events. It is never reset by settlement.
const knowledgeStorageBytes int64 = 4*1024*1024

func (s *Store) reserveKnowledgeStorage(r *auxiliaryResources, id TaskID) error {
	p, digest, err := s.auxiliaryPolicy(); if err != nil { return err }
	if _, ok := r.Inventory.TaskBytes[id]; !ok { return errors.New("knowledge storage baseline is unknown") }
	if err := freezeAuxiliaryPolicy(r, id, p, digest); err != nil { return err }
	projectBytes := r.Inventory.ProjectBytes+auxiliaryControlBytes
	taskBytes := r.Inventory.TaskBytes[id]+auxiliaryControlBytes
	for _, q := range r.Queue { projectBytes += q.PeakBytes; if q.Owner == id { taskBytes += q.PeakBytes } }
	for owner, size := range r.KnowledgeStorage { projectBytes += size; if owner == id { taskBytes += size } }
	extra := knowledgeStorageBytes
	if r.KnowledgeStorage[id] == knowledgeStorageBytes { extra = 0 }
	if projectBytes+extra > p.ProjectBytes || taskBytes+extra > p.TaskBytes { return errors.New("knowledge storage limit reached; retain history") }
	if s.AuxiliaryServices == nil || s.AuxiliaryServices.FreeBytes == nil { return errors.New("knowledge volume capacity is unknown") }
	free, err := s.AuxiliaryServices.FreeBytes(s.Root); if err != nil { return err }
	if free-(projectBytes-r.Inventory.ProjectBytes)-extra < p.FreeBytes { return errors.New("insufficient volume headroom for knowledge") }
	if extra == 0 { return nil }
	if r.KnowledgeStorage == nil { r.KnowledgeStorage = map[TaskID]int64{} }
	r.KnowledgeStorage[id] = knowledgeStorageBytes
	return s.saveAuxiliaryResources(*r)
}

func (s *Store) prepareKnowledgeStorage(id TaskID) error {
	lock, err := s.auxiliaryProjectLock(); if err != nil { return err }; defer systemTaskUnlock(lock)
	r, err := s.readAuxiliaryResources(); if err != nil { return err }
	return s.reserveKnowledgeStorage(&r, id)
}
