package task

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFailureReportCommandNeedsNoSessionOutputs(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	if err := os.Chdir(root); err != nil { t.Fatal(err) }
	t.Cleanup(func() { if err := os.Chdir(previous); err != nil { t.Error(err) } })
	t.Setenv("PATH", "")
	dir := filepath.Join(".ai", "task-1", "reports")
	if err := os.MkdirAll(dir, 0o755); err != nil { t.Fatal(err) }
	report := "# Latest unresolved failure\n\n- Category: compiler\n- Detail: undefined symbol\n- Retryable: false\n- Owner: operator\n- Next action: inspect compile-diagnostics/wi-0001.json\n"
	if err := os.WriteFile(filepath.Join(dir, "latest-failure.md"), []byte(report), 0o644); err != nil { t.Fatal(err) }
	output, err := os.CreateTemp(root, "report-output-")
	if err != nil { t.Fatal(err) }
	defer output.Close()
	previousStdout := os.Stdout
	os.Stdout = output
	defer func() { os.Stdout = previousStdout }()
	if err := runWorkflowCommand([]string{"report", "task-1"}); err != nil { t.Fatal(err) }
	os.Stdout = previousStdout
	content, err := os.ReadFile(output.Name())
	if err != nil { t.Fatal(err) }
	if string(content) != report { t.Fatalf("report output = %q", content) }
	for _, path := range []string{filepath.Join(".ai", "sessions"), filepath.Join(".ai", "task-1", "state.json"), filepath.Join(".ai", "task-1", "events.jsonl")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) { t.Fatalf("read-only report created runtime data at %s: %v", path, err) }
	}
}
