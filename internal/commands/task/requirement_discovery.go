package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"aiw/internal/requirement"
	"aiw/internal/session"
)

// runRequirementDiscovery is the Session adapter. Domain code owns routing and
// validation; Session keeps the actual prompts and raw model outputs.
func runRequirementDiscovery(ctx context.Context, store *session.Store, plan requirementChatPlan, input string) (requirement.ConversationTurn, error) {
	var facts requirement.ConfirmedFacts
	content, err := store.ReadArtifact(plan.SessionID, "confirmed-requirement-facts.json")
	if err == nil {
		if err := json.Unmarshal([]byte(content), &facts); err != nil { return requirement.ConversationTurn{}, err }
	} else if !os.IsNotExist(err) { return requirement.ConversationTurn{}, err }
	history, err := readRequirementHistory(store, plan.SessionID)
	if err != nil { return requirement.ConversationTurn{}, err }
	candidate, notice := requirement.RestoreConversationCandidate(history, plan.RequirementID, facts)
	fmt.Println(notice)
	record := requirement.ConversationEvidence{Version: 1, RequirementID: plan.RequirementID, State: "running", Confirmed: facts}
	record.Turn.Context.UserInput = input
	recordName := fmt.Sprintf("requirement-discussion-%d.json", time.Now().UTC().UnixNano())
	if err := writeRequirementEvidence(store, plan.SessionID, recordName, record); err != nil { return requirement.ConversationTurn{}, err }
	index, err := json.Marshal(recordName)
	if err != nil { return requirement.ConversationTurn{}, err }
	if err := store.WriteArtifact(plan.SessionID, "requirement-discussion-latest.json", index); err != nil { return requirement.ConversationTurn{}, err }
	// Old pending actions must not survive a new answer or a failed assessment.
	if err := store.WriteArtifact(plan.SessionID, "pending-requirement-action.json", []byte("{}")); err != nil { return requirement.ConversationTurn{}, err }
	turn, runErr := requirement.RunConversationTurn(ctx, plan.RequirementID, input, candidate, facts, func(ctx context.Context, phase, prompt string) (string, error) {
		if err := store.WriteArtifact(plan.SessionID, "pending-requirement-action.json", []byte("{}")); err != nil { return "", err }
		before, err := store.Load(plan.SessionID)
		if err != nil { return "", err }
		result, err := session.ExecuteTurnWithOverrides(ctx, store, plan.SessionID, phase, requirementConversationInstructions()+"\n"+prompt, plan.Provider, plan.Model, false)
		after, readErr := store.Load(plan.SessionID)
		if readErr == nil && after.Session.LastTurn == before.Session.LastTurn+1 {
			record.Turns = append(record.Turns, after.Session.LastTurn)
		} else if readErr == nil { readErr = errors.New("Session turn was not durably recorded") }
		err = errors.Join(err, readErr)
		if err == nil && result.ExitCode != 0 { err = fmt.Errorf("model exited with code %d", result.ExitCode) }
		return result.FinalOutput, err
	})
	record.Turn = turn
	record.State = "completed"
	if runErr != nil { record.State, record.Diagnostic = "failed", runErr.Error() }
	if saveErr := writeRequirementEvidence(store, plan.SessionID, recordName, record); saveErr != nil {
		runErr = errors.Join(runErr, fmt.Errorf("save discussion evidence: %w", saveErr))
	}
	phase := turn.Phase
	if runErr != nil {
		phase = "discovery-blocked"
		if err := store.WriteArtifact(plan.SessionID, "pending-requirement-action.json", []byte("{}")); err != nil { return turn, errors.Join(runErr, err) }
	}
	_, err = store.Update(plan.SessionID, func(status *session.Status) error { status.Session.CurrentPhase = phase; return nil })
	return turn, errors.Join(runErr, err)
}

// The latest pointer is installed before model execution. Never search older
// records when it names an incomplete, missing, or invalid latest discussion.
func readRequirementHistory(store *session.Store, id string) (string, error) {
	index, err := store.ReadArtifact(id, "requirement-discussion-latest.json")
	if os.IsNotExist(err) { return "", nil }
	if err != nil { return "", err }
	var name string
	if err := json.Unmarshal([]byte(index), &name); err != nil { return "invalid", nil }
	content, err := store.ReadArtifact(id, name)
	if os.IsNotExist(err) { return "invalid", nil }
	return content, err
}

func writeRequirementEvidence(store *session.Store, id, name string, record requirement.ConversationEvidence) error {
	b, err := json.Marshal(record)
	if err != nil { return err }
	return store.WriteArtifact(id, name, b)
}

func printRequirementDiscovery(turn requirement.ConversationTurn) {
	fmt.Printf("Phase: %s; method: %s (%s)\n", turn.Phase, turn.Context.MethodSelection.Primary, turn.Context.MethodSelection.Status)
	for _, item := range turn.Discovery.Settled { fmt.Printf("[%s] %s: %s\n", item.Status, item.Dimension, item.Conclusion) }
	for i, item := range turn.Discovery.Questions {
		fmt.Printf("%d. [%s] %s\nImpact: %s\n", i+1, item.Status, item.Question, item.Impact)
		for _, ref := range item.Sources { fmt.Printf("  %s: %s\n", ref.Source, ref.Quote) }
		for _, option := range item.Options { fmt.Printf("  %s: %s\n", option.Label, option.Tradeoff) }
	}
	if len(turn.Discovery.Questions) == 0 { fmt.Println("No open questions in this assessment. This is not approval.") }
	printRequirementReadiness(turn.Readiness)
	if turn.Context.Requirement != nil && turn.Context.Requirement.Status == "APPROVED" && !turn.Readiness.Ready {
		fmt.Println("Risk found in an approved Requirement. Existing approval is unchanged; ask the owner for a decision.")
	}
	if review := turn.Discovery.Coverage.Assessment.PlanReview; review != nil {
		for _, section := range review.Sections {
			fmt.Printf("Plan %s: %s\n", section.Section, section.Explanation)
			for _, ref := range section.Sources { fmt.Printf("  %s: %s\n", ref.Source, ref.Quote) }
		}
	}
}

// displayRequirementCheckpoint snapshots exactly what the human sees. The loop
// requires the pending bytes to remain unchanged before accepting confirmation.
func displayRequirementCheckpoint(store *session.Store, plan requirementChatPlan) (string, error) {
	content, err := store.ReadArtifact(plan.SessionID, "pending-requirement-action.json")
	if os.IsNotExist(err) { return "", nil }
	if err != nil { return "", err }
	var action requirementPendingAction
	if err := json.Unmarshal([]byte(content), &action); err != nil { return "", err }
	if action.Kind == "" { return "", nil }
	if plan.RequirementID != "" && action.RequirementID != plan.RequirementID { return "", errors.New("checkpoint targets another Requirement") }
	if action.Kind == "approve" && strings.ToUpper(action.Decision) == "APPROVED" {
		checkpoint, err := currentApprovalCheckpoint(store, plan.SessionID, content, action.RequirementID)
		if err != nil {
			fmt.Printf("Approval suggestion withheld: %v. Continue discussion or save a draft.\n", err)
			return "", store.WriteArtifact(plan.SessionID, "pending-requirement-action.json", []byte("{}"))
		}
		b, err := json.Marshal(checkpoint)
		if err != nil { return "", err }
		if err := store.WriteArtifact(plan.SessionID, "approval-checkpoint.json", b); err != nil { return "", err }
	}
	if action.Kind == "capture" {
		refs, hash, err := requirement.CaptureFactReferences(action.RequirementID, action.Artifact, action.Source, action.Facts)
		if err != nil { return "", err }
		meta, err := requirement.Read(action.RequirementID)
		if err != nil { return "", err }
		checkpoint := captureCheckpoint{Action: content, Digest: hash, References: refs, Revision: meta.Revision}
		b, err := json.Marshal(checkpoint)
		if err != nil { return "", err }
		if err := store.WriteArtifact(plan.SessionID, "capture-checkpoint.json", b); err != nil { return "", err }
		fmt.Printf("Capture draft: %s -> %s/%s\n", action.Source, action.RequirementID, action.Artifact)
		if len(refs) == 0 { fmt.Println("Draft only: no facts will be marked confirmed.") }
		for _, ref := range refs { fmt.Printf("Fact to confirm: %s\n", ref.Quote) }
	}
	fmt.Printf("Pending action: %s\nType confirm or 确认 to accept this checkpoint.\n", content)
	return content, nil
}

type captureCheckpoint struct {
	Revision int
	Action string
	Digest string
	References []requirement.CoverageReference
}

// confirmRequirementCapture never infers confirmed facts from capture alone.
// Facts are installed only if the captured bytes match the displayed digest.
func confirmRequirementCapture(store *session.Store, sessionID string, action requirementPendingAction) error {
	var checkpoint captureCheckpoint
	if content, err := store.ReadArtifact(sessionID, "capture-checkpoint.json"); err == nil {
		if err := json.Unmarshal([]byte(content), &checkpoint); err != nil { return err }
	} else if len(action.Facts) != 0 { return err }
	if len(action.Facts) != 0 && checkpoint.Action == "" { return errors.New("facts require a displayed capture checkpoint") }
	if checkpoint.Action != "" {
		pending, err := store.ReadArtifact(sessionID, "pending-requirement-action.json")
		if err != nil || checkpoint.Action != pending { return errors.New("capture checkpoint changed; prepare it again") }
		meta, err := requirement.Read(action.RequirementID)
		if err != nil { return err }
		if meta.Revision != checkpoint.Revision { return errors.New("Requirement changed after checkpoint; prepare it again") }
		_, hash, err := requirement.CaptureFactReferences(action.RequirementID, action.Artifact, action.Source, action.Facts)
		if err != nil { return err }
		if hash != checkpoint.Digest { return errors.New("draft changed after display; prepare it again") }
	}
	meta, artifact, err := requirement.Capture(action.RequirementID, action.Artifact, action.Source)
	if err != nil { return err }
	if checkpoint.Action != "" && artifact.Digest != checkpoint.Digest { return errors.New("captured draft changed; facts were not confirmed") }
	facts := requirement.ConfirmedFacts{RequirementID: meta.ID, Revision: meta.Revision, References: checkpoint.References}
	// Carry only unchanged fragments from the immediately preceding revision.
	// A replaced source is deliberately dropped, including on draft-only capture.
	if prior, readErr := store.ReadArtifact(sessionID, "confirmed-requirement-facts.json"); readErr == nil {
		var old requirement.ConfirmedFacts
		if err := json.Unmarshal([]byte(prior), &old); err != nil { return err }
		if old.RequirementID == meta.ID && old.Revision == meta.Revision-1 {
			snapshot, err := requirement.LoadConversationContext(meta.ID, "Refresh confirmed facts", nil)
			if err != nil { return err }
			old.Revision = meta.Revision
			for _, ref := range old.Current(snapshot) {
				duplicate := false
				for _, current := range facts.References { duplicate = duplicate || current == ref }
				if !duplicate { facts.References = append(facts.References, ref) }
			}
		}
	} else if !os.IsNotExist(readErr) { return readErr }
	b, err := json.Marshal(facts)
	if err != nil { return err }
	if err := store.WriteArtifact(sessionID, "confirmed-requirement-facts.json", b); err != nil {
		return fmt.Errorf("draft captured but fact record failed; do not repeat capture blindly: %w", err)
	}
	return nil
}
