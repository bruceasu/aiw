package main

import (
	"aiw/internal/issue"
	"aiw/internal/session"
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func issueConversationInstructions() string {
	return `Manage one AIW Issue at a time; req/REQ are legacy names. 

Follow the JSON contract and loaded methods. Never ask users for skills, paths, or CLI arguments. Expose unknowns and conflicts; create no ADRs.

Choose options using evidence and preferences; ask only about close choices, missing critical facts, or changes to scope, risk, or authorization. Record reasons; split Issues with lineage. 

Before creating, capturing, deciding, or promoting, show target, summary, and write scope. Run "aiw issue chat prepare"; require "confirm" or "确认" before writing. Use lowercase slugs; never guess REQ IDs.

Promotion creates a native Task and FD. Do not create an OpenSpec change unless the human explicitly requests it. Promotion is not implementation or release approval.`
}

func dispatchIssueChat(args []string) error {
	if len(args) <= 0 || args[0] == "resume" {
		return resumeIssue(args)
	}

	sessionID := os.Getenv("AIW_ISSUE_SESSION")
	if !issue.ValidID(sessionID) {
		return errors.New("issue action preparation requires an active conversation")
	}
	if len(args) == 0 {
		return errors.New("usage: aiw issue chat prepare <new|capture|approve|archive|cancel> ...")
	}
	action := issuePendingAction{Kind: args[0]}
	switch action.Kind {
	case "new":
		var err error
		action, err = parseIssueCreation(args[1:])
		if err != nil {
			return err
		}
	case "capture":
		if (len(args) != 5 && len(args) != 7) || args[3] != "--file" {
			usage, _ := issueSubcommandUsage("chat")
			return errors.New(usage)
		}
		action.RequirementID, action.Artifact, action.Source = args[1], args[2], args[4]
		if len(args) == 7 {
			if args[5] != "--facts-json" || len(args[6]) > 32768 {
				return errors.New("invalid facts-json")
			}
			if err := json.Unmarshal([]byte(args[6]), &action.Facts); err != nil {
				return err
			}
		}
	case "approve":
		usage, _ := issueSubcommandUsage("approve")
		id, decision, by, reason, err := parseApprovalArgs(args[1:], usage)
		if err != nil {
			return err
		}
		action.RequirementID, action.Decision, action.By, action.Reason = id, decision, by, reason
	case "archive":
		usage, _ := issueSubcommandUsage("archive")
		if len(args) != 6 || args[2] != "--by" || args[4] != "--reason" {
			return errors.New(usage)
		}
		action.RequirementID, action.By, action.Reason = args[1], args[3], args[5]
	case "cancel":
		usage, _ := issueSubcommandUsage("cancel")
		if len(args) != 6 || args[2] != "--by" || args[4] != "--reason" {
			return errors.New(usage)
		}
		action.RequirementID, action.By, action.Reason = args[1], args[3], args[5]
	default:
		return fmt.Errorf("unsupported requirement action: %s", action.Kind)
	}
	if !issue.ValidID(action.RequirementID) {
		return errors.New("invalid requirement or task id")
	}
	b, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return err
	}
	if err := session.NewStore("").WriteArtifact(sessionID, "pending-requirement-action.json", b); err != nil {
		return err
	}
	fmt.Println("Issue action prepared. Ask the human to type confirm or 确认 in the active conversation.")
	return nil

}

func resumeIssue(args []string) error {
	plan, err := prepareResumeIssueLoop(args)
	if err != nil {
		return err
	}
	previous, hadPrevious := os.LookupEnv("AIW_ISSUE_SESSION")
	if err := os.Setenv("AIW_ISSUE_SESSION", plan.SessionID); err != nil {
		return err
	}
	defer func() {
		if hadPrevious {
			_ = os.Setenv("AIW_ISSUE_SESSION", previous)
		} else {
			_ = os.Unsetenv("AIW_ISSUE_SESSION")
		}
	}()
	fmt.Printf("Issue conversation session: %s (phase: %s)\n", plan.SessionID, plan.Phase)
	return resumeIssueLoop(plan)
}

func prepareResumeIssueLoop(args []string) (issueChatPlan, error) {
	var id, provider, model string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider":
			if i+1 >= len(args) {
				return issueChatPlan{}, errors.New("--provider requires a name")
			}
			i++
			provider = args[i]
		case "--model":
			if i+1 >= len(args) {
				return issueChatPlan{}, errors.New("--model requires a model")
			}
			i++
			model = args[i]
		default:
			if id != "" || !issue.ValidID(args[i]) {
				return issueChatPlan{}, errors.New("usage: aiw req chat [requirement-id] [--provider NAME] [--model MODEL]")
			}
			id = args[i]
		}
	}
	workspace, err := os.Getwd()
	if err != nil {
		return issueChatPlan{}, err
	}
	store := session.NewStore("")
	if id == "" {
		sessionID := fmt.Sprintf("issue-chat-%d", time.Now().UTC().UnixNano())
		if _, err := store.Create(sessionID, "Requirement conversation", workspace, "codex", "", issueConversationInstructions()); err != nil {
			return issueChatPlan{}, err
		}
		return issueChatPlan{SessionID: sessionID, Phase: "intake", Provider: provider, Model: model}, nil
	}

	meta, err := issue.Read(id)
	if err != nil {
		return issueChatPlan{}, err
	}
	sessionID := meta.Conversation.SessionID
	if sessionID == "" {
		sessionID = "requirement-" + meta.ID
	}
	if _, err := store.Load(sessionID); errors.Is(err, session.ErrSessionNotFound) {
		if _, err := store.Create(sessionID, "Requirement: "+meta.Title, workspace, "codex", "", issueConversationInstructions()); err != nil {
			return issueChatPlan{}, err
		}
	} else if err != nil {
		return issueChatPlan{}, err
	}
	if _, err := issue.BindConversation(meta.ID, sessionID); err != nil {
		return issueChatPlan{}, err
	}
	phase, err := issueConversationPhase(meta)
	if err != nil {
		return issueChatPlan{}, err
	}
	if _, err := store.Update(sessionID, func(status *session.Status) error { status.Session.CurrentPhase = phase; return nil }); err != nil {
		return issueChatPlan{}, err
	}
	return issueChatPlan{RequirementID: id, SessionID: sessionID, Phase: phase, Provider: provider, Model: model}, nil
}

func issueConversationPhase(meta issue.Meta) (string, error) {
	if meta.Status == "APPROVED" {
		return "promotion", nil
	}
	if meta.Status == "DEFERRED" || meta.Status == "REJECTED" {
		return "human-decision", nil
	}
	for _, artifact := range meta.Artifacts {
		content, err := os.ReadFile(filepath.FromSlash(artifact.Path))
		if err != nil {
			return "", err
		}
		if strings.Contains(string(content), "%% NEEDS_INPUT") {
			return "deep-discovery", nil
		}
	}
	// Only a fresh semantic assessment may choose synthesis, never file existence.
	return "intake", nil
}

func resumeIssueLoop(plan issueChatPlan) error {
	store := session.NewStore("")
	reader := bufio.NewReader(os.Stdin)
	displayedAction, err := displayIssueCheckpoint(store, plan)
	if err != nil {
		return err
	}
	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err == io.EOF && strings.TrimSpace(line) == "" {
			return nil
		}
		if err != nil && err != io.EOF {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			if err == io.EOF {
				return nil
			}
			continue
		}
		if isIssueConversationConfirmation(line) {
			pending, readErr := store.ReadArtifact(plan.SessionID, "pending-requirement-action.json")
			if readErr != nil || displayedAction == "" || pending != displayedAction {
				fmt.Println("No unchanged, displayed checkpoint to confirm. Please describe the action again.")
				continue
			}
			var action issuePendingAction
			if err := json.Unmarshal([]byte(pending), &action); err != nil {
				return err
			}
			if plan.RequirementID != "" && action.RequirementID != plan.RequirementID {
				return errors.New("checkpoint targets another Requirement")
			}
			createdID, err := confirmIssueAction(store, plan.SessionID)
			if err != nil {
				return err
			}
			plan.RequirementID = createdID
			displayedAction = ""
			line = "The human confirmed the displayed action. Reload the Requirement and assess the remaining gaps."
		}
		switch line {
		case "/exit":
			return nil
		case "/status":
			status, statusErr := store.Load(plan.SessionID)
			if statusErr != nil {
				return statusErr
			}
			fmt.Printf("session=%s phase=%s\n", status.Session.ID, status.Session.CurrentPhase)
			continue
		}
		fmt.Println("AI is processing your request...")
		turnCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		result, runErr := runIssueDiscovery(turnCtx, store, plan, line)
		cancel()
		if runErr == nil {
			plan.Phase = result.Phase
			printIssueDiscovery(result)
			displayedAction, runErr = displayIssueCheckpoint(store, plan)
		}
		if runErr != nil || err == io.EOF {
			return runErr
		}
	}
}

func isIssueConversationConfirmation(line string) bool {
	return line == "confirm" || line == "确认"
}

func confirmIssueAction(store *session.Store, sessionID string) (string, error) {
	content, err := store.ReadArtifact(sessionID, "pending-requirement-action.json")
	if err != nil {
		return "", errors.New("no pending Requirement action")
	}
	var action issuePendingAction
	if err := json.Unmarshal([]byte(content), &action); err != nil || action.Kind == "" {
		return "", errors.New("no pending Requirement action")
	}
	switch action.Kind {
	case "new":
		meta, err := createIssueAction(action)
		if err != nil {
			return "", err
		}
		if _, err := issue.BindConversation(meta.ID, sessionID); err != nil {
			return "", fmt.Errorf("Requirement %s was created but Session binding failed; do not repeat creation: %w", meta.ID, err)
		}
		action.RequirementID = meta.ID
		fmt.Println("created requirement:", meta.ID)
	case "capture":
		if err := confirmIssueCapture(store, sessionID, action); err != nil {
			return "", err
		}
	case "approve":
		if err := confirmIssueApproval(store, sessionID, content, action); err != nil {
			return "", err
		}
	case "archive":
		if _, err := issue.Archive(action.RequirementID, action.By, action.Reason); err != nil {
			return "", err
		}
	case "cancel":
		if _, err := issue.Cancel(action.RequirementID, action.By, action.Reason); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported pending requirement action: %s", action.Kind)
	}
	if err := store.WriteArtifact(sessionID, "pending-requirement-action.json", []byte("{}")); err != nil {
		return "", fmt.Errorf("Requirement action completed for %s but pending cleanup failed; do not repeat the action: %w", action.RequirementID, err)
	}
	if err := store.AppendMemory(sessionID, fmt.Sprintf("Confirmed Requirement action: %s for %s", action.Kind, action.RequirementID)); err != nil {
		return "", fmt.Errorf("Requirement action completed for %s but Session memory update failed; do not repeat the action: %w", action.RequirementID, err)
	}
	fmt.Printf("Confirmed Requirement action: %s\n", action.Kind)
	return action.RequirementID, nil
}

func printIssueDiscovery(turn issue.ConversationTurn) {
	fmt.Printf("Phase: %s; method: %s (%s)\n", turn.Phase, turn.Context.MethodSelection.Primary, turn.Context.MethodSelection.Status)
	for _, item := range turn.Discovery.Settled {
		fmt.Printf("[%s] %s: %s\n", item.Status, item.Dimension, item.Conclusion)
	}
	for i, item := range turn.Discovery.Questions {
		fmt.Printf("%d. [%s] %s\nImpact: %s\n", i+1, item.Status, item.Question, item.Impact)
		for _, ref := range item.Sources {
			fmt.Printf("  %s: %s\n", ref.Source, ref.Quote)
		}
		for _, option := range item.Options {
			fmt.Printf("  %s: %s\n", option.Label, option.Tradeoff)
		}
	}
	if len(turn.Discovery.Questions) == 0 {
		fmt.Println("No open questions in this assessment. This is not approval.")
	}
	printIssueReadiness(turn.Readiness)
	if turn.Context.Requirement != nil && turn.Context.Requirement.Status == "APPROVED" && !turn.Readiness.Ready {
		fmt.Println("Risk found in an approved Requirement. Existing approval is unchanged; ask the owner for a decision.")
	}
	if review := turn.Discovery.Coverage.Assessment.PlanReview; review != nil {
		for _, section := range review.Sections {
			fmt.Printf("Plan %s: %s\n", section.Section, section.Explanation)
			for _, ref := range section.Sources {
				fmt.Printf("  %s: %s\n", ref.Source, ref.Quote)
			}
		}
	}
}

// displayIssueCheckpoint snapshots exactly what the human sees. The loop
// requires the pending bytes to remain unchanged before accepting confirmation.
func displayIssueCheckpoint(store *session.Store, plan issueChatPlan) (string, error) {
	content, err := store.ReadArtifact(plan.SessionID, "pending-issue-action.json")
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var action issuePendingAction
	if err := json.Unmarshal([]byte(content), &action); err != nil {
		return "", err
	}
	if action.Kind == "" {
		return "", nil
	}
	if plan.RequirementID != "" && action.RequirementID != plan.RequirementID {
		return "", errors.New("checkpoint targets another Requirement")
	}
	if action.Kind == "approve" && strings.ToUpper(action.Decision) == "APPROVED" {
		checkpoint, err := currentApprovalCheckpoint(store, plan.SessionID, content, action.RequirementID)
		if err != nil {
			fmt.Printf("Approval suggestion withheld: %v. Continue discussion or save a draft.\n", err)
			return "", store.WriteArtifact(plan.SessionID, "pending-issue-action.json", []byte("{}"))
		}
		b, err := json.Marshal(checkpoint)
		if err != nil {
			return "", err
		}
		if err := store.WriteArtifact(plan.SessionID, "approval-checkpoint.json", b); err != nil {
			return "", err
		}
	}
	if action.Kind == "capture" {
		refs, hash, err := issue.CaptureFactReferences(action.RequirementID, action.Artifact, action.Source, action.Facts)
		if err != nil {
			return "", err
		}
		meta, err := issue.Read(action.RequirementID)
		if err != nil {
			return "", err
		}
		checkpoint := captureCheckpoint{Action: content, Digest: hash, References: refs, Revision: meta.Revision}
		b, err := json.Marshal(checkpoint)
		if err != nil {
			return "", err
		}
		if err := store.WriteArtifact(plan.SessionID, "capture-checkpoint.json", b); err != nil {
			return "", err
		}
		fmt.Printf("Capture draft: %s -> %s/%s\n", action.Source, action.RequirementID, action.Artifact)
		if len(refs) == 0 {
			fmt.Println("Draft only: no facts will be marked confirmed.")
		}
		for _, ref := range refs {
			fmt.Printf("Fact to confirm: %s\n", ref.Quote)
		}
	}
	fmt.Printf("Pending action: %s\nType confirm or 确认 to accept this checkpoint.\n", content)
	return content, nil
}

// confirmIssueCapture never infers confirmed facts from capture alone.
// Facts are installed only if the captured bytes match the displayed digest.
func confirmIssueCapture(store *session.Store, sessionID string, action issuePendingAction) error {
	var checkpoint captureCheckpoint
	if content, err := store.ReadArtifact(sessionID, "capture-checkpoint.json"); err == nil {
		if err := json.Unmarshal([]byte(content), &checkpoint); err != nil {
			return err
		}
	} else if len(action.Facts) != 0 {
		return err
	}
	if len(action.Facts) != 0 && checkpoint.Action == "" {
		return errors.New("facts require a displayed capture checkpoint")
	}
	if checkpoint.Action != "" {
		pending, err := store.ReadArtifact(sessionID, "pending-issue-action.json")
		if err != nil || checkpoint.Action != pending {
			return errors.New("capture checkpoint changed; prepare it again")
		}
		meta, err := issue.Read(action.RequirementID)
		if err != nil {
			return err
		}
		if meta.Revision != checkpoint.Revision {
			return errors.New("Requirement changed after checkpoint; prepare it again")
		}
		_, hash, err := issue.CaptureFactReferences(action.RequirementID, action.Artifact, action.Source, action.Facts)
		if err != nil {
			return err
		}
		if hash != checkpoint.Digest {
			return errors.New("draft changed after display; prepare it again")
		}
	}
	meta, artifact, err := issue.Capture(action.RequirementID, action.Artifact, action.Source)
	if err != nil {
		return err
	}
	if checkpoint.Action != "" && artifact.Digest != checkpoint.Digest {
		return errors.New("captured draft changed; facts were not confirmed")
	}
	facts := issue.ConfirmedFacts{RequirementID: meta.ID, Revision: meta.Revision, References: checkpoint.References}
	// Carry only unchanged fragments from the immediately preceding revision.
	// A replaced source is deliberately dropped, including on draft-only capture.
	if prior, readErr := store.ReadArtifact(sessionID, "confirmed-requirement-facts.json"); readErr == nil {
		var old issue.ConfirmedFacts
		if err := json.Unmarshal([]byte(prior), &old); err != nil {
			return err
		}
		if old.RequirementID == meta.ID && old.Revision == meta.Revision-1 {
			snapshot, err := issue.LoadConversationContext(meta.ID, "Refresh confirmed facts", nil)
			if err != nil {
				return err
			}
			old.Revision = meta.Revision
			for _, ref := range old.Current(snapshot) {
				duplicate := false
				for _, current := range facts.References {
					duplicate = duplicate || current == ref
				}
				if !duplicate {
					facts.References = append(facts.References, ref)
				}
			}
		}
	} else if !os.IsNotExist(readErr) {
		return readErr
	}
	b, err := json.Marshal(facts)
	if err != nil {
		return err
	}
	if err := store.WriteArtifact(sessionID, "confirmed-requirement-facts.json", b); err != nil {
		return fmt.Errorf("draft captured but fact record failed; do not repeat capture blindly: %w", err)
	}
	return nil
}

// runIssueDiscovery is the Session adapter. Domain code owns routing and
// validation; Session keeps the actual prompts and raw model outputs.
func runIssueDiscovery(ctx context.Context, store *session.Store, plan issueChatPlan, input string) (issue.ConversationTurn, error) {
	var facts issue.ConfirmedFacts
	content, err := store.ReadArtifact(plan.SessionID, "confirmed-requirement-facts.json")
	if err == nil {
		if err := json.Unmarshal([]byte(content), &facts); err != nil {
			return issue.ConversationTurn{}, err
		}
	} else if !os.IsNotExist(err) {
		return issue.ConversationTurn{}, err
	}
	history, err := readRequirementHistory(store, plan.SessionID)
	if err != nil {
		return issue.ConversationTurn{}, err
	}
	candidate, notice := issue.RestoreConversationCandidate(history, plan.RequirementID, facts)
	fmt.Println(notice)
	record := issue.ConversationEvidence{Version: 1, RequirementID: plan.RequirementID, State: "running", Confirmed: facts}
	record.Turn.Context.UserInput = input
	recordName := fmt.Sprintf("requirement-discussion-%d.json", time.Now().UTC().UnixNano())
	if err := writeRequirementEvidence(store, plan.SessionID, recordName, record); err != nil {
		return issue.ConversationTurn{}, err
	}
	index, err := json.Marshal(recordName)
	if err != nil {
		return issue.ConversationTurn{}, err
	}
	if err := store.WriteArtifact(plan.SessionID, "requirement-discussion-latest.json", index); err != nil {
		return issue.ConversationTurn{}, err
	}
	// Old pending actions must not survive a new answer or a failed assessment.
	if err := store.WriteArtifact(plan.SessionID, "pending-requirement-action.json", []byte("{}")); err != nil {
		return issue.ConversationTurn{}, err
	}
	turn, runErr := issue.RunConversationTurn(ctx, plan.RequirementID, input, candidate, facts, func(ctx context.Context, phase, prompt string) (string, error) {
		if err := store.WriteArtifact(plan.SessionID, "pending-requirement-action.json", []byte("{}")); err != nil {
			return "", err
		}
		before, err := store.Load(plan.SessionID)
		if err != nil {
			return "", err
		}
		result, err := session.ExecuteTurnWithOverrides(ctx, store, plan.SessionID, phase,
			issueConversationInstructions()+"\n"+prompt, plan.Provider, plan.Model, false)
		after, readErr := store.Load(plan.SessionID)
		if readErr == nil && after.Session.LastTurn == before.Session.LastTurn+1 {
			record.Turns = append(record.Turns, after.Session.LastTurn)
		} else if readErr == nil {
			readErr = errors.New("Session turn was not durably recorded")
		}
		err = errors.Join(err, readErr)
		if err == nil && result.ExitCode != 0 {
			err = fmt.Errorf("model exited with code %d", result.ExitCode)
		}
		return result.FinalOutput, err
	})
	record.Turn = turn
	record.State = "completed"
	if runErr != nil {
		record.State, record.Diagnostic = "failed", runErr.Error()
	}
	if saveErr := writeRequirementEvidence(store, plan.SessionID, recordName, record); saveErr != nil {
		runErr = errors.Join(runErr, fmt.Errorf("save discussion evidence: %w", saveErr))
	}
	phase := turn.Phase
	if runErr != nil {
		phase = "discovery-blocked"
		if err := store.WriteArtifact(plan.SessionID, "pending-requirement-action.json", []byte("{}")); err != nil {
			return turn, errors.Join(runErr, err)
		}
	}
	_, err = store.Update(plan.SessionID, func(status *session.Status) error { status.Session.CurrentPhase = phase; return nil })
	return turn, errors.Join(runErr, err)
}

// The latest pointer is installed before model execution. Never search older
// records when it names an incomplete, missing, or invalid latest discussion.
func readRequirementHistory(store *session.Store, id string) (string, error) {
	index, err := store.ReadArtifact(id, "requirement-discussion-latest.json")
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var name string
	if err := json.Unmarshal([]byte(index), &name); err != nil {
		return "invalid", nil
	}
	content, err := store.ReadArtifact(id, name)
	if os.IsNotExist(err) {
		return "invalid", nil
	}
	return content, err
}

func writeRequirementEvidence(store *session.Store, id, name string, record issue.ConversationEvidence) error {
	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return store.WriteArtifact(id, name, b)
}

func confirmIssueApproval(store *session.Store, sessionID, pending string, action issuePendingAction) error {
	if strings.ToUpper(action.Decision) == "APPROVED" {
		content, err := store.ReadArtifact(sessionID, "approval-checkpoint.json")
		if err != nil {
			return err
		}
		var displayed approvalCheckpoint
		if err := json.Unmarshal([]byte(content), &displayed); err != nil {
			return err
		}
		current, err := currentApprovalCheckpoint(store, sessionID, pending, action.RequirementID)
		if err != nil {
			return err
		}
		if current != displayed {
			return errors.New("approval evidence changed after display; reassess and display again")
		}
	}
	_, err := issue.Approve(action.RequirementID, action.Decision, action.By, action.Reason)
	return err
}

// currentApprovalCheckpoint always recomputes readiness from raw output after
// reopening sources. A saved Ready boolean is not trusted.
func currentApprovalCheckpoint(store *session.Store, sessionID, actionText, id string) (approvalCheckpoint, error) {
	history, err := readRequirementHistory(store, sessionID)
	if err != nil {
		return approvalCheckpoint{}, err
	}
	content, err := store.ReadArtifact(sessionID, "confirmed-requirement-facts.json")
	if err != nil {
		return approvalCheckpoint{}, err
	}
	var facts issue.ConfirmedFacts
	if err := json.Unmarshal([]byte(content), &facts); err != nil {
		return approvalCheckpoint{}, err
	}
	if candidate, notice := issue.RestoreConversationCandidate(history, id, facts); candidate == nil {
		return approvalCheckpoint{}, errors.New(notice)
	}
	var saved issue.ConversationEvidence
	if err := json.Unmarshal([]byte(history), &saved); err != nil {
		return approvalCheckpoint{}, err
	}
	report := issue.AssessReadiness(saved.Turn.Discovery.Coverage.Raw, saved.Turn.Context, facts)
	if !report.Ready {
		return approvalCheckpoint{}, errors.New("current coverage and Plan are not ready for approval advice")
	}
	return approvalCheckpoint{Action: actionText, HistoryDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(history))), FactsDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(content)))}, nil
}

func printIssueReadiness(report issue.ReadinessReport) {
	fmt.Printf("Approval advice ready: %t (not approval). Draft capture remains available.\n", report.Ready)
	for _, ref := range report.Confirmed {
		fmt.Printf("Confirmed: %s: %s\n", ref.Source, ref.Quote)
	}
	for _, gap := range report.BusinessBlockers {
		fmt.Printf("Business blocker: %s\n", gap)
	}
	for _, gap := range report.PlanGaps {
		fmt.Printf("Plan gap: %s\n", gap)
	}
	for _, item := range report.UnconfirmedDeferrals {
		fmt.Printf("Needs human postponement confirmation: %s; reason: %s\n", item.Question, item.Reason)
	}
	for _, item := range report.Deferred {
		fmt.Printf("Human-confirmed design postponement: %s; reason: %s\n", item.Question, item.Reason)
	}
}
