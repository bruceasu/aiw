package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Sources live in the same recoverable commit as the facts they describe.
// Queue projection can fail independently; it cannot undo these facts.
type AuxiliarySource struct {
	ID string `json:"id"`
	Owner TaskID `json:"owner"`
	Version string `json:"version"`
	Kind string `json:"kind"`
	Items []AuxiliarySourceItem `json:"items"`
	References []ActorReference `json:"references"`
	Stop *ExecutionStop `json:"stop,omitempty"`
}

type AuxiliarySourceItem struct {
	ID WorkItemID `json:"id"`
	Title string `json:"title"`
	State WorkItemState `json:"state"`
	Accepted *ActorReference `json:"accepted,omitempty"`
}

type TaskAuxiliary struct {
	Sources []AuxiliarySource `json:"sources"`
	Cursor int `json:"cursor"`
	Jobs []AuxiliaryJob `json:"jobs"`
	Memory *TaskMemory `json:"memory,omitempty"`
	Sealed *ActorReference `json:"sealed,omitempty"`
	HostGap string `json:"host_gap,omitempty"`
	Knowledge *TaskKnowledge `json:"knowledge,omitempty"`
}

type TaskMemory struct {
	SourceVersion string `json:"source_version"`
	Output ActorReference `json:"output"`
	JobKey string `json:"job_key"`
}

func captureAuxiliarySource(state *RuntimeState) {
	if state.Protocol == nil { return }
	p := state.Protocol
	source := AuxiliarySource{Owner: state.Task.ID, Kind: "progress", Stop: p.Stop}
	complete := len(state.WorkItems) > 0
	seen := map[ActorReference]bool{}
	add := func(ref *ActorReference) { if ref != nil && !seen[*ref] { source.References = append(source.References, *ref); seen[*ref] = true } }
	if p.Stop != nil { add(&p.Stop.Reference) }
	for _, item := range state.WorkItems {
		source.Items = append(source.Items, AuxiliarySourceItem{item.ID, item.Title, item.State, item.AcceptedReference})
		add(item.AcceptedReference)
		if item.State != WorkItemCompleted || item.AcceptedReference == nil { complete = false }
	}
	for _, item := range p.Items { add(item.ValidatedReport); add(item.AcceptanceCandidate); add(item.VerifierSnapshot) }
	for _, request := range p.Requests {
		// Merely preparing another Session prompt is not a substantive result.
		// Once a terminal report exists, preserve its exact input with it.
		if request.Result == nil { continue }
		add(&request.Request.Input)
		add(request.Result)
		add(request.Attribution)
	}
	if complete { source.Kind = "completed" }
	body, _ := json.Marshal(source)
	source.Version = contentDigest(body)
	source.ID = string(source.Owner)+":"+source.Version
	if p.Auxiliary == nil { p.Auxiliary = &TaskAuxiliary{} }
	a := p.Auxiliary
	if len(a.Sources) > 0 && a.Sources[len(a.Sources)-1].Version == source.Version { return }
	a.Sources = append(a.Sources, source)
}

func auxiliarySource(state RuntimeState, sourceID string) (AuxiliarySource, error) {
	if state.Protocol != nil && state.Protocol.Auxiliary != nil {
		for _, source := range state.Protocol.Auxiliary.Sources { if source.ID == sourceID { return source, nil } }
	}
	return AuxiliarySource{}, errors.New("fixed Task source is unavailable")
}

// TaskMemoryContext reconstructs a Session projection without changing human
// Session memory. Every raw reference is read and hash checked before dispatch.
// A stale/missing summary degrades to the original facts, never to empty text.
func (s *Store) TaskMemoryContext(id TaskID) (string, string, error) {
	state, err := s.Load(id)
	if err != nil { return "", "", err }
	if state.Protocol == nil { return "", "", errors.New("durable Task sources are unavailable") }
	if state.Protocol.Auxiliary == nil || len(state.Protocol.Auxiliary.Sources) == 0 {
		// Pin the initial source BEFORE a prompt can refer to it. A synthetic
		// in-memory version would disappear when freezing adds the next input.
		state, err = s.updateWithEvent(id, &state.StateRevision, Event{Type: "protocol.auxiliary.source-captured"}, func(current *RuntimeState) error {
			captureAuxiliarySource(current)
			return nil
		})
		if err != nil { return "", "", err }
	}
	a := state.Protocol.Auxiliary
	source := a.Sources[len(a.Sources)-1]
	var text strings.Builder
	text.WriteString("[Task facts: "+source.Version+"]\n\nAccepted work:\n")
	for _, item := range source.Items { if item.Accepted != nil { fmt.Fprintf(&text, "- %s: %s\n", item.ID, item.Title) } }
	text.WriteString("\nIn progress, failed, or unaccepted work:\n")
	for _, item := range source.Items { if item.Accepted == nil { fmt.Fprintf(&text, "- %s (%s): %s\n", item.ID, item.State, item.Title) } }
	if source.Stop != nil { fmt.Fprintf(&text, "\nStop: %s\n", source.Stop.Reason) }
	for _, ref := range source.References {
		var raw json.RawMessage
		if err := s.ReadExecutionArtifact(id, ref, &raw); err != nil { return "", source.Version, fmt.Errorf("required Task source %s: %w", ref.Path, err) }
		if text.Len()+len(raw) > 4*1024*1024 { return "", source.Version, errors.New("complete Task sources exceed the synchronous input budget") }
		fmt.Fprintf(&text, "\n[Original %s %s]\n%s\n", ref.Kind, ref.SHA256, raw)
	}
	if a.Memory != nil && a.Memory.SourceVersion == source.Version {
		var output AuxiliaryOutput
		if err := s.ReadExecutionArtifact(id, a.Memory.Output, &output); err == nil && output.JobKey == a.Memory.JobKey && len(output.Text) <= 64*1024 {
			text.WriteString("\n[Generated Task summary; not authorization or a human decision]\n"+output.Text)
			return text.String(), source.Version, nil
		}
	}
	text.WriteString("\n[Degraded: summary unavailable or stale; complete original Task facts loaded]\n")
	return text.String(), source.Version, nil
}

// AuxiliaryReference is a cross-Task pointer. It never copies a retry balance.
type AuxiliaryReference struct {
	SourceOwner TaskID `json:"source_owner_task_id"`
	Key string `json:"key"`
}

func (s *Store) ResolveAuxiliary(ref AuxiliaryReference) (AuxiliaryJob, error) {
	state, err := s.Load(ref.SourceOwner)
	if err != nil { return AuxiliaryJob{}, fmt.Errorf("auxiliary source ledger unavailable: %w", err) }
	if state.PendingEvent != nil { return AuxiliaryJob{}, errors.New("auxiliary source commit needs reconciliation") }
	return findAuxiliaryJob(state, ref.Key)
}

// ValidateFrozenTaskSources rechecks the ORIGINAL version's required artifacts
// before a frozen request is dispatched. A later summary never rewrites input.
func (s *Store) ValidateFrozenTaskSources(input ExecutionInput) error {
	state, err := s.Load(input.TaskID)
	if err != nil { return err }
	if state.Protocol == nil { return nil }
	if state.PendingEvent != nil { return errors.New("Task source requires pending-event reconciliation") }
	if state.Protocol.Auxiliary == nil || len(state.Protocol.Auxiliary.Sources) == 0 { captureAuxiliarySource(&state) }
	found := false
	for _, source := range input.Sources {
		if source.Kind != "task-memory" { continue }
		found = true
		if source.Status != "loaded" || source.SHA256 != contentDigest([]byte(source.Content)) || !strings.HasPrefix(source.Path, "task-source:") { return errors.New("frozen Task source body is unavailable") }
		version := strings.TrimPrefix(source.Path, "task-source:")
		original, err := auxiliarySource(state, string(input.TaskID)+":"+version)
		if err != nil { return err }
		for _, ref := range original.References {
			var raw json.RawMessage
			if err := s.ReadExecutionArtifact(input.TaskID, ref, &raw); err != nil { return err }
		}
	}
	if !found { return errors.New("frozen generation lacks its Task source version") }
	return nil
}
