package openspecgen

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aiw/internal/requirement"
)

func applicationFixture(t *testing.T) (string, requirement.GenerationRequest, *requirement.GenerationRecord, []Artifact) {
	t.Helper()
	request, candidate := businessCandidate()
	for index := range request.Targets {
		request.Targets[index].Digest = "absent"
	}
	artifacts, err := RenderCandidate(request, candidate)
	if err != nil {
		t.Fatal(err)
	}
	record := &requirement.GenerationRecord{
		SchemaVersion: request.SchemaVersion,
		RequestID:     request.RequestID,
		InputDigest:   request.InputDigest,
		State:         requirement.GenerationValidating,
		Candidate:     &candidate,
	}
	return t.TempDir(), request, record, artifacts
}

func TestProtectedApplicationRejectsHumanEditAfterInterruptedWrite(t *testing.T) {
	root, request, record, artifacts := applicationFixture(t)
	persisted, saves := requirement.GenerationRecord{}, 0
	_, err := ApplyCandidate(root, request, record, ApplyHooks{
		Save: func(value requirement.GenerationRecord) error {
			saves++
			persisted = value
			if saves == 2 {
				return errors.New("simulated filesystem checkpoint failure")
			}
			return nil
		},
		CheckContext: func() error { return nil },
	})
	if err == nil || persisted.State != requirement.GenerationApplying {
		t.Fatalf("interrupted application = %+v, %v", persisted, err)
	}
	if err := os.WriteFile(filepath.Join(root, "proposal.md"), []byte("human-authored proposal\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyCandidate(root, request, &persisted, ApplyHooks{
		Save: func(requirement.GenerationRecord) error { return nil }, CheckContext: func() error { return nil },
	}); err == nil || !strings.Contains(err.Error(), "unowned or human edit: proposal.md") {
		t.Fatalf("human edit was not protected: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(root, "proposal.md"))
	if err != nil || string(content) != "human-authored proposal\n" {
		t.Fatalf("human content changed: %q, %v", content, err)
	}
	if _, err := os.Stat(filepath.Join(root, artifacts[1].Path)); !os.IsNotExist(err) {
		t.Fatalf("application continued after human-edit conflict: %v", err)
	}
}

func TestProtectedApplicationRecoversPartialWriteWithoutRecreatingArtifacts(t *testing.T) {
	root, request, record, artifacts := applicationFixture(t)
	persisted, saves := requirement.GenerationRecord{}, 0
	_, err := ApplyCandidate(root, request, record, ApplyHooks{
		Save: func(value requirement.GenerationRecord) error {
			saves++
			persisted = value
			if saves == 2 {
				return errors.New("simulated filesystem checkpoint failure")
			}
			return nil
		},
		CheckContext: func() error { return nil },
	})
	if err == nil {
		t.Fatal("interrupted write succeeded")
	}
	result, err := ApplyCandidate(root, request, &persisted, ApplyHooks{
		Save: func(value requirement.GenerationRecord) error { persisted = value; return nil }, CheckContext: func() error { return nil },
	})
	if err != nil || len(result.Pending) != 0 || len(result.Written) != len(artifacts) {
		t.Fatalf("partial-write recovery = %+v, %v", result, err)
	}
	for _, artifact := range artifacts {
		content, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifact.Path)))
		if readErr != nil || string(content) != string(artifact.Content) {
			t.Fatalf("recovery content for %s = %q, %v", artifact.Path, content, readErr)
		}
	}
}

func TestAcceptanceRecoversSyncFailureWithoutDuplicateApplication(t *testing.T) {
	root, request, record, artifacts := applicationFixture(t)
	persisted, syncCalls := requirement.GenerationRecord{}, 0
	hooks := AcceptanceHooks{
		ApplyHooks: ApplyHooks{Save: func(value requirement.GenerationRecord) error { persisted = value; return nil }, CheckContext: func() error { return nil }},
		SyncChecklist: func() error {
			syncCalls++
			if syncCalls == 1 {
				return errors.New("simulated checklist sync failure")
			}
			return nil
		},
	}
	if _, err := AcceptCandidate(root, request, record, hooks); err == nil || persisted.State != requirement.GenerationApplying {
		t.Fatalf("sync failure accepted generation: %+v, %v", persisted, err)
	}
	if len(persisted.Application.Files) != len(artifacts) {
		t.Fatalf("application manifest = %+v", persisted.Application)
	}
	result, err := AcceptCandidate(root, request, &persisted, hooks)
	if err != nil || persisted.State != requirement.GenerationAccepted || syncCalls != 2 || len(result.Pending) != 0 || len(persisted.Application.Files) != len(artifacts) {
		t.Fatalf("sync recovery = %+v, %+v, calls=%d, err=%v", persisted, result, syncCalls, err)
	}
}
