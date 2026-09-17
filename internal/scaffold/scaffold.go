// Package scaffold generates agent-specific configuration files for Cursor, Claude Code, Codex, and OpenCode.
package scaffold

import (
	"fmt"
	"io"

	"github.com/nqode/qode/internal/config"
)

// Setup generates agent configs for all enabled agents.
func Setup(out io.Writer, root string, cfg *config.Config) error {
	var generated []string

	if cfg.Agents.Cursor.Enabled {
		if err := SetupCursor(out, root); err != nil {
			return fmt.Errorf("cursor setup: %w", err)
		}
		generated = append(generated, "Cursor")
	}

	if cfg.Agents.ClaudeCode.Enabled {
		if err := SetupClaudeCode(out, root); err != nil {
			return fmt.Errorf("claude code setup: %w", err)
		}
		generated = append(generated, "Claude Code")
	}

	if cfg.Agents.Codex.Enabled {
		if err := SetupCodex(out, root); err != nil {
			return fmt.Errorf("codex setup: %w", err)
		}
		generated = append(generated, "Codex")
	}

	if cfg.Agents.OpenCode.Enabled {
		if err := SetupOpenCode(out, root); err != nil {
			return fmt.Errorf("opencode setup: %w", err)
		}
		generated = append(generated, "OpenCode")
	}

	if len(generated) == 0 {
		_, _ = fmt.Fprintln(out, "No agents enabled. Set agents.cursor/claude_code/codex/opencode.enabled: true in qode.yaml")
		return nil
	}

	_, _ = fmt.Fprintf(out, "Generated agent configs for: %v\n", generated)
	return nil
}
