// Package dispatch sends AI prompts to available backends.
//
// RunInteractive() runs the claude CLI interactively when present,
// or returns an error if the binary is not found.
package dispatch

import (
	"time"
)

const defaultTimeout = 5 * time.Minute

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
