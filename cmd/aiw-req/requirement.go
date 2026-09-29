package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	taskcmd "aiw/internal/commands/task"
	"aiw/internal/fsx"
	requirement "aiw/internal/req/domain"
	"aiw/internal/session"
	"aiw/internal/task"
	taskadapter "aiw/internal/task/workflowadapter"
	workflowcli "aiw/internal/workflow/cli"
)

func DispatchRequirement(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Print(requirementUsage)
		return nil
	}
	if len(args) == 2 && isRequirementHelpFlag(args[1]) {
		if usage, ok := requirementSubcommandUsage(args[0]); ok {
			fmt.Print(usage)
			return nil
		}
	}
	switch args[0] {
	case "chat":
		return dispatchRequirementChat(args[1:])
	case "new":
		action, err := parseRequirementCreation(args[1:])
		if err != nil {
			return err
		}
		meta, err := createRequirementAction(action)
		if err != nil {
			return err
		}
		if sessionID := os.Getenv("AIW_REQUIREMENT_SESSION"); sessionID != "" {
			if _, err := requirement.BindConversation(meta.ID, sessionID); err != nil {
				return fmt.Errorf("Requirement %s was created but Session binding failed; do not repeat creation: %w", meta.ID, err)
			}
		}
		fmt.Println("created requirement:", meta.ID)
		return nil
	case "show":
		if len(args) != 2 {
			return errors.New("usage: aiw req show <id>")
		}
		meta, err := requirement.Read(args[1])
		if err != nil {
			return err
		}
		fmt.Printf("%s\t%s\tapproval=%s\tpromotion=%s\ttask=%s\tparent=%s\n", meta.ID, meta.Status, meta.Approval.Status, meta.Promotion.Status, meta.Promotion.TaskID, meta.ParentID)
		return nil
	case "link-parent":
		if len(args) != 3 { return errors.New("usage: aiw issue link-parent <child-id> <parent-id>") }
		meta, err := requirement.LinkParent(args[1], args[2])
		if err != nil { return err }
		fmt.Printf("issue %s parent=%s\n", meta.ID, meta.ParentID)
		return nil
	case "children":
		if len(args) != 2 { return errors.New("usage: aiw issue children <parent-id>") }
		if _, err := requirement.Read(args[1]); err != nil { return err }
		issues, err := requirement.List(requirement.ListAll)
		if err != nil { return err }
		for _, issue := range issues {
			if issue.ParentID == args[1] { fmt.Printf("%s\t%s\t%s\n", issue.ID, issue.Status, issue.Title) }
		}
		return nil
	case "capture":
		return captureRequirement(args[1:])
	case "approve":
		return approveRequirement(args[1:])
	case "promote":
		return promoteRequirement(args[1:])
	case "archive":
		return archiveRequirement(args[1:])
	case "cancel":
		return cancelRequirement(args[1:])
	case "list":
		return listRequirements(args[1:])
	default:
		return fmt.Errorf("unknown requirement command: %s", args[0])
	}
}

// Dispatch is the public entry point for the standalone req program.
func Dispatch(args []string) error { return DispatchRequirement(args) }

func isRequirementHelpFlag(arg string) bool { return arg == "--help" || arg == "-h" }

func requirementSubcommandUsage(command string) (string, bool) {
	usages := map[string]string{
		"chat":    "usage: aiw req chat [requirement-id] [--provider NAME] [--model MODEL]\n",
		"new":     requirementNewUsage + "\n",
		"show":    "usage: aiw req show <id>\n",
		"link-parent": "usage: aiw issue link-parent <child-id> <parent-id>\n",
		"children": "usage: aiw issue children <parent-id>\n",
		"capture": "usage: aiw req capture <id> <artifact> --file <path>\n",
		"approve": "usage: aiw req approve <id> <APPROVED|DEFERRED|REJECTED> [--by <actor>] --reason <reason>\n",
		"promote": "usage: aiw req promote <id> --task <task-id>\n",
		"archive": "usage: aiw req archive <id> --by <actor> --reason <reason>\n",
		"cancel":  "usage: aiw req cancel <id> --by <actor> --reason <reason>\n",
		"list":    "usage: aiw req list [--all|--archived|--cancelled]\n",
	}
	usage, ok := usages[command]
	return usage, ok
}

const requirementUsage = `usage: aiw req <command> ...

commands:
  chat [id] [--provider NAME] [--model MODEL]
  new <slug> [title]                  Auto-number a new Requirement.
  new --id <id> [title]               Create an exact ID for compatibility.
  show <id>
  link-parent <child-id> <parent-id>    Record split lineage before approval.
  children <parent-id>                 List direct child Issues.
  capture <id> <artifact> --file <path>
  approve <id> <APPROVED|DEFERRED|REJECTED> [--by <actor>] --reason <reason>
	promote <id> --task <task-id>
  archive <id> --by <actor> --reason <reason>
  cancel <id> --by <actor> --reason <reason>
  list [--all|--archived|--cancelled]
`

type requirementChatPlan struct {
	RequirementID string
	SessionID     string
	Phase         string
	Provider      string
	Model         string
}

type requirementPendingAction struct {
	AutoNumber          bool     `json:"auto_number,omitempty"`
	Facts               []string `json:"facts,omitempty"`
	Kind                string   `json:"kind"`
	RequirementID       string   `json:"requirement_id"`
	Title               string   `json:"title,omitempty"`
	Artifact            string   `json:"artifact,omitempty"`
	Source              string   `json:"source,omitempty"`
	Decision            string   `json:"decision,omitempty"`
	By                  string   `json:"by,omitempty"`
	Reason              string   `json:"reason,omitempty"`
	TaskID              string   `json:"task_id,omitempty"`
	AllowUnrelatedDirty bool     `json:"allow_unrelated_dirty,omitempty"`
}

func dispatchRequirementChat(args []string) error {
	if len(args) > 0 && args[0] == "prepare" {
		return prepareRequirementAction(args[1:])
	}
	return chatRequirement(args)
}

func chatRequirement(args []string) error {
	plan, err := prepareRequirementChat(args)
	if err != nil {
		return err
	}
	previous, hadPrevious := os.LookupEnv("AIW_REQUIREMENT_SESSION")
	if err := os.Setenv("AIW_REQUIREMENT_SESSION", plan.SessionID); err != nil {
		return err
	}
	defer func() {
		if hadPrevious {
			_ = os.Setenv("AIW_REQUIREMENT_SESSION", previous)
		} else {
			_ = os.Unsetenv("AIW_REQUIREMENT_SESSION")
		}
	}()
	fmt.Printf("Requirement conversation session: %s (phase: %s)\n", plan.SessionID, plan.Phase)
	return requirementChatLoop(plan)
}

func requirementChatLoop(plan requirementChatPlan) error {
	store := session.NewStore("")
	reader := bufio.NewReader(os.Stdin)
	displayedAction, err := displayRequirementCheckpoint(store, plan)
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
		if isRequirementConversationConfirmation(line) {
			pending, readErr := store.ReadArtifact(plan.SessionID, "pending-requirement-action.json")
			if readErr != nil || displayedAction == "" || pending != displayedAction {
				fmt.Println("No unchanged, displayed checkpoint to confirm. Please describe the action again.")
				continue
			}
			var action requirementPendingAction
			if err := json.Unmarshal([]byte(pending), &action); err != nil {
				return err
			}
			if plan.RequirementID != "" && action.RequirementID != plan.RequirementID {
				return errors.New("checkpoint targets another Requirement")
			}
			createdID, err := confirmRequirementAction(store, plan.SessionID)
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
		result, runErr := runRequirementDiscovery(turnCtx, store, plan, line)
		cancel()
		if runErr == nil {
			plan.Phase = result.Phase
			printRequirementDiscovery(result)
			displayedAction, runErr = displayRequirementCheckpoint(store, plan)
		}
		if runErr != nil || err == io.EOF {
			return runErr
		}
	}
}

func isRequirementConversationConfirmation(line string) bool {
	return line == "confirm" || line == "确认"
}

func prepareRequirementAction(args []string) error {
	sessionID := os.Getenv("AIW_REQUIREMENT_SESSION")
	if !requirement.ValidID(sessionID) {
		return errors.New("requirement action preparation requires an active conversation")
	}
	if len(args) == 0 {
		return errors.New("usage: aiw req chat prepare <new|capture|approve|promote> ...")
	}
	action := requirementPendingAction{Kind: args[0]}
	switch action.Kind {
	case "new":
		var err error
		action, err = parseRequirementCreation(args[1:])
		if err != nil {
			return err
		}
	case "capture":
		if (len(args) != 5 && len(args) != 7) || args[3] != "--file" {
			return errors.New("usage: aiw req chat prepare capture <id> <artifact> --file <path>")
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
		id, decision, by, reason, err := parseApprovalArgs(args[1:], "usage: aiw req chat prepare approve <id> <decision> [--by <actor>] --reason <reason>")
		if err != nil {
			return err
		}
		action.RequirementID, action.Decision, action.By, action.Reason = id, decision, by, reason
	case "promote":
		if (len(args) != 4 && len(args) != 5) || args[2] != "--task" || (len(args) == 5 && args[4] != "--allow-unrelated-dirty") {
			return errors.New("usage: aiw req chat prepare promote <id> --task <task-id>")
		}
		action.RequirementID, action.TaskID = args[1], args[3]
		action.AllowUnrelatedDirty = true
	case "archive", "cancel":
		if len(args) != 6 || args[2] != "--by" || args[4] != "--reason" {
			return fmt.Errorf("usage: aiw req chat prepare %s <id> --by <actor> --reason <reason>", action.Kind)
		}
		action.RequirementID, action.By, action.Reason = args[1], args[3], args[5]
	default:
		return fmt.Errorf("unsupported requirement action: %s", action.Kind)
	}
	if !requirement.ValidID(action.RequirementID) || (action.TaskID != "" && !requirement.ValidID(action.TaskID)) {
		return errors.New("invalid requirement or task id")
	}
	b, err := json.MarshalIndent(action, "", "  ")
	if err != nil {
		return err
	}
	if err := session.NewStore("").WriteArtifact(sessionID, "pending-requirement-action.json", b); err != nil {
		return err
	}
	fmt.Println("Requirement action prepared. Ask the human to type confirm or 确认 in the active conversation.")
	return nil
}

func confirmRequirementAction(store *session.Store, sessionID string) (string, error) {
	content, err := store.ReadArtifact(sessionID, "pending-requirement-action.json")
	if err != nil {
		return "", errors.New("no pending Requirement action")
	}
	var action requirementPendingAction
	if err := json.Unmarshal([]byte(content), &action); err != nil || action.Kind == "" {
		return "", errors.New("no pending Requirement action")
	}
	switch action.Kind {
	case "new":
		meta, err := createRequirementAction(action)
		if err != nil {
			return "", err
		}
		if _, err := requirement.BindConversation(meta.ID, sessionID); err != nil {
			return "", fmt.Errorf("Requirement %s was created but Session binding failed; do not repeat creation: %w", meta.ID, err)
		}
		action.RequirementID = meta.ID
		fmt.Println("created requirement:", meta.ID)
	case "capture":
		if err := confirmRequirementCapture(store, sessionID, action); err != nil {
			return "", err
		}
	case "approve":
		if err := confirmRequirementApproval(store, sessionID, content, action); err != nil {
			return "", err
		}
	case "promote":
		args := []string{action.RequirementID, "--task", action.TaskID}
		if action.AllowUnrelatedDirty {
			args = append(args, "--allow-unrelated-dirty")
		}
		if err := promoteRequirement(args); err != nil {
			return "", err
		}
	case "archive":
		if _, err := requirement.Archive(action.RequirementID, action.By, action.Reason); err != nil {
			return "", err
		}
	case "cancel":
		if _, err := requirement.Cancel(action.RequirementID, action.By, action.Reason); err != nil {
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

func prepareRequirementChat(args []string) (requirementChatPlan, error) {
	var id, provider, model string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider":
			if i+1 >= len(args) {
				return requirementChatPlan{}, errors.New("--provider requires a name")
			}
			i++
			provider = args[i]
		case "--model":
			if i+1 >= len(args) {
				return requirementChatPlan{}, errors.New("--model requires a model")
			}
			i++
			model = args[i]
		default:
			if id != "" || !requirement.ValidID(args[i]) {
				return requirementChatPlan{}, errors.New("usage: aiw req chat [requirement-id] [--provider NAME] [--model MODEL]")
			}
			id = args[i]
		}
	}
	workspace, err := os.Getwd()
	if err != nil {
		return requirementChatPlan{}, err
	}
	store := session.NewStore("")
	if id == "" {
		sessionID := fmt.Sprintf("requirement-chat-%d", time.Now().UTC().UnixNano())
		if _, err := store.Create(sessionID, "Requirement conversation", workspace, "codex", "", requirementConversationInstructions()); err != nil {
			return requirementChatPlan{}, err
		}
		return requirementChatPlan{SessionID: sessionID, Phase: "intake", Provider: provider, Model: model}, nil
	}

	meta, err := requirement.Read(id)
	if err != nil {
		return requirementChatPlan{}, err
	}
	sessionID := meta.Conversation.SessionID
	if sessionID == "" {
		sessionID = "requirement-" + meta.ID
	}
	if _, err := store.Load(sessionID); errors.Is(err, session.ErrSessionNotFound) {
		if _, err := store.Create(sessionID, "Requirement: "+meta.Title, workspace, "codex", "", requirementConversationInstructions()); err != nil {
			return requirementChatPlan{}, err
		}
	} else if err != nil {
		return requirementChatPlan{}, err
	}
	if _, err := requirement.BindConversation(meta.ID, sessionID); err != nil {
		return requirementChatPlan{}, err
	}
	phase, err := requirementConversationPhase(meta)
	if err != nil {
		return requirementChatPlan{}, err
	}
	if _, err := store.Update(sessionID, func(status *session.Status) error { status.Session.CurrentPhase = phase; return nil }); err != nil {
		return requirementChatPlan{}, err
	}
	return requirementChatPlan{RequirementID: id, SessionID: sessionID, Phase: phase, Provider: provider, Model: model}, nil
}

func requirementConversationPhase(meta requirement.Meta) (string, error) {
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

func requirementConversationInstructions() string {
	return `You are the AIW Issue Management conversation orchestrator. The req command and REQ IDs are compatibility names.

Guide the human through one bug, feature, or modification Issue at a time. Use the generic discovery baseline and only the domain methods actually loaded in the current input. Follow the current call's JSON response contract. Do not require the human to name a Skill, artifact type, file path, or CLI parameter. Keep conflicts and missing facts explicit. Do not create ADRs at this stage.

For an engineering strategy, use source evidence and the user's small preferences to choose a clearly better option. A bounded sub-agent may compare options when authorized. Record the choice and rationale in the Issue Plan. Ask the human only when options are materially close, critical information is missing, or the choice changes scope, risk, or authorization. A large Issue may be split into smaller Issues with distinct outcomes; preserve parent and child lineage.

Treat every durable action as a confirmation checkpoint. Before creating a Requirement, capturing an artifact, recording APPROVED, DEFERRED, or REJECTED, or promoting a Task, show the action, target, content summary, and write scope. Prepare the action with aiw req chat prepare, then ask the human to type confirm or 确认 in the active conversation. Do not invoke a durable Requirement operation before that confirmation. The active conversation sets AIW_REQUIREMENT_SESSION, so confirmed new Requirements link to this Session. For capture, prepare the artifact source yourself; the human must not need to construct a path or command.

For a new Requirement, prepare new with a lowercase slug such as add-chat-support. The host adds a REQ number only after confirmation. Do not guess the next number. Use --id only when the human asks for an exact ID. After creation, use the full ID returned by the host.

Promotion is separate from implementation: it creates or reuses one AIW Task and its Issue handoff. Feature Design supplies the Task work items; OpenSpec changes are optional and stable specs change only when behavior changes. Do not start implementation or make release decisions.`
}

func captureRequirement(args []string) error {
	if len(args) != 4 || args[2] != "--file" {
		return errors.New("usage: aiw req capture <id> <artifact> --file <path>")
	}
	_, artifact, err := requirement.Capture(args[0], args[1], args[3])
	if err != nil {
		return err
	}
	fmt.Printf("captured %s: %s\n", artifact.Kind, artifact.Path)
	return nil
}

func approveRequirement(args []string) error {
	id, decision, by, reason, err := parseApprovalArgs(args, "usage: aiw req approve <id> <APPROVED|DEFERRED|REJECTED> [--by <actor>] --reason <reason>")
	if err != nil {
		return err
	}
	meta, err := requirement.Approve(id, decision, by, reason)
	if err != nil {
		return err
	}
	fmt.Printf("requirement %s: %s\n", meta.ID, meta.Status)
	return nil
}

func parseApprovalArgs(args []string, usage string) (string, string, string, string, error) {
	if len(args) < 4 || !requirement.ValidID(args[0]) {
		return "", "", "", "", errors.New(usage)
	}
	id, decision, by, reason := args[0], args[1], "", ""
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--by":
			if i+1 >= len(args) || args[i+1] == "" || strings.HasPrefix(args[i+1], "--") {
				return "", "", "", "", errors.New(usage)
			}
			by, i = args[i+1], i+1
		case "--reason":
			if i+1 >= len(args) || args[i+1] == "" || strings.HasPrefix(args[i+1], "--") {
				return "", "", "", "", errors.New(usage)
			}
			reason, i = args[i+1], i+1
		default:
			return "", "", "", "", errors.New(usage)
		}
	}
	if reason == "" {
		return "", "", "", "", errors.New(usage)
	}
	if by == "" {
		current, err := user.Current()
		if err != nil || current.Username == "" {
			return "", "", "", "", errors.New("cannot determine current user; provide --by <actor>")
		}
		by = current.Username
	}
	return id, decision, by, reason, nil
}

func promoteRequirement(args []string) error {
	reqID, taskID, allowUnrelatedDirty, err := parsePromoteArgs(args)
	if err != nil {
		return err
	}
	meta, err := requirement.Read(reqID)
	if err != nil {
		return err
	}
	if (meta.Status != "APPROVED" && meta.Status != "PROMOTED") || meta.Approval.Status != "APPROVED" {
		return errors.New("requirement is not approved")
	}
	if meta.Promotion.TaskID != "" && meta.Promotion.TaskID != taskID {
		return fmt.Errorf("requirement already links to task %s", meta.Promotion.TaskID)
	}
	if meta.Promotion.Status == "SPEC_DRAFTED" || meta.Promotion.Status == "FD_READY" {
		if !fsx.Exists(task.RuntimeTaskDir(taskID)) {
			return fmt.Errorf("promoted Issue task not found: %s", taskID)
		}
		fmt.Printf("issue %s already links to task %s\n", reqID, taskID)
		return nil
	}
	if meta.Promotion.TaskID == "" && !fsx.Exists(task.RuntimeTaskDir(taskID)) {
		if err := taskcmd.CreateTaskFromIssue(taskID, reqID, allowUnrelatedDirty); err != nil {
			return err
		}
	}
	if !fsx.Exists(task.RuntimeTaskDir(taskID)) {
		return fmt.Errorf("promoted task not found: %s", taskID)
	}
	if meta.Promotion.TaskID == "" {
		meta, _, err = requirement.StartPromotion(reqID, taskID)
		if err != nil {
			return err
		}
	}
	snapshot, err := requirement.ArtifactSnapshot(reqID)
	if err != nil {
		return err
	}
	if err := writeRequirementHandoff(meta, snapshot); err != nil {
		return err
	}
	checklistPath := task.FeatureDesignPath(taskID)
	if !fsx.Exists(checklistPath) { checklistPath = filepath.Join(task.TaskDir(taskID), "tasks.md") }
	items, err := task.ReadWorkflowChecklist(checklistPath)
	if errors.Is(err, os.ErrNotExist) || (err == nil && len(items) == 0) {
		fmt.Printf("issue %s linked to task %s; awaiting Feature Design work items\n", reqID, taskID)
		return nil
	}
	if err != nil { return fmt.Errorf("read Feature Design work items: %w", err) }
	if checklistPath == task.FeatureDesignPath(taskID) {
		readiness, err := task.ReadFeatureDesignReadiness(checklistPath)
		if err != nil { return err }
		if readiness == "BLOCKED" {
			fmt.Printf("issue %s linked to task %s; awaiting Feature Design readiness\n", reqID, taskID)
			return nil
		}
	}
	if err := taskcmd.EnsureChecklistMapping(taskID); err != nil {
		return fmt.Errorf("map Feature Design work items: %w", err)
	}
	if err := completePromotion(reqID, taskID); err != nil {
		return err
	}
	fmt.Printf("issue %s promoted to task %s from Feature Design\n", reqID, taskID)
	return nil
}

func completePromotion(requirementID, taskID string) error {
	// Advisory routing is intentionally non-blocking. recommend-routing falls
	// back to deterministic defaults when no LLM is configured; an unexpected
	// routing persistence failure remains visible without downgrading accepted
	// artifact content back to an incomplete promotion.
	if err := workflowcli.RunWorkflowCommand(taskadapter.New(taskadapter.DefaultOperations{}), []string{"recommend-routing", taskID}); err != nil {
		fmt.Fprintf(os.Stderr, "routing recommendation unavailable for task %s: %v\n", taskID, err)
	}
	_, err := requirement.CompletePromotion(requirementID, taskID)
	return err
}

func parseTerminalArgs(args []string, command string) (string, string, string, error) {
	if len(args) < 3 || !requirement.ValidID(args[0]) {
		return "", "", "", fmt.Errorf("usage: aiw req %s <id> [--by <actor>] --reason <reason>", command)
	}
	by, reason := "", ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--by":
			if i+1 >= len(args) || args[i+1] == "" || strings.HasPrefix(args[i+1], "--") {
				return "", "", "", fmt.Errorf("usage: aiw req %s <id> [--by <actor>] --reason <reason>", command)
			}
			by, i = args[i+1], i+1
		case "--reason":
			if i+1 >= len(args) || args[i+1] == "" || strings.HasPrefix(args[i+1], "--") {
				return "", "", "", fmt.Errorf("usage: aiw req %s <id> [--by <actor>] --reason <reason>", command)
			}
			reason, i = args[i+1], i+1
		default:
			return "", "", "", fmt.Errorf("usage: aiw req %s <id> [--by <actor>] --reason <reason>", command)
		}
	}
	if reason == "" {
		return "", "", "", fmt.Errorf("usage: aiw req %s <id> [--by <actor>] --reason <reason>", command)
	}
	if by == "" {
		current, err := user.Current()
		if err != nil || current.Username == "" {
			return "", "", "", fmt.Errorf("cannot determine current user; provide --by <actor>")
		}
		by = current.Username
	}
	return args[0], by, reason, nil
}

func archiveRequirement(args []string) error {
	id, by, reason, err := parseTerminalArgs(args, "archive")
	if err != nil {
		return err
	}
	meta, err := requirement.Archive(id, by, reason)
	if err != nil {
		return err
	}
	fmt.Printf("requirement %s: %s\n", meta.ID, meta.Status)
	return nil
}

func cancelRequirement(args []string) error {
	id, by, reason, err := parseTerminalArgs(args, "cancel")
	if err != nil {
		return err
	}
	meta, err := requirement.Cancel(id, by, reason)
	if err != nil {
		return err
	}
	fmt.Printf("requirement %s: %s\n", meta.ID, meta.Status)
	return nil
}

func listRequirements(args []string) error {
	if len(args) > 1 {
		return errors.New("usage: aiw req list [--all|--archived|--cancelled]")
	}
	filter := requirement.ListActive
	if len(args) == 1 {
		switch args[0] {
		case "--all":
			filter = requirement.ListAll
		case "--archived":
			filter = requirement.ListArchived
		case "--cancelled":
			filter = requirement.ListCancelled
		default:
			return errors.New("usage: aiw req list [--all|--archived|--cancelled]")
		}
	}
	items, err := requirement.List(filter)
	if err != nil {
		return err
	}
	for _, meta := range items {
		fmt.Printf("%s\t%s\t%s\n", meta.ID, meta.Status, meta.Title)
	}
	return nil
}

func parsePromoteArgs(args []string) (string, string, bool, error) {
	if (len(args) != 3 && len(args) != 4) || args[1] != "--task" || !requirement.ValidID(args[2]) || (len(args) == 4 && args[3] != "--allow-unrelated-dirty") {
		return "", "", false, errors.New("usage: aiw req promote <id> --task <task-id>")
	}
	return args[0], args[2], true, nil
}

func writeRequirementHandoff(meta requirement.Meta, artifacts []requirement.Artifact) error {
	dir := filepath.Join(task.RuntimeTaskDir(meta.Promotion.TaskID), "artifacts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "requirement-handoff.md")
	if fsx.Exists(path) {
		return nil
	}
	var rows []string
	for _, artifact := range artifacts {
		rows = append(rows, fmt.Sprintf("| %s | %s | %s |", artifact.Kind, artifact.Path, artifact.Digest))
	}
	content := "# Issue Handoff\n\n## Source\n- Issue ID: " + meta.ID + "\n- Issue revision: " + fmt.Sprint(meta.Revision) + "\n- Approved by: " + meta.Approval.By + "\n- Approved at: " + meta.Approval.At + "\n\n## Referenced Artifacts\n| Artifact | Path | Digest |\n|---|---|---|\n" + strings.Join(rows, "\n") + "\n\n## Approved Scope\n- Use the approved Issue Plan and referenced artifacts as the authoritative scope.\n\n## Open Decisions Carried Into Engineering\n%% NEEDS_INPUT: Review approved artifacts before finalizing the Feature Design.\n\n## Suggested Next Workflow Action\nComplete docs/features/" + meta.Promotion.TaskID + ".md with decisions and ordered work items, then run aiw wf plan " + meta.Promotion.TaskID + ". Update stable OpenSpec specs only when their requirements change.\n"
	return os.WriteFile(path, []byte(content), 0o644)
}
