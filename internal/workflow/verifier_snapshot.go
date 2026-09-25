package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// VerifierSnapshot contains only fixed source bytes, never a live workspace
// pointer for the background reader. The diff is explicitly against Git HEAD
// at acceptance, including staged, unstaged and untracked files in scope.
type VerifierSnapshot struct {
	Version int `json:"version"`
	TaskID TaskID `json:"task_id"`
	WorkItemID WorkItemID `json:"work_item_id"`
	AttemptID AttemptID `json:"attempt_id"`
	Candidate AcceptanceCandidate `json:"candidate"`
	Sources []InputSource `json:"sources"`
	Baseline string `json:"baseline"`
	Diff InputSource `json:"diff"`
	Untracked []InputSource `json:"untracked"`
	Criteria []VerifierCriterion `json:"criteria"`
	Artifacts []verifierArtifact `json:"artifacts"`
}

type VerifierCriterion struct {
	ID string `json:"id"`
	SourceSHA256 string `json:"source_sha256"`
	Text string `json:"text"`
}

type verifierArtifact struct {
	Reference ActorReference `json:"reference"`
	Body json.RawMessage `json:"body"`
}

type verifierBuffer struct { buffer bytes.Buffer }

func (b *verifierBuffer) Write(p []byte) (int, error) {
	if b.buffer.Len()+len(p) > AuxiliaryInputBytes { return 0, errors.New("Verifier snapshot exceeds complete-input byte limit") }
	return b.buffer.Write(p)
}

func verifierGit(ctx context.Context, root string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"--no-optional-locks", "--literal-pathspecs", "-C", root}, args...)...)
	cmd.WaitDelay = time.Second
	var output, diagnostic verifierBuffer
	cmd.Stdout, cmd.Stderr = &output, &diagnostic
	if err := cmd.Run(); err != nil { return nil, fmt.Errorf("Verifier read-only Git snapshot: %w", err) }
	return output.buffer.Bytes(), nil
}

func verifierCriteria(source InputSource) []VerifierCriterion {
	// Requirement/scenario headings define a fixed coverage inventory. For
	// prose sources without these headings, the entire document is one unit.
	var result []VerifierCriterion
	var block strings.Builder
	flush := func() {
		body := strings.TrimSpace(block.String())
		if body != "" { result = append(result, VerifierCriterion{fmt.Sprintf("%s:%s:%d", contentDigest([]byte(source.Path)), source.SHA256, len(result)+1), source.SHA256, body}) }
		block.Reset()
	}
	for _, line := range strings.Split(source.Content, "\n") {
		if strings.HasPrefix(line, "### Requirement:") || strings.HasPrefix(line, "#### Scenario:") { flush() }
		block.WriteString(line+"\n")
	}
	flush()
	return result
}

func (s *Store) captureVerifierSnapshot(state RuntimeState, c AcceptanceCandidate) (VerifierSnapshot, error) {
	v := VerifierSnapshot{Version: 1, TaskID: state.Task.ID, WorkItemID: c.WorkItemID, AttemptID: c.AttemptID, Candidate: c}
	origin, err := stageRecord(&state, c.RequestID)
	if err != nil { return v, err }
	var input ExecutionInput
	if err := s.ReadExecutionArtifact(state.Task.ID, origin.Request.Input, &input); err != nil { return v, err }
	root := executionWorkspace(state)
	if !filepath.IsAbs(root) { root = filepath.Join(filepath.Dir(s.Root), filepath.FromSlash(root)) }
	bytesUsed := 0
	account := func(size int) error {
		bytesUsed += size
		if bytesUsed > AuxiliaryInputBytes { return errors.New("Verifier complete source bytes exceed 128 KiB; source was not truncated") }
		return nil
	}
	for _, source := range input.Sources {
		switch source.Kind { case "task-source", "primary-reference", "requirement", "requirements", "acceptance": default: continue }
		if source.Status != "loaded" || strings.TrimSpace(source.Content) == "" || source.SHA256 != contentDigest([]byte(source.Content)) { return v, errors.New("Verifier original requirement body is unavailable") }
		expected := source.SHA256
		for _, file := range c.Inputs.Files { if file.Path == source.Path { expected = file.SHA256; break } }
		current := ReadInputSource(root, InputSource{Kind: source.Kind, Path: source.Path, ExpectedSHA256: expected, Required: true})
		if current.Status != "loaded" { return v, errors.New("Verifier requirement version changed before acceptance") }
		if err := account(len(current.Content)*2); err != nil { return v, err }
		v.Sources = append(v.Sources, current)
		v.Criteria = append(v.Criteria, verifierCriteria(current)...)
	}
	if len(v.Criteria) == 0 { return v, errors.New("Verifier has no original requirements or acceptance inventory") }
	seen := map[ActorReference]bool{}
	add := func(ref ActorReference) error {
		if seen[ref] { return nil }
		var raw json.RawMessage
		if err := s.ReadExecutionArtifact(state.Task.ID, ref, &raw); err != nil { return err }
		if err := account(len(raw)); err != nil { return err }
		seen[ref] = true
		v.Artifacts = append(v.Artifacts, verifierArtifact{ref, raw})
		return nil
	}
	for _, ref := range []ActorReference{c.Report, c.Plan, c.Policy} { if err := add(ref); err != nil { return v, err } }
	if c.TestsRequired { if err := add(c.Grant); err != nil { return v, err } }
	for _, id := range append([]string{c.CompileRunID}, c.TestRunIDs...) {
		record, err := stageRecord(&state, id)
		if err != nil || record.Result == nil { return v, errors.New("Verifier controlled run evidence is absent") }
		if err := add(*record.Result); err != nil { return v, err }
		var result StageResult
		if err := s.ReadExecutionArtifact(state.Task.ID, *record.Result, &result); err != nil { return v, err }
		for _, ref := range result.Evidence { if err := add(ref); err != nil { return v, err } }
		receipt, err := s.ReadVerificationReceipt(state.Task.ID, id)
		if err != nil { return v, err }
		body, err := json.Marshal(receipt)
		if err != nil { return v, err }
		if err := account(len(body)); err != nil { return v, err }
		v.Artifacts = append(v.Artifacts, verifierArtifact{ActorReference{Kind: "verification-receipt", Path: "reports/verification-host/"+contentDigest([]byte(id))+".json", SHA256: contentDigest(body)}, body})
		for _, check := range receipt.Checks { if err := add(check.Output); err != nil { return v, err } }
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	head, err := verifierGit(ctx, root, "rev-parse", "--verify", "HEAD")
	if err != nil { return v, err }
	v.Baseline = strings.TrimSpace(string(head))
	args := append([]string{"diff", "--binary", "--no-ext-diff", "--no-textconv", v.Baseline, "--"}, c.Inputs.Scope...)
	diff, err := verifierGit(ctx, root, args...)
	if err != nil { return v, err }
	if !utf8.Valid(diff) { return v, errors.New("Verifier diff is not representable as complete UTF-8") }
	if err := account(len(diff)); err != nil { return v, err }
	v.Diff = InputSource{Kind: "accepted-diff", Path: "git:"+v.Baseline, Required: true, Status: "loaded", Content: string(diff), SHA256: contentDigest(diff)}
	names, err := verifierGit(ctx, root, append([]string{"ls-files", "--others", "--exclude-standard", "-z", "--"}, c.Inputs.Scope...)...)
	if err != nil { return v, err }
	for _, name := range strings.Split(string(names), "\x00") {
		if name == "" { continue }
		if err := ctx.Err(); err != nil { return v, err }
		path, err := confinedFile(root, name)
		if err != nil { return v, err }
		file, err := os.Open(path)
		if err != nil { return v, err }
		body, readErr := io.ReadAll(io.LimitReader(file, AuxiliaryInputBytes+1))
		closeErr := file.Close()
		if err := errors.Join(readErr, closeErr); err != nil { return v, err }
		if len(body) > AuxiliaryInputBytes || !utf8.Valid(body) { return v, errors.New("Verifier untracked file exceeds complete-input limit or is not UTF-8") }
		if err := account(len(body)); err != nil { return v, err }
		v.Untracked = append(v.Untracked, InputSource{Kind: "untracked-addition", Path: name, Required: true, Status: "loaded", SHA256: contentDigest(body), Content: string(body)})
	}
	// Recheck all bytes after capture; a raced snapshot cannot claim the
	// accepted version. AcceptExecution repeats its own applicability check.
	current, err := CaptureValidationInputs(root, c.Inputs.Scope, c.Inputs.Bindings, c.Inputs.Toolchain)
	if err != nil { return v, err }
	if current.Digest() != c.Inputs.Digest() { return v, errors.New("Verifier inputs changed during capture") }
	return v, nil
}

// Snapshot failure is an auxiliary gap, never a development Gate. Reserve
// storage before writing, using the existing project -> Task lock order.
func (s *Store) prepareVerifierSnapshot(id TaskID, revision uint64, c AcceptanceCandidate) (ActorReference, error) {
	state, err := s.Load(id)
	if err != nil { return ActorReference{}, err }
	if state.StateRevision != revision || state.Protocol == nil || state.Protocol.Auxiliary == nil || len(state.Protocol.Auxiliary.Sources) == 0 { return ActorReference{}, errors.New("Verifier source revision is unavailable") }
	v, err := s.captureVerifierSnapshot(state, c)
	if err != nil { return ActorReference{}, err }
	body, err := json.Marshal(v)
	if err != nil { return ActorReference{}, err }
	if len(body) > AuxiliaryInputBytes { return ActorReference{}, errors.New("Verifier complete snapshot exceeds 128 KiB; no truncated review was scheduled") }
	lock, err := s.auxiliaryProjectLock()
	if err != nil { return ActorReference{}, err }
	defer systemTaskUnlock(lock)
	r, err := s.readAuxiliaryResources()
	if err != nil { return ActorReference{}, err }
	source := state.Protocol.Auxiliary.Sources[len(state.Protocol.Auxiliary.Sources)-1]
	key := contentDigest(append([]byte("verifier-snapshot:"), body...))
	job := AuxiliaryJob{Key: key, Kind: "verifier-snapshot", Owner: id, Sponsor: id, SourceID: source.ID, SourceVersion: source.Version, InputDigest: contentDigest(body), Prompt: string(body), State: "completed"}
	if err := s.reserveAuxiliaryQueue(&r, job, source); err != nil { return ActorReference{}, err }
	taskLock, err := s.lock(id)
	if err != nil { return ActorReference{}, err }
	defer unlock(taskLock)
	current, err := s.Load(id)
	if err != nil { return ActorReference{}, err }
	if current.StateRevision != revision || current.PendingEvent != nil || !isSystemLock(taskLock) { return ActorReference{}, errors.New("Verifier acceptance revision changed") }
	ref, err := s.persistProtocolArtifactLocked(id, "verifier-snapshot", v)
	if err != nil { return ActorReference{}, err }
	q := r.Queue[key]; q.Terminal = true; r.Queue[key] = q
	return ref, s.saveAuxiliaryResources(r)
}
