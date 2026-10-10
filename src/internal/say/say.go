// Package say contains the provider-neutral translation request and boundary.
package say

import "context"

// Request contains trusted translation options separately from untrusted text.
type Request struct {
	Source    string
	Target    string
	Mode      string
	Style     string
	Polite    string
	Simple    bool
	Profanity string
	Text      string
	Model     string
}

// Provider translates one complete request or returns an error.
type Provider interface {
	Translate(context.Context, Request) (string, error)
}
