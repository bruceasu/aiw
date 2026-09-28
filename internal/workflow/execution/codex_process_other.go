//go:build !linux && !windows

package execution

import "errors"

func managedProcessToken(int, string) (string, error) { return "", errors.New("durable Codex process identity is unsupported on this platform") }
func managedProcessState(int, string, string) (bool, bool, error) { return false, false, errors.New("durable Codex process identity is unsupported on this platform") }
