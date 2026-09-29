package issue

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const defaultContextBytes = 64 * 1024
const maxContextReferences = 32

var errContextBudget = errors.New("context byte budget exceeded")
var errContextPath = errors.New("context path is not allowed")

// ConversationContextOptions limits raw UTF-8 input bytes, not model tokens.
// Zero MaxBytes uses 64 KiB; overrides may reduce but not increase that bound.
// BackgroundPaths are explicit project-relative document references. ModulePaths
// select only README.md and CONTEXT.md; neither option triggers a directory scan.
type ConversationContextOptions struct {
	MaxBytes        int
	BackgroundPaths []string
	ModulePaths     []string
	Method          *ConversationMethodSuggestion
}

type contextReader struct {
	root      *os.Root
	remaining int
}

func newContextReader(options ConversationContextOptions) (*contextReader, error) {
	limit := options.MaxBytes
	if limit == 0 {
		limit = defaultContextBytes
	}
	if limit < 1 || limit > defaultContextBytes {
		return nil, fmt.Errorf("context budget must be between 1 and %d bytes", defaultContextBytes)
	}
	if len(options.BackgroundPaths)+2*len(options.ModulePaths) > maxContextReferences {
		return nil, errors.New("too many context references; narrow the selected scope")
	}
	for _, path := range append(append([]string{}, options.BackgroundPaths...), options.ModulePaths...) {
		if len(path) > 1024 {
			return nil, errors.New("context reference exceeds 1024 bytes")
		}
	}
	root, err := os.OpenRoot(".")
	if err != nil {
		return nil, err
	}
	return &contextReader{root: root, remaining: limit}, nil
}

// contextLocalPath validates before cleaning so traversal cannot disappear.
// Reject platform-specific aliases on all hosts, including Windows ADS names.
func contextLocalPath(path string) (string, error) {
	portable := strings.ReplaceAll(path, "\\", "/")
	if portable == "" || strings.HasPrefix(portable, "/") || strings.ContainsAny(portable, ":\x00") {
		return "", errContextPath
	}
	for _, part := range strings.Split(portable, "/") {
		lower := strings.ToLower(part)
		stem := strings.TrimSuffix(lower, filepath.Ext(lower))
		if part == ".." || strings.TrimRight(part, " .") != part && part != "." {
			return "", errContextPath
		}
		if lower == ".git" || lower == ".ai" || lower == ".ssh" || lower == ".aws" || lower == ".azure" ||
			lower == ".env" || strings.HasPrefix(lower, ".env.") || lower == "secrets" ||
			strings.Contains(lower, "credential") || lower == "id_rsa" || lower == "id_ed25519" ||
			stem == "secret" || stem == "secrets" || stem == "password" || stem == "passwords" || stem == "tokens" {
			return "", errContextPath
		}
	}
	clean := filepath.Clean(filepath.FromSlash(portable))
	if !filepath.IsLocal(clean) {
		return "", errContextPath
	}
	return clean, nil
}

// read bounds allocation before reading and rejects links at every component.
// os.Root also prevents filesystem traversal during the actual open operation.
func (reader *contextReader) read(path string) ([]byte, error) {
	local, err := contextLocalPath(path)
	if err != nil {
		return nil, err
	}
	var info os.FileInfo
	current := ""
	for _, part := range strings.Split(local, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err = reader.root.Lstat(current)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%w: symbolic links are excluded", errContextPath)
		}
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: expected a regular file", errContextPath)
	}
	file, err := reader.root.Open(local)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, fmt.Errorf("%w: source changed during open", errContextPath)
	}
	if opened.Size() > int64(reader.remaining) {
		return nil, errContextBudget
	}
	content, err := io.ReadAll(io.LimitReader(file, int64(reader.remaining)+1))
	if err != nil {
		return nil, err
	}
	if len(content) > reader.remaining {
		return nil, errContextBudget
	}
	if !utf8.Valid(content) || bytes.ContainsRune(content, 0) {
		return nil, errors.New("context source is not UTF-8 text")
	}
	reader.remaining -= len(content)
	return content, nil
}

// readRequirement checks metadata inside the same root and budget as sources;
// the old unrestricted Read/dirFor path is not used for model context.
func (reader *contextReader) readRequirement(id string) (Meta, string, error) {
	for _, dir := range []string{Dir(id), archiveDir(id), cancelledDir(id)} {
		content, err := reader.read(filepath.Join(dir, "requirement.toml"))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return Meta{}, "", err
		}
		meta, err := readMeta(id, bytes.NewReader(content))
		return meta, dir, err
	}
	return Meta{}, "", fmt.Errorf("requirement %s: %w", id, os.ErrNotExist)
}

// finishContext preserves the required-source result. Optional omissions remain
// visible but never turn a complete set of required inputs into a fatal error.
func (snapshot *ConversationContext) finishContext(reader *contextReader, candidate *ConversationCandidate, options ConversationContextOptions) {
	snapshot.attachCandidate(candidate)
	if snapshot.Candidate != nil {
		if len(snapshot.Candidate.Content) > reader.remaining {
			snapshot.Candidate = nil
			snapshot.CandidateNotice = "Candidate omitted: context byte budget exceeded"
		} else {
			reader.remaining -= len(snapshot.Candidate.Content)
		}
	}
	paths := append([]string{}, options.BackgroundPaths...)
	for _, module := range options.ModulePaths {
		// Do not clean module before validation: ../ must remain rejectable.
		prefix := strings.TrimSuffix(strings.ReplaceAll(module, "\\", "/"), "/")
		paths = append(paths, prefix+"/README.md", prefix+"/CONTEXT.md")
	}
	sort.Strings(paths)
	seen := make(map[string]bool)
	for _, source := range snapshot.Sources {
		seen[filepath.Clean(filepath.FromSlash(source.Path))] = true
	}
	for _, path := range paths {
		source := ConversationSource{Kind: "background", Path: path, Purpose: "Relevant local reference; not confirmed requirement text"}
		local, err := contextLocalPath(path)
		if err == nil {
			ext := strings.ToLower(filepath.Ext(local))
			if ext != ".md" && ext != ".txt" && ext != ".rst" {
				err = fmt.Errorf("%w: background must be a document (.md, .txt, .rst)", errContextPath)
			}
		}
		if err != nil {
			source.Status, source.Reason = "rejected", err.Error()
		} else {
			if seen[local] {
				continue
			}
			seen[local] = true
			source.Path = filepath.ToSlash(local)
			_ = source.load(reader, false) // Diagnostic stays on the optional source.
		}
		snapshot.Sources = append(snapshot.Sources, source)
	}
	snapshot.UsedBytes = snapshot.BudgetBytes - reader.remaining
}
