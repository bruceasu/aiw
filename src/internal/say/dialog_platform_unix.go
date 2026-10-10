//go:build !windows

package say

import (
	"errors"
	"os"
)

func zenityNames() []string { return []string{"zenity"} }

func ensureZenitySession() error {
	if os.Getenv("WAYLAND_DISPLAY") == "" && os.Getenv("DISPLAY") == "" {
		return errors.New("Zenity requires an active Wayland or X11 display; on WSL use WSLg or configure an X server")
	}
	return nil
}

func zenityMissingError() error {
	return errors.New("Zenity is unavailable; install zenity in the Linux or WSL environment")
}
