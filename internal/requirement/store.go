package requirement

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const Root = "requirements"
const archiveRoot = "archive"
const cancelledRoot = "cancelled"

type Meta struct {
	ID        string
	Title     string
	Status    string
	Created   string
	Updated   string
	Revision  int
	Approval  Approval
	Promotion Promotion
	Conversation Conversation
	Terminal Terminal
	Artifacts map[string]Artifact
}

type Terminal struct { By, At, Reason string }

type Approval struct {
	Status string
	By     string
	At     string
	Reason string
}

type Promotion struct {
	Status string
	TaskID string
}

type Conversation struct {
	SessionID string
}

type Artifact struct {
	Kind   string
	Path   string
	Digest string
}

var artifactFiles = map[string]string{
	"problem-brief":       "problem-brief.md",
	"business-case":       "business-case.md",
	"metric-brief":        "metric-brief.md",
	"engineering-options": "engineering-options.md",
	"requirement-plan":    "requirement-plan.md",
}

func ValidID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func Dir(id string) string { return filepath.Join(Root, id) }
func archiveDir(id string) string { return filepath.Join(Root, archiveRoot, id) }
func cancelledDir(id string) string { return filepath.Join(Root, cancelledRoot, id) }

func dirFor(id string) string {
	for _, dir := range []string{Dir(id), archiveDir(id), cancelledDir(id)} {
		if _, err := os.Stat(filepath.Join(dir, "requirement.toml")); err == nil { return dir }
	}
	return Dir(id)
}

func Create(id, title string) (Meta, error) {
	if !ValidID(id) {
		return Meta{}, errors.New("invalid requirement id")
	}
	for _, dir := range []string{Dir(id), archiveDir(id), cancelledDir(id)} {
		if _, err := os.Stat(dir); err == nil {
			return Meta{}, fmt.Errorf("requirement already exists: %s", dir)
		} else if !errors.Is(err, os.ErrNotExist) {
			return Meta{}, err
		}
	}
	dir := Dir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Meta{}, err
	}
	today := time.Now().Format("2006-01-02")
	meta := Meta{ID: id, Title: title, Status: "DRAFT", Created: today, Updated: today, Revision: 1, Approval: Approval{Status: "PENDING"}, Promotion: Promotion{Status: "NOT_STARTED"}, Artifacts: map[string]Artifact{}}
	if err := Write(meta); err != nil {
		return Meta{}, err
	}
	return meta, nil
}

func Read(id string) (Meta, error) {
	if !ValidID(id) {
		return Meta{}, errors.New("invalid requirement id")
	}
	path := filepath.Join(dirFor(id), "requirement.toml")
	f, err := os.Open(path)
	if err != nil {
		return Meta{}, err
	}
	defer f.Close()

	meta := Meta{Artifacts: map[string]Artifact{}}
	section := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.Trim(line, "[]")
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := strings.TrimSpace(parts[0]), unquote(strings.TrimSpace(parts[1]))
		switch section {
		case "artifact":
			kind := key
			if _, ok := artifactFiles[kind]; !ok { return Meta{}, fmt.Errorf("unsupported artifact metadata: %s", kind) }
			artifact := meta.Artifacts[kind]
			artifact.Kind = kind
			artifact.Path = value
			meta.Artifacts[kind] = artifact
		case "digest":
			kind := key
			if _, ok := artifactFiles[kind]; !ok { return Meta{}, fmt.Errorf("unsupported digest metadata: %s", kind) }
			artifact := meta.Artifacts[kind]
			artifact.Kind = kind
			artifact.Digest = value
			meta.Artifacts[kind] = artifact
		case "approval":
			switch key { case "status": meta.Approval.Status = value; case "by": meta.Approval.By = value; case "at": meta.Approval.At = value; case "reason": meta.Approval.Reason = value }
		case "promotion":
			switch key { case "status": meta.Promotion.Status = value; case "task_id": meta.Promotion.TaskID = value }
		case "conversation":
			if key == "session_id" { meta.Conversation.SessionID = value }
		case "terminal":
			switch key { case "by": meta.Terminal.By = value; case "at": meta.Terminal.At = value; case "reason": meta.Terminal.Reason = value }
		default:
			switch key { case "id": meta.ID = value; case "title": meta.Title = value; case "status": meta.Status = value; case "created": meta.Created = value; case "updated": meta.Updated = value; case "revision": fmt.Sscanf(value, "%d", &meta.Revision) }
		}
	}
	if err := scanner.Err(); err != nil { return Meta{}, err }
	if meta.ID != id { return Meta{}, fmt.Errorf("requirement metadata id mismatch: %s", meta.ID) }
	if meta.Status == "" || meta.Approval.Status == "" || meta.Promotion.Status == "" { return Meta{}, errors.New("malformed requirement metadata") }
	return meta, nil
}

func Write(meta Meta) error {
	if !ValidID(meta.ID) { return errors.New("invalid requirement id") }
	content := fmt.Sprintf("id = %q\ntitle = %q\nstatus = %q\ncreated = %q\nupdated = %q\nrevision = %d\n\n[approval]\nstatus = %q\nby = %q\nat = %q\nreason = %q\n\n[promotion]\nstatus = %q\ntask_id = %q\n\n[conversation]\nsession_id = %q\n\n[terminal]\nby = %q\nat = %q\nreason = %q\n", meta.ID, meta.Title, meta.Status, meta.Created, meta.Updated, meta.Revision, meta.Approval.Status, meta.Approval.By, meta.Approval.At, meta.Approval.Reason, meta.Promotion.Status, meta.Promotion.TaskID, meta.Conversation.SessionID, meta.Terminal.By, meta.Terminal.At, meta.Terminal.Reason)
	keys := make([]string, 0, len(meta.Artifacts))
	for kind := range meta.Artifacts { keys = append(keys, kind) }
	sort.Strings(keys)
	if len(keys) > 0 {
		content += "\n[artifact]\n"
		for _, kind := range keys { content += fmt.Sprintf("%s = %q\n", kind, meta.Artifacts[kind].Path) }
		content += "\n[digest]\n"
		for _, kind := range keys { content += fmt.Sprintf("%s = %q\n", kind, meta.Artifacts[kind].Digest) }
	}
	return atomicWrite(filepath.Join(dirFor(meta.ID), "requirement.toml"), []byte(content))
}

func BindConversation(id, sessionID string) (Meta, error) {
	if !ValidID(sessionID) { return Meta{}, errors.New("invalid requirement session id") }
	meta, err := Read(id)
	if err != nil { return Meta{}, err }
	if meta.Status == "ARCHIVED" || meta.Status == "CANCELLED" { return Meta{}, fmt.Errorf("requirement is %s", meta.Status) }
	if meta.Conversation.SessionID != "" && meta.Conversation.SessionID != sessionID { return Meta{}, fmt.Errorf("requirement already links to session %s", meta.Conversation.SessionID) }
	if meta.Conversation.SessionID == sessionID { return meta, nil }
	meta.Conversation.SessionID = sessionID
	meta.Updated, meta.Revision = time.Now().Format("2006-01-02"), meta.Revision+1
	if err := Write(meta); err != nil { return Meta{}, err }
	return meta, nil
}

func Capture(id, kind, source string) (Meta, Artifact, error) {
	filename, ok := artifactFiles[kind]
	if !ok { return Meta{}, Artifact{}, fmt.Errorf("unsupported artifact type: %s", kind) }
	meta, err := Read(id)
	if err != nil { return Meta{}, Artifact{}, err }
	if meta.Status == "ARCHIVED" || meta.Status == "CANCELLED" { return Meta{}, Artifact{}, fmt.Errorf("requirement is %s", meta.Status) }
	b, err := os.ReadFile(source)
	if err != nil { return Meta{}, Artifact{}, err }
	target := filepath.Join(dirFor(id), filename)
	if samePath(source, target) { return Meta{}, Artifact{}, errors.New("capture source must not be the destination artifact") }
	if err := atomicWrite(target, b); err != nil { return Meta{}, Artifact{}, err }
	if meta.Status == "DRAFT" { meta.Status = "DISCOVERED" }
	if kind == "requirement-plan" && (meta.Status == "DRAFT" || meta.Status == "DISCOVERED") { meta.Status = "DECIDED" }
	meta.Revision++
	meta.Updated = time.Now().Format("2006-01-02")
	artifact := Artifact{Kind: kind, Path: filepath.ToSlash(target), Digest: digest(b)}
	meta.Artifacts[kind] = artifact
	if err := Write(meta); err != nil { return Meta{}, Artifact{}, err }
	return meta, artifact, nil
}

func Approve(id, decision, by, reason string) (Meta, error) {
	meta, err := Read(id)
	if err != nil { return Meta{}, err }
	if meta.Status == "ARCHIVED" || meta.Status == "CANCELLED" { return Meta{}, fmt.Errorf("requirement is %s", meta.Status) }
	decision = strings.ToUpper(decision)
	if decision != "APPROVED" && decision != "DEFERRED" && decision != "REJECTED" { return Meta{}, errors.New("approval decision must be APPROVED, DEFERRED, or REJECTED") }
	if by == "" || reason == "" { return Meta{}, errors.New("approval requires --by and --reason") }
	if decision == "APPROVED" && meta.Status != "DECIDED" { return Meta{}, fmt.Errorf("requirement must be DECIDED before approval: %s", meta.Status) }
	meta.Status, meta.Approval.Status, meta.Approval.By, meta.Approval.Reason = decision, decision, by, reason
	meta.Approval.At = time.Now().Format(time.RFC3339)
	meta.Updated = time.Now().Format("2006-01-02")
	meta.Revision++
	if err := Write(meta); err != nil { return Meta{}, err }
	entry := fmt.Sprintf("\n## %s\n- Decision: %s\n- By: %s\n- Reason: %s\n", meta.Approval.At, decision, by, reason)
	f, err := os.OpenFile(filepath.Join(dirFor(id), "decision-log.md"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil { return Meta{}, err }
	defer f.Close()
	if _, err := f.WriteString(entry); err != nil { return Meta{}, err }
	return meta, nil
}

func StartPromotion(id, taskID string) (Meta, bool, error) {
	meta, err := Read(id)
	if err != nil { return Meta{}, false, err }
	if meta.Promotion.TaskID != "" {
		if meta.Promotion.TaskID != taskID { return Meta{}, false, fmt.Errorf("requirement already links to task %s", meta.Promotion.TaskID) }
		return meta, false, nil
	}
	if meta.Status != "APPROVED" || meta.Approval.Status != "APPROVED" { return Meta{}, false, errors.New("requirement is not approved") }
	meta.Promotion.TaskID, meta.Promotion.Status = taskID, "TASK_CREATED"
	meta.Status, meta.Updated, meta.Revision = "PROMOTED", time.Now().Format("2006-01-02"), meta.Revision+1
	if err := Write(meta); err != nil { return Meta{}, false, err }
	return meta, true, nil
}

func CompletePromotion(id, taskID string) (Meta, error) {
	meta, err := Read(id)
	if err != nil { return Meta{}, err }
	if meta.Promotion.TaskID != taskID { return Meta{}, fmt.Errorf("requirement does not link to task %s", taskID) }
	if meta.Promotion.Status == "SPEC_DRAFTED" { return meta, nil }
	meta.Promotion.Status = "SPEC_DRAFTED"
	meta.Updated, meta.Revision = time.Now().Format("2006-01-02"), meta.Revision+1
	if err := Write(meta); err != nil { return Meta{}, err }
	return meta, nil
}

func Archive(id, by, reason string) (Meta, error) { return moveTerminal(id, "ARCHIVED", archiveDir(id), by, reason) }
func Cancel(id, by, reason string) (Meta, error) { return moveTerminal(id, "CANCELLED", cancelledDir(id), by, reason) }

func moveTerminal(id, status, target, by, reason string) (Meta, error) {
	meta, err := Read(id); if err != nil { return Meta{}, err }
	if by == "" || reason == "" { return Meta{}, errors.New("terminal transition requires actor and reason") }
	if meta.Status == "ARCHIVED" || meta.Status == "CANCELLED" { return meta, fmt.Errorf("requirement is already %s", meta.Status) }
	if status == "ARCHIVED" && meta.Status != "DECIDED" && meta.Status != "APPROVED" && meta.Status != "PROMOTED" { return Meta{}, fmt.Errorf("requirement must be DECIDED, APPROVED, or PROMOTED before archive: %s", meta.Status) }
	source := dirFor(id)
	if _, err := os.Stat(target); err == nil { return Meta{}, fmt.Errorf("requirement destination already exists: %s", target) } else if !errors.Is(err, os.ErrNotExist) { return Meta{}, err }
	now := time.Now().Format(time.RFC3339)
	meta.Status, meta.Terminal.By, meta.Terminal.At, meta.Terminal.Reason = status, by, now, reason
	meta.Updated, meta.Revision = time.Now().Format("2006-01-02"), meta.Revision+1
	if err := Write(meta); err != nil { return Meta{}, err }
	entry := fmt.Sprintf("\n## %s\n- Decision: %s\n- By: %s\n- Reason: %s\n", now, status, by, reason)
	f, err := os.OpenFile(filepath.Join(source, "decision-log.md"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil { return Meta{}, err }; if _, err := f.WriteString(entry); err != nil { _ = f.Close(); return Meta{}, err }; if err := f.Close(); err != nil { return Meta{}, err }
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { return Meta{}, err }
	if err := os.Rename(source, target); err != nil { return Meta{}, err }
	return meta, nil
}

type ListFilter string
const (
	ListActive ListFilter = "active"
	ListArchived ListFilter = "archived"
	ListCancelled ListFilter = "cancelled"
	ListAll ListFilter = "all"
)

func List(filter ListFilter) ([]Meta, error) {
	roots := []string{Root}; if filter == ListArchived { roots = []string{filepath.Join(Root, archiveRoot)} }; if filter == ListCancelled { roots = []string{filepath.Join(Root, cancelledRoot)} }; if filter == ListAll { roots = []string{Root, filepath.Join(Root, archiveRoot), filepath.Join(Root, cancelledRoot)} }
	var result []Meta
	for _, root := range roots { entries, err := os.ReadDir(root); if errors.Is(err, os.ErrNotExist) { continue }; if err != nil { return nil, err }; for _, entry := range entries { if !entry.IsDir() || entry.Name() == archiveRoot || entry.Name() == cancelledRoot || !ValidID(entry.Name()) { continue }; meta, err := Read(entry.Name()); if err != nil { return nil, err }; result = append(result, meta) } }
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID }); return result, nil
}

func ArtifactSnapshot(id string) ([]Artifact, error) {
	meta, err := Read(id); if err != nil { return nil, err }
	keys := make([]string, 0, len(artifactFiles))
	for key := range artifactFiles { keys = append(keys, key) }
	sort.Strings(keys)
	var result []Artifact
	for _, key := range keys {
		path := filepath.Join(dirFor(id), artifactFiles[key])
		b, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) { continue }
		if err != nil { return nil, err }
		artifact := Artifact{Kind: key, Path: filepath.ToSlash(path), Digest: digest(b)}
		if recorded, ok := meta.Artifacts[key]; ok && recorded.Digest != artifact.Digest { return nil, fmt.Errorf("artifact digest changed since capture: %s", key) }
		result = append(result, artifact)
	}
	return result, nil
}

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { return err }
	tmp, err := os.CreateTemp(filepath.Dir(path), ".requirement-*.tmp")
	if err != nil { return err }
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil { tmp.Close(); return err }
	if err := tmp.Close(); err != nil { return err }
	if err := os.Rename(tmpPath, path); err == nil { return nil }
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) { return err }
	return os.Rename(tmpPath, path)
}

func unquote(value string) string { return strings.Trim(strings.TrimSpace(value), `"`) }
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func samePath(a, b string) bool { left, errA := filepath.Abs(a); right, errB := filepath.Abs(b); return errA == nil && errB == nil && strings.EqualFold(left, right) }
