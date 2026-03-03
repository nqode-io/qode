// Package dispatch sends AI prompts to available backends.
//
// Resolve() returns the best available dispatcher: the claude CLI when
// present, otherwise the clipboard fallback which copies the prompt and
// instructs the user to paste it into their IDE manually.
package dispatch

import (
	"context"
	"errors"
	"time"
)

const defaultTimeout = 5 * time.Minute

// ErrManualDispatch is returned by the clipboard dispatcher to signal that
// the prompt was copied to the clipboard and must be pasted manually.
var ErrManualDispatch = errors.New("manual dispatch: prompt copied to clipboard")

// Options configures a dispatch call.
type Options struct {
	Timeout    time.Duration // 0 means defaultTimeout
	WorkingDir string        // working directory for the subprocess
}

func (o Options) timeout() time.Duration {
	if o.Timeout > 0 {
		return o.Timeout
	}
	return defaultTimeout
}

// Dispatcher sends a prompt to an AI backend.
type Dispatcher interface {
	// Name returns a human-readable name for the dispatcher.
	Name() string
	// Available reports whether the dispatcher can be used on this machine.
	Available() bool
	// Run sends the prompt and returns the AI output. It returns
	// ErrManualDispatch when the dispatcher is clipboard-only.
	Run(ctx context.Context, prompt string, opts Options) (string, error)
}

// Resolve returns the best available dispatcher.
// It prefers the claude CLI; falls back to clipboard.
func Resolve() Dispatcher {
	if d := newClaudeCLI(); d.Available() {
		return d
	}
	return &clipboardDispatcher{}
}
