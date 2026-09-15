package workflow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const CompilerFailureLimit = MaxConsecutiveCompileFailures

// SelectCompileTargets preserves plan order. Shared, unknown, or absent paths
// conservatively select the entire plan, including unavailable targets.
func SelectCompileTargets(plan CompilePlan, changed []string) []CompileTarget {
	if len(changed) == 0 { return append([]CompileTarget(nil), plan.Targets...) }
	selected := make(map[int]bool)
	for _, candidate := range changed {
		candidate = filepath.ToSlash(candidate)
		matched := false
		for index, target := range plan.Targets {
			for _, owner := range target.PathOwners {
				owner = strings.TrimSuffix(filepath.ToSlash(owner), "/")
				if owner == "." || owner == "" { return append([]CompileTarget(nil), plan.Targets...) }
				if candidate == owner || strings.HasPrefix(candidate, owner+"/") {
					selected[index], matched = true, true
				}
			}
		}
		if !matched { return append([]CompileTarget(nil), plan.Targets...) }
	}
	var targets []CompileTarget
	for index, target := range plan.Targets { if selected[index] { targets = append(targets, target) } }
	return targets
}

// ResolveCompileTarget reconstructs commands from the local registry. Plan
// command/args fields are descriptive; they cannot introduce an executable.
func ResolveCompileTarget(workspace string, target CompileTarget) (CompileAdapter, error) {
	switch target.Kind {
	case "script":
		if strings.TrimSpace(target.Script) == "" || filepath.IsAbs(target.Script) || strings.ContainsAny(target.Script, "\x00\r\n\"&|<>^%!:") {
			return CompileAdapter{}, errors.New("compile target requires a repository-relative script")
		}
		root, err := filepath.EvalSymlinks(workspace)
		if err != nil { return CompileAdapter{}, err }
		root, err = filepath.Abs(root)
		if err != nil { return CompileAdapter{}, err }
		path, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(target.Script)))
		if err != nil { return CompileAdapter{}, err }
		relative, err := filepath.Rel(root, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			return CompileAdapter{}, errors.New("compile script escapes the repository")
		}
		info, err := os.Stat(path)
		if err != nil { return CompileAdapter{}, err }
		if !info.Mode().IsRegular() { return CompileAdapter{}, errors.New("compile script is not a regular file") }
		return CompileAdapter{Kind: "script", Name: target.Name, Path: path, Command: path}, nil
	case "go":
		if info, err := os.Stat(filepath.Join(workspace, "go.mod")); err != nil || info.IsDir() {
			return CompileAdapter{}, errors.New("built-in Go adapter requires go.mod; add a repository compile script for nested modules")
		}
		return CompileAdapter{Kind: "go", Name: target.Name, Command: "go", Args: []string{"build", "-o", os.DevNull, "./..."}}, nil
	default:
		return CompileAdapter{}, fmt.Errorf("no built-in adapter for %q; add or repair a repository compile script", target.Kind)
	}
}

func (s *Store) OpenCompilerGate(id TaskID, item WorkItemID, code, reason string) (RuntimeState, error) {
	return s.UpdateWithEvent(id, Event{Type: "compiler.gated", WorkItemID: item, Detail: code}, func(state *RuntimeState) error {
		gateID := GateID(code + "-" + string(item))
		for index := range state.Gates {
			if state.Gates[index].ID == gateID {
				state.Gates[index].State, state.Gates[index].Reason = GateOpen, reason
				return nil
			}
		}
		state.Gates = append(state.Gates, Gate{ID: gateID, WorkItemID: item, Kind: GateValidation, State: GateOpen, Reason: reason})
		return nil
	})
}

// RecordCompilerRequestOutcome persists the result and both compiler counters together.
// Recovery can reuse a result without charging another failure for its request.
func (s *Store) RecordCompilerRequestOutcome(id TaskID, request CompilerRequest, result CompilerResult) (RuntimeState, error) {
	if err := request.Validate(); err != nil { return RuntimeState{}, err }
	if err := result.Validate(); err != nil { return RuntimeState{}, err }
	if request.TaskID != id || result.TaskID != id || result.RequestID != request.ID || result.AttemptID != request.AttemptID || result.WorkItemID != request.WorkItemID { return RuntimeState{}, errors.New("compiler result identity does not match its request") }
	return s.recordCompilerOutcome(id, request.WorkItemID, result.Diagnostics, result.Status == ActorResultAccepted, &result)
}

// recordPreparedCompilerOutcome updates both recovery views in the same Store
// transaction as the Work Item counter. Blocked targets consume no repair budget.
func recordPreparedCompilerOutcome(state *RuntimeState, result *CompilerResult) (bool, error) {
	prepared := state.Automation.PreparedRequest
	if prepared == nil || prepared.TaskID != result.TaskID || prepared.WorkItemID != result.WorkItemID || prepared.AttemptID != result.AttemptID || prepared.Compile == nil || prepared.Compile.Request == nil || prepared.Compile.Request.ID != result.RequestID {
		return false, errors.New("compiler result does not match the prepared request")
	}
	compile := prepared.Compile
	if compile.Result != nil && compile.Result.RequestID == result.RequestID {
		return false, nil
	}
	compile.Result, prepared.CompilerResult = result, result
	if result.Status == ActorResultAccepted {
		compile.Failures = 0
	} else if result.Status == ActorResultFailed {
		compile.Failures++
	}
	if result.Status == ActorResultFailed && compile.Failures >= CompilerFailureLimit {
		state.Gates = append(state.Gates, Gate{ID: GateID("compiler-repair-limit-" + string(result.AttemptID)), WorkItemID: result.WorkItemID, Kind: GateValidation, State: GateOpen, Reason: "three consecutive compiler failures; inspect diagnostics before preparing further work"})
	}
	return true, nil
}
