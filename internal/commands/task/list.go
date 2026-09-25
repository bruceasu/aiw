package task

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"aiw/internal/ui"
)

const listUsage = "usage: aiw list [--all]"

func listTasks(args ...string) error {
	includeArchived := false
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Println(listUsage + "\n\nShow active tasks. --all includes archived tasks and an ACTIVE/ARCHIVED column.")
		return nil
	}
	if len(args) > 0 {
		if len(args) != 1 || args[0] != "--all" {
			return errors.New(listUsage)
		}
		includeArchived = true
	}
	rows, err := collectTaskListRows(includeArchived)
	terminal := ui.NewTerminal(os.Stdout)
	return errors.Join(err, renderTaskList(os.Stdout, rows, includeArchived, terminal.Interactive(), terminal.ColorEnabled()))
}

func renderTaskList(out io.Writer, rows []taskListRow, includeArchived, interactive, colored bool) error {
	if len(rows) == 0 {
		return nil
	}
	headers := []string{"TASK", "STATUS", "PATH"}
	if includeArchived {
		headers = []string{"TASK", "STATUS", "ARCHIVE", "PATH"}
	}
	widths := make([]int, len(headers)-1)
	cells := make([][]string, 0, len(rows)+1)
	if interactive {
		cells = append(cells, headers)
	}
	for _, row := range rows {
		values := []string{row.ID, row.Status}
		if includeArchived {
			archive := "ACTIVE"
			if row.Archived {
				archive = "ARCHIVED"
			}
			values = append(values, archive)
		}
		cells = append(cells, append(values, row.Path))
	}
	for _, values := range cells {
		for column := range widths {
			// Task IDs and Workflow/archive labels are ASCII. The unrestricted
			// path is last and needs no padding. Measure before adding ANSI.
			if width := utf8.RuneCountInString(values[column]); width > widths[column] {
				widths[column] = width
			}
		}
	}
	for index, values := range cells {
		var line strings.Builder
		for column, value := range values {
			code := ""
			if colored && interactive && index > 0 {
				if column == 1 {
					code = taskListStatusColor(value)
				} else if includeArchived && column == 2 && value == "ARCHIVED" {
					code = "90"
				}
			}
			if code != "" {
				fmt.Fprintf(&line, "\033[%sm%s\033[0m", code, value)
			} else {
				line.WriteString(value)
			}
			if column < len(widths) {
				line.WriteString(strings.Repeat(" ", widths[column]-utf8.RuneCountInString(value)+2))
			}
		}
		if _, err := fmt.Fprintln(out, line.String()); err != nil {
			return err
		}
	}
	return nil
}

func taskListStatusColor(status string) string {
	switch status {
	case "DONE":
		return "32"
	case "DRAFT", "CANCELLED":
		return "90"
	case "READY", "RUNNING", "EXECUTING", "IN_PROGRESS":
		return "36"
	case "WAITING", "NEEDS_DECISION", "AWAITING_AUTHORIZATION", "AWAITING_VERIFICATION", "AWAITING_REVIEW", "AWAITING_MERGE":
		return "33"
	case "FAILED", "BLOCKED", "RUNTIME_ERROR":
		return "31"
	default:
		return ""
	}
}
