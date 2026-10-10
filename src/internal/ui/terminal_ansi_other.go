//go:build !windows

package ui

import (
	"os"
	"strings"
)

func terminalSupportsANSI(_ *os.File) bool {
	term := os.Getenv("TERM")
	for _, family := range []string{"ansi", "xterm", "screen", "tmux", "rxvt", "linux", "vt100", "cygwin"} {
		if term == family || strings.HasPrefix(term, family+"-") {
			return true
		}
	}
	return false
}
