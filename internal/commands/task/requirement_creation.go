package task

import (
	"errors"

	"aiw/internal/requirement"
)

const requirementNewUsage = "usage: aiw requirement new <slug> [title] | new --id <id> [title]"

func parseRequirementCreation(args []string) (requirementPendingAction, error) {
	action := requirementPendingAction{Kind: "new", AutoNumber: true}
	if len(args) > 0 && args[0] == "--id" {
		action.AutoNumber = false
		args = args[1:]
	}
	if len(args) < 1 || len(args) > 2 { return action, errors.New(requirementNewUsage) }
	action.RequirementID, action.Title = args[0], args[0]
	if len(args) == 2 { action.Title = args[1] }
	if (action.AutoNumber && !requirement.ValidSlug(action.RequirementID)) || (!action.AutoNumber && !requirement.ValidID(action.RequirementID)) {
		return action, errors.New("invalid Requirement slug or explicit ID; " + requirementNewUsage)
	}
	return action, nil
}

func createRequirementAction(action requirementPendingAction) (requirement.Meta, error) {
	if action.AutoNumber { return requirement.CreateNumbered(action.RequirementID, action.Title) }
	return requirement.Create(action.RequirementID, action.Title)
}
