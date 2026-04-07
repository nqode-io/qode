package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/prompt"
)

var claudeCommands = []string{
	"qode-plan-refine",
	"qode-plan-spec",
	"qode-review-code",
	"qode-review-security",
	"qode-check",
	"qode-start",
	"qode-ticket-fetch",
	"qode-knowledge-add-context",
	"qode-knowledge-add-branch",
}

// SetupClaudeCode generates Claude Code configuration files.
func SetupClaudeCode(root string) error {
	commandsDir := filepath.Join(root, ".claude", "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return err
	}

	engine, err := prompt.NewEngine(root)
	if err != nil {
		return err
	}

	data := prompt.TemplateData{Project: prompt.TemplateProject{Name: filepath.Base(root)}}

	for _, cmd := range claudeCommands {
		content, err := engine.Render("scaffold/"+cmd+".claude", data)
		if err != nil {
			return fmt.Errorf("render %s: %w", cmd, err)
		}
		if err := os.WriteFile(filepath.Join(commandsDir, cmd+".md"), []byte(content), 0644); err != nil {
			return err
		}
	}

	fmt.Printf("  Claude Code: %d slash commands\n", len(claudeCommands))
	return nil
}
