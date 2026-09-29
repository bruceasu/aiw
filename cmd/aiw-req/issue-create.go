package main

import (
	"aiw/internal/issue"
	"errors"
	"fmt"
	"os"
)

func dispatchIssueNew(args []string) error {
	action, err := parseIssueCreation(args[1:])
	if err != nil {
		return err
	}
	meta, err := createIssueAction(action)
	if err != nil {
		return err
	}
	if sessionID := os.Getenv("AIW_REQUIREMENT_SESSION"); sessionID != "" {
		if _, err := issue.BindConversation(meta.ID, sessionID); err != nil {
			return fmt.Errorf("Requirement %s was created but Session binding failed; do not repeat creation: %w",
				meta.ID, err)
		}
	}
	fmt.Println("created requirement:", meta.ID)
	return nil
}
func createIssueAction(action issuePendingAction) (issue.Meta, error) {
	if action.AutoNumber {
		return issue.CreateNumbered(action.RequirementID, action.Title)
	}
	return issue.Create(action.RequirementID, action.Title)
}

func parseIssueCreation(args []string) (issuePendingAction, error) {
	action := issuePendingAction{Kind: "new", AutoNumber: true}
	if len(args) > 0 && args[0] == "--id" {
		action.AutoNumber = false
		args = args[1:]
	}
	newVar, _ := issueSubcommandUsage("new")
	if len(args) < 1 || len(args) > 2 {

		return action, errors.New(newVar)
	}
	action.RequirementID, action.Title = args[0], args[0]
	if len(args) == 2 {
		action.Title = args[1]
	}
	if (action.AutoNumber && !issue.ValidSlug(action.RequirementID)) || (!action.AutoNumber && !issue.ValidID(action.RequirementID)) {
		return action, errors.New("invalid Requirement slug or explicit ID; " + newVar)
	}
	return action, nil
}
