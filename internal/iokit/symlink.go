package iokit

import (
	"fmt"
	"runtime"
)

// WrapSymlinkError wraps a symlink creation error with a platform-specific hint.
func WrapSymlinkError(err error) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("creating symlink requires Developer Mode on Windows; enable it in Settings → Update & Security → For Developers: %w", err)
	}
	return fmt.Errorf("creating symlink: %w", err)
}
