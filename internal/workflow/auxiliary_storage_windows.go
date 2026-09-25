//go:build windows

package workflow

import (
	"errors"
	"golang.org/x/sys/windows"
)

// LocalAuxiliaryFreeBytes reads the actual target volume without a network
// probe. The caller must already have verified the R1 local-disk boundary.
func LocalAuxiliaryFreeBytes(path string) (int64, error) {
	pointer, err := windows.UTF16PtrFromString(path)
	if err != nil { return 0, err }
	var available uint64
	if err := windows.GetDiskFreeSpaceEx(pointer, &available, nil, nil); err != nil { return 0, err }
	if int64(available) < 0 { return 0, errors.New("volume capacity cannot be represented") }
	return int64(available), nil
}
