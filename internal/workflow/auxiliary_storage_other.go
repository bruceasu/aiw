//go:build !windows

package workflow

import "errors"

func LocalAuxiliaryFreeBytes(path string) (int64, error) {
	return 0, errors.New("verified auxiliary volume accounting is unavailable on this platform")
}
