package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// dryRun is set by --dry-run in any top-level dispatch. When true, run()
// prints the command but does not execute it.
var dryRun bool

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func safeID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func today() string {
	return time.Now().Format("2006-01-02")
}

func run(name string, args ...string) error {
	fmt.Fprintf(os.Stderr, "+ %s %s\n", name, strings.Join(args, " "))
	if dryRun {
		fmt.Fprintln(os.Stderr, "  (dry-run: skipped)")
		return nil
	}
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// gitOutput runs a command and returns its trimmed stdout.
func gitOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func ensureTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}
