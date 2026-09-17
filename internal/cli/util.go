package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/iokit"
)

// loadConfigNotifying loads the project config and writes any deprecation notices
// it recorded to errOut, one per line. Notices are advisory: a load that succeeds
// with notices is not an error.
func loadConfigNotifying(errOut io.Writer, root string) (*config.Config, error) {
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	for _, n := range cfg.Notices() {
		_, _ = fmt.Fprintln(errOut, n)
	}
	return cfg, nil
}

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
	return iokit.AtomicWrite(path, []byte(content), 0600)
}
