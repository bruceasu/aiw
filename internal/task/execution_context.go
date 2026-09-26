package task

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"aiw/internal/workflow"
)

var contextLink = regexp.MustCompile(`\]\(([^)]+)\)`)

// LoadExecutionSources expands local source links from the actual Task
// workspace. Optional knowledge is explicit; only primary-source links can
// promote a knowledge version to a required input. No project history scan.
func LoadExecutionSources(id, root, handoff string) ([]workflow.InputSource, error) {
	change := filepath.Join("openspec", "changes", id)
	paths := []string{filepath.Join(change, "proposal.md"), filepath.Join(change, "design.md"), filepath.Join(change, "tasks.md")}
	specs, err := filepath.Glob(filepath.Join(root, change, "specs", "*", "spec.md"))
	if err != nil { return nil, err }
	if len(specs) == 0 { return nil, fmt.Errorf("required capability specs are missing for Task %s", id) }
	for _, path := range specs { relative, err := filepath.Rel(root, path); if err != nil { return nil, err }; paths = append(paths, relative) }
	sort.Strings(paths)
	queue := make([]workflow.InputSource, 0, len(paths))
	for _, path := range paths { queue = append(queue, workflow.InputSource{Kind: "task-source", Path: filepath.ToSlash(path), Required: true}) }
	seen := make(map[string]string)
	var sources []workflow.InputSource
	bytesRead := 0
	for len(queue) > 0 {
		source := queue[0]
		queue = queue[1:]
		source.Path = filepath.ToSlash(filepath.Clean(filepath.FromSlash(source.Path)))
		if prior, ok := seen[source.Path]; ok {
			if source.ExpectedSHA256 != "" && source.ExpectedSHA256 != prior { return nil, fmt.Errorf("conflicting exact source versions: %s", source.Path) }
			continue
		}
		source = workflow.ReadInputSource(root, source)
		if source.Status != "loaded" { return nil, fmt.Errorf("required context %s: %s", source.Path, source.Reason) }
		seen[source.Path] = source.SHA256
		bytesRead += len(source.Content)
		// Bound the synchronous source expansion independently of auxiliary
		// history budgets. Oversize input is a gap, never silent truncation.
		if bytesRead > 4*1024*1024 || len(sources) >= 128 { return nil, fmt.Errorf("required context exceeds 4 MiB or 128 source files") }
		sources = append(sources, source)
		for _, link := range contextLink.FindAllStringSubmatch(source.Content, -1) {
			target := strings.Trim(link[1], " <>")
			path, fragment, _ := strings.Cut(target, "#")
			if path == "" || strings.Contains(path, "://") || !strings.EqualFold(filepath.Ext(path), ".md") { continue }
			expected := ""
			if strings.HasPrefix(fragment, "sha256=") { expected = strings.TrimPrefix(fragment, "sha256=") }
			queue = append(queue, workflow.InputSource{Kind: "primary-reference", Path: filepath.ToSlash(filepath.Join(filepath.Dir(source.Path), filepath.FromSlash(path))), Required: true, ExpectedSHA256: expected})
		}
	}
	// Handoff content is context, not a source of new requirement links.
	handoffRoot, err := filepath.Abs(RuntimeTaskDir(id))
	if err != nil { return nil, err }
	handoff, err = filepath.Abs(handoff)
	if err != nil { return nil, err }
	relative, err := filepath.Rel(handoffRoot, handoff)
	if err != nil { return nil, err }
	source := workflow.ReadInputSource(handoffRoot, workflow.InputSource{Kind: "handoff", Path: filepath.ToSlash(relative), Required: true})
	if source.Status != "loaded" { return nil, fmt.Errorf("required handoff: %s", source.Reason) }
	sources = append(sources, source)
	// The absence of an optional summary means raw sources above are used.
	summary := workflow.ReadInputSource(handoffRoot, workflow.InputSource{Kind: "optional-summary", Path: "artifacts/summary.md"})
	if summary.Status != "loaded" { summary.Reason = "raw-source fallback: " + summary.Reason }
	sources = append(sources, summary)
	return sources, nil
}

func RenderExecutionSources(sources []workflow.InputSource) string {
	var text strings.Builder
	text.WriteString("\n\n[Fixed source contents]\nThese source documents define the task. Handoff and optional summaries do not add requirements.\n")
	for _, source := range sources {
		fmt.Fprintf(&text, "\nSource: %s (%s)\nRead status: %s\nSHA256: %s\n", source.Path, source.Kind, source.Status, source.SHA256)
		if source.Reason != "" { fmt.Fprintf(&text, "Reason: %s\n", source.Reason) }
		if source.Status == "loaded" { text.WriteString(source.Content); text.WriteString("\n[End source]\n") }
	}
	return text.String()
}

// ValidateDependencyInputs only exposes completed, currently applicable
// dependency artifacts. Tester context is handled separately by its adapter.
func ValidateDependencyInputs(state workflow.RuntimeState, itemID workflow.WorkItemID, root string) ([]workflow.InputSource, error) {
	var dependencies []workflow.WorkItemID
	for _, item := range state.WorkItems { if item.ID == itemID { dependencies = item.Dependencies } }
	var sources []workflow.InputSource
	for _, id := range dependencies {
		found := false
		for _, item := range state.WorkItems {
			if item.ID != id { continue }
			found = true
			if item.State != workflow.WorkItemCompleted || item.LastOutputReference == "" { return nil, fmt.Errorf("dependency %s lacks accepted output", id) }
			if item.AcceptedReference == nil { return nil, fmt.Errorf("dependency %s needs accepted input-version evidence before downstream use", id) }
			store := workflow.NewStore("")
			evidence, err := store.ReadAcceptedExecution(state.Task.ID, item, *item.AcceptedReference, root)
			if err != nil { return nil, err }
			var report workflow.ExecutionReport
			if err := store.ReadExecutionArtifact(state.Task.ID, *evidence.Report, &report); err != nil { return nil, err }
			sources = append(sources, workflow.InputSource{Kind: "accepted-dependency", Path: evidence.Report.Path, Required: true, SHA256: report.OutputSHA256, Status: "loaded", Content: report.Output})
		}
		if !found { return nil, fmt.Errorf("required dependency %s is missing", id) }
	}
	return sources, nil
}

// Keep filesystem provenance explicit for adapters without importing CLI UI.
func ExecutionWorkspace(workspace string) (string, error) {
	if !filepath.IsAbs(workspace) { workspace = filepath.Join(RuntimeRoot(), filepath.FromSlash(workspace)) }
	root, err := filepath.Abs(workspace)
	if err != nil { return "", err }
	if info, err := os.Stat(root); err != nil || !info.IsDir() { return "", fmt.Errorf("execution workspace is unavailable: %s", root) }
	return root, nil
}
