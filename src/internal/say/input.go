package say

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// ReadInput accepts either one text argument or UTF-8 text from stdin.
func ReadInput(args []string, stdin io.Reader) (string, error) {
	if len(args) > 1 {
		return "", errors.New("provide one text argument or stdin")
	}
	if len(args) == 1 {
		if args[0] == "" {
			return "", errors.New("input text is empty")
		}
		if !utf8.ValidString(args[0]) {
			return "", errors.New("input text is not valid UTF-8")
		}
		return args[0], nil
	}
	if stdin == nil {
		return "", errors.New("stdin is unavailable")
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	if !utf8.Valid(data) {
		return "", errors.New("stdin is not valid UTF-8")
	}
	text := string(data)
	if strings.TrimSpace(text) == "" {
		return "", errors.New("input text is empty")
	}
	return text, nil
}

// ValidateInputText rejects empty or non-UTF-8 input from external sources.
func ValidateInputText(text string) (string, error) {
	if !utf8.ValidString(text) {
		return "", errors.New("input text is not valid UTF-8")
	}
	if strings.TrimSpace(text) == "" {
		return "", errors.New("input text is empty")
	}
	return text, nil
}
