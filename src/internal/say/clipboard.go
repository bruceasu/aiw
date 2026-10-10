package say

import (
	"context"
	"errors"
)

// Clipboard reads and writes plain text without retaining clipboard contents.
type Clipboard interface {
	Read(context.Context) (string, error)
	Write(context.Context, string) error
}

var ErrClipboardUnavailable = errors.New("clipboard access is unavailable on this platform")
