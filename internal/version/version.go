package version

import "strings"

// Version is replaced by release builds through -ldflags.
var Version = "dev"

// Value returns the normalized version value used by commands and plugins.
func Value() string {
	value := strings.TrimSpace(Version)
	if value == "" {
		return "dev"
	}
	return value
}

// Label returns the user-facing version label used in help and --version.
func Label() string {
	value := Value()
	if value == "dev" || strings.HasPrefix(value, "v") {
		return value
	}
	return "v" + value
}
