package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	AuxiliaryInputTokens = 32768
	AuxiliaryInputBytes = 128*1024
	AuxiliaryOutputTokens = 4096
	AuxiliaryOutputBytes = 64*1024
	auxiliaryControlBytes = 1024*1024
)

// Actual capability and complete-prompt measurement come from a local,
// evidence-backed adapter. Profile names and byte/token guesses are not proofs.
type AuxiliaryCapability struct {
	Selection AISelection `json:"selection"`
	Reference ActorReference `json:"reference"`
	CounterVersion string `json:"counter_version"`
	ContextTokens int64 `json:"context_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	EnforcedOutput bool `json:"enforced_output"`
	ClosedInput bool `json:"closed_input"`
}

type AuxiliaryMeasurement struct {
	Capability AuxiliaryCapability `json:"capability"`
	PromptDigest string `json:"prompt_digest"`
	InputTokens int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

type AuxiliaryResourcePolicy struct {
	Version int `json:"version"`
	Reason string `json:"reason"`
	TaskTokens int64 `json:"task_tokens"`
	TaskDispatches int `json:"task_dispatches"`
	TaskQueue int `json:"task_queue"`
	ProjectQueue int `json:"project_queue"`
	TaskBytes int64 `json:"task_bytes"`
	ProjectBytes int64 `json:"project_bytes"`
	FreeBytes int64 `json:"free_bytes"`
}

func DefaultAuxiliaryResourcePolicy() AuxiliaryResourcePolicy {
	return AuxiliaryResourcePolicy{1, "R3 initial policy", 524288, 64, 128, 1024, 512*1024*1024, 4*1024*1024*1024, 256*1024*1024}
}

// AuxiliaryServices perform only read-only local checks while Core holds the
// resource lock. No model, process launch, network, or Task mutation is allowed.
type AuxiliaryServices struct {
	// Counts the complete receiving prompt using the exact model's locally
	// reviewed counter. Capacity excludes the receiving output reservation.
	MeasureKnowledge func(*AISelection, string) (int64, int64, error)
	// StartHost must only start a detached managed process and return. It is
	// called after the Task lock is released, never as a foreground worker.
	StartHost func(TaskID) error
	ReportGap func(TaskID, error)
	FreeBytes func(string) (int64, error)
	Authorize func(RuntimeState, AuxiliaryJob) error
	VerifyCapability func(AuxiliaryCapability) error
	VerifyObservation func(RuntimeState, AuxiliaryReservation, AuxiliaryObservation) error
	VerifyInventory func(AuxiliaryInventory) error
}

type AuxiliaryInventory struct {
	Known bool `json:"known"`
	Evidence ActorReference `json:"evidence"`
	TaskBytes map[TaskID]int64 `json:"task_bytes"`
	ProjectBytes int64 `json:"project_bytes"`
}

type AuxiliaryReservation struct {
	ID string `json:"id"`
	Key string `json:"key"`
	Owner TaskID `json:"owner"`
	Sponsor TaskID `json:"sponsor"`
	PolicyDigest string `json:"policy_digest"`
	Measurement AuxiliaryMeasurement `json:"measurement"`
	Tokens int64 `json:"tokens"`
	Released bool `json:"released"`
	Terminal bool `json:"terminal"`
	Executor string `json:"executor"`
	Evidence []ActorReference `json:"evidence,omitempty"`
}

type AuxiliaryQueueReservation struct {
	Key string `json:"key"`
	Owner TaskID `json:"owner"`
	Sponsor TaskID `json:"sponsor"`
	SourceID string `json:"source_id"`
	Kind string `json:"kind"`
	InputDigest string `json:"input_digest"`
	PeakBytes int64 `json:"peak_bytes"`
	Terminal bool `json:"terminal"`
	KnowledgeTags []string `json:"knowledge_tags,omitempty"`
}

type auxiliaryResources struct {
	Version int `json:"version"`
	Inventory AuxiliaryInventory `json:"inventory"`
	Policies map[TaskID]string `json:"policies"`
	PolicyHistory map[string]AuxiliaryResourcePolicy `json:"policy_history"`
	Queue map[string]AuxiliaryQueueReservation `json:"queue"`
	Reservations map[string]AuxiliaryReservation `json:"reservations"`
	Active string `json:"active,omitempty"`
	KnowledgeDecisions map[string]KnowledgeDecisionPointer `json:"knowledge_decisions,omitempty"`
	KnowledgeStorage map[TaskID]int64 `json:"knowledge_storage,omitempty"`
	HumanKnowledge map[string]KnowledgeHumanIndex `json:"human_knowledge,omitempty"`
}

func (s *Store) auxiliaryProjectLock() (*os.File, error) {
	if !durablePlatformSupported() { return nil, errors.New("auxiliary durable storage is unavailable") }
	path := filepath.Join(s.Root, "locks", "auxiliary-resources.lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { return nil, err }
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil { return nil, err }
	if err := systemTaskLock(f); err != nil { _ = f.Close(); return nil, err }
	return f, nil
}

func (s *Store) readAuxiliaryResources() (auxiliaryResources, error) {
	var r auxiliaryResources
	data, err := os.ReadFile(filepath.Join(s.Root, "auxiliary-resources.json"))
	if err != nil { return r, fmt.Errorf("auxiliary resource history is unknown: %w", err) }
	if err := json.Unmarshal(data, &r); err != nil { return r, err }
	if r.Version != 1 || !r.Inventory.Known || r.Queue == nil || r.Reservations == nil || r.Policies == nil || r.PolicyHistory == nil { return r, errors.New("auxiliary resource history is incomplete") }
	return r, nil
}

func (s *Store) saveAuxiliaryResources(r auxiliaryResources) error {
	data, err := json.Marshal(r)
	if err != nil { return err }
	// Control headroom is part of the persistent reservation, not extra space.
	if len(data)*2 > auxiliaryControlBytes { return errors.New("auxiliary resource control headroom exhausted") }
	if s.AuxiliaryServices == nil || s.AuxiliaryServices.FreeBytes == nil { return errors.New("volume capacity is unknown") }
	free, err := s.AuxiliaryServices.FreeBytes(s.Root)
	if err != nil { return err }
	if free-int64(len(data)*2) < 256*1024*1024 { return errors.New("insufficient volume space for auxiliary control commit") }
	return durableWrite(filepath.Join(s.Root, "auxiliary-resources.json"), data)
}

// InitializeAuxiliaryResources is a maintenance boundary. Missing ledgers on
// restart cannot be treated as zero usage. The adapter must verify the complete
// existing inventory and absence of old dispatches before initialization.
func (s *Store) InitializeAuxiliaryResources(inventory AuxiliaryInventory) error {
	if s.AuxiliaryServices == nil || s.AuxiliaryServices.VerifyInventory == nil || !inventory.Known || !validProtocolReference(inventory.Evidence) || inventory.TaskBytes == nil || inventory.ProjectBytes < 0 { return errors.New("verified auxiliary inventory is required") }
	lock, err := s.auxiliaryProjectLock()
	if err != nil { return err }
	defer systemTaskUnlock(lock)
	if _, err := os.Stat(filepath.Join(s.Root, "auxiliary-resources.json")); !errors.Is(err, os.ErrNotExist) { return errors.New("existing or unreadable resource ledger must be preserved") }
	if err := s.AuxiliaryServices.VerifyInventory(inventory); err != nil { return err }
	var total int64
	for id, size := range inventory.TaskBytes {
		if validateTaskID(id) != nil || size < 0 || size > 4*1024*1024*1024 { return errors.New("invalid auxiliary inventory") }
		total += size
	}
	if total > inventory.ProjectBytes { return errors.New("project inventory omits Task bytes") }
	if inventory.Evidence.Kind == "auxiliary-inventory" && inventory.Evidence.Path == "auxiliary-inventory.json" {
		inventory, err = s.freezeLocalAuxiliaryInventory(inventory)
		if err != nil { return err }
	}
	r := auxiliaryResources{Version: 1, Inventory: inventory, Policies: map[TaskID]string{}, PolicyHistory: map[string]AuxiliaryResourcePolicy{}, Queue: map[string]AuxiliaryQueueReservation{}, Reservations: map[string]AuxiliaryReservation{}}
	return s.saveAuxiliaryResources(r)
}

func (s *Store) auxiliaryPolicy() (AuxiliaryResourcePolicy, string, error) {
	var p AuxiliaryResourcePolicy
	data, err := os.ReadFile(filepath.Join(s.Root, "resource-policy.json"))
	if err != nil { return p, "", err }
	if err := json.Unmarshal(data, &p); err != nil { return p, "", err }
	if err := validateAuxiliaryPolicy(p); err != nil { return p, "", err }
	return p, contentDigest(data), nil
}

func validateAuxiliaryPolicy(p AuxiliaryResourcePolicy) error {
	max := DefaultAuxiliaryResourcePolicy()
	if p.Version < 1 || p.Reason == "" || p.TaskTokens <= 0 || p.TaskTokens > max.TaskTokens || p.TaskDispatches <= 0 || p.TaskDispatches > max.TaskDispatches || p.TaskQueue <= 0 || p.TaskQueue > max.TaskQueue || p.ProjectQueue <= 0 || p.ProjectQueue > max.ProjectQueue || p.TaskBytes <= auxiliaryControlBytes || p.TaskBytes > max.TaskBytes || p.ProjectBytes <= auxiliaryControlBytes || p.ProjectBytes > max.ProjectBytes || p.FreeBytes < max.FreeBytes { return errors.New("resource policy is missing limits or exceeds the approved R3 ceiling") }
	return nil
}

func freezeAuxiliaryPolicy(r *auxiliaryResources, id TaskID, p AuxiliaryResourcePolicy, digest string) error {
	if oldDigest := r.Policies[id]; oldDigest != "" && oldDigest != digest {
		old, ok := r.PolicyHistory[oldDigest]
		if !ok || p.Version <= old.Version || p.Reason == "" { return errors.New("resource policy changes need a newer version and a reason") }
	}
	r.Policies[id], r.PolicyHistory[digest] = digest, p
	return nil
}

func (s *Store) reserveAuxiliaryQueue(r *auxiliaryResources, job AuxiliaryJob, source AuxiliarySource) error {
	if old, ok := r.Queue[job.Key]; ok {
		if old.Owner != source.Owner || old.InputDigest != job.InputDigest || old.SourceID != source.ID { return errors.New("auxiliary fixed key conflict") }
		return nil
	}
	p, digest, err := s.auxiliaryPolicy()
	if err != nil { return err }
	if _, known := r.Inventory.TaskBytes[source.Owner]; !known { return errors.New("source Task auxiliary storage baseline is unknown") }
	if _, known := r.Inventory.TaskBytes[job.Sponsor]; !known { return errors.New("sponsor Task auxiliary history is unknown") }
	if err := freezeAuxiliaryPolicy(r, job.Sponsor, p, digest); err != nil { return err }
	projectQueue, taskQueue := 0, 0
	projectBytes := r.Inventory.ProjectBytes+auxiliaryControlBytes
	taskBytes := r.Inventory.TaskBytes[source.Owner]+auxiliaryControlBytes
	for owner, size := range r.KnowledgeStorage { projectBytes += size; if owner == source.Owner { taskBytes += size } }
	for _, q := range r.Queue {
		projectBytes += q.PeakBytes
		if q.Owner == source.Owner { taskBytes += q.PeakBytes }
		if !q.Terminal { projectQueue++; if q.Owner == source.Owner { taskQueue++ } }
	}
	// Keep conservative peak reservations, including crash orphans, until a
	// separately verified inventory can settle them. Never infer deletion.
	body, _ := json.Marshal(job)
	peak := int64(len(body)*2+24*AuxiliaryOutputBytes+256*1024)
	if taskQueue >= p.TaskQueue || projectQueue >= p.ProjectQueue || taskBytes+peak > p.TaskBytes || projectBytes+peak > p.ProjectBytes { return errors.New("auxiliary queue or storage limit reached; retain the source cursor") }
	if s.AuxiliaryServices == nil || s.AuxiliaryServices.FreeBytes == nil { return errors.New("volume capacity is unknown") }
	free, err := s.AuxiliaryServices.FreeBytes(s.Root)
	if err != nil { return err }
	// Existing reservations may not yet have reached disk.
	if free-(projectBytes-r.Inventory.ProjectBytes)-peak < p.FreeBytes { return errors.New("auxiliary write reservation would exhaust volume headroom") }
	r.Queue[job.Key] = AuxiliaryQueueReservation{Key: job.Key, Owner: source.Owner, Sponsor: job.Sponsor, SourceID: source.ID, Kind: job.Kind, InputDigest: job.InputDigest, PeakBytes: peak}
	return s.saveAuxiliaryResources(*r)
}

func (s *Store) reserveAuxiliaryCall(r *auxiliaryResources, job AuxiliaryJob, executor string, m AuxiliaryMeasurement, now, deadline time.Time) (AuxiliaryReservation, error) {
	var reservation AuxiliaryReservation
	cap := m.Capability
	if now.Add(120*time.Second).After(deadline) { return reservation, errors.New("insufficient bounded host time") }
	if s.AuxiliaryServices == nil || s.AuxiliaryServices.VerifyCapability == nil { return reservation, errors.New("capability-unavailable") }
	if cap.Selection.Provider == "" || cap.Selection.Model == "" || cap.Selection.Digest == "" || !validProtocolReference(cap.Reference) || cap.CounterVersion == "" || !cap.EnforcedOutput || !cap.ClosedInput || cap.ContextTokens <= 0 || cap.OutputTokens <= 0 { return reservation, errors.New("capability-unavailable") }
	if err := s.AuxiliaryServices.VerifyCapability(cap); err != nil { return reservation, err }
	if m.PromptDigest != job.InputDigest || m.InputTokens <= 0 || m.InputTokens > AuxiliaryInputTokens || m.OutputTokens <= 0 || m.OutputTokens > AuxiliaryOutputTokens || m.OutputTokens > cap.OutputTokens || m.InputTokens+m.OutputTokens > cap.ContextTokens || len(job.Prompt) > AuxiliaryInputBytes { return reservation, errors.New("auxiliary complete input or output allowance exceeds capability") }
	if job.Model != nil && *job.Model != cap.Selection { return reservation, errors.New("auxiliary recovery cannot change model") }
	if r.Active != "" { return reservation, errors.New("project auxiliary slot needs reconciliation") }
	p, digest, err := s.auxiliaryPolicy()
	if err != nil { return reservation, err }
	if err := freezeAuxiliaryPolicy(r, job.Sponsor, p, digest); err != nil { return reservation, err }
	var tokens int64
	count := 0
	for _, call := range r.Reservations { if call.Sponsor == job.Sponsor && !call.Released { tokens += call.Tokens; count++ } }
	if count >= p.TaskDispatches || tokens+m.InputTokens+m.OutputTokens > p.TaskTokens { return reservation, errors.New("Task auxiliary budget exhausted") }
	id := fmt.Sprintf("%s:%d", job.Key, len(job.Calls)+1)
	if _, exists := r.Reservations[id]; exists { return reservation, errors.New("existing dispatch intent must be reconciled") }
	reservation = AuxiliaryReservation{ID: id, Key: job.Key, Owner: job.Owner, Sponsor: job.Sponsor, PolicyDigest: digest, Measurement: m, Tokens: m.InputTokens+m.OutputTokens, Executor: executor}
	r.Reservations[id], r.Active = reservation, id
	if err := s.saveAuxiliaryResources(*r); err != nil { return AuxiliaryReservation{}, err }
	return reservation, nil
}
