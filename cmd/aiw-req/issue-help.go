package main

const requirementUsage = `usage: aiw issue <command> ...

commands:
  chat [id] [--provider NAME] [--model MODEL]
  new <slug> [title]                    Auto-number a new Requirement.
  new --id <id> [title]                 Create an exact ID for compatibility.
  show <id>
  link-parent <child-id> <parent-id>    Record split lineage before approval.
  children <parent-id>                  List direct child Issues.
  capture <id> <artifact> --file <path>
  approve <id> <APPROVED|DEFERRED|REJECTED> [--by <actor>] --reason <reason>
  archive <id> --by <actor> --reason <reason>
  cancel <id> --by <actor> --reason <reason>
  list [--all|--archived|--cancelled]
`

func isIssueHelpFlag(arg string) bool { return arg == "--help" || arg == "-h" }

func issueSubcommandUsage(command string) (string, bool) {
	usages := map[string]string{
		"resume":      "usage: aiw issue resume [issue-id] [--provider NAME] [--model MODEL]\n",
		"new":         "usage: aiw issue new <slug> [title] | new --id <id> [title]\n",
		"show":        "usage: aiw issue show <id>\n",
		"link-parent": "usage: aiw issue link-parent <child-id> <parent-id>\n",
		"children":    "usage: aiw issue children <parent-id>\n",
		"capture":     "usage: aiw issue capture <id> <artifact> --file <path>\n",
		"approve":     "usage: aiw issue approve <id> <APPROVED|DEFERRED|REJECTED> [--by <actor>] --reason <reason>\n",
		"archive":     "usage: aiw issue archive <id> --by <actor> --reason <reason>\n",
		"cancel":      "usage: aiw issue cancel <id> --by <actor> --reason <reason>\n",
		"list":        "usage: aiw issue list [--all|--archived|--cancelled]\n",
	}
	usage, ok := usages[command]
	return usage, ok
}
