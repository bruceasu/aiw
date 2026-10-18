package main

const requirementUsage = `usage: aiw issue <command> ...

commands:
  chat [id] [--provider NAME] [--model MODEL]
  new <slug> [title]                    Create an ISSUE-00001 record (slug is not its ID).
  new --id <id> [title]                 Create an exact ID for compatibility.
  show <id> [--json]                   Resolve a canonical ID or unique REQ number.
  link-parent <child-id> <parent-id>    Record split lineage before approval.
  children <parent-id>                  List direct child Issues.
  capture <id> <artifact> --file <path>
  approve <id> <APPROVED|DEFERRED|REJECTED> [--by <actor>] --reason <reason>
  promote <id>                         Create an FD for an approved Issue.
  archive <id> --by <actor> --reason <reason>
  cancel <id> --by <actor> --reason <reason>
  list [--all|--archived|--cancelled]

IDs are case-insensitive. Ambiguous REQ numbers are rejected.
Use issue-plan (or requirement-plan) for the Plan artifact.
Successful capture consumes its unchanged source in the project draft directories.
External sources and other drafts are kept. Cleanup warnings go to stderr.
`

func isIssueHelpFlag(arg string) bool { return arg == "--help" || arg == "-h" }

func issueSubcommandUsage(command string) (string, bool) {
	usages := map[string]string{
		"resume":      "usage: aiw issue resume [issue-id] [--provider NAME] [--model MODEL]\n",
		"new":         "usage: aiw issue new <slug> [title] | new --id <id> [title]\n",
		"show":        "usage: aiw issue show <id> [--json]\n",
		"link-parent": "usage: aiw issue link-parent <child-id> <parent-id>\n",
		"children":    "usage: aiw issue children <parent-id>\n",
		"capture":     "usage: aiw issue capture <id> <artifact> --file <path>\n\nAfter saving the artifact and metadata, remove only this unchanged source in\n.ai/issues/drafts/ or .ai/requirements/drafts/. Keep external sources and other drafts.\nCleanup failure keeps capture successful and reports the retained or recovery path on stderr.\n",
		"approve":     "usage: aiw issue approve <id> <APPROVED|DEFERRED|REJECTED> [--by <actor>] --reason <reason>\n",
		"promote":     "usage: aiw issue promote <id>\n",
		"archive":     "usage: aiw issue archive <id> --by <actor> --reason <reason>\n",
		"cancel":      "usage: aiw issue cancel <id> --by <actor> --reason <reason>\n",
		"list":        "usage: aiw issue list [--all|--archived|--cancelled]\n",
	}
	usage, ok := usages[command]
	return usage, ok
}
