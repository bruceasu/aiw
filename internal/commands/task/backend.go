package task

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"aiw/internal/gitx"
	"aiw/internal/taskx"
)

type backendMode string

const (
	backendAuto     backendMode = "auto"
	backendOpenSpec backendMode = "openspec"
	backendNative   backendMode = "native"
)

func selectBackend(operation string, args []string) (backendMode, []string, error) {
	mode := backendAuto
	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] != "--backend" {
			remaining = append(remaining, args[i])
			continue
		}
		if i+1 >= len(args) {
			return "", nil, errors.New("missing value for --backend (auto, openspec, or native)")
		}
		switch backendMode(args[i+1]) {
		case backendAuto, backendOpenSpec, backendNative:
			mode = backendMode(args[i+1])
		default:
			return "", nil, fmt.Errorf("unknown backend %q (expected auto, openspec, or native)", args[i+1])
		}
		i++
	}
	if mode == backendNative {
		return mode, remaining, nil
	}
	bin, err := findOpenSpec()
	if err != nil {
		if mode == backendOpenSpec {
			return "", nil, err
		}
		fmt.Fprintf(os.Stderr, "workflow backend: native fallback (%v)\n", err)
		return backendNative, remaining, nil
	}
	if !supportsOpenSpec(operation) {
		message := fmt.Sprintf("OpenSpec has no direct mapping for task %s", operation)
		if mode == backendOpenSpec {
			return "", nil, errors.New(message + "; use --backend native")
		}
		fmt.Fprintf(os.Stderr, "workflow backend: native fallback (%s)\n", message)
		return backendNative, remaining, nil
	}
	fmt.Fprintf(os.Stderr, "workflow backend: openspec (%s)\n", bin)
	return backendOpenSpec, append([]string{bin}, remaining...), nil
}

func findOpenSpec() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("AIW_OPENSPEC_BIN")); configured != "" {
		path, err := exec.LookPath(configured)
		if err == nil {
			probe := exec.Command(path, "--version")
			if probe.Run() == nil {
				return path, nil
			}
		}
		return "", fmt.Errorf("configured OpenSpec executable is not usable: %s", configured)
	}
	candidates := []string{}
	candidates = append(candidates, "openspec")
	if runtime.GOOS == "windows" {
		candidates = append(candidates, "openspec.cmd")
	}
	for _, candidate := range candidates {
		path, err := exec.LookPath(candidate)
		if err != nil {
			continue
		}
		probe := exec.Command(path, "--version")
		if err := probe.Run(); err == nil {
			return path, nil
		}
	}
	return "", errors.New("verified OpenSpec executable not found; install OpenSpec or set AIW_OPENSPEC_BIN")
}

func supportsOpenSpec(operation string) bool {
	return operation == "new" || operation == "archive"
}

func runOpenSpec(bin string, operation string, args []string) error {
	command := []string{}
	switch operation {
	case "new":
		id, allowDirty, err := parseNewArgs(args)
		if err != nil { return err }
		primary, primaryPath, err := gitx.IsPrimaryWorktree(); if err != nil { return err }
		if !primary { return fmt.Errorf("ordinary Tasks must be created from the primary workspace: %s", primaryPath) }
		dirty, err := gitx.IsDirty(); if err != nil { return err }
		if dirty && !allowDirty { return errors.New("working tree has uncommitted changes; commit or clean them, or rerun with --allow-dirty") }
		command = []string{"new", "change", id}
	case "archive":
		if len(args) != 1 {
			return errors.New("OpenSpec archive delegation supports only: archive <change-id>")
		}
		meta, err := taskx.ReadTaskMeta(taskx.ResolveTaskMetaPath(args[0]))
		if err != nil { return err }
		if meta.Status != "DONE" && meta.Status != "CANCELLED" { return fmt.Errorf("task must be DONE or CANCELLED before archive: %s", meta.Status) }
		if resolvedWorkspaceKind(meta) != "primary" && !(meta.Status == "CANCELLED" && meta.Delivery == "discarded") { return errors.New("delegated archive supports only primary or discarded Tasks; clean isolated delivery first") }
		command = []string{"archive", "--yes", args[0]}
	default:
		return fmt.Errorf("unsupported OpenSpec operation: %s", operation)
	}
	cmd := exec.Command(bin, command...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("OpenSpec %s failed: %w", operation, err)
	}
	if operation == "new" {
		id, _, _ := parseNewArgs(args)
		if err := ensureTaskMeta(id); err != nil {
			return fmt.Errorf("create AIW task metadata: %w", err)
		}
	}
	return nil
}
