package cli

import (
	"fmt"
	"os"

	"github.com/nqode/qode/internal/iokit"
)

// resolveRoot returns the effective project root, preferring the --root flag,
// then the current working directory.
func resolveRoot() (string, error) {
	if flagRoot != "" {
		return flagRoot, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}
	return wd, nil
}

// writePromptToFile atomically writes content to path, creating parent dirs as needed.
// On template render error the caller should return before calling this.
func writePromptToFile(path, content string) error {
	return iokit.AtomicWrite(path, []byte(content), 0644)
}
