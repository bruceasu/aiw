package task

import "testing"

func TestParseAIWorkflowOptionsRejectsSessionAndLast(t *testing.T) {
	_, err := parseAIWorkflowOptions([]string{"--session", "abc", "--last"})
	if err == nil {
		t.Fatal("expected session+last to fail")
	}
}

func TestParseAIWorkflowOptionsRejectsApplyAndDryRun(t *testing.T) {
	_, err := parseAIWorkflowOptions([]string{"--apply", "--dry-run"})
	if err == nil {
		t.Fatal("expected apply+dry-run to fail")
	}
}

func TestParseAIWorkflowOptionsParsesValidFlags(t *testing.T) {
	opts, err := parseAIWorkflowOptions([]string{"--session", "abc", "--apply", "--prompt", "hello"})
	if err != nil {
		t.Fatalf("parse options: %v", err)
	}
	if opts.Session != "abc" {
		t.Fatalf("unexpected session: %q", opts.Session)
	}
	if !opts.Apply {
		t.Fatal("expected apply=true")
	}
	if opts.DryRun {
		t.Fatal("expected dryRun=false")
	}
	if opts.Prompt != "hello" {
		t.Fatalf("unexpected prompt: %q", opts.Prompt)
	}
}

func TestParseAIArchiveOptionsAllowsArchiveAndAIFlags(t *testing.T) {
	aopts, aiopts, err := parseAIArchiveOptions([]string{"--finalize", "--last", "--prompt", "p"})
	if err != nil {
		t.Fatalf("parse archive ai options: %v", err)
	}
	if !aopts.Push || !aopts.CleanupWT || !aopts.DeleteBranch {
		t.Fatalf("finalize flags not expanded: %+v", aopts)
	}
	if !aiopts.Last {
		t.Fatal("expected aiopts.Last=true")
	}
	if aiopts.Prompt != "p" {
		t.Fatalf("unexpected prompt: %q", aiopts.Prompt)
	}
}

func TestDryRunOutputPathNotEmpty(t *testing.T) {
	p := dryRunOutputPath()
	if p == "" {
		t.Fatal("expected non-empty output path")
	}
}

func TestIsHelpFlag(t *testing.T) {
	if !isHelpFlag("-h") || !isHelpFlag("--help") || !isHelpFlag("help") {
		t.Fatal("expected help flags to be recognized")
	}
	if isHelpFlag("new") {
		t.Fatal("did not expect non-help token to be recognized as help")
	}
}

func TestAIActionUsage(t *testing.T) {
	usage, ok := aiActionUsage("new")
	if !ok {
		t.Fatal("expected new action to have usage")
	}
	if usage == "" {
		t.Fatal("expected non-empty usage for new action")
	}
	_, ok = aiActionUsage("unknown")
	if ok {
		t.Fatal("expected unknown action to have no usage")
	}
}

func TestRunAIWorkflowRejectsInvalidID(t *testing.T) {
	err := runAIWorkflow([]string{"new", "bad id", "--dry-run"})
	if err == nil {
		t.Fatal("expected invalid id to fail")
	}
}
