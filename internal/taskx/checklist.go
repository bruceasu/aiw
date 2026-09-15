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
	dependencyCommentPattern = regexp.MustCompile(`\s*<!--\s*aiw:depends-on=([^>]+)-->\s*$`)
	dependencyReferencePattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)+$`)
)

// ChecklistItem is a supported, human-authored numbered checkbox from an
// OpenSpec tasks.md file. Number stays human-readable; Workflow Core assigns
// the stable WorkItemID when the item is first planned.
type ChecklistItem struct {
	Number    string
	Title     string
	Completed bool
	Line      int
	DependsOn []string
}

// ChecklistDiagnostic reports a non-mutating parse problem. Callers must not
// plan or reconcile a checklist when diagnostics are present.
type ChecklistDiagnostic struct {
	Code    string
	Line    int
	Message string
}

// ChecklistProjectionError identifies a checklist condition that prevents an
// adapter from projecting a completed Work Item. Its Code is suitable for a
// durable reconciliation Gate; callers must not guess which authored entry to
// change when the identifier is duplicated or absent.
type ChecklistProjectionError struct {
	Code    string
	Item    string
	Message string
}

func (e *ChecklistProjectionError) Error() string {
	return e.Message
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
		return nil, &ChecklistProjectionError{
			Code:    diagnostics[0].Code,
			Message: "cannot read checklist: " + diagnostics[0].Message,
		}
	}
	return items, nil
}

// ProjectWorkflowChecklistCompletion marks exactly one numbered checklist
// entry complete. It preserves all other authored content and refuses to write
// when parsing detects an ambiguous identifier or the mapped item was deleted.
func ProjectWorkflowChecklistCompletion(path, number string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	items, diagnostics := ParseNumberedChecklist(string(content))
	if len(diagnostics) > 0 {
		return &ChecklistProjectionError{
			Code:    diagnostics[0].Code,
			Item:    number,
			Message: "cannot project checklist completion: " + diagnostics[0].Message,
		}
	}
	for _, item := range items {
		if item.Number != number {
			continue
		}
		if item.Completed {
			return nil
		}
		lines := strings.Split(string(content), "\n")
		line := lines[item.Line-1]
		matches := numberedChecklistPattern.FindStringSubmatch(line)
		if matches == nil {
			return fmt.Errorf("checklist item %s changed during projection", number)
		}
		lines[item.Line-1] = strings.Replace(line, "[ ]", "[x]", 1)
		return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
	}
	return &ChecklistProjectionError{
		Code:    "deleted-checklist-identifier",
		Item:    number,
		Message: fmt.Sprintf("cannot project checklist completion: mapped item %s is absent from tasks.md", number),
	}
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
		title, dependencies, dependencyDiagnostic := parseChecklistDependencies(matches[3], lineNumber)
		if dependencyDiagnostic != nil {
			diagnostics = append(diagnostics, *dependencyDiagnostic)
			continue
		}
		items = append(items, ChecklistItem{
			Number:    number,
			Title:     title,
			Completed: strings.EqualFold(matches[1], "x"),
			Line:      lineNumber,
			DependsOn: dependencies,
		})
	}
	return items, diagnostics
}

func parseChecklistDependencies(title string, line int) (string, []string, *ChecklistDiagnostic) {
	matches := dependencyCommentPattern.FindStringSubmatch(title)
	if matches == nil {
		return title, nil, nil
	}
	dependencies := []string{}
	seen := map[string]bool{}
	for _, dependency := range strings.Split(matches[1], ",") {
		dependency = strings.TrimSpace(dependency)
		if !dependencyReferencePattern.MatchString(dependency) || seen[dependency] {
			return "", nil, &ChecklistDiagnostic{Code: "invalid-checklist-dependency", Line: line, Message: "expected unique dotted checklist identifiers in aiw:depends-on"}
		}
		seen[dependency] = true
		dependencies = append(dependencies, dependency)
	}
	if len(dependencies) == 0 {
		return "", nil, &ChecklistDiagnostic{Code: "invalid-checklist-dependency", Line: line, Message: "aiw:depends-on requires at least one dotted checklist identifier"}
	}
	return strings.TrimSpace(strings.TrimSuffix(title, matches[0])), dependencies, nil
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
		for _, dependency := range item.DependsOn {
			hash.Write([]byte(dependency))
			hash.Write([]byte{0})
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}
