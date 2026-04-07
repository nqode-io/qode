package scaffold

import (
	"fmt"
	"io"

	"github.com/nqode/qode/internal/config"
)

// Setup generates IDE configs for all enabled IDEs.
func Setup(out io.Writer, root string, cfg *config.Config) error {
	var generated []string

	if cfg.IDE.Cursor.Enabled {
		if err := SetupCursor(out, root); err != nil {
			return fmt.Errorf("cursor setup: %w", err)
		}
		generated = append(generated, "Cursor")
	}

	if cfg.IDE.ClaudeCode.Enabled {
		if err := SetupClaudeCode(out, root); err != nil {
			return fmt.Errorf("claude code setup: %w", err)
		}
		generated = append(generated, "Claude Code")
	}

	if len(generated) == 0 {
		_, _ = fmt.Fprintln(out, "No IDEs enabled. Set ide.cursor/claude_code.enabled: true in qode.yaml")
		return nil
	}

	_, _ = fmt.Fprintf(out, "Generated IDE configs for: %v\n", generated)
	return nil
}
