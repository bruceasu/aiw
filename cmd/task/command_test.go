package task

import "testing"

func TestParseInitOptionsRequiresPromptsWhenUsingTemplate(t *testing.T) {
	_, err := parseInitOptions([]string{"--template", "go"})
	if err == nil {
		t.Fatal("expected template without --prompts to fail")
	}
}

func TestParsePromptOptionsRejectsListWithMerge(t *testing.T) {
	_, err := parsePromptOptions([]string{"list", "--merge"})
	if err == nil {
		t.Fatal("expected prompts list with merge to fail")
	}
}

func TestParseArchiveOptionsFinalizeEnablesAllFlags(t *testing.T) {
	opts, err := parseArchiveOptions([]string{"--finalize"})
	if err != nil {
		t.Fatalf("parse archive options: %v", err)
	}
	if !opts.Push || !opts.CleanupWT || !opts.DeleteBranch {
		t.Fatalf("expected finalize to enable all flags, got %+v", opts)
	}
}
