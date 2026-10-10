package issue

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const IssueRoot = "docs/issues"

var numberedIssue = regexp.MustCompile(`(?i)^ISSUE-([0-9]{3,})$`)
var shortRequirement = regexp.MustCompile(`(?i)^REQ[0-9]+$`)

type recordLocation struct {
	id string
	dir string
	metadata string
}

func recordRoot(id string) string {
	if numberedIssue.MatchString(id) { return IssueRoot }
	return Root
}

func metadataName(root string) string {
	if root == IssueRoot { return "issue.toml" }
	return "requirement.toml"
}

// All storage callers use the same exact-first identity resolution. The
// directory name, not a caller's abbreviation, is the canonical identity.
func locateRecord(id string) (recordLocation, error) {
	if !ValidID(id) || id == "." || id == ".." || strings.TrimRight(id, " .") != id {
		return recordLocation{}, errors.New("invalid Issue id")
	}
	locations, err := recordLocations(ListAll)
	if err != nil { return recordLocation{}, err }
	var exact, abbreviated []recordLocation
	for _, location := range locations {
		if strings.EqualFold(location.id, id) {
			exact = append(exact, location)
		} else if shortRequirement.MatchString(id) && strings.HasPrefix(strings.ToUpper(location.id), strings.ToUpper(id)+"-") {
			abbreviated = append(abbreviated, location)
		}
	}
	matches := exact
	if len(matches) == 0 { matches = abbreviated }
	if len(matches) == 0 { return recordLocation{}, fmt.Errorf("Issue %s: %w", id, os.ErrNotExist) }
	if len(matches) > 1 { return recordLocation{}, fmt.Errorf("ambiguous Issue id %s (%d records)", id, len(matches)) }
	return matches[0], nil
}

func recordLocations(filter ListFilter) ([]recordLocation, error) {
	var result []recordLocation
	for _, root := range []string{IssueRoot, Root} {
		suffixes := []string{""}
		switch filter {
		case ListArchived: suffixes = []string{archiveRoot}
		case ListCancelled: suffixes = []string{cancelledRoot}
		case ListAll: suffixes = []string{"", archiveRoot, cancelledRoot}
		}
		for _, suffix := range suffixes {
			base := filepath.Join(root, suffix)
			if err := checkRecordPath(base); errors.Is(err, os.ErrNotExist) { continue } else if err != nil { return nil, err }
			entries, err := os.ReadDir(base)
			if err != nil { return nil, err }
			for _, entry := range entries {
				if entry.Name() == archiveRoot || entry.Name() == cancelledRoot || !ValidID(entry.Name()) { continue }
				if entry.Type()&os.ModeSymlink != 0 { return nil, fmt.Errorf("Issue directory must not be a link: %s", filepath.Join(base, entry.Name())) }
				if !entry.IsDir() { continue }
				dir := filepath.Join(base, entry.Name())
				metadata := filepath.Join(dir, metadataName(root))
				if err := checkRecordPath(metadata); errors.Is(err, os.ErrNotExist) { continue } else if err != nil { return nil, err }
				info, err := os.Stat(metadata)
				if err != nil { return nil, err }
				if !info.Mode().IsRegular() { return nil, fmt.Errorf("Issue metadata must be a regular file: %s", metadata) }
				result = append(result, recordLocation{entry.Name(), dir, metadata})
			}
		}
	}
	return result, nil
}

// Reject links in every component, including a redirected docs/ root.
func checkRecordPath(path string) error {
	if !filepath.IsLocal(path) { return errors.New("Issue path must be project-relative") }
	current := ""
	for _, part := range strings.Split(filepath.Clean(path), string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil { return err }
		if info.Mode()&os.ModeSymlink != 0 { return fmt.Errorf("Issue path must not be a link: %s", current) }
	}
	return nil
}

func artifactFilename(dir, kind string) string {
	if kind == "issue-plan" { kind = "requirement-plan" }
	if kind == "requirement-plan" && strings.HasPrefix(filepath.ToSlash(dir), IssueRoot+"/") { return "issue-plan.md" }
	return artifactFiles[kind]
}

func canonicalArtifactKind(kind string) string {
	if kind == "issue-plan" { return "requirement-plan" }
	return kind
}
