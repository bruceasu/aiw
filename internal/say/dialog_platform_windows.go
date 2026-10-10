//go:build windows

package say

import "errors"

func zenityNames() []string { return []string{"zenity.exe", "zenity"} }

func ensureZenitySession() error { return nil }

func zenityMissingError() error {
	return errors.New("native Windows Zenity is unavailable; install Zenity and add zenity.exe to PATH")
}
