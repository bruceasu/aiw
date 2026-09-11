package taskx

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	numberedChecklistPattern = regexp.MustCompile(`^- \[([ xX])\] ([0-9]+(?:\.[0-9]+)+)\s+(.+?)\s*$`)
	numberedChecklistPrefix  = regexp.MustCompile(`^- \[[^\]]*\] [0-9]`)
)

// ChecklistItem is a supported, human-authored numbered checkbox from an
// OpenSpec tasks.md file. Number stays human-readable; Workflow Core assigns
// the stable WorkItemID when the item is first planned.
type ChecklistItem struct {
	Number    string
	Title     string
	Completed bool
	Line      int
}

// ChecklistDiagnostic reports a non-mutating parse problem. Callers must not
// plan or reconcile a checklist when diagnostics are present.
type ChecklistDiagnostic struct {
	Code    string
	Line    int
	Message string
}

// ReadWorkflowChecklist reads an authored OpenSpec checklist without
// modifying the source file.
func ReadWorkflowChecklist(path string) ([]ChecklistItem, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	items, diagnostics := ParseNumberedChecklist(string(content))
	if len(diagnostics) > 0 {
		return nil, fmt.Errorf("cannot read checklist: %s", diagnostics[0].Message)
	}
	return items, nil
}

// ParseNumberedChecklist accepts only top-level checkbox entries using a
// dotted numeric identifier (for example "1.2").
func ParseNumberedChecklist(content string) ([]ChecklistItem, []ChecklistDiagnostic) {
	var items []ChecklistItem
	var diagnostics []ChecklistDiagnostic
	seen := make(map[string]int)

	for index, line := range strings.Split(content, "\n") {
		lineNumber := index + 1
		matches := numberedChecklistPattern.FindStringSubmatch(line)
		if matches == nil {
			if numberedChecklistPrefix.MatchString(line) {
				diagnostics = append(diagnostics, ChecklistDiagnostic{
					Code:    "unsupported-numbered-checklist",
					Line:    lineNumber,
					Message: "expected a top-level checkbox with a dotted numeric identifier and title",
				})
			}
			continue
		}

		number := matches[2]
		if previousLine, exists := seen[number]; exists {
			diagnostics = append(diagnostics, ChecklistDiagnostic{
				Code:    "duplicate-checklist-number",
				Line:    lineNumber,
				Message: fmt.Sprintf("checklist number %s duplicates line %d", number, previousLine),
			})
			continue
		}
		seen[number] = lineNumber
		items = append(items, ChecklistItem{
			Number:    number,
			Title:     matches[3],
			Completed: strings.EqualFold(matches[1], "x"),
			Line:      lineNumber,
		})
	}
	return items, diagnostics
}

// ChecklistFingerprint is stable for equivalent supported checklist content.
// It intentionally excludes source line positions so reordering headings does
// not create a different plan identity.
func ChecklistFingerprint(items []ChecklistItem) string {
	hash := sha256.New()
	for _, item := range items {
		hash.Write([]byte(item.Number))
		hash.Write([]byte{0})
		hash.Write([]byte(item.Title))
		hash.Write([]byte{0})
		if item.Completed {
			hash.Write([]byte("x"))
		} else {
			hash.Write([]byte(" "))
		}
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
