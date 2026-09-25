package task

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"aiw/internal/session"
	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

func workflowSessionRecoveryFixture(t *testing.T, sessionID string) (taskx.TaskMeta, *workflow.Store) {
	t.Helper()
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AIW_ROOT", root)

	meta := taskx.TaskMeta{ID: "task-1", Status: "TODO", Worktree: root, WorkspaceKind: "primary", Delivery: "unmanaged", Session: sessionID}
	if err := os.MkdirAll(taskx.TaskDir(meta.ID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := taskx.WriteTaskMeta(taskx.TaskMetaPath(meta.ID), meta); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskx.TaskDir(meta.ID), "tasks.md"), []byte("- [ ] 1.1 Prepare Session\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, store, err := compatibleWorkflow(meta.ID)
	if err != nil {
		t.Fatal(err)
	}
	return loaded, store
}

func TestAdvanceWorkflowCreatesMissingTaskSessionAndReusesPreparedRequest(t *testing.T) {
	meta, store := workflowSessionRecoveryFixture(t, "task-1")
	first, err := advanceWorkflow(meta.ID, meta, store, false, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Automation.PreparedRequest == nil || len(first.Attempts) != 1 {
		t.Fatalf("first preparation = %+v", first)
	}
	if _, err := session.NewStore("").Load(meta.Session); err != nil {
		t.Fatalf("created Session: %v", err)
	}
	second, err := advanceWorkflow(meta.ID, meta, store, false, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Attempts) != 1 || second.Automation.PreparedRequest == nil || second.Automation.PreparedRequest.AttemptID != first.Automation.PreparedRequest.AttemptID {
		t.Fatalf("repeated preparation replaced ownership: %+v", second)
	}
}

func TestAdvanceWorkflowDoesNotRebuildNonMissingSessionErrors(t *testing.T) {
	for _, tc := range []struct {
		name      string
		sessionID string
		prepare   func(t *testing.T, meta taskx.TaskMeta)
		missing   bool
	}{
		{name: "different-session-is-missing", sessionID: "other-session", missing: true},
		{
			name:      "status-is-missing",
			sessionID: "task-1",
			prepare: func(t *testing.T, meta taskx.TaskMeta) {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(session.NewStore("").Root, "sessions", meta.Session), 0o755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:      "session-identity-conflicts",
			sessionID: "task-1",
			prepare: func(t *testing.T, meta taskx.TaskMeta) {
				t.Helper()
				path := filepath.Join(session.NewStore("").Root, "sessions", meta.Session)
				if err := os.MkdirAll(path, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(path, "status.json"), []byte(`{"session":{"id":"other-session"}}`), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			meta, store := workflowSessionRecoveryFixture(t, tc.sessionID)
			if tc.prepare != nil {
				tc.prepare(t, meta)
			}
			_, err := advanceWorkflow(meta.ID, meta, store, false, "", "")
			if err == nil {
				t.Fatal("expected Session preparation failure")
			}
			if errors.Is(err, session.ErrSessionNotFound) != tc.missing {
				t.Fatalf("missing classification for %v = %t", err, errors.Is(err, session.ErrSessionNotFound))
			}
			state, loadErr := store.Load(workflow.TaskID(meta.ID))
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			if len(state.Attempts) != 0 || state.Automation.PreparedRequest != nil {
				t.Fatalf("failed preparation retained execution ownership: %+v", state)
			}
			if _, loadErr := session.NewStore("").Load(meta.ID); tc.sessionID != meta.ID && !errors.Is(loadErr, session.ErrSessionNotFound) {
				t.Fatalf("different binding created task-named Session: %v", loadErr)
			}
		})
	}
}

func TestAdvanceWorkflowCreationFailureDoesNotCreateAttempt(t *testing.T) {
	meta, store := workflowSessionRecoveryFixture(t, "task-1")
	if err := os.WriteFile(filepath.Join(taskx.RuntimeTaskDir(meta.ID), "artifacts"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := advanceWorkflow(meta.ID, meta, store, false, "", ""); err == nil {
		t.Fatal("expected Session creation failure")
	}
	state, err := store.Load(workflow.TaskID(meta.ID))
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Attempts) != 0 || state.Automation.PreparedRequest != nil {
		t.Fatalf("creation failure retained execution ownership: %+v", state)
	}
}

func TestAdvanceWorkflowDoesNotRebuildArchivedSession(t *testing.T) {
	meta, store := workflowSessionRecoveryFixture(t, "task-1")
	sessions := session.NewStore("")
	if _, err := sessions.Create(meta.Session, "fixture", ".", "codex", "", "fixture"); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(sessions.Root, "archive", meta.Session)
	if err := os.MkdirAll(filepath.Dir(archive), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(sessions.Root, "sessions", meta.Session), archive); err != nil {
		t.Fatal(err)
	}
	if _, err := advanceWorkflow(meta.ID, meta, store, false, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(sessions.Root, "sessions", meta.Session)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("archived Session was recreated in active storage: %v", err)
	}
}

func TestRepairWorkflowStateRepairsOnlyMissingSessionAttempt(t *testing.T) {
	for _, tc := range []struct {
		name    string
		prepare func(t *testing.T, meta taskx.TaskMeta)
		repair  bool
	}{
		{name: "missing", repair: true},
		{
			name: "corrupt",
			prepare: func(t *testing.T, meta taskx.TaskMeta) {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(session.NewStore("").Root, "sessions", meta.Session), 0o755); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			meta, store := workflowSessionRecoveryFixture(t, "task-1")
			state, err := syncWorkflowChecklist(meta.ID, store)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.StartAttempt(workflow.TaskID(meta.ID), workflow.Attempt{ID: "attempt-1", WorkItemID: state.WorkItems[0].ID, SessionID: meta.Session, Workspace: "."}); err != nil {
				t.Fatal(err)
			}
			if _, err := store.UpdateWithEvent(workflow.TaskID(meta.ID), workflow.Event{Type: "test.paused"}, func(current *workflow.RuntimeState) error {
				current.Automation.Supervisor.Result = "runner-paused"
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if tc.prepare != nil {
				tc.prepare(t, meta)
			}
			err = repairWorkflowState(meta.ID)
			if tc.repair && err != nil {
				t.Fatal(err)
			}
			if !tc.repair && err == nil {
				t.Fatal("expected non-missing Session error")
			}
			updated, loadErr := store.Load(workflow.TaskID(meta.ID))
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			if len(updated.Attempts) != 1 {
				t.Fatalf("repair lost Attempt audit: %+v", updated)
			}
			if tc.repair {
				if updated.Attempts[0].State != workflow.AttemptCancelled || updated.WorkItems[0].State != workflow.WorkItemReady || updated.WriteLease != nil {
					t.Fatalf("missing Session repair = %+v", updated)
				}
			} else if updated.Attempts[0].State != workflow.AttemptRunning || updated.WriteLease == nil {
				t.Fatalf("non-missing Session changed Attempt: %+v", updated)
			}
		})
	}
}
