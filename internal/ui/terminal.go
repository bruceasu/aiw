package ui

import (
	"fmt"
	"io"
	"os"
)

type Terminal struct {
	out     io.Writer
	colored bool
}

func NewTerminal(out io.Writer) Terminal {
	return Terminal{out: out, colored: supportsColor(out)}
}

func (t Terminal) Section(title string) {
	if t.colored {
		fmt.Fprintf(t.out, "\n\033[1;36m%s\033[0m\n", title)
		return
	}
	fmt.Fprintf(t.out, "%s:\n", title)
}

func (t Terminal) Field(label, value string) {
	if t.colored {
		fmt.Fprintf(t.out, "  \033[36m%-16s\033[0m %s\n", label, value)
		return
	}
	fmt.Fprintf(t.out, "%s: %s\n", label, value)
}

func (t Terminal) State(label, value string) {
	if t.colored {
		color := "32"
		if value == "stopped" || value == "lease-expired" || value == "lost" || value == "unavailable" {
			color = "31"
		} else if value == "waiting" || value == "prepared" || value == "starting" || value == "pending" {
			color = "33"
		}
		t.Field(label, fmt.Sprintf("\033[%sm%s\033[0m", color, value))
		return
	}
	t.Field(label, value)
}

func (t Terminal) Line(text string) {
	fmt.Fprintln(t.out, text)
}

func supportsColor(out io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
