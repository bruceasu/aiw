package issue

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const Root = "docs/requirements"
const archiveRoot = "archive"
const cancelledRoot = "cancelled"

type Meta struct {
	ID        string
	Title     string
	ParentID  string
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
	SourceDigest string
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

func Dir(id string) string { return filepath.Join(recordRoot(id), id) }
func archiveDir(id string) string { return filepath.Join(recordRoot(id), archiveRoot, id) }
func cancelledDir(id string) string { return filepath.Join(recordRoot(id), cancelledRoot, id) }

func dirFor(id string) string {
	if location, err := locateRecord(id); err == nil { return location.dir }
	return Dir(id)
}

func Create(id, title string) (Meta, error) {
	return createWithSequence(id, title, false)
}

func createExact(id, title string) (Meta, error) {
	if !ValidID(id) {
		return Meta{}, errors.New("invalid requirement id")
	}
	if _, err := locateRecord(id); err == nil { return Meta{}, fmt.Errorf("Issue already exists: %s", id) } else if !errors.Is(err, os.ErrNotExist) { return Meta{}, err }
	for _, dir := range []string{Dir(id), archiveDir(id), cancelledDir(id)} {
		if _, err := os.Stat(dir); err == nil {
			return Meta{}, fmt.Errorf("requirement already exists: %s", dir)
		} else if !errors.Is(err, os.ErrNotExist) {
			return Meta{}, err
		}
	}
	dir := Dir(id)
	if err := os.MkdirAll(recordRoot(id), 0o755); err != nil {
		return Meta{}, err
	}
	if err := checkRecordPath(recordRoot(id)); err != nil { return Meta{}, err }
	if err := os.Mkdir(dir, 0o755); err != nil {
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
	location, err := locateRecord(id)
	if err != nil { return Meta{}, err }
	dir, path := location.dir, location.metadata
	f, err := os.Open(path)
	if err != nil {
		return Meta{}, err
	}
	defer f.Close()
	meta, err := readMeta(location.id, f)
	if err != nil { return Meta{}, err }
	// Stored paths are historical hints. The registered kind resolves the
	// current location after a directory move without rewriting the source file.
	for kind, artifact := range meta.Artifacts {
		if filename := artifactFilename(dir, kind); filename != "" {
			artifact.Path = filepath.ToSlash(filepath.Join(dir, filename))
			meta.Artifacts[kind] = artifact
		}
	}
	return meta, nil
}

// readMeta shares metadata decoding with the bounded, project-rooted context
// reader without changing the existing lifecycle storage API.
func readMeta(id string, input io.Reader) (Meta, error) {
	meta := Meta{Artifacts: map[string]Artifact{}}
	section := ""
	scanner := bufio.NewScanner(input)
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
			if key == "source_digest" { meta.Approval.SourceDigest = value }
			switch key { case "status": meta.Approval.Status = value; case "by": meta.Approval.By = value; case "at": meta.Approval.At = value; case "reason": meta.Approval.Reason = value }
		case "promotion":
			switch key { case "status": meta.Promotion.Status = value; case "task_id": meta.Promotion.TaskID = value }
		case "conversation":
			if key == "session_id" { meta.Conversation.SessionID = value }
		case "terminal":
			switch key { case "by": meta.Terminal.By = value; case "at": meta.Terminal.At = value; case "reason": meta.Terminal.Reason = value }
		default:
			switch key { case "id": meta.ID = value; case "title": meta.Title = value; case "parent_id": meta.ParentID = value; case "status": meta.Status = value; case "created": meta.Created = value; case "updated": meta.Updated = value; case "revision": fmt.Sscanf(value, "%d", &meta.Revision) }
		}
	}
	if err := scanner.Err(); err != nil { return Meta{}, err }
	if meta.ID != id { return Meta{}, fmt.Errorf("requirement metadata id mismatch: %s", meta.ID) }
	if meta.Status == "" || meta.Approval.Status == "" || meta.Promotion.Status == "" { return Meta{}, errors.New("malformed requirement metadata") }
	return meta, nil
}

func Write(meta Meta) error {
	if !ValidID(meta.ID) { return errors.New("invalid requirement id") }
	content := fmt.Sprintf("id = %q\ntitle = %q\nparent_id = %q\nstatus = %q\ncreated = %q\nupdated = %q\nrevision = %d\n\n[approval]\nstatus = %q\nby = %q\nat = %q\nreason = %q\n\n[promotion]\nstatus = %q\ntask_id = %q\n\n[conversation]\nsession_id = %q\n\n[terminal]\nby = %q\nat = %q\nreason = %q\n", meta.ID, meta.Title, meta.ParentID, meta.Status, meta.Created, meta.Updated, meta.Revision, meta.Approval.Status, meta.Approval.By, meta.Approval.At, meta.Approval.Reason, meta.Promotion.Status, meta.Promotion.TaskID, meta.Conversation.SessionID, meta.Terminal.By, meta.Terminal.At, meta.Terminal.Reason)
	content = strings.Replace(content, "\n[promotion]\n", fmt.Sprintf("source_digest = %q\n\n[promotion]\n", meta.Approval.SourceDigest), 1)
	keys := make([]string, 0, len(meta.Artifacts))
	for kind := range meta.Artifacts { keys = append(keys, kind) }
	sort.Strings(keys)
	if len(keys) > 0 {
		content += "\n[artifact]\n"
		for _, kind := range keys { content += fmt.Sprintf("%s = %q\n", kind, meta.Artifacts[kind].Path) }
		content += "\n[digest]\n"
		for _, kind := range keys { content += fmt.Sprintf("%s = %q\n", kind, meta.Artifacts[kind].Digest) }
	}
	location, err := locateRecord(meta.ID)
	if err != nil {
		// Only createExact's new empty directory has no metadata yet.
		if !errors.Is(err, os.ErrNotExist) { return err }
		dir := Dir(meta.ID)
		if err := checkRecordPath(dir); err != nil { return err }
		location = recordLocation{meta.ID, dir, filepath.Join(dir, metadataName(recordRoot(meta.ID)))}
	}
	if location.id != meta.ID { return errors.New("write requires a canonical Issue id") }
	return atomicWrite(location.metadata, []byte(content))
}

// LinkParent records split lineage on a child before its approval is fixed.
// Existing REQ records without parent_id remain valid roots.
func LinkParent(childID, parentID string) (Meta, error) {
	if !ValidID(childID) || !ValidID(parentID) || childID == parentID {
		return Meta{}, errors.New("child and parent must be distinct valid Issue IDs")
	}
	child, err := Read(childID)
	if err != nil { return Meta{}, err }
	parent, err := Read(parentID)
	if err != nil { return Meta{}, err }
	childID, parentID = child.ID, parent.ID
	if childID == parentID { return Meta{}, errors.New("child and parent must be distinct Issue records") }
	if child.ParentID == parentID { return child, nil }
	if child.ParentID != "" { return Meta{}, fmt.Errorf("Issue %s already has parent %s", childID, child.ParentID) }
	if child.Approval.Status != "PENDING" || child.Status == "ARCHIVED" || child.Status == "CANCELLED" {
		return Meta{}, fmt.Errorf("Issue %s lineage is fixed after approval or closure", childID)
	}
	seen := map[string]bool{childID: true}
	for current := parentID; current != ""; {
		parent, err := Read(current)
		if err != nil { return Meta{}, err }
		if seen[parent.ID] { return Meta{}, errors.New("Issue parent link would create a cycle") }
		seen[parent.ID] = true
		current = parent.ParentID
	}
	child.ParentID, child.Updated, child.Revision = parentID, time.Now().Format("2006-01-02"), child.Revision+1
	if err := Write(child); err != nil { return Meta{}, err }
	return child, nil
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
	kind = canonicalArtifactKind(kind)
	_, ok := artifactFiles[kind]
	if !ok { return Meta{}, Artifact{}, fmt.Errorf("unsupported artifact type: %s", kind) }
	meta, err := Read(id)
	if err != nil { return Meta{}, Artifact{}, err }
	if meta.Status == "ARCHIVED" || meta.Status == "CANCELLED" { return Meta{}, Artifact{}, fmt.Errorf("requirement is %s", meta.Status) }
	b, err := os.ReadFile(source)
	if err != nil { return Meta{}, Artifact{}, err }
	location, err := locateRecord(meta.ID)
	if err != nil { return Meta{}, Artifact{}, err }
	target := filepath.Join(location.dir, artifactFilename(location.dir, kind))
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
	meta.Approval.SourceDigest = approvalSourceDigest(meta)
	meta.Updated = time.Now().Format("2006-01-02")
	meta.Revision++
	if err := Write(meta); err != nil { return Meta{}, err }
	entry := fmt.Sprintf("\n## %s\n- Decision: %s\n- By: %s\n- Reason: %s\n", meta.Approval.At, decision, by, reason)
	location, err := locateRecord(meta.ID)
	if err != nil { return Meta{}, err }
	f, err := os.OpenFile(filepath.Join(location.dir, "decision-log.md"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil { return Meta{}, err }
	defer f.Close()
	if _, err := f.WriteString(entry); err != nil { return Meta{}, err }
	return meta, nil
}

// approvalSourceDigest binds an approval to the Requirement content that was
// present when the decision was recorded. It belongs to the Requirement
// lifecycle, not to OpenSpec artifact generation.
func approvalSourceDigest(meta Meta) string {
	b, _ := json.Marshal(struct {
		ID        string
		Title     string
		ParentID  string `json:"ParentID,omitempty"`
		Artifacts map[string]Artifact
	}{meta.ID, meta.Title, meta.ParentID, meta.Artifacts})
	return digest(b)
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
	if meta.Promotion.Status == "SPEC_DRAFTED" || meta.Promotion.Status == "FD_READY" { return meta, nil }
	meta.Promotion.Status = "FD_READY"
	meta.Updated, meta.Revision = time.Now().Format("2006-01-02"), meta.Revision+1
	if err := Write(meta); err != nil { return Meta{}, err }
	return meta, nil
}

func Archive(id, by, reason string) (Meta, error) { return moveTerminal(id, "ARCHIVED", by, reason) }
func Cancel(id, by, reason string) (Meta, error) { return moveTerminal(id, "CANCELLED", by, reason) }

func moveTerminal(id, status, by, reason string) (Meta, error) {
	meta, err := Read(id); if err != nil { return Meta{}, err }
	if by == "" || reason == "" { return Meta{}, errors.New("terminal transition requires actor and reason") }
	if meta.Status == "ARCHIVED" || meta.Status == "CANCELLED" { return meta, fmt.Errorf("requirement is already %s", meta.Status) }
	if status == "ARCHIVED" && meta.Status != "DECIDED" && meta.Status != "APPROVED" && meta.Status != "PROMOTED" { return Meta{}, fmt.Errorf("requirement must be DECIDED, APPROVED, or PROMOTED before archive: %s", meta.Status) }
	location, err := locateRecord(meta.ID); if err != nil { return Meta{}, err }
	source := location.dir
	suffix := archiveRoot; if status == "CANCELLED" { suffix = cancelledRoot }
	target := filepath.Join(filepath.Dir(source), suffix, meta.ID)
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
	locations, err := recordLocations(filter)
	if err != nil { return nil, err }
	var result []Meta
	for _, location := range locations {
		meta, err := Read(location.id)
		if err != nil { return nil, err }
		result = append(result, meta)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID }); return result, nil
}

func ArtifactSnapshot(id string) ([]Artifact, error) {
	meta, err := Read(id); if err != nil { return nil, err }
	location, err := locateRecord(meta.ID); if err != nil { return nil, err }
	keys := make([]string, 0, len(artifactFiles))
	for key := range artifactFiles { keys = append(keys, key) }
	sort.Strings(keys)
	var result []Artifact
	for _, key := range keys {
		path := filepath.Join(location.dir, artifactFilename(location.dir, key))
		if err := checkRecordPath(path); err != nil {
			if _, recorded := meta.Artifacts[key]; !recorded && errors.Is(err, os.ErrNotExist) { continue }
			return nil, err
		}
		b, err := os.ReadFile(path)
		if err != nil { return nil, err }
		artifact := Artifact{Kind: key, Path: filepath.ToSlash(path), Digest: digest(b)}
		if recorded, ok := meta.Artifacts[key]; ok && recorded.Digest != artifact.Digest { return nil, fmt.Errorf("artifact digest changed since capture: %s", key) }
		result = append(result, artifact)
	}
	return result, nil
}

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { return err }
	temporaryRoot := filepath.Join(runtimeRoot(), "temporary")
	if strings.HasPrefix(filepath.ToSlash(path), IssueRoot+"/") { temporaryRoot = filepath.Join(filepath.Dir(runtimeRoot()), "issues", "temporary") }
	if err := os.MkdirAll(temporaryRoot, 0o700); err != nil { return err }
	tmp, err := os.CreateTemp(temporaryRoot, ".requirement-*.tmp")
	if err != nil { return err }
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil { tmp.Close(); return err }
	if err := tmp.Sync(); err != nil { tmp.Close(); return err }
	if err := tmp.Close(); err != nil { return err }
	return os.Rename(tmpPath, path)
}

func unquote(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		if decoded, err := strconv.Unquote(value); err == nil {
			return decoded
		}
	}
	return strings.Trim(value, `"`)
}
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func samePath(a, b string) bool { left, errA := filepath.Abs(a); right, errB := filepath.Abs(b); return errA == nil && errB == nil && strings.EqualFold(left, right) }
