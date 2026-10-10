package main

import (
	"aiw/internal/issue"
	"errors"
	"encoding/json"
	"fmt"
	"os"
)

func showIssue(args []string) error {
	structured := len(args) == 3 && args[2] == "--json"
	if len(args) != 2 && !structured {
		usage, _ := issueSubcommandUsage("show")
		return errors.New(usage)
	}
	meta, err := issue.Read(args[1])
	if err != nil {
		return err
	}
	if structured {
		if _, err := issue.ArtifactSnapshot(meta.ID); err != nil { return err }
		return json.NewEncoder(os.Stdout).Encode(struct {
			ID string `json:"id"`
			Status string `json:"status"`
			ApprovalStatus string `json:"approval_status"`
		}{meta.ID, meta.Status, meta.Approval.Status})
	}
	fmt.Printf("%s\t%s\tapproval=%s\tpromotion=%s\ttask=%s\tparent=%s\n", meta.ID, meta.Status, meta.Approval.Status, meta.Promotion.Status, meta.Promotion.TaskID, meta.ParentID)
	return nil
}
