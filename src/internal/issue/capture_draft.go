package issue

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// capturedDraft owns only this capture's source, never a directory-wide sweep.
// The designated .ai storage may be shared by managed worktrees. Once opened,
// all cleanup operations stay beneath one fixed draft root.
type capturedDraft struct {
	root *os.Root
	rootPath string
	local string
	source string
	familyInfo os.FileInfo
	rootInfo os.FileInfo
	info os.FileInfo
	warning string
}

func readCaptureSource(source string) ([]byte, *capturedDraft, error) {
	draft := prepareCapturedDraft(source)
	if draft == nil || draft.warning != "" {
		content, err := os.ReadFile(source)
		return content, draft, err
	}
	file, err := draft.root.Open(draft.local)
	if err != nil {
		draft.warning = draft.message("source could not be opened safely: %v", err)
		content, readErr := os.ReadFile(source)
		return content, draft, readErr
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !sameDraftFile(draft.info, opened) {
		draft.warning = draft.message("source changed while opening; kept source")
	}
	content, err := io.ReadAll(file)
	if err != nil { return nil, draft, err }
	after, statErr := file.Stat()
	if statErr != nil || !sameDraftFile(draft.info, after) {
		draft.warning = draft.message("source changed while reading; kept source")
	}
	return content, draft, nil
}

func prepareCapturedDraft(source string) *capturedDraft {
	absolute, err := filepath.Abs(source)
	if err != nil { return nil }
	project, err := os.Getwd()
	if err != nil { return nil }
	for _, family := range []string{"issues", "requirements"} {
		rootPath := filepath.Join(project, ".ai", family, "drafts")
		local, err := filepath.Rel(rootPath, absolute)
		if err != nil || local == "." || !filepath.IsLocal(local) { continue }
		draft := &capturedDraft{rootPath: rootPath, local: local, source: source}
		for _, part := range strings.Split(filepath.ToSlash(source), "/") {
			if part == ".." {
				draft.warning = draft.message("parent traversal is excluded; kept source")
				return draft
			}
		}
		if strings.Contains(local, ":") {
			draft.warning = draft.message("alternate file streams are excluded; kept source")
			return draft
		}
		// Only .ai itself may be the managed shared-storage link. Neither the
		// family directory nor the draft directory may redirect cleanup.
		for index, path := range []string{filepath.Dir(rootPath), rootPath} {
			info, err := os.Lstat(path)
			if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				draft.warning = draft.message("draft storage is not a regular directory; kept source")
				return draft
			}
			if index == 0 { draft.familyInfo = info } else { draft.rootInfo = info }
		}
		draft.root, err = os.OpenRoot(rootPath)
		if err != nil {
			draft.warning = draft.message("cannot open draft storage: %v", err)
			return draft
		}
		err = draft.checkStorage()
		if err == nil { draft.info, err = draft.sourceInfo() }
		if err != nil {
			draft.warning = draft.message("source is not eligible for cleanup: %v", err)
		}
		return draft
	}
	return nil
}

func (draft *capturedDraft) close() {
	if draft.root != nil { draft.root.Close() }
}

// Bind the root handle to the identities checked before OpenRoot. Never adopt
// a replacement directory as a new snapshot, or follow a replacement link.
func (draft *capturedDraft) checkStorage() error {
	for index, path := range []string{filepath.Dir(draft.rootPath), draft.rootPath} {
		current, err := os.Lstat(path)
		if err != nil { return err }
		expected := draft.rootInfo
		if index == 0 { expected = draft.familyInfo }
		if !current.IsDir() || current.Mode()&os.ModeSymlink != 0 || !os.SameFile(expected, current) {
			return errors.New("draft storage identity changed or became a symbolic link")
		}
	}
	opened, err := draft.root.Stat(".")
	if err != nil { return err }
	if !os.SameFile(draft.rootInfo, opened) { return errors.New("opened draft root differs from the original directory") }
	return nil
}

func (draft *capturedDraft) message(format string, args ...any) string {
	return fmt.Sprintf("capture succeeded; draft cleanup warning for %q: %s; do not repeat capture", draft.source, fmt.Sprintf(format, args...))
}

func sameDraftFile(original, current os.FileInfo) bool {
	return original != nil && current != nil && current.Mode().IsRegular() &&
		os.SameFile(original, current) && original.Size() == current.Size() &&
		original.Mode() == current.Mode() && original.ModTime().Equal(current.ModTime())
}

func (draft *capturedDraft) sourceInfo() (os.FileInfo, error) {
	current := ""
	var info os.FileInfo
	for _, part := range strings.Split(draft.local, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		var err error
		info, err = draft.root.Lstat(current)
		if err != nil { return nil, err }
		if info.Mode()&os.ModeSymlink != 0 { return nil, errors.New("symbolic links are excluded") }
	}
	if info == nil || !info.Mode().IsRegular() { return nil, errors.New("expected a regular source file") }
	return info, nil
}

// cleanup runs only after both the artifact and metadata have been saved.
// Quarantine makes a replaced source recoverable instead of unlinking it by
// its old name. This is not a lock against edits by other same-user processes.
func (draft *capturedDraft) cleanup(content []byte) (warning string) {
	if draft.warning != "" { return draft.warning }
	if err := draft.checkStorage(); err != nil {
		return draft.message("draft storage changed: %v; kept source", err)
	}
	current, err := draft.sourceInfo()
	if err != nil || !sameDraftFile(draft.info, current) {
		return draft.message("source changed before cleanup; kept source")
	}
	directory := ".capture-cleanup-" + rand.Text()
	if err := draft.root.Mkdir(directory, 0o700); err != nil {
		return draft.message("cannot create private cleanup directory: %v; kept source", err)
	}
	defer func() {
		if err := draft.root.Remove(directory); err != nil && !errors.Is(err, os.ErrNotExist) {
			if warning != "" { warning += "; " }
			warning += draft.message("cleanup directory remains at %q: %v", filepath.Join(draft.rootPath, directory), err)
		}
	}()
	held := filepath.Join(directory, "source")
	if err := draft.root.Rename(draft.local, held); err != nil {
		return draft.message("cannot isolate source: %v; kept source", err)
	}
	if err := draft.verifyHeld(held, content); err != nil {
		return draft.restore(held, err)
	}
	if err := draft.root.Remove(held); err != nil {
		return draft.restore(held, fmt.Errorf("cannot delete captured source: %w", err))
	}
	return ""
}

func (draft *capturedDraft) verifyHeld(held string, content []byte) error {
	info, err := draft.root.Lstat(held)
	if err != nil { return err }
	if !sameDraftFile(draft.info, info) { return errors.New("isolated source identity or attributes changed") }
	file, err := draft.root.Open(held)
	if err != nil { return err }
	defer file.Close()
	opened, err := file.Stat()
	if err != nil { return err }
	if !sameDraftFile(draft.info, opened) { return errors.New("isolated source changed while opening") }
	// A growing source cannot cause an unbounded second read.
	actual, err := io.ReadAll(io.LimitReader(file, int64(len(content))+1))
	if err != nil { return err }
	after, err := file.Stat()
	if err != nil { return err }
	if !sameDraftFile(draft.info, after) || !bytes.Equal(actual, content) {
		return errors.New("isolated source content changed")
	}
	return nil
}

func (draft *capturedDraft) restore(held string, cause error) string {
	// Link fails if another file now occupies the original path. Rename would
	// overwrite it on some platforms, so unsupported links retain recovery data.
	if err := draft.root.Link(held, draft.local); err != nil {
		return draft.message("%v; source retained at recovery path %q; restore failed: %v", cause, filepath.Join(draft.rootPath, held), err)
	}
	if err := draft.root.Remove(held); err != nil {
		return draft.message("%v; source restored; recovery link also remains at %q: %v", cause, filepath.Join(draft.rootPath, held), err)
	}
	return draft.message("%v; source restored and kept", cause)
}
