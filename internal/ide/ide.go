package ide

import (
	"fmt"

	"github.com/nqode/qode/internal/config"
)

// Setup generates IDE configs for all enabled IDEs.
func Setup(root string, cfg *config.Config) error {
	var generated []string

	if cfg.IDE.Cursor.Enabled {
		if err := SetupCursor(root, cfg); err != nil {
			return fmt.Errorf("cursor setup: %w", err)
		}
		generated = append(generated, "Cursor")
	}

	if cfg.IDE.VSCode.Enabled {
		if err := SetupVSCode(root, cfg); err != nil {
			return fmt.Errorf("vscode setup: %w", err)
		}
		generated = append(generated, "VS Code")
	}

	if cfg.IDE.ClaudeCode.Enabled {
		if err := SetupClaudeCode(root, cfg); err != nil {
			return fmt.Errorf("claude code setup: %w", err)
		}
		generated = append(generated, "Claude Code")
	}

	if len(generated) == 0 {
		fmt.Println("No IDEs enabled. Set ide.cursor/vscode/claude_code.enabled: true in qode.yaml")
		return nil
	}

	fmt.Printf("Generated IDE configs for: %v\n", generated)
	return nil
}

// managedStart / managedEnd are markers used to demarcate qode-managed sections
// in existing config files, so they can be updated without overwriting custom content.
const (
	managedStart = "// qode:managed-start"
	managedEnd   = "// qode:managed-end"
)
