package gitx

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Run(name string, args ...string) error {
	fmt.Fprintf(os.Stderr, "+ %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func HasRemote(name string) bool {
	cmd := exec.Command("git", "remote", "get-url", name)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func RefExists(ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func DetectBaseBranch() (string, error) {
	for _, candidate := range []string{"origin/main", "origin/master", "main", "master"} {
		if RefExists(candidate) {
			return candidate, nil
		}
	}
	return "", errors.New("cannot detect base branch; pass one explicitly, e.g.: aiw wt <task-id> main")
}
