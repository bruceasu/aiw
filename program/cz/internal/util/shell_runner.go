package util

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"aiw-cz/internal/cz"
)

func Run(name string, args ...string) error {
	fmt.Fprintf(os.Stderr, "+ %s %s\n", name, strings.Join(args, " "))
	if cz.DryRun {
		fmt.Fprintln(os.Stderr, "  (dry-run: skipped)")
		return nil
	}
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func RunAndWaitForOuput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}
