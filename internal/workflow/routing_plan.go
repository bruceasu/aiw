package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const RoutingPlanSchemaVersion = 1

// RoutingPlan is the reviewable, task-local recommendation consumed by
// supervised execution. Compile targets are selected locally; no LLM output
// can introduce a command into this plan.
type RoutingPlan struct {
	SchemaVersion int               `json:"schema_version"`
	TaskID        TaskID            `json:"task_id"`
	Source        string            `json:"source"`
	Profiles      map[string]string `json:"profiles"`
	Compile       CompilePlan       `json:"compile_plan"`
	RecordedAt    string            `json:"recorded_at"`
}

type CompilePlan struct {
	Available bool            `json:"available"`
	Targets   []CompileTarget `json:"targets,omitempty"`
	Reason    string          `json:"reason,omitempty"`
}

// CompileTarget holds only a repository-relative script or a registered
// built-in adapter. Later compiler execution must resolve it again locally.
type CompileTarget struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	Script     string   `json:"script,omitempty"`
	Command    string   `json:"command,omitempty"`
	Args       []string `json:"args,omitempty"`
	PathOwners []string `json:"path_owners"`
}

func DefaultRoutingProfiles() map[string]string {
	return map[string]string{"analysis": "fast", "coder": "balanced", "tester": "balanced", "verifier": "reasoning"}
}

func CompilePlanForWorkspace(workspace string) CompilePlan {
	adapter, err := SelectCompileAdapter(workspace)
	if err != nil { return CompilePlan{Reason: err.Error()} }
	target := CompileTarget{Name: adapter.Name, Kind: adapter.Kind, Command: adapter.Command, Args: append([]string(nil), adapter.Args...), PathOwners: []string{"."}}
	if adapter.Kind == "script" { target.Script, target.Command, target.Args = filepath.ToSlash(adapter.Name), "", nil }
	return CompilePlan{Available: true, Targets: []CompileTarget{target}}
}

func (p RoutingPlan) Validate() error {
	if p.SchemaVersion != RoutingPlanSchemaVersion || p.TaskID == "" || (p.Source != "llm" && p.Source != "defaults") || strings.TrimSpace(p.RecordedAt) == "" { return fmt.Errorf("routing plan requires schema, Task, source, and recorded time") }
	for actor := range DefaultRoutingProfiles() {
		if strings.TrimSpace(p.Profiles[actor]) == "" { return fmt.Errorf("routing plan profile for %s is required", actor) }
	}
	if !p.Compile.Available {
		if strings.TrimSpace(p.Compile.Reason) == "" { return fmt.Errorf("unavailable compile plan requires a reason") }
		return nil
	}
	if len(p.Compile.Targets) == 0 { return fmt.Errorf("available compile plan requires a target") }
	for _, target := range p.Compile.Targets {
		if strings.TrimSpace(target.Name) == "" || len(target.PathOwners) == 0 { return fmt.Errorf("compile target requires a name and path ownership") }
		if target.Kind == "script" && strings.TrimSpace(target.Script) == "" { return fmt.Errorf("script compile target requires a repository script") }
		if target.Kind != "script" && strings.TrimSpace(target.Command) == "" { return fmt.Errorf("built-in compile target requires a command") }
	}
	return nil
}

// PersistRoutingPlan writes an inspectable task-local artifact and records its
// digest in the event log. It is advisory: no review flag blocks supervision.
func (s *Store) PersistRoutingPlan(id TaskID, plan RoutingPlan) (RuntimeState, string, error) {
	if plan.TaskID != id { return RuntimeState{}, "", fmt.Errorf("routing plan Task does not match runtime Task") }
	if err := plan.Validate(); err != nil { return RuntimeState{}, "", err }
	content, err := json.MarshalIndent(plan, "", "  ")
	if err != nil { return RuntimeState{}, "", fmt.Errorf("encode routing plan: %w", err) }
	content = append(content, '\n')
	const relative = "routing-plan.json"
	if err := atomicWrite(s.path(id, relative), content); err != nil { return RuntimeState{}, "", err }
	digest := sha256.Sum256(content)
	state, err := s.UpdateWithEvent(id, Event{Type: "routing-plan.persisted", Detail: relative + "#" + hex.EncodeToString(digest[:])}, func(*RuntimeState) error { return nil })
	if err != nil { return RuntimeState{}, "", err }
	return state, relative, nil
}

// LoadRoutingPlan returns the Task-local advisory routing plan after
// validating its durable representation.
func (s *Store) LoadRoutingPlan(id TaskID) (RoutingPlan, error) {
	content, err := os.ReadFile(s.path(id, "routing-plan.json"))
	if err != nil {
		return RoutingPlan{}, err
	}
	var plan RoutingPlan
	if err := json.Unmarshal(content, &plan); err != nil {
		return RoutingPlan{}, fmt.Errorf("decode routing plan: %w", err)
	}
	if plan.TaskID != id {
		return RoutingPlan{}, fmt.Errorf("routing plan Task does not match runtime Task")
	}
	if err := plan.Validate(); err != nil {
		return RoutingPlan{}, err
	}
	return plan, nil
}

func NewRoutingPlan(id TaskID, source string, profiles map[string]string, compile CompilePlan) RoutingPlan {
	copyProfiles := DefaultRoutingProfiles()
	for actor := range copyProfiles {
		if profile := strings.TrimSpace(profiles[actor]); profile != "" { copyProfiles[actor] = profile }
	}
	return RoutingPlan{SchemaVersion: RoutingPlanSchemaVersion, TaskID: id, Source: source, Profiles: copyProfiles, Compile: compile, RecordedAt: time.Now().UTC().Format(time.RFC3339)}
}

// Reference uses the same canonical encoding as PersistRoutingPlan. The full
// compile snapshot is embedded in the request so later plan edits are harmless.
func (p RoutingPlan) Reference() ActorReference {
	content, _ := json.MarshalIndent(p, "", "  ")
	digest := sha256.Sum256(append(content, '\n'))
	return ActorReference{Kind: "routing-plan", Path: "routing-plan.json", SHA256: hex.EncodeToString(digest[:])}
}
