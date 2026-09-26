package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"aiw/internal/workflow"
	"aiw/internal/workflow/execution"
)

func readAuxiliaryJSON(path string, target any) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 64*1024+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 64*1024 {
		return nil, errors.New("auxiliary configuration exceeds 64 KiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, errors.New("auxiliary configuration contains trailing data")
	}
	return data, nil
}

// A JSON request cannot by itself impersonate an interactive human reviewer.
func RunKnowledgeCommand(args []string) error {
	if len(args) < 2 { return errors.New("usage: workflow knowledge <show|review|import> <task> [workspace request.json]") }
	store := workflow.NewStore("")
	execution.ConfigureProductionAuxiliary(store)
	store.AuxiliaryServices.StartHost = func(workflow.TaskID) error { return nil }
	id := workflow.TaskID(args[1])
	state, err := store.Load(id); if err != nil { return err }
	if state.Protocol == nil || state.Protocol.Auxiliary == nil { return errors.New("Task has no durable knowledge sources") }
	actor := "current project operator (identity unverified)"
	if u, err := user.Current(); err == nil { actor = u.Username+" (identity unverified)" }
	var preferences struct { Owner string `json:"owner"` }
	if _, err := readAuxiliaryJSON(filepath.Join(store.Root, "knowledge-review.json"), &preferences); err != nil && !errors.Is(err, os.ErrNotExist) { return err }
	preferred := preferences.Owner
	if preferred == "" { preferred = actor }
	if args[0] == "show" {
		if len(args) != 2 { return errors.New("usage: workflow knowledge show <task>") }
		return json.NewEncoder(os.Stdout).Encode(struct { Knowledge *workflow.TaskKnowledge `json:"knowledge"`; PreferredReviewer string `json:"preferred_reviewer"`; Note string `json:"note"` }{state.Protocol.Auxiliary.Knowledge, preferred, "Ordinary review is a non-blocking todo. Only unsafe requirement, compatibility, or permission conflicts need a decision. Confirmation never updates formal documents."})
	}
	if len(args) != 4 || !filepath.IsAbs(args[2]) { return errors.New("review/import requires an absolute workspace and an exact JSON request file") }
	terminal, err := os.Stdin.Stat()
	if err != nil || terminal.Mode()&os.ModeCharDevice == 0 { return errors.New("knowledge review/import requires an interactive human terminal") }
	var request workflow.KnowledgeReviewRequest
	var imported struct { SourcePath string `json:"source_path"`; Content workflow.KnowledgeContent `json:"content"` }
	switch args[0] {
	case "review":
		if _, err := readAuxiliaryJSON(args[3], &request); err != nil { return err }
	case "import":
		if _, err := readAuxiliaryJSON(args[3], &imported); err != nil { return err }
	default: return errors.New("unknown knowledge operation")
	}
	fmt.Fprintf(os.Stderr, "Preferred reviewer: %s\nRecorded operator: %s\n", preferred, actor)
	if args[0] == "review" {
		if err := json.NewEncoder(os.Stderr).Encode(request); err != nil { return err }
		found := false
		if state.Protocol.Auxiliary.Knowledge != nil {
			for _, v := range state.Protocol.Auxiliary.Knowledge.Versions { if v.Version == request.Version { found = true; if err := json.NewEncoder(os.Stderr).Encode(v); err != nil { return err } } }
		}
		if !found { return errors.New("reviewed knowledge version is unavailable") }
	} else { if err := json.NewEncoder(os.Stderr).Encode(imported); err != nil { return err } }
	fmt.Fprint(os.Stderr, "Type review to record this human action: ")
	answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil || strings.TrimSpace(answer) != "review" { return errors.New("human knowledge action cancelled") }
	var result workflow.KnowledgeVersion
	if args[0] == "review" {
		result, err = store.ReviewKnowledge(id, args[2], request, workflow.KnowledgeReviewer{Actor: actor, IdentityVerified: false})
	} else {
		source := workflow.ReadInputSource(args[2], workflow.InputSource{Kind: "original-human-text", Path: imported.SourcePath, Required: true})
		result, err = store.ImportHumanKnowledge(id, source, imported.Content)
	}
	if err != nil { return err }
	return json.NewEncoder(os.Stdout).Encode(result)
}
