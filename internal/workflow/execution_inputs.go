package workflow

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// InputSource records actual bytes, including an explicit read failure for
// optional inputs. A path or an empty search result is not a source version.
type InputSource struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
	Required bool `json:"required"`
	ExpectedSHA256 string `json:"expected_sha256,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	Content string `json:"content"`
}

type ExecutionInput struct {
	WorkspaceInputs *ValidationInputs `json:"workspace_inputs,omitempty"`
	SchemaVersion int `json:"schema_version"`
	RequestID string `json:"request_id"`
	TaskID TaskID `json:"task_id"`
	WorkItemID WorkItemID `json:"work_item_id"`
	AttemptID AttemptID `json:"attempt_id"`
	SessionID string `json:"session_id"`
	Turn int `json:"turn"`
	Actor ActorKind `json:"actor"`
	Workspace string `json:"workspace"`
	AISelection *AISelection `json:"ai_selection,omitempty"`
	AllowedPaths []string `json:"allowed_paths"`
	Sources []InputSource `json:"sources"`
	Prompt string `json:"prompt"`
}

// ExecutionRequestID includes the turn: a repair never overwrites its parent.
func ExecutionRequestID(r PreparedAgentRequest) string {
	identity, _ := json.Marshal([]any{r.TaskID, r.WorkItemID, r.AttemptID, r.SessionID, r.ExpectedSessionTurn})
	return contentDigest(identity)
}

func contentDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func (s *Store) PersistExecutionInput(r PreparedAgentRequest, input ExecutionInput) (ActorReference, error) {
	if input.SchemaVersion != 1 || input.RequestID != ExecutionRequestID(r) || input.TaskID != r.TaskID || input.WorkItemID != r.WorkItemID || input.AttemptID != r.AttemptID || input.SessionID != r.SessionID || input.Turn != r.ExpectedSessionTurn || input.Workspace != r.Workspace || !validActor(input.Actor) || strings.TrimSpace(input.Prompt) == "" || len(input.Sources) == 0 || len(input.AllowedPaths) == 0 {
		return ActorReference{}, errors.New("execution input requires exact request identity, role, scope, sources and prompt")
	}
	if len(input.Prompt) > 4*1024*1024 { return ActorReference{}, errors.New("complete execution prompt exceeds the 4 MiB input ceiling") }
	for _, source := range input.Sources {
		if source.Path == "" || source.Kind == "" { return ActorReference{}, errors.New("input source identity is missing") }
		if source.Status == "loaded" {
			if source.SHA256 != contentDigest([]byte(source.Content)) || (source.ExpectedSHA256 != "" && source.ExpectedSHA256 != source.SHA256) { return ActorReference{}, fmt.Errorf("input version mismatch: %s", source.Path) }
		} else if source.Required || source.Status != "unavailable" || source.Reason == "" {
			return ActorReference{}, fmt.Errorf("required input is unavailable or unexplained: %s", source.Path)
		}
	}
	return s.persistExecutionArtifact(r, "execution-input", "inputs/"+input.RequestID+".json", input)
}

// persistExecutionArtifact publishes once under the existing Task lock. E02
// owns the platform durability upgrade; this does not activate that protocol.
func (s *Store) persistExecutionArtifact(r PreparedAgentRequest, kind, relative string, value any) (ActorReference, error) {
	if err := validateTaskID(r.TaskID); err != nil { return ActorReference{}, err }
	if r.TaskID == "." || r.TaskID == ".." || r.SessionID == "" || r.ExpectedSessionTurn <= 0 { return ActorReference{}, errors.New("artifact request binding is incomplete") }
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil { return ActorReference{}, err }
	content = append(content, '\n')
	lock, err := s.lock(r.TaskID)
	if err != nil { return ActorReference{}, err }
	defer unlock(lock)
	state, err := s.Load(r.TaskID)
	if err != nil { return ActorReference{}, err }
	attempt, found := findAttempt(state.Attempts, r.AttemptID)
	if !found || attempt.WorkItemID != r.WorkItemID || attempt.Workspace != r.Workspace { return ActorReference{}, errors.New("artifact does not match persisted Attempt") }
	if state.SchemaVersion == DurableSchemaVersion {
		item, err := executionItem(&state, r.WorkItemID)
		if err != nil || item.AttemptID != r.AttemptID || item.Phase == PhaseAccepted || !isSystemLock(lock) { return ActorReference{}, errors.New("artifact does not match the durable execution") }
	} else if current := state.Automation.PreparedRequest; current == nil || ExecutionRequestID(*current) != ExecutionRequestID(r) { return ActorReference{}, errors.New("artifact does not match the prepared generation") }
	if _, err := os.Stat(s.path(r.TaskID, runtimeStateFile)); err != nil { return ActorReference{}, err }
	relative = filepath.ToSlash(filepath.Join("reports", relative))
	target := s.path(r.TaskID, filepath.FromSlash(relative))
	prior, err := os.ReadFile(target)
	if err == nil {
		if !bytes.Equal(prior, content) { return ActorReference{}, fmt.Errorf("immutable artifact conflict: %s", relative) }
	} else if errors.Is(err, os.ErrNotExist) {
		if err := writeExecutionReport(target, content); err != nil { return ActorReference{}, err }
	} else { return ActorReference{}, err }
	file, err := os.OpenFile(target, os.O_RDWR, 0)
	if err != nil { return ActorReference{}, err }
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(syncErr, closeErr); err != nil { return ActorReference{}, err }
	return ActorReference{Kind: kind, Path: relative, SHA256: contentDigest(content)}, nil
}

func (s *Store) ReadExecutionArtifact(id TaskID, ref ActorReference, value any) error {
	if err := validateTaskID(id); err != nil { return err }
	if id == "." || id == ".." || ref.SHA256 == "" || ref.Kind == "" { return errors.New("artifact identity and digest are required") }
	root := s.path(id, "reports")
	target, err := confinedFile(s.path(id, ""), ref.Path)
	if err != nil { return fmt.Errorf("resolve execution artifact: %w", err) }
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) { return errors.New("artifact must be Task-owned") }
	content, err := os.ReadFile(target)
	if err != nil { return fmt.Errorf("read execution artifact content: %w", err) }
	if contentDigest(content) != ref.SHA256 { return errors.New("artifact content digest mismatch") }
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	return decoder.Decode(value)
}

func confinedFile(root, path string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil { return "", fmt.Errorf("resolve absolute input root: %w", err) }
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil { return "", &os.PathError{Op: "resolve input root links", Path: root, Err: err} }
	root = resolvedRoot
	if !filepath.IsAbs(path) { path = filepath.Join(root, filepath.FromSlash(path)) }
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil { return "", &os.PathError{Op: "resolve input target links", Path: path, Err: err} }
	path = resolvedPath
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) { return "", errors.New("input path escapes its declared root") }
	return path, nil
}

// ReadInputSource resolves links before reading; exact-version references
// never silently fall back to the latest version.
func ReadInputSource(root string, source InputSource) InputSource {
	path, err := confinedFile(root, source.Path)
	var content []byte
	if err == nil { content, err = os.ReadFile(path) }
	if err == nil && source.ExpectedSHA256 != "" && contentDigest(content) != source.ExpectedSHA256 { err = errors.New("required source version is unavailable") }
	if err != nil {
		source.Status, source.Reason, source.Content, source.SHA256 = "unavailable", err.Error(), "", ""
		return source
	}
	source.Status, source.Content, source.SHA256 = "loaded", string(content), contentDigest(content)
	return source
}

// ValidationInputs is a deterministic content manifest. Scope is explicit;
// directory discovery includes additions and deletions, even at unchanged HEAD.
type ValidationInputs struct {
	SchemaVersion int `json:"schema_version"`
	Scope []string `json:"scope"`
	Files []ActorReference `json:"files"`
	Bindings []ActorReference `json:"bindings"`
	Toolchain string `json:"toolchain"`
}

func CaptureValidationInputs(root string, scope []string, bindings []ActorReference, toolchain string) (ValidationInputs, error) {
	manifest := ValidationInputs{SchemaVersion: 1, Scope: append([]string(nil), scope...), Bindings: append([]ActorReference(nil), bindings...), Toolchain: toolchain}
	if len(scope) == 0 || toolchain == "" { return manifest, errors.New("validation scope and toolchain identity are required") }
	for _, ref := range bindings { if ref.Kind == "" || ref.Path == "" || ref.SHA256 == "" { return manifest, errors.New("validation binding lacks an exact version") } }
	files := make(map[string]ActorReference)
	for index, item := range manifest.Scope {
		clean := filepath.Clean(filepath.FromSlash(item))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) { return manifest, errors.New("validation scope escapes workspace") }
		manifest.Scope[index] = filepath.ToSlash(clean)
		err := filepath.WalkDir(filepath.Join(root, clean), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil { return walkErr }
			relative, err := filepath.Rel(root, path)
			if err != nil { return err }
			// These are runtime projections, not implementation inputs. Other
			// documents and executable prompts remain in the manifest.
			if entry.IsDir() && (entry.Name() == ".git" || relative == ".ai" || relative == ".wt") { return filepath.SkipDir }
			if entry.IsDir() { return nil }
			resolved, err := confinedFile(root, relative)
			if err != nil { return err }
			info, err := os.Stat(resolved)
			if err != nil { return err }
			if !info.Mode().IsRegular() { return fmt.Errorf("validation input is not a regular file: %s", relative) }
			content, err := os.ReadFile(resolved)
			if err != nil { return err }
			relative = filepath.ToSlash(relative)
			files[relative] = ActorReference{Kind: "content", Path: relative, SHA256: contentDigest(content)}
			return nil
		})
		if err != nil { return manifest, err }
	}
	for _, ref := range files { manifest.Files = append(manifest.Files, ref) }
	sort.Strings(manifest.Scope)
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	sort.Slice(manifest.Bindings, func(i, j int) bool { a, b := manifest.Bindings[i], manifest.Bindings[j]; return a.Kind+"\x00"+a.Path+"\x00"+a.SHA256 < b.Kind+"\x00"+b.Path+"\x00"+b.SHA256 })
	return manifest, nil
}

func (m ValidationInputs) Digest() string { data, _ := json.Marshal(m); return contentDigest(data) }

// ValidationToolchainIdentity reads executable bytes without running probes.
// E03 adapters must add any extra interpreter/tool identities to their plan.
func ValidationToolchainIdentity() (string, error) {
	identities := []string{runtime.GOOS, runtime.GOARCH, runtime.Version()}
	for _, name := range []string{"PATH", "GOROOT", "GOTOOLCHAIN", "GOFLAGS", "GOOS", "GOARCH", "CGO_ENABLED", "CC", "CXX", "PYTHONPATH", "VIRTUAL_ENV"} { identities = append(identities, name+"="+os.Getenv(name)) }
	for _, name := range []string{"go", "python", "powershell.exe", "sh", "cc", "c++"} {
		path, err := exec.LookPath(name)
		if err != nil { identities = append(identities, name+"=unavailable"); continue }
		content, err := os.ReadFile(path)
		if err != nil { return "", err }
		identities = append(identities, name+"="+path+":"+contentDigest(content))
	}
	sort.Strings(identities)
	return strings.Join(identities, "\n"), nil
}

func CompileInputBindings(plan *CompilePlan) ([]ActorReference, error) {
	if plan == nil { return nil, errors.New("current compile plan is missing") }
	content, err := json.Marshal(plan)
	if err != nil { return nil, err }
	return []ActorReference{{Kind: "compile-plan", Path: "frozen-compile-plan", SHA256: contentDigest(content)}}, nil
}

// EvidenceApplicability is also the legacy-result boundary. A missing manifest
// is unknown, never an implicit pass or permission to rerun a command.
func EvidenceApplicability(previous *ValidationInputs, current ValidationInputs, controlled, passed bool) (bool, string) {
	if !controlled || !passed { return false, "a controlled passed execution is required" }
	if previous == nil || previous.SchemaVersion != 1 || current.SchemaVersion != 1 || len(previous.Scope) == 0 || previous.Toolchain == "" { return false, "legacy input provenance is unknown" }
	if strings.HasPrefix(previous.Toolchain, "incomplete:") || strings.HasPrefix(current.Toolchain, "incomplete:") { return false, "execution adapter has not fixed the complete toolchain environment" }
	if previous.Digest() != current.Digest() { return false, "relevant input content, discovery scope, plan or toolchain changed" }
	return true, "exact inputs remain applicable"
}
